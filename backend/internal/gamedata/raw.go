// Package gamedata holds versioned, static game data (skills, class models,
// Daevanion node values, targets) embedded into the binary, plus the parsers
// that turn raw tooltip text into numbers.
package gamedata

// SkillFile is one class's imported skill tooltips (data/<region>/skills/<class>.json).
type SkillFile struct {
	Class     string     `json:"class"`
	Source    string     `json:"source"`
	FetchedAt string     `json:"fetchedAt"`
	Skills    []RawSkill `json:"skills"`
}

type RawSkill struct {
	ID              int                 `json:"id"`
	Slug            string              `json:"slug"`
	Name            string              `json:"name"`
	Category        string              `json:"category"` // active, passive, stigma
	UnlockLevel     int                 `json:"unlockLevel"`
	MaxLevel        int                 `json:"maxLevel"`
	Levels          []RawLevel          `json:"levels"`
	Specializations []RawSpecialization `json:"specializations,omitempty"`
}

type RawLevel struct {
	Level     int     `json:"level"`
	Text      string  `json:"text"`
	Cooldown  *string `json:"cooldown"`
	MP        *string `json:"mp"`
	NeedLevel *string `json:"needLevel"`
}

type RawSpecialization struct {
	Index         int    `json:"index"`
	MinSkillLevel int    `json:"minSkillLevel"`
	Text          string `json:"text"`
}

// BoardFile is one class's Daevanion boards (data/<region>/boards/<class>.json).
// Percentage stats are in basis points (150 = 1.5%), as in the client.
type BoardFile struct {
	Class      string            `json:"class"`
	Source     string            `json:"source"`
	FetchedAt  string            `json:"fetchedAt"`
	Boards     []RawBoard        `json:"boards"`
	StatLabels map[string]string `json:"statLabels"`
}

type RawBoard struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	NeedLevel int       `json:"needLevel"`
	Nodes     []RawNode `json:"nodes"`
}

type RawNode struct {
	ID          int       `json:"id"`
	Row         int       `json:"row"`
	Col         int       `json:"col"`
	Type        string    `json:"type"` // Start, Stat, SkillLevel
	Grade       *string   `json:"grade"`
	Cost        int       `json:"cost"`
	NeedLevel   int       `json:"needLevel"`
	Auto        bool      `json:"auto"`
	Label       string    `json:"label"`
	Stats       []RawStat `json:"stats"`
	SkillID     *int      `json:"skillId"`
	SkillLevels int       `json:"skillLevels"`
}

type RawStat struct {
	Stat  string  `json:"stat"`
	Value float64 `json:"value"`
}
