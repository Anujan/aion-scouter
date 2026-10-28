package stats

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Contribution is one source's share of a stat.
type Contribution struct {
	Source string  `json:"source"` // e.g. "Gear", "Daevanion", "Title", "Passive"
	Detail string  `json:"detail"` // e.g. "Spiritforged Orb (+15)"
	Value  float64 `json:"value"`
}

// Sheet is a set of stat totals with provenance.
type Sheet struct {
	values  map[Key]float64
	sources map[Key][]Contribution
}

func NewSheet() *Sheet {
	return &Sheet{values: map[Key]float64{}, sources: map[Key][]Contribution{}}
}

func (s *Sheet) Add(k Key, v float64, source, detail string) {
	if v == 0 {
		return
	}
	k = Canonical(k)
	s.values[k] += v
	s.sources[k] = append(s.sources[k], Contribution{source, detail, v})
}

// Set replaces a stat's total, recording the override as its only source.
func (s *Sheet) Set(k Key, v float64, source string) {
	s.values[k] = v
	s.sources[k] = []Contribution{{source, "override", v}}
}

func (s *Sheet) Get(k Key) float64 { return s.values[k] }

// Pct returns a percentage stat as a fraction (12.5% -> 0.125).
func (s *Sheet) Pct(k Key) float64 { return s.values[k] / 100 }

func (s *Sheet) Clone() *Sheet {
	c := NewSheet()
	for k, v := range s.values {
		c.values[k] = v
	}
	for k, v := range s.sources {
		c.sources[k] = append([]Contribution(nil), v...)
	}
	return c
}

// Entry is one stat for display.
type Entry struct {
	Key     Key            `json:"key"`
	Label   string         `json:"label"`
	Group   string         `json:"group"`
	Pct     bool           `json:"pct"`
	Value   float64        `json:"value"`
	Sources []Contribution `json:"sources"`
}

// Entries lists every non-zero stat, offense first.
func (s *Sheet) Entries() []Entry {
	var out []Entry
	for k, v := range s.values {
		info := Registry[k]
		if info.Label == "" {
			info = Info{Label: string(k), Group: "other"}
		}
		out = append(out, Entry{k, info.Label, info.Group, info.Pct, round(v), s.sources[k]})
	}
	order := map[string]int{"offense": 0, "utility": 1, "defense": 2, "pvp": 3, "other": 4}
	sort.Slice(out, func(i, j int) bool {
		if order[out[i].Group] != order[out[j].Group] {
			return order[out[i].Group] < order[out[j].Group]
		}
		return out[i].Label < out[j].Label
	})
	return out
}

func round(v float64) float64 { return math.Round(v*1000) / 1000 }

var (
	labelFirst = regexp.MustCompile(`^(.+?)\s*([+-])\s*([\d,]+(?:\.\d+)?)\s*(%?)$`)
	valueFirst = regexp.MustCompile(`^([+-])\s*([\d,]+(?:\.\d+)?)\s*(%?)\s+(.+?)(?:\s+when\s+(.+))?$`)
)

// Phrase is a stat bonus parsed from free text such as "PvE Damage Boost +4.5%"
// or "+150 PvE Attack when HP is 70% or more".
type Phrase struct {
	Key       Key
	Value     float64
	Condition string // non-empty when the bonus only applies sometimes
}

// ParsePhrase understands the two shapes the game uses for flat bonuses.
func ParsePhrase(text string) (Phrase, bool) {
	text = strings.TrimSpace(text)
	var label, sign, num, condition string
	if m := labelFirst.FindStringSubmatch(text); m != nil {
		label, sign, num = m[1], m[2], m[3]
	} else if m := valueFirst.FindStringSubmatch(text); m != nil {
		sign, num, label, condition = m[1], m[2], m[4], m[5]
	} else {
		return Phrase{}, false
	}
	k, ok := KeyForLabel(label)
	if !ok {
		return Phrase{}, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(num, ",", ""), 64)
	if err != nil {
		return Phrase{}, false
	}
	if sign == "-" {
		v = -v
	}
	// "Cooldown -0.5%" and "MP Cost -3.1%" are reductions written as negatives.
	if (k == CooldownRed || k == "MPCostReduction") && v < 0 {
		v = -v
	}
	return Phrase{Key: k, Value: v, Condition: condition}, true
}
