package api

import (
	"sort"
	"strconv"
	"time"

	"github.com/anujan/aion-scouter/backend/internal/engine"
	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
	"github.com/anujan/aion-scouter/backend/internal/stats"
)

// CharacterView is everything the character page shows. The frontend renders
// it as-is; no game logic happens in the browser.
type CharacterView struct {
	Ref       ncsoft.Ref    `json:"ref"`
	FetchedAt time.Time     `json:"fetchedAt"`
	Profile   ProfileView   `json:"profile"`
	Class     ClassView     `json:"class"`
	Equipment []GearView    `json:"equipment"`
	Skills    []SkillView   `json:"skills"`
	Boards    []BoardView   `json:"boards"`
	Stats     []stats.Entry `json:"stats"`
	Report    stats.Report  `json:"report"`
}

type ProfileView struct {
	Name         string `json:"name"`
	Level        int    `json:"level"`
	ClassName    string `json:"className"`
	Server       string `json:"server"`
	Race         string `json:"race"`
	Title        string `json:"title"`
	CombatPower  int    `json:"combatPower"`
	ItemLevel    int    `json:"itemLevel"`
	ProfileImage string `json:"profileImage"`
}

type ClassView struct {
	Slug        string   `json:"slug"`
	Supported   bool     `json:"supported"`
	Status      string   `json:"status"`
	Assumptions []string `json:"assumptions"`
}

type StatLine struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Extra string `json:"extra,omitempty"`
}

type GearView struct {
	Slot       string     `json:"slot"`
	SlotPos    int        `json:"slotPos"`
	Name       string     `json:"name"`
	Grade      string     `json:"grade"`
	Category   string     `json:"category"`
	Enchant    int        `json:"enchant"`
	MaxEnchant int        `json:"maxEnchant"`
	Exceed     int        `json:"exceed"`
	Icon       string     `json:"icon,omitempty"`
	Main       []StatLine `json:"main"`
	Options    []StatLine `json:"options"`
	Manastones []StatLine `json:"manastones"`
	Skills     []StatLine `json:"skills"`
	Theostones []string   `json:"theostones"`
	Set        string     `json:"set,omitempty"`
}

type SkillView struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Level    int    `json:"level"`
	Equipped bool   `json:"equipped"`
	Icon     string `json:"icon,omitempty"`
}

type BoardView struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	OpenNodes   int        `json:"openNodes"`
	TotalNodes  int        `json:"totalNodes"`
	PointsSpent int        `json:"pointsSpent"`
	PointsTotal int        `json:"pointsTotal"`
	Stats       []StatLine `json:"stats"`
}

func buildView(snap *ncsoft.Snapshot, class *gamedata.Class, sheet *stats.Sheet, rep stats.Report) *CharacterView {
	p := snap.Info.Profile
	v := &CharacterView{
		Ref:       snap.Ref,
		FetchedAt: snap.FetchedAt,
		Profile: ProfileView{
			Name: p.CharacterName, Level: p.CharacterLevel, ClassName: p.ClassName, Server: p.ServerName,
			Race: p.RaceName, Title: p.TitleName, CombatPower: p.CombatPower, ProfileImage: p.ProfileImage,
		},
		Stats:  sheet.Entries(),
		Report: rep,
	}
	for _, st := range snap.Info.Stat.StatList {
		if st.Type == "ItemLevel" {
			v.Profile.ItemLevel = st.Value
		}
	}
	if class != nil {
		v.Class = ClassView{Slug: class.Slug, Supported: engine.Supported(class.Slug), Status: class.Model.Status, Assumptions: class.Model.Assumptions}
	}

	slots := append([]ncsoft.EquipmentSlot(nil), snap.Equipment.Equipment.EquipmentList...)
	sort.Slice(slots, func(i, j int) bool { return slots[i].SlotPos < slots[j].SlotPos })
	for _, sl := range slots {
		g := GearView{Slot: sl.SlotPosName, SlotPos: sl.SlotPos, Name: sl.Name, Grade: sl.Grade, Enchant: sl.EnchantLevel, Exceed: sl.ExceedLevel, Icon: sl.Icon}
		if it := snap.Items[strconv.Itoa(sl.SlotPos)]; it != nil {
			g.Category, g.MaxEnchant = it.CategoryName, it.MaxEnchantLevel
			for _, st := range it.MainStats {
				line := StatLine{Label: st.Name, Value: st.Value}
				if st.MinValue != "" {
					line.Value = st.MinValue + "–" + st.Value
				}
				if st.Extra != "" && st.Extra != "0" && st.Extra != "0%" {
					line.Extra = "+" + st.Extra
				}
				g.Main = append(g.Main, line)
			}
			for _, st := range it.SubStats {
				g.Options = append(g.Options, StatLine{Label: st.Name, Value: st.Value})
			}
			for _, st := range it.MagicStoneStat {
				g.Manastones = append(g.Manastones, StatLine{Label: st.Name, Value: st.Value, Extra: st.Grade})
			}
			for _, sk := range it.SubSkills {
				g.Skills = append(g.Skills, StatLine{Label: sk.Name, Value: "+" + strconv.Itoa(sk.Level)})
			}
			for _, gs := range it.GodStoneStat {
				g.Theostones = append(g.Theostones, gs.Name)
			}
			if it.Set != nil {
				g.Set = it.Set.Name
			}
		}
		v.Equipment = append(v.Equipment, g)
	}

	for _, sk := range snap.Equipment.Skill.SkillList {
		if sk.Acquired != 1 {
			continue
		}
		v.Skills = append(v.Skills, SkillView{ID: sk.ID, Name: sk.Name, Category: sk.Category, Level: sk.SkillLevel, Equipped: sk.Equip == 1, Icon: sk.Icon})
	}

	for _, b := range snap.Info.Daevanion.BoardList {
		bv := BoardView{ID: b.ID, Name: b.Name, OpenNodes: b.OpenNodeCount, TotalNodes: b.TotalNodeCount}
		totals := map[string]float64{}
		var order []string
		if class != nil {
			for _, rb := range class.Boards.Boards {
				if rb.ID == b.ID {
					for _, n := range rb.Nodes {
						bv.PointsTotal += n.Cost
					}
				}
			}
			if d := snap.Daevanion[strconv.Itoa(b.ID)]; d != nil {
				for _, n := range d.NodeList {
					node := class.Node(n.NodeID)
					if n.Open == 0 || node == nil {
						continue
					}
					bv.PointsSpent += node.Cost
					label := node.Label
					if node.Type == "Stat" && len(node.Stats) == 1 {
						label = stats.Label(stats.Key(node.Stats[0].Stat))
						if _, ok := totals[label]; !ok {
							order = append(order, label)
						}
						val := node.Stats[0].Value
						if stats.IsPct(stats.Key(node.Stats[0].Stat)) {
							val /= 100
						}
						totals[label] += val
					}
				}
			}
		}
		for _, l := range order {
			bv.Stats = append(bv.Stats, StatLine{Label: l, Value: strconv.FormatFloat(totals[l], 'f', -1, 64)})
		}
		v.Boards = append(v.Boards, bv)
	}
	return v
}
