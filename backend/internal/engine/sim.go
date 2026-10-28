package engine

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
)

// Sim is one simulated fight against one target. Damage is computed as
// expected values per hit (crit, double, multi-hit folded in); the random
// number generator only decides chance-based procs.
type Sim struct {
	T     float64
	End   float64
	Rng   *rand.Rand
	Env   Env
	Model *gamedata.ClassModel
	MP    float64
	MPMax float64

	base     Offense
	queue    eventQueue
	seq      int
	cds      map[string]float64
	icds     map[string]float64
	buffs    map[string]*Buff
	buffList []string // application order, so float sums are deterministic
	hooks    []func(Landing)
	sources  map[string]*sourceTotals
	timeline []TimelineEntry
	record   bool
	module   Module
}

// Buff is a timed modifier on the caster's Offense.
type Buff struct {
	Name      string
	Until     float64
	Stacks    int
	MaxStacks int
	Apply     func(o *Offense, stacks int)
}

// Action is something the caster does when free.
type Action struct {
	Name     string
	Cast     float64 // base cast time in seconds, before Combat Speed
	Cooldown float64 // base cooldown, before Cooldown Reduction
	NoCDR    bool    // "Unaffected by cooldown reduction effects"
	MP       float64
	Do       func(s *Sim)
}

// Landing describes a hit that landed, for proc hooks.
type Landing struct {
	Source      string
	FromSpirit  bool
	SpiritSkill bool
	CritChance  float64
	Proc        bool // procs don't trigger other procs
	DoT         bool
}

// Module is a class's rotation: it sets up passives and procs, then picks an
// action whenever the caster is free.
type Module interface {
	Setup(s *Sim)
	Next(s *Sim) *Action
}

type sourceTotals struct {
	kind   string
	damage float64
	hits   int
	casts  int
}

// TimelineEntry is one cast, for the rotation chart.
type TimelineEntry struct {
	T     float64 `json:"t"`
	Skill string  `json:"skill"`
	Cast  float64 `json:"cast"`
}

func newSim(env Env, model *gamedata.ClassModel, base Offense, mod Module, duration float64, seed int64, mpMax float64, record bool) *Sim {
	return &Sim{
		End:     duration,
		Rng:     rand.New(rand.NewSource(seed)),
		Env:     env,
		Model:   model,
		MP:      mpMax,
		MPMax:   mpMax,
		base:    base,
		cds:     map[string]float64{},
		icds:    map[string]float64{},
		buffs:   map[string]*Buff{},
		sources: map[string]*sourceTotals{},
		record:  record,
		module:  mod,
	}
}

func (s *Sim) run() {
	s.module.Setup(s)
	regen := s.Model.Param("mpRegenPerSecond", 0)
	var tick func()
	tick = func() {
		s.MP = math.Min(s.MPMax, s.MP+regen)
		s.At(s.T+1, tick)
	}
	s.At(1, tick)
	s.At(0, s.actorReady)
	for s.queue.Len() > 0 {
		ev := heap.Pop(&s.queue).(*event)
		if ev.t > s.End {
			break
		}
		s.T = ev.t
		ev.fn()
	}
}

func (s *Sim) actorReady() {
	a := s.module.Next(s)
	if a == nil {
		s.At(s.T+0.1, s.actorReady)
		return
	}
	cast := s.CastTime(a.Cast)
	if a.Cooldown > 0 {
		s.StartCooldown(a.Name, a.Cooldown, a.NoCDR)
	}
	s.MP -= a.MP
	s.source(a.Name, "skill").casts++
	if s.record {
		s.timeline = append(s.timeline, TimelineEntry{T: round2(s.T), Skill: a.Name, Cast: round2(cast)})
	}
	do := a.Do
	s.At(s.T+cast, func() { do(s) })
	// Reaction delay varies per cast; a fixed delay makes cast cycles perfectly
	// periodic and creates false breakpoints against 1s proc cooldowns.
	delay := s.Model.Param("latency", 0.1) + (s.Rng.Float64()-0.5)*s.Model.Param("latencyJitter", 0)
	s.At(s.T+cast+math.Max(0, delay), s.actorReady)
}

// At schedules fn at time t.
func (s *Sim) At(t float64, fn func()) {
	s.seq++
	heap.Push(&s.queue, &event{t: t, seq: s.seq, fn: fn})
}

// Offense is the caster's current offense, including active buffs.
func (s *Sim) Offense() Offense {
	o := s.base
	for _, name := range s.buffList {
		if b := s.buffs[name]; b.Until > s.T {
			b.Apply(&o, b.Stacks)
		}
	}
	return o
}

// CastTime scales a base cast time by Combat Speed.
func (s *Sim) CastTime(base float64) float64 {
	return base / (1 + math.Max(0, s.Offense().CombatSpeed))
}

