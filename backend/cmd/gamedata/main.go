// Command gamedata imports static game data for one or more classes:
//
//   - per-level skill tooltips -> internal/gamedata/data/global/skills/<class>.json
//   - Daevanion boards (grid, costs, node stats) -> .../boards/<class>.json
//
// Source: metabot.gg's AION 2 pages, which publish values read from the Global
// client. Skill tooltips are stored as raw text and parsed at load time, so
// parser fixes never need a re-import.
//
//	go run ./cmd/gamedata -classes spiritmaster,chanter
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
)

const base = "https://metabot.gg/en/aion-2"

func main() {
	classes := flag.String("classes", "spiritmaster,chanter", "comma-separated class slugs")
	out := flag.String("out", "internal/gamedata/data/global", "output directory")
	flag.Parse()

	for _, class := range strings.Split(*classes, ",") {
		class = strings.TrimSpace(class)
		skills, err := importClass(class)
		if err != nil {
			log.Fatalf("%s skills: %v", class, err)
		}
		write(filepath.Join(*out, "skills", class+".json"), skills)
		log.Printf("%s: %d skills", class, len(skills.Skills))

		boards, err := importBoards(class)
		if err != nil {
			log.Fatalf("%s boards: %v", class, err)
		}
		write(filepath.Join(*out, "boards", class+".json"), boards)
		log.Printf("%s: %d boards", class, len(boards.Boards))
	}
}

func write(path string, v any) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal(err)
	}
	b, _ := json.MarshalIndent(v, "", " ")
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
}

func importBoards(class string) (*gamedata.BoardFile, error) {
	page, err := fetch(base + "/daevanion/" + class)
	if err != nil {
		return nil, err
	}
	payload := flightPayload(page)
	i := strings.Index(payload, `{"boards":[`)
	if i < 0 {
		return nil, errors.New("no board data in page")
	}
	file := &gamedata.BoardFile{
		Class:     class,
		Source:    "metabot.gg Daevanion planner (Global client values)",
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := json.NewDecoder(strings.NewReader(payload[i:])).Decode(file); err != nil {
		return nil, fmt.Errorf("decode boards: %w", err)
	}
	return file, nil
}

func flightPayload(page string) string {
	var flight strings.Builder
	for _, m := range flightChunk.FindAllStringSubmatch(page, -1) {
		var s string
		if err := json.Unmarshal([]byte(`"`+m[1]+`"`), &s); err == nil {
			flight.WriteString(s)
		}
	}
	return flight.String()
}

var skillLink = regexp.MustCompile(`href="/en/aion-2/skills/([a-z0-9-]+)"`)

func importClass(class string) (*gamedata.SkillFile, error) {
	page, err := fetch(base + "/classes/" + class)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var slugs []string
	for _, m := range skillLink.FindAllStringSubmatch(page, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			slugs = append(slugs, m[1])
		}
	}
	if len(slugs) == 0 {
		return nil, errors.New("no skills found on class page")
	}
	file := &gamedata.SkillFile{
		Class:     class,
		Source:    "metabot.gg (Global client tooltip text)",
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}
	for _, slug := range slugs {
		s, err := importSkill(slug)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", slug, err)
		}
		file.Skills = append(file.Skills, *s)
		time.Sleep(400 * time.Millisecond)
	}
	sort.Slice(file.Skills, func(i, j int) bool { return file.Skills[i].ID < file.Skills[j].ID })
	return file, nil
}

var (
	flightChunk = regexp.MustCompile(`self\.__next_f\.push\(\[1,"((?:[^"\\]|\\.)*)"\]\)`)
	iconID      = regexp.MustCompile(`aion2/skills/(\d{8})\.webp`)
	kindLine    = regexp.MustCompile(`(?i)\b(active|passive|stigma) skill · unlocks at level (\d+) · max level (\d+)`)
	titleTag    = regexp.MustCompile(`<title>([^<]+?)(?: – | \| )`)
	childText   = regexp.MustCompile(`"children":"((?:[^"\\]|\\.)*)"`)
	specLevel   = regexp.MustCompile(`^requires skill level (\d+)$`)
	classSuffix = regexp.MustCompile(` \([A-Z][a-z]+\)$`)
)

func importSkill(slug string) (*gamedata.RawSkill, error) {
	page, err := fetch(base + "/skills/" + slug)
	if err != nil {
		return nil, err
	}
	payload := flightPayload(page)

	s := &gamedata.RawSkill{Slug: slug}
	if m := titleTag.FindStringSubmatch(page); m != nil {
		s.Name = classSuffix.ReplaceAllString(htmlUnescape(m[1]), "")
	}
	if m := iconID.FindStringSubmatch(page); m != nil {
		s.ID, _ = strconv.Atoi(m[1])
	}
	if m := kindLine.FindStringSubmatch(payload); m != nil {
		s.Category = strings.ToLower(m[1])
		s.UnlockLevel, _ = strconv.Atoi(m[2])
		s.MaxLevel, _ = strconv.Atoi(m[3])
	}

	i := strings.Index(payload, `{"levels":[`)
	if i < 0 {
		return nil, errors.New("no level table in page")
	}
	var lv struct {
		Levels []gamedata.RawLevel `json:"levels"`
	}
	if err := json.NewDecoder(strings.NewReader(payload[i:])).Decode(&lv); err != nil {
		return nil, fmt.Errorf("decode levels: %w", err)
	}
	s.Levels = lv.Levels

	// Specializations: a run of "requires skill level N" / "<effect>" pairs
	// inside the section with id "specializations".
	if j := strings.Index(payload, `"id":"specializations"`); j >= 0 {
		seg := payload[j:min(len(payload), j+20000)]
		if k := strings.Index(seg, `"id":"levels"`); k > 0 {
			seg = seg[:k]
		}
		var texts []string
		for _, m := range childText.FindAllStringSubmatch(seg, -1) {
			var t string
			if json.Unmarshal([]byte(`"`+m[1]+`"`), &t) == nil {
				texts = append(texts, t)
			}
		}
		for n := 0; n+1 < len(texts); n++ {
			if m := specLevel.FindStringSubmatch(texts[n]); m != nil {
				lvl, _ := strconv.Atoi(m[1])
				s.Specializations = append(s.Specializations, gamedata.RawSpecialization{
					Index: len(s.Specializations) + 1, MinSkillLevel: lvl, Text: texts[n+1],
				})
			}
		}
	}
	if s.Name == "" {
		return nil, errors.New("missing name")
	}
	if s.ID == 0 {
		// A few pages carry no icon; the loader falls back to matching by name.
		log.Printf("warning: %s has no skill id", slug)
	}
	return s, nil
}

func htmlUnescape(s string) string {
	return strings.NewReplacer("&#x27;", "'", "&amp;", "&", "&quot;", `"`, "&#39;", "'").Replace(s)
}

func fetch(url string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "aion-scouter-gamedata/0.1 (+https://github.com/anujan/aion-scouter)")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %d", url, res.StatusCode)
	}
	b, err := io.ReadAll(res.Body)
	return string(b), err
}
