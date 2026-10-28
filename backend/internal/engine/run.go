package engine

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/stats"
)

var modules = map[string]func(*Kit) Module{
	"spiritmaster": newSpiritmaster,
	"chanter":      newChanter,
}

// Supported reports whether a class has a rotation module.
func Supported(classSlug string) bool { return modules[classSlug] != nil }

// Scenario is everything about a fight that isn't the character.
type Scenario struct {
	TargetID   string  `json:"targetId"`
	Duration   float64 `json:"duration"`   // seconds
	Iterations int     `json:"iterations"` // Monte Carlo runs (procs only)
	PowerShard bool    `json:"powerShard"`
	Seed       int64   `json:"seed"`
}

// DefaultScenario is a 3-minute fight against the training dummy.
func DefaultScenario() Scenario {
	return Scenario{TargetID: "dummy", Duration: 180, Iterations: 40, PowerShard: true, Seed: 1}
}

func (sc *Scenario) normalize() {
	if sc.Duration <= 0 {
		sc.Duration = 180
	}
	sc.Duration = math.Min(sc.Duration, 900)
	if sc.Iterations <= 0 {
		sc.Iterations = 40
	}
	sc.Iterations = min(sc.Iterations, 200)
	if sc.TargetID == "" {
		sc.TargetID = "dummy"
	}
}

// Input is a fully resolved character ready to simulate.
type Input struct {
	Region  *gamedata.Region
	Class   *gamedata.Class
	Sheet   *stats.Sheet
	Loadout Loadout
	Procs   []string // Theostone effect texts from gear
}

// Result is the outcome of a simulation.
type Result struct {
	DPS         float64            `json:"dps"`
	DPSStdErr   float64            `json:"dpsStdErr"` // Monte Carlo standard error of DPS (proc luck)
	TotalDamage float64            `json:"totalDamage"`
	Duration    float64            `json:"duration"`
	Iterations  int                `json:"iterations"`
	Target      gamedata.Target    `json:"target"`
	Summary     Summary            `json:"summary"`
	Sources     []SourceResult     `json:"sources"`
	Timeline    []TimelineEntry    `json:"timeline"`
	Assumptions []string           `json:"assumptions"`
	Constants   gamedata.Constants `json:"constants"`
}

// Summary explains the multipliers behind the number.
type Summary struct {
	EffectiveAttack   float64 `json:"effectiveAttack"`
	CritChance        float64 `json:"critChance"`
	CritMultiplier    float64 `json:"critMultiplier"`
	DamageBoost       float64 `json:"damageBoost"`
	DoubleChance      float64 `json:"doubleChance"`
	MultiHitChance    float64 `json:"multiHitChance"`
	CombatSpeed       float64 `json:"combatSpeed"`
	CooldownReduction float64 `json:"cooldownReduction"`
	DefenseReduction  float64 `json:"defenseReduction"` // flat damage removed per hit by target defense
}

// SourceResult is one line of the damage breakdown.
type SourceResult struct {
	Name   string  `json:"name"`
	Kind   string  `json:"kind"` // skill, dot, spirit, proc
	Damage float64 `json:"damage"`
	Share  float64 `json:"share"`
	DPS    float64 `json:"dps"`
	Casts  float64 `json:"casts"`
	Hits   float64 `json:"hits"`
	AvgHit float64 `json:"avgHit"`
}

