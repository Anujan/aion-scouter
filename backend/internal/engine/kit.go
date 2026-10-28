package engine

import (
	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
)

// SkillState is one skill on a character: its total level and whether it is
// slotted. The optimizer will edit these.
type SkillState struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"` // Active, Passive, Dp (stigma)
	Level    int    `json:"level"`
	Equipped bool   `json:"equipped"`
}

// Loadout is a character's skills keyed by skill ID.
type Loadout map[int]SkillState

// LoadoutFrom reads skills from a snapshot.
func LoadoutFrom(snap *ncsoft.Snapshot) Loadout {
	l := Loadout{}
	for _, sk := range snap.Equipment.Skill.SkillList {
		if sk.Acquired != 1 {
			continue
		}
		l[sk.ID] = SkillState{ID: sk.ID, Name: sk.Name, Category: sk.Category, Level: sk.SkillLevel, Equipped: sk.Equip == 1}
	}
	return l
}

// Skill is a usable skill at the character's level, with tooltip numbers
// parsed out.
type Skill struct {
	ID       int
	Name     string
	Level    int
	Parts    []gamedata.Part
	Cooldown float64
	MP       float64
	Text     string
}

// Part returns damage part i, or a zero part when the tooltip has fewer.
func (k *Skill) Part(i int) gamedata.Part {
	if i < len(k.Parts) {
		return k.Parts[i]
	}
	return gamedata.Part{}
}

// Kit is the set of skills a character can use in a fight.
type Kit struct {
	skills map[int]*Skill
}

// NewKit resolves a loadout against the class's skill data. Passives count
// when learned; actives and stigmas only when slotted.
func NewKit(class *gamedata.Class, l Loadout) *Kit {
	k := &Kit{skills: map[int]*Skill{}}
	for id, st := range l {
		if st.Level <= 0 || (st.Category != "Passive" && !st.Equipped) {
			continue
		}
		raw := class.Skill(id, st.Name)
		if raw == nil {
			continue
		}
		lv := raw.At(st.Level)
		k.skills[id] = &Skill{
			ID:       id,
			Name:     raw.Name,
			Level:    st.Level,
			Parts:    gamedata.Parts(lv.Text),
			Cooldown: gamedata.Seconds(lv.Cooldown),
			MP:       gamedata.Number(lv.MP),
			Text:     lv.Text,
		}
	}
	return k
}

// Get returns a usable skill, or nil.
func (k *Kit) Get(id int) *Skill { return k.skills[id] }

// Has reports whether a skill is usable.
func (k *Kit) Has(id int) bool { return k.skills[id] != nil }

// nameOf returns a usable skill's name, or "".
func (k *Kit) nameOf(id int) string {
	if s := k.skills[id]; s != nil {
		return s.Name
	}
	return ""
}
