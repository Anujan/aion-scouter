package gamedata

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
)

//go:embed data
var embedded embed.FS

// Region is the data set for one game region (currently only "global").
type Region struct {
	Name      string
	Constants Constants
	Targets   []Target
	Classes   map[string]*Class // keyed by class slug, e.g. "spiritmaster"
}

// Class bundles everything we know about one class.
type Class struct {
	Slug   string
	Model  *ClassModel
	Skills *SkillFile
	Boards *BoardFile

	skillByID   map[int]*RawSkill
	skillByName map[string]*RawSkill
	nodeByID    map[int]*RawNode
}

// Load reads the embedded data for a region.
func Load(region string) (*Region, error) {
	root, err := fs.Sub(embedded, path.Join("data", region))
	if err != nil {
		return nil, err
	}
	r := &Region{Name: region, Classes: map[string]*Class{}}
	if err := readJSON(root, "constants.json", &r.Constants); err != nil {
		return nil, err
	}
	var targets struct {
		Targets []Target `json:"targets"`
	}
	if err := readJSON(root, "targets.json", &targets); err != nil {
		return nil, err
	}
	r.Targets = targets.Targets

	models, err := fs.Glob(root, "classes/*.json")
	if err != nil {
		return nil, err
	}
	for _, m := range models {
		slug := strings.TrimSuffix(path.Base(m), ".json")
		c := &Class{Slug: slug, Model: &ClassModel{}, Skills: &SkillFile{}, Boards: &BoardFile{}}
		if err := readJSON(root, m, c.Model); err != nil {
			return nil, err
		}
		if err := readJSON(root, "skills/"+slug+".json", c.Skills); err != nil {
			return nil, err
		}
		if err := readJSON(root, "boards/"+slug+".json", c.Boards); err != nil {
			return nil, err
		}
		c.index()
		r.Classes[slug] = c
	}
	return r, nil
}

func readJSON(root fs.FS, name string, v any) error {
	b, err := fs.ReadFile(root, name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (c *Class) index() {
	c.skillByID = map[int]*RawSkill{}
	c.skillByName = map[string]*RawSkill{}
	for i := range c.Skills.Skills {
		s := &c.Skills.Skills[i]
		if s.ID != 0 {
			c.skillByID[s.ID] = s
		}
		c.skillByName[strings.ToLower(s.Name)] = s
	}
	c.nodeByID = map[int]*RawNode{}
	for bi := range c.Boards.Boards {
		for ni := range c.Boards.Boards[bi].Nodes {
			n := &c.Boards.Boards[bi].Nodes[ni]
			c.nodeByID[n.ID] = n
		}
	}
}

// Skill finds a skill by ID, falling back to its name (a few imported skills
// have no ID).
func (c *Class) Skill(id int, name string) *RawSkill {
	if s, ok := c.skillByID[id]; ok {
		return s
	}
	return c.skillByName[strings.ToLower(name)]
}

// Node finds a Daevanion node by ID.
func (c *Class) Node(id int) *RawNode { return c.nodeByID[id] }

// ClassForName maps the API's className ("Spiritmaster") to a class slug.
func (r *Region) ClassForName(className string) (*Class, bool) {
	c, ok := r.Classes[strings.ToLower(strings.ReplaceAll(className, " ", ""))]
	return c, ok
}

// Target finds a target profile by ID.
func (r *Region) Target(id string) (Target, bool) {
	for _, t := range r.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return Target{}, false
}

// At returns the tooltip for a skill level, clamped to the known range.
func (s *RawSkill) At(level int) RawLevel {
	if len(s.Levels) == 0 {
		return RawLevel{}
	}
	i := min(max(level, 1), len(s.Levels)) - 1
	return s.Levels[i]
}

// Part is one "<flat> (+<pct>% Attack)" damage or healing amount in a tooltip.
type Part struct {
	Flat float64 `json:"flat"`
	Coef float64 `json:"coef"` // fraction of Attack (1.2 for +120%)
}

var partRe = regexp.MustCompile(`([\d,]+(?:\.\d+)?) \(\+([\d.]+)% Attack\)`)

// Parts extracts every damage/heal amount from tooltip text, in order.
func Parts(text string) []Part {
	var out []Part
	for _, m := range partRe.FindAllStringSubmatch(text, -1) {
		out = append(out, Part{Flat: num(m[1]), Coef: num(m[2]) / 100})
	}
	return out
}

// Capture returns the first submatch of pattern in text as a number.
func Capture(text, pattern string) (float64, bool) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, false
	}
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, false
	}
	return num(m[1]), true
}

// Seconds parses "15s" style durations.
func Seconds(s *string) float64 {
	if s == nil {
		return 0
	}
	return num(strings.TrimSuffix(strings.TrimSpace(*s), "s"))
}

// Number parses a tooltip number ("1,148", "150").
func Number(s *string) float64 {
	if s == nil {
		return 0
	}
	return num(*s)
}

func num(s string) float64 {
	v, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", ""), 64)
	return v
}