// Ready reports whether a cooldown has elapsed.
func (s *Sim) Ready(name string) bool { return s.cds[name] <= s.T+1e-9 }

// StartCooldown puts name on cooldown, scaled by Cooldown Reduction (capped at 60%).
func (s *Sim) StartCooldown(name string, cd float64, noCDR bool) {
	if !noCDR {
		cd *= 1 - clamp(s.Offense().CooldownReduction, 0, 0.6)
	}
	s.cds[name] = s.T + cd
}

// ReduceCooldown takes seconds off a running cooldown.
func (s *Sim) ReduceCooldown(name string, sec float64) {
	if s.cds[name] > s.T {
		s.cds[name] = math.Max(s.T, s.cds[name]-sec)
	}
}

// CooldownLeft is the time until name is ready.
func (s *Sim) CooldownLeft(name string) float64 { return math.Max(0, s.cds[name]-s.T) }

// AddBuff applies or refreshes a buff, adding a stack up to MaxStacks.
func (s *Sim) AddBuff(name string, duration float64, maxStacks int, apply func(o *Offense, stacks int)) {
	b, ok := s.buffs[name]
	if !ok {
		s.buffList = append(s.buffList, name)
	}
	if !ok || b.Until <= s.T {
		b = &Buff{Name: name, MaxStacks: max(maxStacks, 1), Apply: apply}
		s.buffs[name] = b
	}
	b.Until = s.T + duration
	b.Stacks = min(b.Stacks+1, b.MaxStacks)
}

// BuffActive reports whether a buff is up.
func (s *Sim) BuffActive(name string) bool {
	b, ok := s.buffs[name]
	return ok && b.Until > s.T
}

// OnLand registers a proc hook, called for every landed hit.
func (s *Sim) OnLand(fn func(Landing)) { s.hooks = append(s.hooks, fn) }

// Proc rolls a chance-based effect with an internal cooldown.
func (s *Sim) Proc(name string, chance, icd float64) bool {
	if s.icds[name] > s.T+1e-9 {
		return false
	}
	if s.Rng.Float64() >= chance {
		return false
	}
	s.icds[name] = s.T + icd
	return true
}

// Deal applies one damage part to the target, records it under name and
// fires landing hooks.
func (s *Sim) Deal(name, kind string, part gamedata.Part, opts HitOpts, land Landing) HitResult {
	r := s.Offense().Hit(s.Env, part, opts)
	src := s.source(name, kind)
	src.damage += r.Expected
	src.hits++
	land.Source = name
	land.CritChance = r.CritChance
	for _, h := range s.hooks {
		h(land)
	}
	return r
}

// DoT spreads total damage over ticks every second for duration seconds.
// Ticks don't trigger landing procs.
func (s *Sim) DoT(name string, part gamedata.Part, duration float64, opts HitOpts, perTick bool) {
	ticks := int(math.Max(1, math.Round(duration)))
	tick := part
	if !perTick {
		tick = gamedata.Part{Flat: part.Flat / float64(ticks), Coef: part.Coef / float64(ticks)}
	}
	opts.NoMultiHit = true
	key := "dot:" + name
	s.cds[key] = s.T + duration // marks the target as afflicted
	for i := 1; i <= ticks; i++ {
		s.At(s.T+float64(i), func() {
			r := s.Offense().Hit(s.Env, tick, opts)
			src := s.source(name, "dot")
			src.damage += r.Expected
			src.hits++
		})
	}
}

// TargetHasDoT reports whether any damage over time is ticking on the target.
func (s *Sim) TargetHasDoT() bool {
	for k, until := range s.cds {
		if len(k) > 4 && k[:4] == "dot:" && until > s.T {
			return true
		}
	}
	return false
}

func (s *Sim) source(name, kind string) *sourceTotals {
	src, ok := s.sources[name]
	if !ok {
		src = &sourceTotals{kind: kind}
		s.sources[name] = src
	}
	return src
}

// totalDamage sums every source, in name order so the float sum is stable.
func (s *Sim) totalDamage() float64 {
	names := make([]string, 0, len(s.sources))
	for n := range s.sources {
		names = append(names, n)
	}
	sort.Strings(names)
	var t float64
	for _, n := range names {
		t += s.sources[n].damage
	}
	return t
}

type event struct {
	t   float64
	seq int
	fn  func()
}

type eventQueue []*event

func (q eventQueue) Len() int { return len(q) }
func (q eventQueue) Less(i, j int) bool {
	if q[i].t != q[j].t {
		return q[i].t < q[j].t
	}
	return q[i].seq < q[j].seq
}
func (q eventQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *eventQueue) Push(x any)   { *q = append(*q, x.(*event)) }
func (q *eventQueue) Pop() any {
	old := *q
	n := len(old)
	e := old[n-1]
	*q = old[:n-1]
	return e
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
