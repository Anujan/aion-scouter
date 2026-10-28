# Aion Scouter

A MapleScouter-style tool for AION 2. Look up a character and get an estimated DPS built from the character's real gear, Daevanion boards and skills, a breakdown of where the damage comes from, and a ranking of which stat to raise next.

Supported classes: **Spiritmaster** (priority) and **Chanter**. Other classes show their profile and rebuilt stats, without DPS.

See [`docs/maplescouter-deep-dive.md`](docs/maplescouter-deep-dive.md) for the research and roadmap behind it.

## Layout

```
backend/    Go: API, stat rebuild, damage engine (all calculations)
frontend/   SvelteKit SPA: visuals only, every number comes from the API
docs/       Research and design notes
```

### Backend packages

| Package | What it does |
|---|---|
| `internal/ncsoft` | Throttled, cached client for the JSON endpoints behind the official character site (`aion2.plaync.com`). It loads a full character snapshot: profile, gear, every item's rolls, and Daevanion nodes. |
| `internal/gamedata` | Embedded, versioned game data under `data/global/`:<br>• skill tooltips for every level, Daevanion boards (grid, costs, node stats);<br>• damage constants and target profiles;<br>• per-class model files holding every tunable that isn't in a tooltip (cast times, spirit attack rate…), so calibration doesn't need code changes. |
| `internal/stats` | Rebuilds the final stat sheet. NCSOFT doesn't publish the stat window, so it sums gear, manastones, set bonuses, titles, Daevanion nodes, passives and primary/god stats, and records where each value came from. |
| `internal/engine` | Expected-value hit model (community damage formula) inside an event-driven rotation sim. Class modules live in `spiritmaster.go` and `chanter.go`. Also produces the stat-efficiency table. |
| `internal/api` | HTTP API: `GET /api/meta`, `GET /api/search`, `GET /api/character/{region}/{serverId}/{characterId}`, `POST /api/simulate`. |
| `cmd/server` | Runs the API. With `-static`, it also serves the built frontend. |
| `cmd/gamedata` | Re-imports skill and board data. |

## Running locally

Requires Go 1.24+ and Node 22+.

```sh
make dev-api   # terminal 1: API on :8080
make dev-web   # terminal 2: http://localhost:5173 (proxies /api to :8080)
```

The **Demo characters** region serves three anonymized real characters from `backend/internal/testdata`, so the site works without calling NCSOFT.

For a single production binary that serves both:

```sh
make run       # builds frontend + backend, serves on :8080
```

Run `make check` for Go vet and tests plus the Svelte type check.

## How the estimate works

1. **Snapshot.** About 30 requests to the official site, cached for 10 minutes and spaced 250 ms apart.
2. **Stat rebuild.** Every value keeps its sources, which the Stats tab shows. The API's skill levels already include Daevanion and gear bonuses.
3. **Simulation.** The class module chooses actions; for Spiritmaster that means summon swaps, Four Elements into Elemental Fusion, jointstrikes and fillers. Every hit uses the expected value of crit, Double and Multi-hit. Procs (Spirit's Descent, Corrode, Theostones…) are rolled with a seeded RNG over 40 runs, so results are reproducible.
4. **Efficiency.** The fight is re-simulated with each stat raised one step. The gain is expressed in flat Attack, like MapleScouter's converted stat.

### Known limits (shown in the UI as assumptions)

- **Cast times and spirit basic attacks:** estimates. The client doesn't publish cast times. Spirit basic-attack rate and damage are placeholders.
- **Boss profile:** a generic estimate. Boss stats live on NCSOFT's servers.
- **Not exposed by the API:** stigma specializations, wings and class base stats. The DPS tab has a **Match your stat window** panel where players can override any rebuilt stat.
- **Calibration:** not yet calibrated against real fights. The next step is to compare against DPS-meter logs (Abyss Logs, aion2t), per the design doc.

## Data sources

- Character data: NCSOFT's official AION 2 character site (unofficial, undocumented API).
- Skill tooltips and Daevanion boards: Global client values as published by metabot.gg, imported with `cmd/gamedata`.
- Damage formula and constants: community research. Each constant's note in `constants.json` cites its source.

Unofficial fan project, not affiliated with NCSOFT.
