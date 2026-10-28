// Package testdata holds anonymized character snapshots captured from the
// official API, used as test fixtures across packages.
package testdata

import (
	"embed"
	"encoding/json"

	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
)

//go:embed *.json
var files embed.FS

// Snapshot loads a fixture by name ("spiritmaster_high", "spiritmaster_mid",
// "chanter_high").
func Snapshot(name string) (*ncsoft.Snapshot, error) {
	b, err := files.ReadFile(name + ".json")
	if err != nil {
		return nil, err
	}
	var s ncsoft.Snapshot
	return &s, json.Unmarshal(b, &s)
}
