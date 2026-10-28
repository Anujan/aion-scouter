package engine

import (
	"github.com/anujan/aion-scouter/backend/internal/stats"
)

// StatValue is how much one step of a stat is worth: MapleScouter's
// "converted stat" idea, with flat Attack as the common currency.
type StatValue struct {
	Key              stats.Key `json:"key"`
	Label            string    `json:"label"`
	Step             float64   `json:"step"`
	Pct              bool      `json:"pct"`
	DPSGain          float64   `json:"dpsGain"`
	DPSGainPct       float64   `json:"dpsGainPct"`
	AttackEquivalent float64   `json:"attackEquivalent"` // flat Attack giving the same gain
}

var efficiencySteps = []struct {
	key  stats.Key
	step float64
	boss bool // only meaningful against bosses
}{
	{stats.Attack, 50, false},
	{stats.AttackBonus, 20, false},
	{stats.AttackIncrease, 1, false},
	{stats.WeaponDmgBoost, 1, false},
	{stats.PvEAttack, 20, false},
	{stats.PvEDamageBoost, 1, false},
	{stats.BossDamageBoost, 1, true},
	{stats.Critical, 50, false},
	{stats.CritDmgBoost, 1, false},
	{stats.DoubleChance, 1, false},
	{stats.PerfectChance, 1, false},
	{stats.MultiHitChance, 1, false},
	{stats.CombatSpeed, 5, false},
	{stats.CooldownRed, 5, false},
	{stats.Penetration, 100, false},
}

// Efficiency re-simulates with each stat nudged up one step. Every run uses
// the same seeds to limit proc noise. Combat Speed and Cooldown Reduction
// shift every event's timing, so they use a bigger step to stay above noise.
func Efficiency(in Input, sc Scenario, baseline *Result) ([]StatValue, error) {
	var out []StatValue
	var perAttack float64
	for _, e := range efficiencySteps {
		if e.boss && !baseline.Target.Boss {
			continue
		}
		sheet := in.Sheet.Clone()
		sheet.Add(e.key, e.step, "What-if", "efficiency probe")
		alt := in
		alt.Sheet = sheet
		r, err := Simulate(alt, sc)
		if err != nil {
			return nil, err
		}
		gain := r.DPS - baseline.DPS
		if e.key == stats.Attack {
			perAttack = gain / e.step
		}
		v := StatValue{Key: e.key, Label: stats.Label(e.key), Step: e.step, Pct: stats.IsPct(e.key), DPSGain: gain}
		if baseline.DPS > 0 {
			v.DPSGainPct = gain / baseline.DPS * 100
		}
		if perAttack > 0 {
			v.AttackEquivalent = gain / perAttack
		}
		out = append(out, v)
	}
	return out, nil
}
