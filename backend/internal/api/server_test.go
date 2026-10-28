package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	data, err := gamedata.Load("global")
	if err != nil {
		t.Fatal(err)
	}
	return (&Server{Data: data, Source: &Source{Live: ncsoft.NewClient()}}).Handler()
}

func do(t *testing.T, h http.Handler, method, path, body string, out any) int {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: %v: %s", method, path, err, rec.Body.String())
		}
	}
	return rec.Code
}

func TestDemoFlow(t *testing.T) {
	h := newTestServer(t)

	var search struct {
		Results []searchRow `json:"results"`
	}
	if code := do(t, h, "GET", "/api/search?region=demo", "", &search); code != 200 || len(search.Results) != 3 {
		t.Fatalf("search: %d %+v", code, search)
	}

	var view CharacterView
	if code := do(t, h, "GET", "/api/character/demo/1/spiritmaster_high", "", &view); code != 200 {
		t.Fatalf("character: %d", code)
	}
	if !view.Class.Supported || len(view.Equipment) == 0 || len(view.Stats) == 0 || len(view.Boards) == 0 {
		t.Fatalf("character view incomplete: %+v", view.Class)
	}

	var sim simulateResponse
	body := `{"region":"demo","serverId":1,"characterId":"spiritmaster_high","efficiency":true,
	          "scenario":{"targetId":"boss-generic","duration":120,"iterations":10,"powerShard":true,"seed":1}}`
	if code := do(t, h, "POST", "/api/simulate", body, &sim); code != 200 {
		t.Fatalf("simulate: %d", code)
	}
	if sim.Result.DPS <= 0 || len(sim.Efficiency) == 0 || sim.Result.Target.ID != "boss-generic" {
		t.Fatalf("simulate result incomplete: dps=%v eff=%d", sim.Result.DPS, len(sim.Efficiency))
	}

	// An override from the in-game stat window replaces the rebuilt value.
	var capped simulateResponse
	body = `{"region":"demo","serverId":1,"characterId":"spiritmaster_high","overrides":{"Critical":0},
	         "scenario":{"targetId":"boss-generic","duration":120,"iterations":10,"powerShard":true,"seed":1}}`
	do(t, h, "POST", "/api/simulate", body, &capped)
	if capped.Result.Summary.CritChance != 0 || capped.Result.DPS >= sim.Result.DPS {
		t.Fatalf("override not applied: crit %v dps %v", capped.Result.Summary.CritChance, capped.Result.DPS)
	}
}

func TestErrors(t *testing.T) {
	h := newTestServer(t)
	var e map[string]string
	if code := do(t, h, "GET", "/api/character/mars/1/x", "", &e); code != 400 {
		t.Errorf("unknown region: %d", code)
	}
	if code := do(t, h, "GET", "/api/character/demo/1/nobody", "", &e); code != 404 {
		t.Errorf("unknown demo character: %d", code)
	}
	if code := do(t, h, "POST", "/api/simulate", "{", &e); code != 400 {
		t.Errorf("bad json: %d", code)
	}
	if code := do(t, h, "GET", "/api/search?region=eu&name=a", "", &e); code != 400 {
		t.Errorf("short name: %d", code)
	}
}
