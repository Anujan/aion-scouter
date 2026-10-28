// Package api is the HTTP API the Svelte frontend talks to. All game logic
// (stat rebuild, simulation, efficiency) runs here; responses are ready to
// render.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anujan/aion-scouter/backend/internal/engine"
	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
	"github.com/anujan/aion-scouter/backend/internal/stats"
)

// Server wires game data and a character source to HTTP handlers.
type Server struct {
	Data   *gamedata.Region
	Source *Source
	// StaticDir, when set, serves the built frontend with SPA fallback.
	StaticDir string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/meta", s.meta)
	mux.HandleFunc("GET /api/search", s.search)
	mux.HandleFunc("GET /api/character/{region}/{serverId}/{characterId}", s.character)
	mux.HandleFunc("POST /api/simulate", s.simulate)
	if s.StaticDir != "" {
		mux.Handle("/", spa(s.StaticDir))
	}
	return logRequests(cors(mux))
}

type metaResponse struct {
	Regions   []regionView       `json:"regions"`
	Targets   []gamedata.Target  `json:"targets"`
	Classes   []ClassView        `json:"classes"`
	Constants gamedata.Constants `json:"constants"`
	Scenario  engine.Scenario    `json:"defaultScenario"`
	Overrides []overrideView     `json:"overrideStats"`
}

type regionView struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type overrideView struct {
	Key   stats.Key `json:"key"`
	Label string    `json:"label"`
	Pct   bool      `json:"pct"`
}

// overrideKeys are the stats a player can type in from their in-game stat
// window to correct the rebuild.
var overrideKeys = []stats.Key{
	stats.Critical, stats.CritDmgBoost, stats.PvEDamageBoost, stats.DamageBoost, stats.BossDamageBoost,
	stats.WeaponDmgBoost, stats.AttackIncrease, stats.DoubleChance, stats.PerfectChance,
	stats.MultiHitChance, stats.CombatSpeed, stats.CooldownRed, stats.Penetration,
}

func (s *Server) meta(w http.ResponseWriter, r *http.Request) {
	m := metaResponse{Targets: s.Data.Targets, Constants: s.Data.Constants, Scenario: engine.DefaultScenario()}
	for _, id := range []string{"eu", "nae", "naw", "la", "as"} {
		m.Regions = append(m.Regions, regionView{id, ncsoft.Regions[id]})
	}
	m.Regions = append(m.Regions, regionView{DemoRegion, "Demo characters"})
	for _, slug := range []string{"spiritmaster", "chanter"} {
		if c := s.Data.Classes[slug]; c != nil {
			m.Classes = append(m.Classes, ClassView{Slug: slug, Supported: engine.Supported(slug), Status: c.Model.Status, Assumptions: c.Model.Assumptions})
		}
	}
	for _, k := range overrideKeys {
		m.Overrides = append(m.Overrides, overrideView{k, stats.Label(k), stats.IsPct(k)})
	}
	writeJSON(w, http.StatusOK, m)
}

type searchRow struct {
	ncsoft.SearchResult
	ClassName string `json:"className"`
	Supported bool   `json:"supported"`
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	region, name := q.Get("region"), strings.TrimSpace(q.Get("name"))
	if region != DemoRegion && len(name) < 2 {
		writeError(w, http.StatusBadRequest, "name must be at least 2 characters")
		return
	}
	page, _ := strconv.Atoi(q.Get("page"))
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	res, err := s.Source.Search(ctx, region, name, page)
	if err != nil {
		writeUpstream(w, err)
		return
	}
	rows := make([]searchRow, 0, len(res.List))
	for _, it := range res.List {
		cn := ncsoft.ClassForPCID(it.PCID)
		_, known := s.Data.ClassForName(cn)
		rows = append(rows, searchRow{it, cn, known && engine.Supported(strings.ToLower(cn))})
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": rows, "pagination": res.Pagination})
}

// load fetches a character and rebuilds its stat sheet.
func (s *Server) load(ctx context.Context, ref ncsoft.Ref) (*ncsoft.Snapshot, *gamedata.Class, *stats.Sheet, stats.Report, error) {
	snap, err := s.Source.Snapshot(ctx, ref)
	if err != nil {
		return nil, nil, nil, stats.Report{}, err
	}
	class, ok := s.Data.ClassForName(snap.Info.Profile.ClassName)
	if !ok {
		return snap, nil, stats.NewSheet(), stats.Report{}, nil
	}
	sheet, rep := stats.Rebuild(snap, class)
	return snap, class, sheet, rep, nil
}

func refFrom(region, server, character string) (ncsoft.Ref, error) {
	id, err := strconv.Atoi(server)
	if err != nil {
		return ncsoft.Ref{}, errors.New("serverId must be a number")
	}
	if _, ok := ncsoft.Regions[region]; !ok && region != DemoRegion {
		return ncsoft.Ref{}, fmt.Errorf("unknown region %q", region)
	}
	if character == "" {
		return ncsoft.Ref{}, errors.New("characterId is required")
	}
	return ncsoft.Ref{Region: region, ServerID: id, CharacterID: character}, nil
}

func (s *Server) character(w http.ResponseWriter, r *http.Request) {
	ref, err := refFrom(r.PathValue("region"), r.PathValue("serverId"), r.PathValue("characterId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	snap, class, sheet, rep, err := s.load(ctx, ref)
	if err != nil {
		writeUpstream(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildView(snap, class, sheet, rep))
}

type simulateRequest struct {
	Region      string                `json:"region"`
	ServerID    int                   `json:"serverId"`
	CharacterID string                `json:"characterId"`
	Scenario    *engine.Scenario      `json:"scenario"`
	Overrides   map[stats.Key]float64 `json:"overrides"`
	Efficiency  bool                  `json:"efficiency"`
}

type simulateResponse struct {
	Result     *engine.Result        `json:"result"`
	Efficiency []engine.StatValue    `json:"efficiency,omitempty"`
	Scenario   engine.Scenario       `json:"scenario"`
	Overrides  map[stats.Key]float64 `json:"overrides"`
}

func (s *Server) simulate(w http.ResponseWriter, r *http.Request) {
	var req simulateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	ref, err := refFrom(req.Region, strconv.Itoa(req.ServerID), req.CharacterID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	snap, class, sheet, rep, err := s.load(ctx, ref)
	if err != nil {
		writeUpstream(w, err)
		return
	}
	if class == nil || !engine.Supported(class.Slug) {
		writeError(w, http.StatusUnprocessableEntity, snap.Info.Profile.ClassName+" is not supported yet")
		return
	}
	for k, v := range req.Overrides {
		if _, ok := stats.Registry[k]; ok {
			sheet.SetFinal(k, v, "Your stat window")
		}
	}
	sc := engine.DefaultScenario()
	if req.Scenario != nil {
		sc = *req.Scenario
	}
	in := engine.Input{Region: s.Data, Class: class, Sheet: sheet, Loadout: engine.LoadoutFrom(snap), Procs: rep.Procs}
	res, err := engine.Simulate(in, sc)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out := simulateResponse{Result: res, Scenario: sc, Overrides: req.Overrides}
	if req.Efficiency {
		if out.Efficiency, err = engine.Efficiency(in, sc, res); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeUpstream(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ncsoft.ErrNotFound):
		writeError(w, http.StatusNotFound, "character not found")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "the official site took too long to answer")
	default:
		log.Printf("upstream: %v", err)
		writeError(w, http.StatusBadGateway, "couldn't reach the official character site")
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

// spa serves files from dir, falling back to index.html for client routes.
func spa(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
}
