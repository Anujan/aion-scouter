package stats

import (
	"testing"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/testdata"
)

func TestParsePhrase(t *testing.T) {
	cases := []struct {
		in   string
		key  Key
		val  float64
		cond bool
	}{
		{"PvE Damage Boost +4.5%", PvEDamageBoost, 4.5, false},
		{"Attack Bonus +34", AttackBonus, 34, false},
		{"Cooldown -0.5%", CooldownRed, 0.5, false},
		{"Attack increase +2.5%", AttackIncrease, 2.5, false},
		{"Critical Hit increase +4.2%", CriticalRatio, 4.2, false},
		{"+150 PvE Attack when HP is 70% or more", PvEAttack, 150, true},
		{"Penetration +210", Penetration, 210, false},
		{"HP +900", HPMax, 900, false},
	}
	for _, c := range cases {
		p, ok := ParsePhrase(c.in)
		if !ok || p.Key != c.key || p.Value != c.val || (p.Condition != "") != c.cond {
			t.Errorf("ParsePhrase(%q) = %+v, %v; want %s %v cond=%v", c.in, p, ok, c.key, c.val, c.cond)
		}
	}
}

func TestRebuildFixtures(t *testing.T) {
	region, err := gamedata.Load("global")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"spiritmaster_high", "spiritmaster_mid", "chanter_high"} {
		snap, err := testdata.Snapshot(name)
		if err != nil {
			t.Fatal(err)
		}
		class, ok := region.ClassForName(snap.Info.Profile.ClassName)
		if !ok {
			t.Fatalf("%s: no class for %q", name, snap.Info.Profile.ClassName)
		}
		sheet, rep := Rebuild(snap, class)
		if sheet.Get(WeaponMaxAttack) <= sheet.Get(WeaponMinAttack) || sheet.Get(WeaponMinAttack) <= 0 {
			t.Errorf("%s: weapon range %v-%v", name, sheet.Get(WeaponMinAttack), sheet.Get(WeaponMaxAttack))
		}
		if sheet.Get(Critical) <= 0 || sheet.Get(PvEDamageBoost) <= 0 {
			t.Errorf("%s: crit %v, PvE damage boost %v", name, sheet.Get(Critical), sheet.Get(PvEDamageBoost))
		}
		if testing.Verbose() {
			for _, e := range sheet.Entries() {
				if e.Group == "offense" || e.Group == "utility" {
					t.Logf("%s %-26s %10.2f", name, e.Label, e.Value)
				}
			}
			t.Logf("%s unparsed: %v", name, rep.Unparsed)
			t.Logf("%s conditional: %v procs: %v", name, rep.Conditional, rep.Procs)
		}
	}
}
