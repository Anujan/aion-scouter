// Package stats rebuilds a character's final stat sheet from the pieces the
// official API exposes (gear, Daevanion nodes, titles, set bonuses, primary and
// god stats). NCSOFT does not publish the in-game stat window, so every value
// here is a sum we compute, and each keeps a record of where it came from.
package stats

import "strings"

// Key is a stat identifier. Keys reuse the game client's own stat IDs (as seen
// in item and Daevanion data) so data files map onto them without translation.
type Key string

const (
	WeaponMinAttack Key = "WeaponMinAttack" // main-hand weapon min attack (incl. enchant)
	WeaponMaxAttack Key = "WeaponMaxAttack" // main-hand weapon max attack (incl. enchant)
	Attack          Key = "WeaponFixingDamage"
	AttackBonus     Key = "FixingDamage"
	AttackIncrease  Key = "DamageRatio" // "Attack increase %"
	WeaponDmgBoost  Key = "AmplifyWeaponDamage"
	PvEAttack       Key = "PvEAddDamage"
	BossAttack      Key = "BossNpcAddDamage"
	Penetration     Key = "DefensePierce"
	DamageBoost     Key = "AmplifyAllDamage"
	PvEDamageBoost  Key = "PvEAmplifyDamage"
	BossDamageBoost Key = "BossNpcAmplifyDamage"
	Critical        Key = "Critical"
	CriticalRatio   Key = "CriticalRatio" // "Critical Hit increase %" (multiplies rating)
	CritDmgBoost    Key = "AmplifyCriticalDamage"
	DoubleChance    Key = "HardHit"
	PerfectChance   Key = "Perfect"
	MultiHitChance  Key = "AdditionalHitRate"
	CombatSpeed     Key = "CombatSpeed"
	CooldownRed     Key = "CoolTimeDecrease"
	PowerShardBonus Key = "SealStoneAddDamage"
	Accuracy        Key = "WeaponAccuracy"
	AccuracyRatio   Key = "AccuracyRatio"
	MPMax           Key = "MPMax"
	HPMax           Key = "HPMax"
	Defense         Key = "Defense"
	ItemLevel       Key = "ItemLevel"
)

// Info describes how a key is displayed and combined.
type Info struct {
	Label string `json:"label"`
	Pct   bool   `json:"pct"`   // value is a percentage (stored as 1.5 for 1.5%)
	Group string `json:"group"` // offense, utility, defense, other
}

// Registry lists every key we understand. Unknown keys still sum, they just
// display with their raw ID.
var Registry = map[Key]Info{
	WeaponMinAttack: {"Weapon Min Attack", false, "offense"},
	WeaponMaxAttack: {"Weapon Max Attack", false, "offense"},
	Attack:          {"Attack", false, "offense"},
	AttackBonus:     {"Attack Bonus", false, "offense"},
	AttackIncrease:  {"Attack Increase", true, "offense"},
	WeaponDmgBoost:  {"Weapon Damage Boost", true, "offense"},
	PvEAttack:       {"PvE Attack", false, "offense"},
	BossAttack:      {"Boss Attack", false, "offense"},
	Penetration:     {"Penetration", false, "offense"},
	DamageBoost:     {"Damage Boost", true, "offense"},
	PvEDamageBoost:  {"PvE Damage Boost", true, "offense"},
	BossDamageBoost: {"Boss Damage Boost", true, "offense"},
	Critical:        {"Critical Hit", false, "offense"},
	CriticalRatio:   {"Critical Hit Increase", true, "offense"},
	CritDmgBoost:    {"Critical Damage Boost", true, "offense"},
	DoubleChance:    {"Double Chance", true, "offense"},
	PerfectChance:   {"Perfect Chance", true, "offense"},
	MultiHitChance:  {"Multi-hit Chance", true, "offense"},
	CombatSpeed:     {"Combat Speed", true, "utility"},
	CooldownRed:     {"Cooldown Reduction", true, "utility"},
	PowerShardBonus: {"Power Shard Damage Bonus", false, "offense"},
	Accuracy:        {"Accuracy", false, "utility"},
	AccuracyRatio:   {"Accuracy Increase", true, "utility"},
	MPMax:           {"MP", false, "utility"},
	HPMax:           {"HP", false, "defense"},
	Defense:         {"Defense", false, "defense"},
	ItemLevel:       {"Item Level", false, "other"},

	"PvPAmplifyDamage":       {"PvP Damage Boost", true, "pvp"},
	"PvPAddDamage":           {"PvP Attack", false, "pvp"},
	"PvPCritical":            {"PvP Critical Hit", false, "pvp"},
	"PvPAccuracy":            {"PvP Accuracy", false, "pvp"},
	"PvEAccuracy":            {"PvE Accuracy", false, "utility"},
	"DecreaseDamage":         {"Damage Tolerance", true, "defense"},
	"PvEDecreaseDamage":      {"PvE Damage Tolerance", true, "defense"},
	"BossNpcDecreaseDamage":  {"Boss Damage Tolerance", true, "defense"},
	"DecreaseCriticalDamage": {"Critical Damage Tolerance", true, "defense"},
	"AdditionalHitResistRate": {
		"Multi-hit Resist", true, "defense",
	},
	"CriticalResist":     {"Critical Hit Resist", false, "defense"},
	"AbnormalAccuracy":   {"Status Effect Chance", true, "utility"},
	"AbnormalResistance": {"Status Effect Resist", true, "defense"},
	"AmplifyFrontAttack": {"Front Attack Damage Boost", true, "offense"},
	"AmplifyBackAttack":  {"Back Attack Damage Boost", true, "offense"},
	"MoveSpeed":          {"Move Speed", true, "utility"},
	"Block":              {"Block", false, "defense"},
	"Evasion":            {"Evasion", false, "defense"},
	"IgnoreIronWall":     {"Endurance Penetration", true, "offense"},
	"MPCostReduction":    {"MP Cost Reduction", true, "utility"},

	"EvasionRatio":             {"Evasion Increase", true, "defense"},
	"BlockRatio":               {"Block Increase", true, "defense"},
	"CriticalResistRatio":      {"Critical Hit Resist Increase", true, "defense"},
	"HPRatio":                  {"HP Increase", true, "defense"},
	"MPRatio":                  {"MP Increase", true, "utility"},
	"DefenseRatio":             {"Defense Increase", true, "defense"},
	"Restoration":              {"Regeneration", true, "defense"},
	"RestorationPierce":        {"Regeneration Penetration", true, "offense"},
	"HardHitResist":            {"Double Resist", true, "defense"},
	"PerfectResist":            {"Perfect Resist", true, "defense"},
	"IronWall":                 {"Endurance", true, "defense"},
	"StaminaMax":               {"Stamina", false, "defense"},
	"HPRegen":                  {"Natural HP Regen", false, "defense"},
	"HpPotionRate":             {"HP Potion Recovery", false, "defense"},
	"PvEDamageDefense":         {"PvE Defense", false, "defense"},
	"BackAttackDecreaseDamage": {"Back Attack Damage Tolerance", true, "defense"},
}

