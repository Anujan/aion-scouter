package engine

import (
	"sort"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
)

// Spiritmaster skill IDs (Global client).
const (
	smColdShock      = 16010000
	smCombustion     = 16040000
	smSiphon         = 16060000
	smSoulsCry       = 16070000
	smCryOfTerror    = 16080000
	smFireSpirit     = 16100000
	smWaterSpirit    = 16110000
	smWindSpirit     = 16120000
	smEarthSpirit    = 16130000
	smCurse          = 16140000
	smJSCorrode      = 16150000
	smBenediction    = 16190000
	smCursedCloud    = 16220000
	smSeizeMagic     = 16230000
	smDestructive    = 16240000
	smAncientSpirit  = 16250000
	smMagicBlock     = 16260000
	smFusion         = 16300000
	smDimensional    = 16330000
	smFlameBlessing  = 16370000
	smAssaultTerror  = 16700000
	smDescent        = 16730000
	smCorrodePassive = 16740000
	smRevitalization = 16750000
	smUnification    = 16780000
	smCountercurrent = 16800000
)

type spirit struct {
	skill         int
	element       string
	procChance    float64 // chance per basic attack to use its skill
	attackGrowth  bool    // "+2% Attack per skill level"
	mpPerAttack   float64
	jointPartBase int // offset of this element in jointstrike tooltips (Fire, Water, Earth, Wind, Ancient)
}

var spirits = []spirit{
	{smFireSpirit, "Fire", 0.15, true, 0, 0},
	{smWaterSpirit, "Water", 0.15, true, 20, 1},
	{smEarthSpirit, "Earth", 0.10, false, 0, 2},
	{smWindSpirit, "Wind", 0.10, false, 0, 3},
}

const ancientJointIndex = 4

type spiritmaster struct {
	kit *Kit

	active        *spirit
	elements      int
	fourElements  bool
	descentUntil  float64
	ancientUntil  float64
	ancientBasics int
	corrodeUntil  float64
	spiritSkillAt map[string]float64
}

func newSpiritmaster(kit *Kit) Module {
	return &spiritmaster{kit: kit, spiritSkillAt: map[string]float64{}}
}

func (m *spiritmaster) param(s *Sim, name string, def float64) float64 {
	return s.Model.Param(name, def)
}

// spiritOpts are hit options for damage dealt by a spirit.
func (m *spiritmaster) spiritOpts(s *Sim, sp *spirit, skillID int) HitOpts {
	o := HitOpts{AttackMult: 1}
	if sk := m.kit.Get(skillID); sk != nil && (sp == nil || sp.attackGrowth) {
		o.AttackMult = 1 + m.param(s, "spiritAttackPerLevel", 0.02)*float64(sk.Level)
	}
	if s.T < m.corrodeUntil {
		o.ExtraBoost = m.param(s, "corrodeSpiritDamageTaken", 10) / 100
	}
	return o
}

func (m *spiritmaster) Setup(s *Sim) {
	procCD := m.param(s, "procCooldown", 1)

	// The active spirit's basic attacks.
	interval := m.param(s, "spiritBasicInterval", 2)
	var basic func()
	basic = func() {
		if sp := m.active; sp != nil {
			part := gamedata.Part{Coef: m.param(s, "spiritBasicCoef", 0.5)}
			s.Deal(sp.element+" Spirit (basic)", "spirit", part, m.spiritOpts(s, sp, sp.skill), Landing{FromSpirit: true})
			s.MP = min(s.MPMax, s.MP+sp.mpPerAttack)
			if s.Rng.Float64() < sp.procChance {
				m.spiritSkill(s, sp)
			}
		}
		s.At(s.T+interval, basic)
	}
	s.At(interval, basic)

	if sk := m.kit.Get(smDescent); sk != nil {
		s.OnLand(func(l Landing) {
			if !l.Proc && !l.FromSpirit && s.T < m.descentUntil && s.Proc("descent", 0.5, procCD) {
				s.Deal(sk.Name, "proc", sk.Part(0), HitOpts{NoMultiHit: true}, Landing{Proc: true})
			}
		})
	}
	if sk := m.kit.Get(smCorrodePassive); sk != nil {
		s.OnLand(func(l Landing) {
			if !l.Proc && !l.FromSpirit && s.Proc("corrode", 0.5*l.CritChance, procCD) {
				s.Deal(sk.Name, "proc", sk.Part(0), HitOpts{NoMultiHit: true}, Landing{Proc: true})
			}
		})
	}
	if sk := m.kit.Get(smCountercurrent); sk != nil {
		s.OnLand(func(l Landing) {
			if !l.Proc && !l.FromSpirit && s.TargetHasDoT() && s.Proc("countercurrent", 0.5, procCD) {
				s.Deal(sk.Name, "proc", sk.Part(0), HitOpts{NoMultiHit: true}, Landing{Proc: true})
			}
		})
	}
	if sk := m.kit.Get(smFlameBlessing); sk != nil {
		s.OnLand(func(l Landing) {
			if !l.Proc && !l.FromSpirit && s.BuffActive("Flame Blessing") && s.Proc("flameblessing", 0.5, procCD) {
				s.Deal(sk.Name, "proc", sk.Part(0), HitOpts{NoMultiHit: true}, Landing{Proc: true})
			}
		})
	}
	if sk := m.kit.Get(smUnification); sk != nil {
		per, _ := gamedata.Capture(sk.Text, `Critical Damage Boost by ([\d.]+)%`)
		maxStacks := int(m.param(s, "elementUnificationMaxStacks", 5))
		dur := m.param(s, "elementUnificationDuration", 10)
		s.OnLand(func(l Landing) {
			if l.SpiritSkill && s.Proc("unification", 1, procCD) {
				s.AddBuff("Element Unification", dur, maxStacks, func(o *Offense, n int) {
					o.CritDamageBoost += per / 100 * float64(n)
				})
			}
		})
	}
	if sk := m.kit.Get(smRevitalization); sk != nil {
		chance, _ := gamedata.Capture(sk.Text, `Has a ([\d.]+)% chance`)
		s.OnLand(func(l Landing) {
			if !l.Proc && s.Proc("revitalization", chance/100, procCD) {
				for _, sp := range spirits {
					s.ReduceCooldown(m.kit.nameOf(sp.skill), 1)
				}
			}
		})
	}
}

