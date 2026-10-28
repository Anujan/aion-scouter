package ncsoft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	siteBase   = "https://aion2.plaync.com"
	searchBase = "https://api-search.plaync.com/aion2global/search/v2/character"
	lang       = "en-US"
	userAgent  = "aion-scouter/0.1 (+https://github.com/anujan/aion-scouter)"

	maxCacheEntries = 20000
)

// Regions are the Global shards the official site exposes.
var Regions = map[string]string{
	"eu":  "Europe",
	"nae": "NA East",
	"naw": "NA West",
	"la":  "Latin America",
	"as":  "Asia / Japan",
}

// ErrNotFound is returned when NCSOFT answers 404.
var ErrNotFound = errors.New("not found upstream")

// Client is a polite, cached client for the official site's JSON API.
// It spaces requests by MinGap and caches every successful response for TTL.
type Client struct {
	HTTP   *http.Client
	MinGap time.Duration
	TTL    time.Duration

	mu    sync.Mutex
	next  time.Time
	cache map[string]cacheEntry
}

type cacheEntry struct {
	body []byte
	exp  time.Time
}

func NewClient() *Client {
	return &Client{
		HTTP:   &http.Client{Timeout: 20 * time.Second},
		MinGap: 250 * time.Millisecond,
		TTL:    10 * time.Minute,
		cache:  map[string]cacheEntry{},
	}
}

func (c *Client) get(ctx context.Context, rawURL string, out any) error {
	c.mu.Lock()
	if e, ok := c.cache[rawURL]; ok && time.Now().Before(e.exp) {
		c.mu.Unlock()
		return json.Unmarshal(e.body, out)
	}
	wait := time.Until(c.next)
	if wait < 0 {
		wait = 0
	}
	c.next = time.Now().Add(wait + c.MinGap)
	c.mu.Unlock()

	if wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", siteBase+"/")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("ncsoft: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("ncsoft: read: %w", err)
	}
	if res.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("ncsoft: upstream answered %d", res.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("ncsoft: decode: %w", err)
	}
	c.mu.Lock()
	if len(c.cache) >= maxCacheEntries {
		now := time.Now()
		for k, e := range c.cache {
			if now.After(e.exp) {
				delete(c.cache, k)
			}
		}
	}
	if len(c.cache) < maxCacheEntries {
		c.cache[rawURL] = cacheEntry{body: body, exp: time.Now().Add(c.TTL)}
	}
	c.mu.Unlock()
	return nil
}

func siteURL(path string, params url.Values) string {
	params.Set("lang", lang)
	return siteBase + path + "?" + params.Encode()
}

// Ref identifies one character.
type Ref struct {
	Region      string `json:"region"`
	ServerID    int    `json:"serverId"`
	CharacterID string `json:"characterId"`
}

func (r Ref) params() url.Values {
	return url.Values{
		"region":      {r.Region},
		"serverId":    {strconv.Itoa(r.ServerID)},
		"characterId": {r.CharacterID},
	}
}

var highlight = regexp.MustCompile(`<[^>]*>`)

// Search finds characters by name in one region. The search API highlights the
// hit with <strong>; that markup is stripped and the IDs are URL-decoded.
func (c *Client) Search(ctx context.Context, region, name string, page int) (*SearchResponse, error) {
	if _, ok := Regions[region]; !ok {
		return nil, fmt.Errorf("unknown region %q", region)
	}
	q := url.Values{
		"keyword":    {name},
		"page":       {strconv.Itoa(max(page, 1))},
		"size":       {"40"},
		"region":     {region},
		"localeInfo": {lang},
	}
	var out SearchResponse
	if err := c.get(ctx, searchBase+"?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	for i := range out.List {
		r := &out.List[i]
		r.Name = highlight.ReplaceAllString(r.Name, "")
		if id, err := url.QueryUnescape(r.CharacterID); err == nil {
			r.CharacterID = id
		}
	}
	return &out, nil
}

func (c *Client) Info(ctx context.Context, ref Ref) (*InfoResponse, error) {
	var out InfoResponse
	return &out, c.get(ctx, siteURL("/api/character/info", ref.params()), &out)
}

func (c *Client) Equipment(ctx context.Context, ref Ref) (*EquipmentResponse, error) {
	var out EquipmentResponse
	return &out, c.get(ctx, siteURL("/api/character/equipment", ref.params()), &out)
}

func (c *Client) Item(ctx context.Context, ref Ref, slot EquipmentSlot) (*ItemResponse, error) {
	p := ref.params()
	p.Set("id", strconv.Itoa(slot.ID))
	p.Set("enchantLevel", strconv.Itoa(slot.EnchantLevel))
	p.Set("slotPos", strconv.Itoa(slot.SlotPos))
	var out ItemResponse
	return &out, c.get(ctx, siteURL("/api/character/equipment/item", p), &out)
}

func (c *Client) Daevanion(ctx context.Context, ref Ref, boardID int) (*DaevanionResponse, error) {
	p := ref.params()
	p.Set("boardId", strconv.Itoa(boardID))
	var out DaevanionResponse
	return &out, c.get(ctx, siteURL("/api/character/daevanion/detail", p), &out)
}

// ClassForPCID maps a search result's pcId (class × gender × race) to the
// class name. IDs come in blocks of four, starting at 5 with Gladiator.
func ClassForPCID(pcID int) string {
	classes := []string{"Gladiator", "Templar", "Ranger", "Assassin", "Spiritmaster", "Sorcerer", "Cleric", "Chanter"}
	i := (pcID - 5) / 4
	if pcID < 5 || i >= len(classes) {
		return ""
	}
	return classes[i]
}

func (c *Client) Servers(ctx context.Context, region string) ([]Server, error) {
	var out struct {
		ServerList []Server `json:"serverList"`
	}
	err := c.get(ctx, siteBase+"/en-us/api/gameinfo/servers?"+url.Values{"lang": {lang}, "region": {region}}.Encode(), &out)
	return out.ServerList, err
}
