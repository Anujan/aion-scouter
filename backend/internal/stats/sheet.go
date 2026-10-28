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

// ratios maps rating stats to the "X increase %" stat that multiplies them.
// The in-game stat window shows the multiplied value.
var ratios = map[Key]Key{
	Critical:         CriticalRatio,
	Accuracy:         AccuracyRatio,
	"CriticalResist": "CriticalResistRatio",
	"Evasion":        "EvasionRatio",
	"Block":          "BlockRatio",
	HPMax:            "HPRatio",
	MPMax:            "MPRatio",
	Defense:          "DefenseRatio",
}

// Final is a stat as the in-game stat window shows it: flat total times its
// "increase %" stat, if it has one.
func (s *Sheet) Final(k Key) float64 {
	if r, ok := ratios[k]; ok {
		return s.values[k] * (1 + s.Pct(r))
	}
	return s.values[k]
}

// SetFinal overrides a stat with a value read from the in-game stat window,
// backing out its "increase %" so Final(k) returns v.
func (s *Sheet) SetFinal(k Key, v float64, source string) {
	if r, ok := ratios[k]; ok {
		v /= 1 + s.Pct(r)
	}
	s.Set(k, v, source)
}

// Entries lists every non-zero stat as the stat window shows it, offense first.
func (s *Sheet) Entries() []Entry {
	var out []Entry
	for k, v := range s.values {
		info := Registry[k]
		if info.Label == "" {
			info = Info{Label: string(k), Group: "other"}
		}
		sources := s.sources[k]
		if r, ok := ratios[k]; ok && s.values[r] != 0 {
			final := s.Final(k)
			sources = append(append([]Contribution(nil), sources...),
				Contribution{"Increase", Label(r) + " " + strconv.FormatFloat(round(s.values[r]), 'f', -1, 64) + "%", final - v})
			v = final
		}
		out = append(out, Entry{k, info.Label, info.Group, info.Pct, round(v), sources})
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
