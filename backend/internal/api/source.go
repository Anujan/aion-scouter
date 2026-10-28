package api

import (
	"context"
	"strings"

	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
	"github.com/anujan/aion-scouter/backend/internal/testdata"
)

// DemoRegion serves the bundled, anonymized fixture characters, so the site
// works offline and in development without calling NCSOFT.
const DemoRegion = "demo"

var demoCharacters = []struct {
	fixture string
	pcID    int
	level   int
	server  string
}{
	{"spiritmaster_high", 24, 45, "Demo"},
	{"spiritmaster_mid", 24, 45, "Demo"},
	{"chanter_high", 36, 45, "Demo"},
}

// Source is where characters come from: the live site or the demo fixtures.
type Source struct {
	Live *ncsoft.Client
}

func (s *Source) Search(ctx context.Context, region, name string, page int) (*ncsoft.SearchResponse, error) {
	if region != DemoRegion {
		return s.Live.Search(ctx, region, name, page)
	}
	out := &ncsoft.SearchResponse{}
	for _, d := range demoCharacters {
		snap, err := testdata.Snapshot(d.fixture)
		if err != nil {
			return nil, err
		}
		p := snap.Info.Profile
		if name != "" && !strings.Contains(strings.ToLower(p.CharacterName), strings.ToLower(name)) {
			continue
		}
		out.List = append(out.List, ncsoft.SearchResult{
			CharacterID: d.fixture, Name: p.CharacterName, PCID: d.pcID, Level: p.CharacterLevel,
			ServerID: 1, ServerName: d.server, Region: DemoRegion,
		})
	}
	out.Pagination.Page, out.Pagination.EndPage, out.Pagination.Total = 1, 1, len(out.List)
	return out, nil
}

func (s *Source) Snapshot(ctx context.Context, ref ncsoft.Ref) (*ncsoft.Snapshot, error) {
	if ref.Region != DemoRegion {
		return s.Live.LoadSnapshot(ctx, ref)
	}
	for _, d := range demoCharacters {
		if d.fixture == ref.CharacterID {
			snap, err := testdata.Snapshot(d.fixture)
			if err != nil {
				return nil, err
			}
			snap.Ref = ref
			return snap, nil
		}
	}
	return nil, ncsoft.ErrNotFound
}