// spiritSkill fires a spirit's skill if its internal cooldown allows, granting
// an element and triggering Dimensional Control.
func (m *spiritmaster) spiritSkill(s *Sim, sp *spirit) {
	if s.T < m.spiritSkillAt[sp.element] {
		return
	}
	m.spiritSkillAt[sp.element] = s.T + m.param(s, "spiritSkillCooldown", 3)
	sk := m.kit.Get(sp.skill)
	s.Deal(sk.Name+" (spirit skill)", "spirit", sk.Part(0), m.spiritOpts(s, sp, sp.skill), Landing{FromSpirit: true, SpiritSkill: true})
	m.gainElement()
	if dc := m.kit.Get(smDimensional); dc != nil {
		s.Deal(dc.Name, "skill", dc.Part(0), HitOpts{}, Landing{})
		s.MP = min(s.MPMax, s.MP+100)
	}
}

func (m *spiritmaster) gainElement() {
	if m.fourElements {
		return
	}
	m.elements++
	if m.elements >= 4 {
		m.fourElements, m.elements = true, 0
	}
}

func (m *spiritmaster) ancientSkill(s *Sim) {
	sk := m.kit.Get(smAncientSpirit)
	s.Deal("Ancient Spirit (spirit skill)", "spirit", sk.Part(0), m.spiritOpts(s, nil, smAncientSpirit), Landing{FromSpirit: true, SpiritSkill: true})
	m.fourElements, m.elements = true, 0
}

func (m *spiritmaster) summonAncient(s *Sim) {
	dur := m.param(s, "ancientDuration", 30)
	m.ancientUntil = s.T + dur
	m.descentUntil = max(m.descentUntil, s.T+m.param(s, "descentWindowAncient", 30))
	m.ancientBasics = 0
	m.ancientSkill(s)
	interval := m.param(s, "spiritBasicInterval", 2)
	every := int(m.param(s, "ancientBasicsPerSkill", 7))
	until := m.ancientUntil
	var basic func()
	basic = func() {
		if s.T >= until {
			return
		}
		part := gamedata.Part{Coef: m.param(s, "spiritBasicCoef", 0.5)}
		s.Deal("Ancient Spirit (basic)", "spirit", part, m.spiritOpts(s, nil, smAncientSpirit), Landing{FromSpirit: true})
		m.ancientBasics++
		if m.ancientBasics%every == 0 {
			m.ancientSkill(s)
		}
		s.At(s.T+interval, basic)
	}
	s.At(s.T+interval, basic)
}

// jointstrike deals a jointstrike's own hit, its damage over time, and the
// active spirit's follow-up.
func (m *spiritmaster) jointstrike(s *Sim, sk *Skill, spiritOffset int, dot float64) {
	s.Deal(sk.Name, "skill", sk.Part(0), HitOpts{}, Landing{})
	if dot > 0 {
		s.DoT(sk.Name+" (DoT)", sk.Part(1), dot, HitOpts{}, m.param(s, "dotIsTotal", 1) == 0)
	}
	idx := -1
	var sp *spirit
	switch {
	case s.T < m.ancientUntil:
		idx = ancientJointIndex
	case m.active != nil:
		sp = m.active
		idx = sp.jointPartBase
	}
	if idx >= 0 {
		skill := smAncientSpirit
		if sp != nil {
			skill = sp.skill
		}
		s.Deal(sk.Name+" (spirit)", "spirit", sk.Part(spiritOffset+idx), m.spiritOpts(s, sp, skill), Landing{FromSpirit: true})
	}
}

