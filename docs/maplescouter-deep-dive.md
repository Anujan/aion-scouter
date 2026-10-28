# MapleScouter Deep Dive → Aion Scouter Plan

_Research date: 2026-10-28._

This document covers three things:

1. How MapleScouter (maplescouter.com) works, and why players trust it.
2. What Aion 2 gives us to build the same thing, using real API responses and the community damage model.
3. A concrete architecture and roadmap for Aion Scouter, ordered by the four goals:
   1. Estimated DPS
   2. Skill and Daevanion board optimizer
   3. Equipment comparison
   4. Estimated % of a boss

> **Confidence note.** MapleScouter is closed-source and sits behind a Cloudflare bot
> challenge, so its live site could not be read directly. Everything about its internals
> comes from its public disclaimers, community write-ups, and one project that
> reverse-engineered it by probing its outputs (MapleDoro, PR #183). The confidence of each
> claim is marked as **[confirmed]**, **[reverse-engineered]** or **[inferred]**.

---

## Part 1: How MapleScouter works

### 1.1 What it is

MapleScouter is an unofficial MapleStory spec analyzer. You type a character name and it shows:

| Output | What it means |
|---|---|
| **Combat Power (전투력)** | The official in-game number, read from Nexon's API. |
| **Converted main stat (환산 주스탯)** | Every damage stat (boss %, IED, crit damage, ATT…) collapsed into one "equivalent main stat" number, so different builds can be compared on a single axis. **[confirmed concept]** |
| **HEXA score (헥사 환산)** | The converted number with 6th-job HEXA skill levels included. It is the GMS community's de facto "how strong are you" number, used for party recruiting. **[confirmed]** |
| **Boss clear % (보스 컷)** | For each boss: the % of the boss's HP you can deal in its time limit, solo and in a party. **[confirmed]** |
| **Stat efficiency table** | How much 1% boss damage, 1% crit damage, 1% IED and so on are worth in main-stat terms. **[confirmed concept]** |
| **Equipment / spec simulator** | Edit gear, potentials and stars, toggle buffs, and see every number above change. **[confirmed]** |

Its own disclaimers say:

- Combat power is computed against a **300% defense training dummy (허수아비)** baseline.
- It uses the **displayed IED, not real IED**, so IED can be overvalued.
- Boss and hunting cut-offs **"may be inaccurate"**.

**[confirmed, from the site's disclaimer text as quoted by search indexes]**

### 1.2 The data layer: Nexon Open API

MapleScouter depends entirely on Nexon's **official** Open API (`open.api.nexon.com`, launched for KMS in 2024).

| Step | Endpoint |
|---|---|
| Name → `ocid` | `/maplestory/v1/id` |
| Basic info | `/character/basic` (class, level, world) |
| **Final stats** | `/character/stat`: final, already-summed stats including Combat Power, boss %, IED, final damage, crit damage and cooldown reduction |
| Gear | `/character/item-equipment` (per-item potentials, stars, flames), `/character/set-effect` |
| Skills | `/character/skill`, `/character/link-skill`, `/character/hexamatrix`, `/character/hexamatrix-stat` |
| Everything else | Symbols, union, ability, hyper stats, and more |

**Why this matters.** Nexon hands MapleScouter the **final stat window**, already summed. MapleScouter never has to rebuild a character's stats from gear, so its hard problem is only "stats → damage". **Aion 2 does not give us this** (see §2.1). That is the biggest structural difference between the two projects.

Known limitations:

- The API refreshes only when the player changes channel or logs out.
- Snapshots lag what the player sees in game.

### 1.3 The damage model

The core loop has three steps:

```
character stats ──► per-class damage function ──► "damage vs. reference target"
                                                     │
                     invert around main stat ◄───────┘   (= converted main stat)
```

1. **Multiplicative stat buckets [confirmed concept, reverse-engineered shape].** MapleStory damage is a product of independent groups:

   ```
   damage ∝ (4·MainStat + SubStat) · ATT · (1 + ATT%) · (1 + Dmg% + Boss%) · (1 + FD%)
            · (1.35 + CritDmg%) · (1 − Def·(1 − IED)) · HexaCoreMultipliers · …
   ```

   MapleDoro's fitted engine models MapleScouter's output as **"a per-class scale × a product of groups"**: stat, flat ATT, ATT%, damage, crit damage, defense, HEXA cores, and a remainder. Each group starts from a **hidden per-class baseline** (the class's own passives and buffs) plus the sheet's inputs. **[reverse-engineered]**

2. **Per-class skill cycle [inferred, strongly supported].** The class baseline and "remainder" constants come from a modeled **burst rotation**. Since the April 2025 patch every class runs a 2-minute burst cycle. The model includes:

   - cooldown reduction, which MapleScouter adds to its HEXA figure;
   - buff uptime;
   - the damage share of each skill, so HEXA core levels are weighted by how much of the cycle each skill makes up.

   HEXA cores follow per-skill level curves, with stepped bonuses at specific levels taken from the wiki's skill tables.

3. **Reference target.** All headline numbers are computed against a fixed dummy with 300% defense, later 380% for "Boss 380" HEXA. This makes them comparable across players. Defense enters as `1 − Def × (1 − IED)`.

4. **Converted stat = inversion.** Find the main-stat value K such that a character with K extra main stat (and nothing else changed) deals the same damage as the real one. That gives one comparable number. The marginal version, "how much main stat is 1% boss worth?", produces the stat-efficiency table.

### 1.4 Boss clear %

```
clear% = (character DPS vs. that boss's defense) × fight time ÷ boss HP     [per boss, solo/party]
```

- **Boss data:** HP, defense, time limit, and level/force requirements. Below-requirement penalties on level gap or Arcane/Authentic Force are applied as multipliers.
- **HEXA requirement:** MapleScouter scales the character's damage until a boss reads exactly 100%. That yields a "HEXA needed for this boss", which it shows on each boss tile. **[reverse-engineered]**
- **Party mode:** A party clear % uses a party-size assumption. Community guides describe the numbers as "the % of HP cut within a 30-minute window". **[confirmed in community guides]**
- **Calibration:** Clear cut-offs are **tuned against real clears**. A MapleScouter cut update moved every class by 0.1 to 0.3. The MapleDoro work shows the community validates against video-logged clears, per class and per boss. **[reverse-engineered]**

### 1.5 Why it won

| Factor | Why it matters |
|---|---|
| **Zero input** | A name is enough. The official API does the work. |
| **One number** | HEXA score became the social currency for recruiting and comparing players. |
| **Answers the real question** | "Can I clear boss X?" matters more to players than "what is my damage?". |
| **What-if simulator** | "Is this upgrade worth it?" is the most common question in a gear-progression game. |
| **Honest disclaimers** | Players trust a tool that tells them where it is wrong. |
| **Calibrated to reality** | Theory is fitted to real kills, so the numbers match player experience. |
| **Ecosystem** | Others build on it: MapleDoro reimplements its engine, MemoFam scrapes its result page, and planners import or export its files. A shareable, stable result format pays off. |

---

## Part 2: What Aion 2 gives us

### 2.1 Character data (unofficial API) — a real response was examined

There is **no official Aion 2 developer API**. NCSOFT's own character page (`aion2.plaync.com`) calls JSON endpoints that the community has documented (the `nuriland/aion2-api` Go client, and `nklsgod/AION-2-Character-Check`).

| Purpose | Endpoint (params) |
|---|---|
| Search | `GET https://api-search.plaync.com/aion2global/search/v2/character?keyword=&page=&size=&region=&race=&localeInfo=en-US` |
| Profile + stats + titles + boards summary | `GET https://aion2.plaync.com/api/character/info?lang=en-US&region=&serverId=&characterId=` |
| Equipment + skills + pet/wing | `GET …/api/character/equipment?…` |
| One item's full rolls | `GET …/api/character/equipment/item?…&id=&enchantLevel=&slotPos=` |
| One Daevanion board, node by node | `GET …/api/character/daevanion/detail?…&boardId=` |

**Regions:**

- Global has five shards: `eu`, `nae`, `naw`, `la`, `as`.
- KR and TW have their own hosts.
- KR returns 404 to requests from outside Korea.

**Operational constraints:**

- Browsers cannot call these endpoints directly because of CORS, so we need a server-side proxy.
- The endpoints are unofficial and field names can change.
- Existing projects throttle to about 4 requests per second and cache for 5–10 minutes. One full character load is about 30 requests: info, equipment, about 25 item calls and about 5 board calls.

#### What a real character response contains

Data below is from an EU Cleric snapshot taken 2026-10-26.

- **`character.profile`:** name, level, class, server, race, title, and `combatPower` (the official number).
- **`character.stat.statList`:** **only 17 entries**, as `{type, name, value, statSecondList}`:
  - **Six primary stats:** Might (`STR`), Dexterity (`DEX`), Intelligence (`INT`), Constitution (`CON`), Precision (`AGI`), Willpower (`WIS`). Each comes with its derived % lines, for example Might 2 → "Attack increase +0.2%".
  - **Ten god stats:** Justice, Freedom, Illusion, Life, Time, Destruction, Death, Wisdom, Destiny, Space. Again with their derived lines, for example Time 62 → "Combat Speed +6.2%, Double Resist +6.2%".
  - **`ItemLevel`.**
- ⚠️ **Missing:** total Attack, Critical Hit, Damage Boost, Weapon Damage Boost, Crit Damage, Multi-Hit, Penetration, PvE/Boss Damage, Combat Speed, Cooldown Reduction, and so on. **Unlike Nexon's API, NCSOFT gives no final combat stat window.**
- **`character.title`:** owned titles, with the equipped ones' `equipStatList`, for example `"Cooldown Reduction +3%"`.
- **`character.daevanion.boardList`:** per board, `{id, name, openNodeCount, totalNodeCount}`. Board IDs: 71 Nezekan, 72 Zikel, 73 Vaizel, 74 Triniel, 76 Azphel (PvP); 75 is presumably Ariel.
- **`daevanion/detail`:** every node as `{nodeId, type: Stat|SkillLevel|Start|None, grade, open, name}`.
  - Names look like "Attack", "Critical Hit", "Damage Boost" or "Skill Level Up - Judgment Thunder".
  - ⚠️ **Node values and adjacency are not included.** Both must come from our own static data.
- **`equipment.equipmentList`:** 25 slots, from MainHand through Arcana5. Each has `{id, name, grade, enchantLevel, exceedLevel, slotPosName}`.
- **`equipment.skill.skillList`:** every skill as `{id, name, category: Active|Passive|Dp, skillLevel, equip, acquired}`. `Dp` is the stigma/DP skill category. `equip` shows which are slotted.
- **`equipment.petwing`:** the pet and wing, with wing enchant level.
- **Item detail:**
  - `mainStats[]` as `{id, value, extra, minValue, exceed}`. `extra` is the enchant bonus; weapons have `minValue`/`value` as min and max Attack.
  - `subStats[]` (random rolls), `magicStoneStat[]` (manastones), `subSkills[]` (skill-level lines), `soulBindRate`, `maxEnchantLevel`, `maxExceedEnchantLevel`.
  - Stat IDs are stable internal keys, which is good for mapping: `WeaponFixingDamage` (Attack), `AmplifyWeaponDamage` (Weapon Damage Boost), `PvEAmplifyDamage`, `DefensePierce` (Penetration), `AdditionalHitRate` (Multi-hit), `Perfect`, `CombatSpeed`, `Critical`, `SealStoneAddDamage` (Power Shard bonus), and so on.

**Consequence:** Aion Scouter must **rebuild the final stat sheet** by summing:

- gear (main, enchant `extra`, exceed, sub-stats, manastones);
- set effects;
- Daevanion nodes;
- equipped titles;
- god stats and primary stats (the derived % lines are given to us);
- wings, pets and arcana;
- class passives, and buffs as a scenario toggle.

This is the single biggest piece of new work compared with MapleScouter. It is also a moat: the site that gets it right owns the space.

**Validation.** There is no official final-stat number to check against. Two ways to check the rebuild:

1. **(Preferred)** Let users enter, or OCR a screenshot of, their in-game stat window once. Diff it against our rebuild to find missing sources. Collect these diffs, anonymized, to fix the model for everyone.
2. Use `combatPower` as a weak sanity signal. Its formula is unknown and known to be buggy (penetration was over-weighted, as NCSOFT acknowledged), but it correlates at about 0.95–0.97 with item level and nodes opened.

### 2.2 The damage formula (community model, KR/TW-derived)

This is the most widely cited PvE model (gameplay.tips, cross-checked with aion2.akunito.com and the BUSON2 / 딜 효율 계산기 tools):

```
Base   = [ ( (PureAttack × WeaponDmgBoost) × MultiHit × PowerShard + AttackBonus ) × AttackIncrease%
           + PvEAttack + BossAttack + SpeciesAttack + FrontBackAttack ]
Hit    = Base × SkillCoefficient + SkillFlatDamage − (Defense − Penetration) × 0.1      (PvP: × 0.01)
Final  = Hit × (1 + DamageBoost + PvEDamageBoost + BossDamageBoost + SpeciesDamageBoost − TargetTolerance)
             × CritMult × DoubleMult × FrontBackDmgMult × SkillSpecialty
```

**Expected-value terms per hit:**

- **Crit:**
  - Crit chance comes from (Crit rating − target Crit Resist). 100 points = 10%, capped at 80%.
  - Crit multiplier = 1.5 + CritDmgBoost − target CritDmgTolerance.
  - Some skills always crit.
- **Double (KR name: Smite / 강타):** ×2 on proc. Bosses from Dying Dramata onward have 30% Double Resist.
- **Perfect:** rolls weapon max Attack instead of a min–max roll.
- **Multi-hit:** 1–4 extra hits at about 5% each. Its place in the formula is still unconfirmed.
- **Front/Back:** weighted by the player's back-attack uptime, a scenario input (about 75% is typical).
- **Block/Parry vs Accuracy:** monsters cannot evade, only block. Back attacks bypass block. A blocked hit deals half damage.
- **Damage Boost vs Tolerance:** they cancel 1:1. Damage, PvE and Boss Damage Boost are **additive with each other**.

**Throughput modifiers (not in the per-hit formula):**

- **Combat Speed:** shortens cast and animation. There are class-specific breakpoints.
- **Cooldown Reduction:** the target band is about 33–36%, with a cap of about 60%.

**What bosses don't have:**

- Front/Back Damage Boost, Weapon Damage Boost, Crit and Multi-Hit. So the matching **tolerances on the player side are worthless in PvE**.
- Endurance Penetration.

**Global vs KR/TW:** values differ. For example, a Daevanion orange node gives 1.5% on Global vs 2.5% on KR, a crit node 5 vs 10, and titles 3.5% vs 5%.
→ **All constants must be versioned per region and per patch.**

### 2.3 Skills

**Sources for skill coefficients:**

- **aion2.gaming.tools:** per skill, "base + % of Attack" per level, cooldown, and shard costs.
- **aion2timers.com/db:** flat value and rate per level, cooldown, MP and range for every active, passive and stigma skill. Taken from Global client text.
- **metabot.gg** and **6sword.com.**

**Skill levels:**

- Skill points raise a skill to level 10.
- The four shared Daevanion boards add +1 each, for a maximum of 14.
- Gear `subSkills` lines and arcana add more.
- **Specialty** options unlock at skill levels 8, 12 and 16; specialty slots unlock at 8, 12 and 20.

**Class roster:** the eight launch classes are Gladiator, Templar, Assassin, Ranger, Sorcerer, Spiritmaster, Cleric and Chanter. Sources disagree on whether a ninth exists, so check this before building.

### 2.4 Daevanion boards (optimizer input)

| Board | Unlock | Cost to fill | Theme (orange corners) |
|---|---|---|---|
| Nezekan | 12 | 134 | Combat Speed ×2, Cooldown Reduction ×2 |
| Zikel | 20 | 134 | Damage Boost ×2, Damage Tolerance ×2 |
| Vaizel | 30 | 134 | Crit Damage Boost ×2, Crit Damage Tolerance ×2 |
| Triniel | 40 | 168 | Multi-Hit Chance ×3, Multi-Hit Resist ×3 |
| Ariel (PvE) | 45 | 232 (own currency) | PvE Damage Boost ×4, PvE Damage Tolerance ×4; Boss Damage Boost nodes |
| Azphel (PvP) | 45 | 232 (own currency) | PvP Damage Boost ×4, PvP Damage Tolerance ×4 |

**Node costs:**

| Node | Cost | Effect |
|---|---|---|
| Grey | 1 | Flat stat |
| Green | 2 | +1 passive skill level |
| Blue | 3 | +1 active skill level |
| Orange | 4 | +1.5% themed stat (Global) |

**Rules:**

- Nodes unlock outward from a central Start node, by adjacency.
- Players earn about 316 shared points through normal play, against a combined cost of 570 for the four shared boards. **So allocation is a real choice, and an optimizer is valuable.**
- Resets go node by node, outside-in, for a Kina cost.
- KR has eight boards (adding Marchutan and Yustiel), and its shared boards cost about 210 each.

The API gives us each board's node list (id, type, grade, name, open). **Board geometry (adjacency) and node values** must be built once, by hand or from client data, and stored as static JSON.

### 2.5 Bosses and ground truth — Aion's big advantage

- Boss HP and stats are **server-side**. No client datamine can give exact HP.
- **However, several open-source DPS meters exist.** They read network packets. Examples: Abyss Logs, aion2t.com, A2Tools, AionFlex, NOIA2, and the KR A2DIL.
- These meters observe **real boss HP** per fight (for example, one Krao Cave run showed 13,908,075 HP).
- They also observe per-skill damage, crit/double/perfect/multi-hit rates, buff uptime, and leaderboards **per boss × class × combat-power bracket**.

MapleScouter had to calibrate against hand-logged videos. **We can calibrate against thousands of parsed fights.** Partnering with one meter (for example Abyss Logs) for aggregate data is the highest-leverage move in this whole plan. We should **not** ship our own packet sniffer, because of ToS and anti-cheat risk.

### 2.6 Existing Aion 2 tools (the competition)

| Tool | What it does | Gap |
|---|---|---|
| BUSON2, 아이온2 딜 효율 계산기 | Manual stat entry → relative stat efficiency | No API import, no skills, no DPS |
| AZANG (aion2zang.info) | Equipment simulator, Daevanion optimizer, stigma statistics | Korean only; method undisclosed |
| AZAS (azas2.com) | Character search, CP simulator | Combat power, not damage |
| Abyss Logs / aion2t / AionFlex | Real parses, leaderboards | Measures what happened, can't answer "what if" |
| Character Check, aion2data, aion2tracker | Armory lookups | No math |

**Nobody combines** API import, a full stat rebuild, a rotation-based DPS model, an optimizer and calibrated boss % in English for Global. **That gap is Aion Scouter.**

---

## Part 3: Aion Scouter design

### 3.1 Architecture

```
┌──────────────┐   ┌───────────────────────────┐   ┌───────────────────────────┐
│ NCSOFT site  │◄──│ Edge proxy (CF Worker)     │◄──│ Web app (Next.js / TS)     │
│ JSON (unoff.)│   │ • throttle ~4 rps/region   │   │ • character view           │
└──────────────┘   │ • cache 10 min (KV)        │   │ • simulator / compare      │
                   │ • normalize → CharacterDTO │   │ • optimizer (Web Worker)   │
                   │ • snapshot → DB (D1/PG)    │   └──────────▲────────────────┘
                   └───────────────────────────┘              │ imports
                                                    ┌──────────┴────────────────┐
                                                    │ @aion-scouter/engine (pure │
                                                    │ TS, no I/O, unit-tested)   │
                                                    │  1 stat rebuild            │
                                                    │  2 damage per hit (EV)     │
                                                    │  3 rotation sim → DPS      │
                                                    │  4 optimizers              │
                                                    │  5 boss %                  │
                                                    └──────────▲────────────────┘
                                                               │ reads
                                                    ┌──────────┴────────────────┐
                                                    │ /data (versioned JSON)     │
                                                    │ region/patch/{skills,      │
                                                    │ boards, items, sets,       │
                                                    │ bosses, constants}.json    │
                                                    └───────────────────────────┘
```

Key principles, each learned from MapleScouter:

1. **The engine is a pure function:**
   `simulate(characterDTO, scenario, gameData) → { stats, perSkill[], dps, bossPct[] }`.
   This gives us:
   - instant what-if in the browser, with no server round trip;
   - testability;
   - re-use by other tools, as MapleScouter got through MapleDoro.
2. **Everything numeric is data.** Constants, skills, node values and boss profiles live in JSON keyed by `region` and `patch`. A patch should mean a data PR, not a code change.
3. **The scenario is explicit and shown to the user:**
   - target: dummy or a named boss;
   - back-attack uptime;
   - party buffs (Chanter, Cleric);
   - consumables;
   - fight length;
   - latency/GCD slack.

   MapleScouter's credibility comes from saying what it assumed.
4. **Show the honesty box.** List what is modeled, what is estimated, and the known error band per class, measured during calibration.

### 3.2 Engine modules

**1. Stat rebuild** — `CharacterDTO → StatSheet`. A table of `{statId: value}` plus provenance: which source contributed what, so the UI can show "Crit 1,230 = gear 610 + boards 300 + manastones 180 + …".

- Map API stat IDs onto a canonical stat enum.
- Add up all the sources in §2.1. Daevanion node values come from `boards.json`.
- Apply class passives at their current skill level.

**2. Hit model** — `(StatSheet, skillRank, target) → E[damage per hit]`.

- Implements §2.2 as expected value: no RNG, so the result is deterministic and fast.
- Exposes **partial derivatives** for the stat-efficiency table: "+1% Damage Boost = +x% DPS = y Attack-equivalent". This is our version of 환산.

**3. Rotation simulator** — `(skills, StatSheet, scenario) → timeline, DPS`.

- Discrete-event simulation over 60–300 s:
  - cooldowns scaled by Cooldown Reduction;
  - cast and animation times scaled by Combat Speed;
  - GCD, buff and debuff windows, charges, proc skills, and DP skills gated by DP generation.
- Per-class **priority lists** stored as data (APL-style, like SimulationCraft), so class experts can contribute rotations without touching engine code.
- Output: DPS, per-skill damage share, and buff uptimes. These are directly comparable to DPS-meter breakdowns, which is what calibration needs.

**4. Optimizers:**

- **Daevanion:** choose a connected node set per board (rooted at Start), within the point budget, to maximize DPS.
  - This is a budget-constrained connected-subgraph problem (prize-collecting Steiner tree). It is NP-hard in general, but boards are only about 225 nodes.
  - Approach:
    1. Linearize: value each node by its marginal DPS from the hit model's derivatives (or a finite difference through the sim for skill-level nodes).
    2. Solve an ILP with flow-based connectivity constraints using HiGHS compiled to WASM, in a Web Worker.
    3. Re-evaluate with the full sim. Re-linearize around the new solution and repeat 2–3 times until it converges, which handles diminishing returns such as the crit cap.
    4. Fall back to a greedy "best value per point including the path to reach it" algorithm with lookahead.
  - Output: the node set, a diff against the current board ("unlock these 14, reset these 3"), and the Kina reset cost.
- **Skills:** stigma and DP choice, skill-point allocation, and specialty picks. The space is small per class, so enumerate or beam-search and score each option with the rotation sim. The skill and board optimizers interact (blue nodes raise skill levels), so run them in alternation: optimize board, then skills, then board again.

**5. Boss %** — `DPS(vs boss profile) × time ÷ HP`.

- Uses a `bosses.json` profile per boss: HP from meter data, Defense, Damage Tolerance, Crit/Double Resist, time limit or enrage, and phase/uptime factor.
- Solo % and party %. Party mode takes a class composition and shares buffs.
- Inverted, it answers MapleScouter's "HEXA needed" question: "Attack-equivalent needed to clear".

### 3.3 Our "one number"

The site needs its own shareable headline number, as HEXA score is for MapleScouter:

- **Recommended:** a **Boss DPS Score** — modeled DPS against a fixed reference boss profile, under a fixed scenario, with no party buffs.
- It must be versioned. It is fine for it to move between patches, as long as we say so.
- Combat Power is not suitable, because players already know it doesn't measure damage.

### 3.4 Calibration plan

1. **Collect.** Build a dataset of (character snapshot, real parse), with consent, through one of:
   - users pasting their Abyss Logs or aion2t fight link next to their character;
   - a data partnership with a meter project.
2. **Compare.** For each pair, compare predicted against observed values for:
   - per-skill damage share;
   - crit, double and perfect rates;
   - DPS.
3. **Fit.** Fit uncertain constants per class: the multi-hit position, the weapon-boost form, the rotation efficiency (human factor), and boss defense/tolerance values.
4. **Publish.** Publish the error per class and keep a regression set. When a patch lands, re-run it and flag classes that drift (MapleDoro's `drift-check` idea).

### 3.5 Roadmap (follows the requested priority)

| Phase | Ships | Needs |
|---|---|---|
| **0. Foundation** | Proxy and cache, search, character page (gear, skills, boards, titles), stat rebuild showing where each stat comes from, and "compare with your in-game stat window" | Endpoint wrapper, canonical stat map, item/set/title/node value data |
| **1. Estimated DPS** ★ | DPS against a dummy and per-skill breakdown for 2 classes (start with the most played DPS, for example Sorcerer and Gladiator), stat-efficiency table, honesty box | Hit model, rotation sim, skill data and priority lists for 2 classes |
| **2. Optimizer** ★ | Daevanion optimizer (ILP + greedy), skill/stigma/specialty optimizer, current-vs-optimal diff | Board geometry and node values, HiGHS-WASM |
| **3. Equipment compare** ★ | Swap or edit any slot (item, enchant, exceed, manastones, sub-stats) and see ΔDPS, Δstats and the "upgrade worth X Attack" figure; A/B builds side by side; shareable link | Mostly UI. The engine already supports it. **Can ship alongside Phase 1 cheaply.** |
| **4. Boss %** ★ | Boss list with solo and party %, needed-score per boss, and Boss DPS Score | Boss profiles and calibration data from meters |
| 5. Scale | Remaining classes, KR/TW data packs, snapshots and progress history, public engine package and API | Contributors and data upkeep |

**Suggested Phase 0–1 issue breakdown:**

1. `proxy/`: Cloudflare Worker with routes `search`, `character`, `equipment`, `item`, `daevanion`. Region map, about 4 rps throttle, KV cache, normalized `CharacterDTO`. Store recorded fixtures for tests.
2. `engine/stats`: canonical stat enum plus the API-ID → enum map. Summation with provenance.
3. `data/global/<patch>/`: schema and first data for items/sets, titles, node values, constants.
4. `engine/hit`: §2.2 formula with expected-value crit, double, perfect, multi-hit and front/back. Unit tests against hand-worked examples from the community guides.
5. `engine/sim`: event-driven rotation with priority-list DSL. Two classes.
6. `web/`: search → character → DPS page. Scenario panel. Honesty box.

### 3.6 Risks and mitigations

| Risk | Mitigation |
|---|---|
| Unofficial API changes or blocks us | Respect rate limits, send a clear User-Agent, cache aggressively, don't crawl. Keep the parser isolated with fixture tests. Have manual stat entry as a fallback. |
| ToS: NCSOFT is not Nexon and offers no developer program | Read-only, on-demand lookups only. No packet capture of our own. Get explicit permission for any meter data. Add a takedown contact. |
| Formula uncertainty (multi-hit placement, weapon boost) | Make each one a constant. Calibrate against parses. Show error bands. |
| Global and KR patches diverge | Version data by `region/patch`, and keep each region's constants separate. |
| Optimizer seems "wrong" because the scenario differs | Show scenario assumptions next to every result. Let users change them. |
| One-person data upkeep burden | Keep skills, rotations and boards as data. Accept community PRs, as MapleDoro's calibration was community work. |

---

## Sources

**MapleScouter and MapleStory**

- MapleDoro "Scouter calibration" PR (reverse-engineering MapleScouter): https://github.com/chablades/mapledoro/pull/183
- MapleScouter disclaimer text (300% dummy, displayed IED, boss-cut accuracy), via search snippets of https://maplescouter.vercel.app/ and maplescouter.com
- maple-scout-worker (MemoFam extension imports Boss380 HEXA and multipliers from MapleScouter): https://github.com/Kimoing/maple-scout-worker
- Boss cut-off "30-minute" interpretation: https://game.theplace.io.kr/maple-hex-stat-calculation-boss-clear-time-guide/
- 환산 주스탯 concept: https://namu.wiki/w/환산%20주스탯 , https://www.fmkorea.com/3259149693
- MapleStory combat power inputs: https://maplestorywiki.net/w/Combat_Power
- Nexon Open API: https://openapi.nexon.com/ , https://developers.nexon.com/Maplestory
- MapleSpecUp methodology notes (boss-condition damage, skill cycle, uptime): https://maplespecup.com/

**Aion 2 data and API**

- nuriland/aion2-api (unofficial Go client): https://github.com/nuriland/aion2-api
- nklsgod/AION-2-Character-Check (endpoints, proxy, real response fixture): https://github.com/nklsgod/AION-2-Character-Check

**Aion 2 mechanics**

- Damage formula: https://gameplay.tips/guides/aion-2-how-stats-work-damage-formula-guide.html
- Stats and caps: https://aion2.akunito.com/en/systems/stats/
- Community formula notes: https://expcarry.com/aion-2-damage-formula , https://thegameswiki.com/aion-2/wiki/damage-formula-and-stat-efficiency
- Daevanion boards: https://timesaver.gg/blog/aion-2-daevanion-guide , https://aion2maps.com/guides/daevanion-boards/
- Combat power: https://metabot.gg/en/aion-2/guides/combat-power-guide , https://timesaver.gg/blog/aion-2-combat-power-gear-guide

**Aion 2 skill data**

- https://aion2.gaming.tools/skills/17190000
- https://www.aion2timers.com/db/skills/
- https://metabot.gg/en/aion-2/skills

**Aion 2 DPS meters**

- https://github.com/karim-mo/aion2-abysslogs-dps-meter , https://abysslogs.com/
- https://aion2t.com/dps-top
- https://aionflex.gg/
- https://github.com/ZDYoung0519/NOIA2

**Existing Aion 2 calculators**

- https://buson2.com/
- https://bongbong99999-debug.github.io/aion2ssow/
- https://aion2zang.info/
- https://azas2.com/
