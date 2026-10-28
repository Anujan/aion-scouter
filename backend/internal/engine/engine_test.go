package engine

import (
	"math"
	"testing"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/stats"
	"github.com/anujan/aion-scouter/backend/internal/testdata"
)

func loadInput(t *testing.T, fixture string) Input {
	t.Helper()
	region, err := gamedata.Load("global")
	if err != nil {
		t.Fatal(err)
	}
	snap, err := testdata.Snapshot(fixture)
	if err != nil {
		t.Fatal(err)
	}
	class, ok := region.ClassForName(snap.Info.Profile.ClassName)
	if !ok {
		t.Fatalf("no class %q", snap.Info.Profile.ClassName)
	}
	sheet, rep := stats.Rebuild(snap, class)
	return Input{Region: region, Class: class, Sheet: sheet, Loadout: LoadoutFrom(snap), Procs: rep.Procs}
}

func TestHitFormula(t *testing.T) {
	env := Env{
		Target: gamedata.Target{Defense: 200, CritResist: 0},
		Const:  gamedata.Constants{DefenseFactor: 0.1, CritPerPoint: 0.001, CritCap: 0.8, CritDamageBase: 0.5},
	}
	o := Offense{WeaponMin: 100, WeaponMax: 100, Penetration: 100}
	// Attack 100, coefficient 100%, flat 0, defense reduction (200-100)*0.1 = 10.
	r := o.Hit(env, gamedata.Part{Coef: 1}, HitOpts{})
	if math.Abs(r.Expected-90) > 1e-9 {
		t.Fatalf("expected 90, got %v", r.Expected)
	}
	// 500 crit rating = 50% chance of a 1.5x hit: 90 * 1.25.
	o.CritRating = 500
	r = o.Hit(env, gamedata.Part{Coef: 1}, HitOpts{})
	if math.Abs(r.Expected-112.5) > 1e-9 || r.CritChance != 0.5 {
		t.Fatalf("expected 112.5 at 50%% crit, got %v (%v)", r.Expected, r.CritChance)
	}
}

func TestSimulateFixtures(t *testing.T) {
	for _, fx := range []string{"spiritmaster_high", "spiritmaster_mid", "chanter_high"} {
		in := loadInput(t, fx)
		for _, target := range []string{"dummy", "boss-generic"} {
			sc := DefaultScenario()
			sc.TargetID = target
			r, err := Simulate(in, sc)
			if err != nil {
				t.Fatal(err)
			}
			if r.DPS <= 0 || len(r.Sources) == 0 || len(r.Timeline) == 0 {
				t.Fatalf("%s/%s: empty result %+v", fx, target, r)
			}
			var share float64
			for _, s := range r.Sources {
				share += s.Share
			}
			if math.Abs(share-1) > 1e-6 {
				t.Errorf("%s: shares sum to %v", fx, share)
			}
			if testing.Verbose() {
				t.Logf("%s vs %s: DPS %.0f, crit %.1f%%, attack %.0f", fx, target, r.DPS, r.Summary.CritChance*100, r.Summary.EffectiveAttack)
				for _, s := range r.Sources {
					t.Logf("   %-40s %-6s %5.1f%%  casts %5.1f hits %6.1f avg %8.0f", s.Name, s.Kind, s.Share*100, s.Casts, s.Hits, s.AvgHit)
				}
			}
		}
	}
}

func TestSimulateDeterministic(t *testing.T) {
	in := loadInput(t, "spiritmaster_high")
	a, _ := Simulate(in, DefaultScenario())
	b, _ := Simulate(in, DefaultScenario())
	if a.DPS != b.DPS {
		t.Fatalf("same seed gave %v and %v", a.DPS, b.DPS)
	}
}

func TestBetterGearMoreDPS(t *testing.T) {
	high, _ := Simulate(loadInput(t, "spiritmaster_high"), DefaultScenario())
	mid, _ := Simulate(loadInput(t, "spiritmaster_mid"), DefaultScenario())
	if high.DPS <= mid.DPS {
		t.Fatalf("high-geared Spiritmaster (%v) should out-damage mid (%v)", high.DPS, mid.DPS)
	}
}

func TestEfficiency(t *testing.T) {
	in := loadInput(t, "spiritmaster_high")
	sc := DefaultScenario()
	sc.Iterations = 10
	base, err := Simulate(in, sc)
	if err != nil {
		t.Fatal(err)
	}
	vals, err := Efficiency(in, sc, base)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vals {
		if v.Key == stats.Attack && math.Abs(v.AttackEquivalent-v.Step) > 1e-6 {
			t.Errorf("Attack should be worth itself, got %v", v.AttackEquivalent)
		}
		if testing.Verbose() {
			t.Logf("%-26s +%-4v  %+6.2f%%  = %6.1f Attack", v.Label, v.Step, v.DPSGainPct, v.AttackEquivalent)
		}
	}
}