func (m *spiritmaster) Next(s *Sim) *Action {
	k := m.kit
	ready := func(id int) *Skill {
		sk := k.Get(id)
		if sk == nil || !s.Ready(sk.Name) || s.MP < sk.MP {
			return nil
		}
		return sk
	}
	cast := func(name string) float64 { return s.Model.CastTime(name) }

	if sk := k.Get(smFusion); sk != nil && m.fourElements {
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Do: func(s *Sim) {
			m.fourElements = false
			s.Deal(sk.Name, "skill", sk.Part(0), HitOpts{}, Landing{})
		}}
	}
	if sk := ready(smBenediction); sk != nil {
		boost, _ := gamedata.Capture(sk.Text, `Increases PvE Damage Boost by ([\d.]+)%`)
		dur := m.param(s, "benedictionDuration", 10)
		return &Action{Name: sk.Name, Cast: cast("buff"), Cooldown: sk.Cooldown, Do: func(s *Sim) {
			s.AddBuff(sk.Name, dur, 1, func(o *Offense, _ int) { o.DamageBoost += boost / 100 })
		}}
	}
	if sk := ready(smFlameBlessing); sk != nil {
		dur := m.param(s, "flameBlessingDuration", 16.5)
		return &Action{Name: sk.Name, Cast: cast("buff"), Cooldown: sk.Cooldown, Do: func(s *Sim) {
			s.AddBuff("Flame Blessing", dur, 1, func(*Offense, int) {})
		}}
	}
	if sk := ready(smAncientSpirit); sk != nil {
		return &Action{Name: sk.Name, Cast: cast("summon"), Cooldown: sk.Cooldown, MP: sk.MP, Do: m.summonAncient}
	}
	// Swap to whichever summon is ready, highest skill level first.
	var summons []*Skill
	for _, sp := range spirits {
		if sk := ready(sp.skill); sk != nil {
			summons = append(summons, sk)
		}
	}
	sort.Slice(summons, func(i, j int) bool { return summons[i].Level > summons[j].Level })
	if len(summons) > 0 {
		sk := summons[0]
		sp := spiritFor(sk.ID)
		return &Action{Name: sk.Name, Cast: cast("summon"), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) {
			m.active = sp
			m.descentUntil = max(m.descentUntil, s.T+m.param(s, "descentWindow", 10))
			m.spiritSkillAt[sp.element] = 0 // a fresh summon always uses its skill
			m.spiritSkill(s, sp)
		}}
	}
	if sk := ready(smDestructive); sk != nil {
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) { m.jointstrike(s, sk, 1, 0) }}
	}
	if sk := ready(smJSCorrode); sk != nil {
		dur := m.param(s, "corrodeDuration", 20)
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) {
			m.corrodeUntil = s.T + dur
			m.jointstrike(s, sk, 2, dur)
		}}
	}
	if sk := ready(smCurse); sk != nil {
		dur := m.param(s, "curseDuration", 5)
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) { m.jointstrike(s, sk, 2, dur) }}
	}
	for _, id := range []int{smSoulsCry, smMagicBlock, smSeizeMagic, smCryOfTerror, smAssaultTerror, smSiphon} {
		if sk := ready(id); sk != nil {
			return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) {
				s.Deal(sk.Name, "skill", sk.Part(0), HitOpts{}, Landing{})
			}}
		}
	}
	if sk := ready(smCursedCloud); sk != nil {
		dur := m.param(s, "cursedCloudDuration", 10)
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) {
			s.Deal(sk.Name, "skill", sk.Part(0), HitOpts{}, Landing{})
			s.DoT(sk.Name+" (DoT)", sk.Part(1), dur, HitOpts{}, m.param(s, "dotIsTotal", 1) == 0)
		}}
	}
	if sk := k.Get(smCombustion); sk != nil && s.MP >= sk.MP+m.param(s, "mpReserve", 200) {
		return &Action{Name: sk.Name, Cast: cast(sk.Name), MP: sk.MP, Do: func(s *Sim) {
			s.Deal(sk.Name, "skill", sk.Part(0), HitOpts{}, Landing{})
		}}
	}
	if sk := k.Get(smColdShock); sk != nil {
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Do: func(s *Sim) {
			s.Deal(sk.Name, "skill", sk.Part(0), HitOpts{}, Landing{})
			s.MP = min(s.MPMax, s.MP+100)
		}}
	}
	return nil
}

func spiritFor(id int) *spirit {
	for i := range spirits {
		if spirits[i].skill == id {
			return &spirits[i]
		}
	}
	return nil
}
