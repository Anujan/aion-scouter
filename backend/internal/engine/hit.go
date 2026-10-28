// Package engine turns a stat sheet into damage: an expected-value hit model
// (community damage formula) driven by a per-class rotation simulation.
package engine

import (
	"math"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/stats"
)

// Offense is the subset of a stat sheet the hit formula reads, with every
// percentage converted to a fraction. Buffs add to it during a fight.
type Offense struct {
	WeaponMin, WeaponMax float64
	Attack               float64 // flat Attack from gear other than the weapon range
	AttackBonus          float64
	AttackIncrease       float64
	WeaponDamageBoost    float64
	PvEAttack            float64
	BossAttack           float64
	Penetration          float64
	DamageBoost          float64 // Damage + PvE Damage Boost (Boss added vs bosses)
	BossDamageBoost      float64
	CritRating           float64 // after Critical Hit Increase %
	CritDamageBoost      float64
	DoubleChance         float64
	PerfectChance        float64
	MultiHitChance       float64
	CombatSpeed          float64
	CooldownReduction    float64
	PowerShardBonus      float64
}

// OffenseFrom reads an Offense out of a sheet.
func OffenseFrom(s *stats.Sheet) Offense {
	return Offense{
		WeaponMin:         s.Get(stats.WeaponMinAttack),
		WeaponMax:         s.Get(stats.WeaponMaxAttack),
		Attack:            s.Get(stats.Attack),
		AttackBonus:       s.Get(stats.AttackBonus),
		AttackIncrease:    s.Pct(stats.AttackIncrease),
		WeaponDamageBoost: s.Pct(stats.WeaponDmgBoost),
		PvEAttack:         s.Get(stats.PvEAttack),
		BossAttack:        s.Get(stats.BossAttack),
		Penetration:       s.Get(stats.Penetration),
		DamageBoost:       s.Pct(stats.DamageBoost) + s.Pct(stats.PvEDamageBoost),
		BossDamageBoost:   s.Pct(stats.BossDamageBoost),
		CritRating:        s.Get(stats.Critical) * (1 + s.Pct(stats.CriticalRatio)),
		CritDamageBoost:   s.Pct(stats.CritDmgBoost),
		DoubleChance:      s.Pct(stats.DoubleChance),
		PerfectChance:     s.Pct(stats.PerfectChance),
		MultiHitChance:    s.Pct(stats.MultiHitChance),
		CombatSpeed:       s.Pct(stats.CombatSpeed),
		CooldownReduction: s.Pct(stats.CooldownRed),
		PowerShardBonus:   s.Get(stats.PowerShardBonus),
	}
}

// Env is the fixed context of a fight.
type Env struct {
	Target     gamedata.Target
	Const      gamedata.Constants
	PowerShard bool
	// TargetToleranceDown lowers the target's Damage Tolerance (debuffs), in %.
	TargetToleranceDown float64
}

// HitOpts tweaks one hit.
type HitOpts struct {
	AttackMult float64 // multiplies effective Attack (spirit Attack bonuses); 0 means 1
	ForceCrit  bool
	ExtraBoost float64 // extra Damage Boost fraction for this hit (e.g. "damage taken from Spirit +10%")
	NoMultiHit bool    // procs and damage over time don't multi-hit
	NoVariance bool    // fixed damage (Theostones): skip the attack formula
}

// HitResult is the expected outcome of one hit.
type HitResult struct {
	Expected   float64 `json:"expected"`
	NonCrit    float64 `json:"nonCrit"`
	Crit       float64 `json:"crit"`
	CritChance float64 `json:"critChance"`
}

// EffectiveAttack is Attack after every attack-side term in the formula.
func (o Offense) EffectiveAttack(env Env) float64 {
	p := clamp(o.PerfectChance, 0, 1)
	weapon := (1-p)*(o.WeaponMin+o.WeaponMax)/2 + p*o.WeaponMax
	pure := (weapon + o.Attack) * (1 + o.WeaponDamageBoost)
	bonus := o.AttackBonus
	if env.PowerShard {
		pure *= 1 + env.Const.PowerShardMultiplier
		bonus += o.PowerShardBonus
	}
	a := (pure+bonus)*(1+o.AttackIncrease) + o.PvEAttack
	if env.Target.Boss {
		a += o.BossAttack
	}
	return a
}

// CritChance is the chance a normal hit crits against the target.
func (o Offense) CritChance(env Env) float64 {
	return clamp((o.CritRating-env.Target.CritResist)*env.Const.CritPerPoint, 0, env.Const.CritCap)
}

// Multiplier is everything after the base hit, excluding crit: the Damage
// Boost pool, Double and Multi-hit, as expected values.
func (o Offense) multiplier(env Env, opts HitOpts) float64 {
	boost := o.DamageBoost + opts.ExtraBoost
	if env.Target.Boss {
		boost += o.BossDamageBoost
	}
	tol := math.Max(0, env.Target.DamageTolerance-env.TargetToleranceDown) / 100
	m := math.Max(0, 1+boost-tol)
	m *= 1 + clamp(o.DoubleChance-env.Target.DoubleResist/100, 0, 1)
	if !opts.NoMultiHit {
		m *= 1 + clamp(o.MultiHitChance, 0, 1)*env.Const.MultiHitExtraHits*env.Const.MultiHitHitFraction
	}
	return m
}

// Hit computes the expected damage of one damage part.
func (o Offense) Hit(env Env, part gamedata.Part, opts HitOpts) HitResult {
	var base float64
	if opts.NoVariance {
		base = part.Flat
	} else {
		atk := o.EffectiveAttack(env)
		if opts.AttackMult > 0 {
			atk *= opts.AttackMult
		}
		base = atk*part.Coef + part.Flat
	}
	base -= math.Max(0, env.Target.Defense-o.Penetration) * env.Const.DefenseFactor
	base = math.Max(0, base)

	nonCrit := base * o.multiplier(env, opts)
	critMult := math.Max(1, 1+env.Const.CritDamageBase+o.CritDamageBoost-env.Target.CritDamageTolerance/100)
	cc := o.CritChance(env)
	if opts.ForceCrit {
		cc = 1
	}
	return HitResult{
		Expected:   nonCrit * (1 + cc*(critMult-1)),
		NonCrit:    nonCrit,
		Crit:       nonCrit * critMult,
		CritChance: cc,
	}
}

func clamp(v, lo, hi float64) float64 { return math.Min(hi, math.Max(lo, v)) }
