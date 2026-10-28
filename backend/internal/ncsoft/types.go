// Package ncsoft talks to the JSON endpoints behind NCSOFT's official AION 2
// character site (aion2.plaync.com). The API is unofficial and undocumented;
// the structs below mirror the responses as observed in October 2026.
package ncsoft

// SearchResult is one row of the character search.
type SearchResult struct {
	CharacterID string `json:"characterId"`
	Name        string `json:"name"`
	Race        int    `json:"race"`
	PCID        int    `json:"pcId"`
	Level       int    `json:"level"`
	ServerID    int    `json:"serverId"`
	ServerName  string `json:"serverName"`
	Region      string `json:"region"`
	ProfileImg  string `json:"profileImageUrl"`
}

type SearchResponse struct {
	List       []SearchResult `json:"list"`
	Pagination struct {
		Page    int `json:"page"`
		Size    int `json:"size"`
		Total   int `json:"total"`
		EndPage int `json:"endPage"`
	} `json:"pagination"`
}

type Profile struct {
	CharacterName  string `json:"characterName"`
	CharacterLevel int    `json:"characterLevel"`
	ClassName      string `json:"className"`
	ServerName     string `json:"serverName"`
	CombatPower    int    `json:"combatPower"`
	TitleName      string `json:"titleName"`
	RaceName       string `json:"raceName"`
	RaceID         int    `json:"raceId"`
	ProfileImage   string `json:"profileImage"`
	PCID           int    `json:"pcId"`
	ServerID       int    `json:"serverId"`
	CharacterID    string `json:"characterId"`
	RegionName     string `json:"regionName"`
}

type Stat struct {
	Type           string   `json:"type"`
	Name           string   `json:"name"`
	Value          int      `json:"value"`
	StatSecondList []string `json:"statSecondList"`
}

type Title struct {
	Name          string `json:"name"`
	EquipCategory string `json:"equipCategory"`
	Grade         string `json:"grade"`
	EquipStatList []struct {
		Desc string `json:"desc"`
	} `json:"equipStatList"`
}

type Board struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Open           int    `json:"open"`
	OpenNodeCount  int    `json:"openNodeCount"`
	OpenPercent    int    `json:"openPercent"`
	TotalNodeCount int    `json:"totalNodeCount"`
	Icon           string `json:"icon"`
}

// InfoResponse is /api/character/info.
type InfoResponse struct {
	Profile Profile `json:"profile"`
	Stat    struct {
		StatList []Stat `json:"statList"`
	} `json:"stat"`
	Title struct {
		TitleList []Title `json:"titleList"`
	} `json:"title"`
	Daevanion struct {
		BoardList []Board `json:"boardList"`
	} `json:"daevanion"`
}

type EquipmentSlot struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Grade        string `json:"grade"`
	EnchantLevel int    `json:"enchantLevel"`
	ExceedLevel  int    `json:"exceedLevel"`
	SlotPos      int    `json:"slotPos"`
	SlotPosName  string `json:"slotPosName"`
	Icon         string `json:"icon"`
}

type Skill struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"` // Active, Passive, Dp
	SkillLevel int    `json:"skillLevel"`
	Equip      int    `json:"equip"`
	Acquired   int    `json:"acquired"`
	NeedLevel  int    `json:"needLevel"`
	Icon       string `json:"icon"`
}

type PetWing struct {
	Pet *struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Level int    `json:"level"`
	} `json:"pet"`
	Wing *struct {
		ID           *int    `json:"id"`
		Name         *string `json:"name"`
		Grade        *string `json:"grade"`
		EnchantLevel *int    `json:"enchantLevel"`
	} `json:"wing"`
}

// EquipmentResponse is /api/character/equipment.
type EquipmentResponse struct {
	Equipment struct {
		EquipmentList []EquipmentSlot `json:"equipmentList"`
	} `json:"equipment"`
	Skill struct {
		SkillList []Skill `json:"skillList"`
	} `json:"skill"`
	PetWing PetWing `json:"petwing"`
}

type ItemStat struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	MinValue string `json:"minValue,omitempty"`
	Extra    string `json:"extra,omitempty"`
	Exceed   bool   `json:"exceed,omitempty"`
	Grade    string `json:"grade,omitempty"`
	SlotPos  int    `json:"slotPos,omitempty"`
}

type ItemSkill struct {
	ID    int    `json:"id"`
	Level int    `json:"level"`
	Name  string `json:"name"`
}

// ItemResponse is /api/character/equipment/item: the rolled item in one slot.
type ItemResponse struct {
	Name                  string      `json:"name"`
	Grade                 string      `json:"grade"`
	Level                 int         `json:"level"`
	EnchantLevel          int         `json:"enchantLevel"`
	MaxEnchantLevel       int         `json:"maxEnchantLevel"`
	MaxExceedEnchantLevel *int        `json:"maxExceedEnchantLevel"`
	CategoryName          string      `json:"categoryName"`
	Type                  string      `json:"type"`
	MagicStoneSlotCount   *int        `json:"magicStoneSlotCount"`
	SoulBindRate          string      `json:"soulBindRate"`
	MainStats             []ItemStat  `json:"mainStats"`
	SubStats              []ItemStat  `json:"subStats"`
	MagicStoneStat        []ItemStat  `json:"magicStoneStat"`
	SubSkills             []ItemSkill `json:"subSkills"`
	GodStoneStat          []GodStone  `json:"godStoneStat"`
	Set                   *ItemSet    `json:"set"`
}

// GodStone is a Theostone engraved on a weapon. Its effect is free text.
type GodStone struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Grade string `json:"grade"`
}

type ItemSet struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	EquippedCount int    `json:"equippedCount"`
	Bonuses       []struct {
		Degree       int      `json:"degree"`
		Descriptions []string `json:"descriptions"`
	} `json:"bonuses"`
}

type DaevanionNode struct {
	BoardID int    `json:"boardId"`
	NodeID  int    `json:"nodeId"`
	Type    string `json:"type"` // Stat, SkillLevel, Start, None
	Grade   string `json:"grade"`
	Open    int    `json:"open"`
	Name    string `json:"name"`
}

// DaevanionResponse is /api/character/daevanion/detail for one board.
type DaevanionResponse struct {
	NodeList []DaevanionNode `json:"nodeList"`
}

type PCData struct {
	ClassName string `json:"className"`
	ClassText string `json:"classText"`
	Gender    string `json:"genderName"`
	ID        int    `json:"id"`
	RaceName  string `json:"raceName"`
}

type Server struct {
	RaceID     int    `json:"raceId"`
	ServerID   int    `json:"serverId"`
	ServerName string `json:"serverName"`
	ShortName  string `json:"serverShortName"`
}
