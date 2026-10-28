package gamedata

// Constants are the global damage-formula constants (data/<region>/constants.json).
// Most are community-measured; Notes says where each came from.
type Constants struct {
	DefenseFactor        float64           `json:"defenseFactor"`        // PvE damage reduction per point of (Defense - Penetration)
	CritPerPoint         float64           `json:"critPerPoint"`         // crit chance per Critical Hit point above the target's resist
	CritCap              float64           `json:"critCap"`              // maximum crit chance
	CritDamageBase       float64           `json:"critDamageBase"`       // bonus on a crit before Critical Damage Boost (0.5 = 1.5x)
	MultiHitExtraHits    float64           `json:"multiHitExtraHits"`    // average extra hits per multi-hit proc
	MultiHitHitFraction  float64           `json:"multiHitHitFraction"`  // damage of each extra hit, as a fraction of the hit
	PowerShardMultiplier float64           `json:"powerShardMultiplier"` // regular Power Shard attack multiplier
	Notes                map[string]string `json:"notes"`
}

// Target is a damage-taking profile: a training dummy or a boss.
type Target struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Description         string  `json:"description"`
	Boss                bool    `json:"boss"`
	Defense             float64 `json:"defense"`
	CritResist          float64 `json:"critResist"`
	CritDamageTolerance float64 `json:"critDamageTolerance"` // %
	DamageTolerance     float64 `json:"damageTolerance"`     // % (all Damage/PvE/Boss Tolerance combined)
	DoubleResist        float64 `json:"doubleResist"`        // %
	HP                  float64 `json:"hp,omitempty"`
	Estimated           bool    `json:"estimated"`
	Source              string  `json:"source"`
}

// ClassModel holds per-class tunables the engine's class module reads
// (data/<region>/classes/<class>.json). Anything not in the tooltips lives
// here so it can be calibrated without code changes.
type ClassModel struct {
	Class       string             `json:"class"`
	DisplayName string             `json:"displayName"`
	Status      string             `json:"status"`
	Base        map[string]float64 `json:"base"` // class base stats not exposed by the API
	CastTimes   map[string]float64 `json:"castTimes"`
	Params      map[string]float64 `json:"params"`
	Passives    []PassiveModel     `json:"passives"`
	Assumptions []string           `json:"assumptions"`
}

// PassiveModel pulls permanent stats out of a passive skill's tooltip.
type PassiveModel struct {
	Skill int           `json:"skill"`
	Stats []PassiveStat `json:"stats"`
}

type PassiveStat struct {
	Key     string `json:"key"`
	Pattern string `json:"pattern"` // regexp; first group is the value
}

// CastTime returns the modelled base cast time for a skill (before Combat Speed).
func (m *ClassModel) CastTime(skill string) float64 {
	if v, ok := m.CastTimes[skill]; ok {
		return v
	}
	return m.CastTimes["default"]
}

// Param returns a tunable, or def when the model doesn't set it.
func (m *ClassModel) Param(name string, def float64) float64 {
	if v, ok := m.Params[name]; ok {
		return v
	}
	return def
}
