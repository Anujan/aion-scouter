package stats

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
)

// Report lists what the rebuild could not account for, so the UI can be
// honest about it.
type Report struct {
	Unparsed    []string `json:"unparsed"`    // bonus text we don't understand yet
	Conditional []string `json:"conditional"` // bonuses counted as always-on
	Procs       []string `json:"procs"`       // proc effects handled by the engine, not the sheet
}

// primaryStats are reported by the API as totals that already include gear, so
// gear lines for them are skipped to avoid double counting.
var primaryStats = map[string]bool{
	"STR": true, "DEX": true, "INT": true, "CON": true, "AGI": true, "WIS": true,
	"Justice": true, "Freedom": true, "Illusion": true, "Life": true, "Time": true,
	"Destruction": true, "Death": true, "Wisdom": true, "Destiny": true, "Space": true,
}

// Rebuild sums every stat source the API exposes into one sheet.
func Rebuild(snap *ncsoft.Snapshot, class *gamedata.Class) (*Sheet, Report) {
	s := NewSheet()
	var rep Report

	for k, v := range class.Model.Base {
		s.Add(Key(k), v, "Base", "level 45 base (calibrated)")
	}

	// Primary and god stats: the API gives their derived effects as text lines.
	for _, st := range snap.Info.Stat.StatList {
		if st.Type == "ItemLevel" {
			s.Add(ItemLevel, float64(st.Value), "Character", "item level")
			continue
		}
		for _, line := range st.StatSecondList {
			p, ok := ParsePhrase(line)
			if !ok {
				rep.Unparsed = append(rep.Unparsed, st.Name+": "+line)
				continue
			}
			s.Add(p.Key, p.Value, "Stats", fmt.Sprintf("%s %d", st.Name, st.Value))
		}
	}

	// Gear, in slot order.
	slots := make([]ncsoft.EquipmentSlot, len(snap.Equipment.Equipment.EquipmentList))
	copy(slots, snap.Equipment.Equipment.EquipmentList)
	sort.Slice(slots, func(i, j int) bool { return slots[i].SlotPos < slots[j].SlotPos })
	seenSets := map[string]bool{}
	for _, slot := range slots {
		it := snap.Items[strconv.Itoa(slot.SlotPos)]
		if it == nil {
			continue
		}
		name := it.Name
		if it.EnchantLevel > 0 {
			name = fmt.Sprintf("%s +%d", it.Name, it.EnchantLevel)
		}
		addItem(s, slot, it, name)
		for _, g := range it.GodStoneStat {
			rep.Procs = append(rep.Procs, fmt.Sprintf("%s (%s): %s", g.Name, it.Name, firstLine(g.Desc)))
		}
		if it.Set != nil && !seenSets[it.Set.ID] {
			seenSets[it.Set.ID] = true
			for _, b := range it.Set.Bonuses {
				if b.Degree > it.Set.EquippedCount {
					continue
				}
				for _, d := range b.Descriptions {
					detail := fmt.Sprintf("%s (%d-piece)", it.Set.Name, b.Degree)
					addPhrase(s, &rep, d, "Set bonus", detail)
				}
			}
		}
	}

	// Equipped titles.
	for _, t := range snap.Info.Title.TitleList {
		for _, st := range t.EquipStatList {
			addPhrase(s, &rep, st.Desc, "Title", t.Name)
		}
	}

	// Daevanion: every open node, valued from the board data.
	for _, board := range snap.Daevanion {
		for _, n := range board.NodeList {
			if n.Open == 0 {
				continue
			}
			node := class.Node(n.NodeID)
			if node == nil {
				rep.Unparsed = append(rep.Unparsed, fmt.Sprintf("Daevanion node %d (%s)", n.NodeID, n.Name))
				continue
			}
			for _, st := range node.Stats {
				k := Key(st.Stat)
				v := st.Value
				if IsPct(k) {
					v /= 100 // board data stores percentages in basis points
				}
				s.Add(k, v, "Daevanion", boardName(class, n.BoardID))
			}
		}
	}

	// Class passives, at the character's current skill level.
	levels := SkillLevels(snap)
	for _, pm := range class.Model.Passives {
		lvl := levels[pm.Skill]
		skill := class.Skill(pm.Skill, "")
		if lvl == 0 || skill == nil {
			continue
		}
		text := skill.At(lvl).Text
		for _, ps := range pm.Stats {
			if v, ok := gamedata.Capture(text, ps.Pattern); ok {
				s.Add(Key(ps.Key), v, "Passive", fmt.Sprintf("%s Lv %d", skill.Name, lvl))
			}
		}
	}
	return s, rep
}

func addItem(s *Sheet, slot ncsoft.EquipmentSlot, it *ncsoft.ItemResponse, name string) {
	isWeapon := slot.SlotPosName == "MainHand"
	for _, st := range it.MainStats {
		if primaryStats[st.ID] {
			continue
		}
		k := Key(st.ID)
		extra := parseValue(k, st.Extra)
		if isWeapon && k == Attack && st.MinValue != "" {
			s.Add(WeaponMinAttack, parseValue(k, st.MinValue)+extra, "Gear", name)
			s.Add(WeaponMaxAttack, parseValue(k, st.Value)+extra, "Gear", name)
			continue
		}
		s.Add(k, parseValue(k, st.Value)+extra, "Gear", name)
	}
	for _, st := range it.SubStats {
		if primaryStats[st.ID] {
			continue
		}
		k := Key(st.ID)
		s.Add(k, parseValue(k, st.Value), "Gear", name+" (option)")
	}
	for _, st := range it.MagicStoneStat {
		k := Key(st.ID)
		s.Add(k, parseValue(k, st.Value), "Manastone", name)
	}
}

func addPhrase(s *Sheet, rep *Report, text, source, detail string) {
	p, ok := ParsePhrase(text)
	if !ok {
		rep.Unparsed = append(rep.Unparsed, detail+": "+text)
		return
	}
	if p.Condition != "" {
		rep.Conditional = append(rep.Conditional, detail+": "+text)
	}
	s.Add(p.Key, p.Value, source, detail)
}

// parseValue reads an item stat string ("+7", "1.5%", "1,100"). Percentage
// stats written without a % sign (manastones) are basis points.
func parseValue(k Key, v string) float64 {
	v = strings.TrimSpace(strings.TrimPrefix(v, "+"))
	pct := strings.HasSuffix(v, "%")
	f, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSuffix(v, "%"), ",", ""), 64)
	if IsPct(Canonical(k)) && !pct {
		f /= 100
	}
	return f
}

// SkillLevels maps skill ID to the character's current (total) level. The API
// level already includes Daevanion and gear bonuses.
func SkillLevels(snap *ncsoft.Snapshot) map[int]int {
	out := map[int]int{}
	for _, sk := range snap.Equipment.Skill.SkillList {
		if sk.Acquired == 1 {
			out[sk.ID] = sk.SkillLevel
		}
	}
	return out
}

func boardName(c *gamedata.Class, id int) string {
	for _, b := range c.Boards.Boards {
		if b.ID == id {
			return b.Name
		}
	}
	return strconv.Itoa(id)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
