package engine

import "github.com/anujan/aion-scouter/backend/internal/gamedata"

// Chanter skill IDs (Global client).
const (
	chOnslaught        = 18010000
	chIncandescentBlow = 18040000
	chImpactfulCrush   = 18060000
	chRushingSmash     = 18090000
	chDarkCrush        = 18100000
	chFracturingBlow   = 18130000
	chHeatWaveBlow     = 18150000
	chUndefeatedMantra = 18190000
	chTremorCrush      = 18210000
	chObliterate       = 18220000
	chEnsnaringMark    = 18230000
	chPowerOfTheStorm  = 18250000
	chSpinningStrike   = 18290000
	chMarchutansWrath  = 18330000
	chAssaultShock     = 18700000
	chEarthsPromise    = 18780000
	chWindsPromise     = 18800000
)

type chanter struct {
	kit       *Kit
	darkCrush float64 // Dark Crush usable until this time
}

func newChanter(kit *Kit) Module { return &chanter{kit: kit} }

func (m *chanter) Setup(s *Sim) {
	procCD := s.Model.Param("procCooldown", 1)
	if sk := m.kit.Get(chUndefeatedMantra); sk != nil {
		boost, _ := gamedata.Capture(sk.Text, `PvE Damage Boost and PvE Damage Tolerance by ([\d.]+)%`)
		s.AddBuff(sk.Name, 1e9, 1, func(o *Offense, _ int) { o.DamageBoost += boost / 100 })
	}
	if sk := m.kit.Get(chEarthsPromise); sk != nil {
		// Re-applied every 10s on any hit while attacking: effectively permanent.
		down, _ := gamedata.Capture(sk.Text, `PvE Damage Tolerance by ([\d.]+)%`)
		s.Env.TargetToleranceDown += down
	}
	if sk := m.kit.Get(chWindsPromise); sk != nil {
		s.OnLand(func(l Landing) {
			if !l.Proc && s.Proc("windspromise", 0.5*l.CritChance, procCD) {
				s.Deal(sk.Name, "proc", sk.Part(0), HitOpts{NoMultiHit: true}, Landing{Proc: true})
			}
		})
	}
}

func (m *chanter) hit(sk *Skill, opens float64) func(s *Sim) {
	return func(s *Sim) {
		s.Deal(sk.Name, "skill", sk.Part(0), HitOpts{}, Landing{})
		if opens > 0 {
			m.darkCrush = s.T + opens
		}
	}
}

func (m *chanter) Next(s *Sim) *Action {
	k := m.kit
	ready := func(id int) *Skill {
		sk := k.Get(id)
		if sk == nil || !s.Ready(sk.Name) || s.MP < sk.MP {
			return nil
		}
		return sk
	}
	cast := func(name string) float64 { return s.Model.CastTime(name) }
	window := s.Model.Param("darkCrushWindow", 2)
	stigmaWindow := s.Model.Param("darkCrushWindowStigma", 3)

	if sk := ready(chPowerOfTheStorm); sk != nil {
		cs := s.Model.Param("stormCombatSpeed", 20) / 100
		cdr := s.Model.Param("stormCooldownReduction", 20) / 100
		dur := s.Model.Param("stormDuration", 16.5)
		return &Action{Name: sk.Name, Cast: cast("buff"), Cooldown: sk.Cooldown, Do: func(s *Sim) {
			s.AddBuff(sk.Name, dur, 1, func(o *Offense, _ int) {
				o.CombatSpeed += cs
				o.CooldownReduction += cdr
			})
		}}
	}
	if sk := ready(chSpinningStrike); sk != nil {
		crit := s.Model.Param("spinningStrikeCritBuff", 15) / 100
		dur := s.Model.Param("spinningStrikeBuffDuration", 30)
		stacks := int(s.Model.Param("spinningStrikeMaxStacks", 2))
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) {
			m.hit(sk, window)(s)
			s.AddBuff(sk.Name, dur, stacks, func(o *Offense, n int) { o.CritDamageBoost += crit * float64(n) })
		}}
	}
	if sk := ready(chDarkCrush); sk != nil && s.T < m.darkCrush {
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, Do: func(s *Sim) {
			m.darkCrush = 0
			m.hit(sk, 0)(s)
		}}
	}
	for _, c := range []struct {
		id    int
		opens float64
	}{
		{chImpactfulCrush, window},
		{chMarchutansWrath, stigmaWindow},
		{chEnsnaringMark, stigmaWindow},
		{chFracturingBlow, 0},
		{chObliterate, 0},
		{chAssaultShock, 0},
		{chHeatWaveBlow, 0},
		{chRushingSmash, 0},
		{chTremorCrush, 0},
	} {
		if sk := ready(c.id); sk != nil {
			return &Action{Name: sk.Name, Cast: cast(sk.Name), Cooldown: sk.Cooldown, MP: sk.MP, Do: func(s *Sim) {
				m.hit(sk, c.opens)(s)
				if c.id == chRushingSmash {
					s.MP = min(s.MPMax, s.MP+100)
				}
			}}
		}
	}
	if sk := k.Get(chIncandescentBlow); sk != nil && s.MP >= sk.MP+s.Model.Param("mpReserve", 250) {
		return &Action{Name: sk.Name, Cast: cast(sk.Name), MP: sk.MP, Do: m.hit(sk, 0)}
	}
	if sk := k.Get(chOnslaught); sk != nil {
		return &Action{Name: sk.Name, Cast: cast(sk.Name), Do: func(s *Sim) {
			m.hit(sk, 0)(s)
			s.MP = min(s.MPMax, s.MP+100)
		}}
	}
	return nil
}