// Simulate runs the class rotation against the scenario's target.
func Simulate(in Input, sc Scenario) (*Result, error) {
	sc.normalize()
	newModule, ok := modules[in.Class.Slug]
	if !ok {
		return nil, fmt.Errorf("class %q is not supported yet", in.Class.Slug)
	}
	target, ok := in.Region.Target(sc.TargetID)
	if !ok {
		return nil, fmt.Errorf("unknown target %q", sc.TargetID)
	}
	env := Env{Target: target, Const: in.Region.Constants, PowerShard: sc.PowerShard}
	base := OffenseFrom(in.Sheet)
	mpMax := in.Class.Model.Param("mpBase", 0) + in.Sheet.Get(stats.MPMax)
	procs := parseTheostones(in.Procs)

	totals := map[string]*sourceTotals{}
	var timeline []TimelineEntry
	runDPS := make([]float64, 0, sc.Iterations)
	for i := 0; i < sc.Iterations; i++ {
		kit := NewKit(in.Class, in.Loadout)
		sim := newSim(env, in.Class.Model, base, newModule(kit), sc.Duration, sc.Seed+int64(i), mpMax, i == 0)
		addTheostones(sim, procs)
		sim.run()
		runDPS = append(runDPS, sim.totalDamage()/sc.Duration)
		for name, src := range sim.sources {
			t, ok := totals[name]
			if !ok {
				t = &sourceTotals{kind: src.kind}
				totals[name] = t
			}
			t.damage += src.damage
			t.hits += src.hits
			t.casts += src.casts
		}
		if i == 0 {
			timeline = sim.timeline
		}
	}

	n := float64(sc.Iterations)
	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	sort.Strings(names)
	var total float64
	for _, name := range names {
		total += totals[name].damage
	}
	total /= n
	res := &Result{
		TotalDamage: total,
		DPS:         total / sc.Duration,
		DPSStdErr:   stdErr(runDPS),
		Duration:    sc.Duration,
		Iterations:  sc.Iterations,
		Target:      target,
		Timeline:    timeline,
		Assumptions: in.Class.Model.Assumptions,
		Constants:   in.Region.Constants,
		Summary:     summarize(base, env),
	}
	for _, name := range names {
		t := totals[name]
		if t.damage == 0 && t.casts == 0 {
			continue
		}
		r := SourceResult{
			Name:   name,
			Kind:   t.kind,
			Damage: t.damage / n,
			DPS:    t.damage / n / sc.Duration,
			Casts:  float64(t.casts) / n,
			Hits:   float64(t.hits) / n,
		}
		if total > 0 {
			r.Share = r.Damage / total
		}
		if t.hits > 0 {
			r.AvgHit = t.damage / float64(t.hits)
		}
		res.Sources = append(res.Sources, r)
	}
	sort.Slice(res.Sources, func(i, j int) bool { return res.Sources[i].Damage > res.Sources[j].Damage })
	return res, nil
}

func summarize(o Offense, env Env) Summary {
	crit := 1 + env.Const.CritDamageBase + o.CritDamageBoost - env.Target.CritDamageTolerance/100
	boost := o.DamageBoost
	if env.Target.Boss {
		boost += o.BossDamageBoost
	}
	return Summary{
		EffectiveAttack:   o.EffectiveAttack(env),
		CritChance:        o.CritChance(env),
		CritMultiplier:    math.Max(1, crit),
		DamageBoost:       boost,
		DoubleChance:      clamp(o.DoubleChance-env.Target.DoubleResist/100, 0, 1),
		MultiHitChance:    o.MultiHitChance,
		CombatSpeed:       o.CombatSpeed,
		CooldownReduction: o.CooldownReduction,
		DefenseReduction:  math.Max(0, env.Target.Defense-o.Penetration) * env.Const.DefenseFactor,
	}
}

type theostone struct {
	name   string
	chance float64
	damage float64
	icd    float64
}

var (
	theoChance = regexp.MustCompile(`([\d.]+)% chance to deal ([\d,]+) `)
	theoCD     = regexp.MustCompile(`Cooldown: ([\d.]+)s`)
)

// parseTheostones reads "Theostone: X (item): Has a 1% chance to deal 5000
// Fire damage ... Cooldown: 1s" lines.
func parseTheostones(lines []string) []theostone {
	var out []theostone
	for _, l := range lines {
		m := theoChance.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		chance, _ := strconv.ParseFloat(m[1], 64)
		dmg, _ := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", ""), 64)
		icd := 1.0
		if c := theoCD.FindStringSubmatch(l); c != nil {
			icd, _ = strconv.ParseFloat(c[1], 64)
		}
		name := l
		if i := strings.Index(l, " ("); i > 0 {
			name = l[:i]
		}
		out = append(out, theostone{name, chance / 100, dmg, icd})
	}
	return out
}

func addTheostones(s *Sim, stones []theostone) {
	for _, t := range stones {
		t := t
		s.OnLand(func(l Landing) {
			if !l.Proc && !l.FromSpirit && s.Proc(t.name, t.chance, t.icd) {
				s.Deal(t.name, "proc", gamedata.Part{Flat: t.damage}, HitOpts{NoVariance: true, NoMultiHit: true}, Landing{Proc: true})
			}
		})
	}
}

// stdErr is the standard error of the mean of xs.
func stdErr(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var mean, ss float64
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	return math.Sqrt(ss/float64(len(xs)-1)) / math.Sqrt(float64(len(xs)))
}
