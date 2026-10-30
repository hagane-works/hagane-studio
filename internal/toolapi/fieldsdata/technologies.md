# Technology fields — `/technologies`

Author custom / overriding research technologies: nodes, prerequisites, unlocks, granted modifiers, and
tree placement.

Custom root technologies must be attached to a certified technology-surface branch so the export can
generate their grid boxes in `interface/countrytechtreeview.gui`. First inspect
`get_technology_surface_environment`, confirm the detected DLC environment, create a branch with
`create_technology_surface_branch`, and add its nodes with `add_technology_surface_node`. Export validation
rejects a project-owned root that has no generated grid box. Dependency mods with their own replacement
technology interface remain unsupported because their private grid layout cannot be patched safely.

| Field | Meaning |
|---|---|
| `tech_id` | unique per-project id (auto-derived from `name` lowercased if blank; auto-suffixed on collision). Ownership is inferred from the id **prefix** (`USA_…`) — the `country_tag` column stays empty for a global tech |
| `country_tag` | `""` = global tech (vanilla default) |
| `tech_line` | logical line (`infantry` / `armor` / `artillery` / `naval` / …) → mapped to an in-game folder |
| `folder` | in-game tech-tree folder id. ⚠️ **armour = the British spelling `armour_folder`** (`armor_folder` → in-game error); air = `air_techs_folder`. Defaults from `tech_line` |
| `research_cost` | research-time cost multiplier (default 1.0) |
| `start_year` | earliest research year / tree column (default 1936) |
| `position_x` / `position_y` | node grid coordinates (⚠️ `position_y` is transformed on export — verify placement in-game, can't check offline) |
| `position_y_is_explicit` | row intent: `true` emits the stored row (including `0`); `false` derives from `start_year`; legacy `null` rows use the line compatibility heuristic |
| `prerequisites` | `list` of required tech_ids |
| `categories` | tech-category tags (drive category-scoped bonuses) |
| `modifiers` | granted modifiers — **see the shape note below** |
| `enable_equipments` | equipment archetypes unlocked (siblings: `enable_subunits` / `enable_equipment_modules` / `enable_building` / `enable_tactic`) |
| `doctrine` | bool — marks the node as a doctrine tech |
| `xp_research_bonus` | dict — XP-boosted research bonus |
| `on_research_complete` | **raw effect** — runs when researched |
| `allow` / `allow_branch` / `on_research_complete_limit` | **raw triggers** |
| `ai_will_do` | AI weight body (`factor = 1`), NOT an effect |
| `name` | ⚠️ defaults to the literal `"New Technology"` — **set `name = <tech_id>`** or the bareword rides into the output |

### 🔴 The `modifiers` shape (a flat dict is silently ignored)
`modifiers` is a dict, but the shape matters:
- **A flat country modifier on the tech** → use the escape-hatch key:
  `modifiers = {"__modifier": "army_attack_factor = 0.05\nresearch_speed_factor = 0.1"}`.
- **Per-equipment bonuses** → nested `{archetype: {stat: value}}`.
- A plain `{stat: value}` dict is **silently ignored** (tech emits with no modifier). Use `__modifier`.

Raw-script fields → `_raw-script-fields.md`; modifier stats → `tokens.md`.

### Common quirks
- The route `{tech_id}` path param is the **integer DB id**, not the string `tech_id`.
- `add_tech_bonus` category (when unlocking via a focus) must be a real category — `construction` /
  `infrastructure` are NOT categories (use `industry`).
