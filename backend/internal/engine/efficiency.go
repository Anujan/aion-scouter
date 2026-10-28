package engine

import (
	"math"
	"sync"

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
	WithinNoise      bool      `json:"withinNoise"`      // gain is smaller than proc-luck noise
}

// efficiencyIterations is the minimum number of runs per probe. Proc luck
// moves DPS by roughly ±0.3% at 40 runs; small stat steps need more.
const efficiencyIterations = 200

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
	sc.Iterations = max(sc.Iterations, efficiencyIterations)
	if baseline.Iterations < sc.Iterations {
		var err error
		if baseline, err = Simulate(in, sc); err != nil {
			return nil, err
		}
	}
	// Runs share seeds, so their noise is correlated; treating the two errors
	// as independent (sqrt 2) is a conservative bound.
	noise := 2 * math.Sqrt2 * baseline.DPSStdErr
	var steps []int
	for i, e := range efficiencySteps {
		if !e.boss || baseline.Target.Boss {
			steps = append(steps, i)
		}
	}
	// Probes are independent: run them in parallel.
	gains := make([]float64, len(steps))
	errs := make([]error, len(steps))
	var wg sync.WaitGroup
	for j, i := range steps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := efficiencySteps[i]
			sheet := in.Sheet.Clone()
			sheet.Add(e.key, e.step, "What-if", "efficiency probe")
			alt := in
			alt.Sheet = sheet
			r, err := Simulate(alt, sc)
			if err != nil {
				errs[j] = err
				return
			}
			gains[j] = r.DPS - baseline.DPS
		}()
	}
	wg.Wait()

	var perAttack float64 // efficiencySteps starts with Attack
	out := make([]StatValue, 0, len(steps))
	for j, i := range steps {
		if errs[j] != nil {
			return nil, errs[j]
		}
		e, gain := efficiencySteps[i], gains[j]
		if e.key == stats.Attack {
			perAttack = gain / e.step
		}
		v := StatValue{Key: e.key, Label: stats.Label(e.key), Step: e.step, Pct: stats.IsPct(e.key), DPSGain: gain, WithinNoise: math.Abs(gain) < noise}
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