// IsPct reports whether k is a percentage stat.
func IsPct(k Key) bool { return Registry[k].Pct }

// Label returns the display label for k.
func Label(k Key) string {
	if i, ok := Registry[k]; ok {
		return i.Label
	}
	return string(k)
}

// aliases maps item stat IDs that mean the same thing as a canonical key.
var aliases = map[Key]Key{
	"ArmorDefense":   Defense,
	"ArmorEvasion":   "Evasion",
	"WeaponAccuracy": Accuracy,
}

// Canonical folds item-specific stat IDs onto canonical keys.
func Canonical(k Key) Key {
	if a, ok := aliases[k]; ok {
		return a
	}
	return k
}

// labelKeys maps the English labels used in free-text bonuses (titles, set
// effects, primary/god stat lines) to keys. Matching is case-insensitive and
// prefers the longest label, so "PvE Damage Boost" wins over "Damage Boost".
var labelKeys = map[string]Key{
	"attack":                       Attack,
	"attack bonus":                 AttackBonus,
	"attack increase":              AttackIncrease,
	"weapon damage boost":          WeaponDmgBoost,
	"pve attack":                   PvEAttack,
	"boss attack":                  BossAttack,
	"penetration":                  Penetration,
	"damage boost":                 DamageBoost,
	"pve damage boost":             PvEDamageBoost,
	"boss damage boost":            BossDamageBoost,
	"critical hit":                 Critical,
	"critical hit increase":        CriticalRatio,
	"critical damage boost":        CritDmgBoost,
	"double chance":                DoubleChance,
	"perfect chance":               PerfectChance,
	"multi-hit chance":             MultiHitChance,
	"combat speed":                 CombatSpeed,
	"cooldown reduction":           CooldownRed,
	"cooldown":                     CooldownRed, // god stat line "Cooldown -0.5%" (sign flipped below)
	"accuracy":                     Accuracy,
	"accuracy bonus":               Accuracy,
	"accuracy increase":            AccuracyRatio,
	"mp":                           MPMax,
	"hp":                           HPMax,
	"defense bonus":                Defense,
	"defense":                      Defense,
	"pvp damage boost":             "PvPAmplifyDamage",
	"pvp attack":                   "PvPAddDamage",
	"damage tolerance":             "DecreaseDamage",
	"pve damage tolerance":         "PvEDecreaseDamage",
	"boss damage tolerance":        "BossNpcDecreaseDamage",
	"critical damage tolerance":    "DecreaseCriticalDamage",
	"critical hit resist":          "CriticalResist",
	"status effect chance":         "AbnormalAccuracy",
	"status effect resist":         "AbnormalResistance",
	"move speed":                   "MoveSpeed",
	"block":                        "Block",
	"evasion":                      "Evasion",
	"mp cost":                      "MPCostReduction",
	"endurance penetration":        "IgnoreIronWall",
	"power shard damage bonus":     PowerShardBonus,
	"front attack damage boost":    "AmplifyFrontAttack",
	"back attack damage boost":     "AmplifyBackAttack",
	"pve accuracy":                 "PvEAccuracy",
	"back attack damage tolerance": "BackAttackDecreaseDamage",
	"evasion increase":             "EvasionRatio",
	"block increase":               "BlockRatio",
	"critical hit resist increase": "CriticalResistRatio",
	"hp increase":                  "HPRatio",
	"mp increase":                  "MPRatio",
	"defense increase":             "DefenseRatio",
	"regeneration":                 "Restoration",
	"regeneration penetration":     "RestorationPierce",
	"double resist":                "HardHitResist",
	"perfect resist":               "PerfectResist",
	"endurance":                    "IronWall",
	"stamina":                      "StaminaMax",
	"natural hp regen":             "HPRegen",
	"hp potion recovery":           "HpPotionRate",
	"pve defense":                  "PvEDamageDefense",
}

// KeyForLabel resolves a free-text label to a key.
func KeyForLabel(label string) (Key, bool) {
	k, ok := labelKeys[strings.ToLower(strings.TrimSpace(label))]
	return k, ok
}
