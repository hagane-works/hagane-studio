# UI-panel fields — `/ui-panels` (界面机制 Builder)

Author a project-owned `scripted_gui` interface panel — a window + its controls. You fill
structured fields; **the program compiles them into the `.gui` window, the `scripted_gui` `.txt`, and the
merged localisation** (preview == export). You never write GUI markup — that's exactly what this
workbench replaces.

## Panel (`POST /api/go/projects/{pid}/ui-panels`)
| Field | Meaning |
|---|---|
| `token` | **required** snake_case naming seed (e.g. `mymod_stability`) — the sole input that derives every window/gui/loc name. Auto-suffixed on collision |
| `country_tag` | owner scope (uppercased) |
| `context_type` | the scripted_gui context — one of: `player_context` · `selected_country_context` · `selected_state_context` · `decision_category` · `diplomatic_action` · `national_focus_context` · `country_mapicon` · `state_mapicon` (default `decision_category`; an invalid value is a 422) |
| `parent_window_token` | token of a parent window to nest under (a token ref, not an FK) |
| `visible_trigger` | **raw trigger** — gates whether the whole window shows |
| `window_props` | geometry `{x, y, width, height}` plus optional `background_sprite`, `orientation`, `origo`, `moveable` |
| `controls` | complete initial control tree; use guarded operations to edit it after creation |
| `name_en` / `name_zh` / `display_name_locs` | window title loc |

`visible_trigger` → `_raw-script-fields.md`; trigger tokens → `tokens.md`.

## Control shapes (each dict in `controls`)
Every control needs a unique short `name` and a `kind`. Common fields are `x`,
`y`, `w`, `h`, `visible_trigger` and validated `raw_extras`. `width`/`height`
are accepted aliases for `w`/`h`. Unknown fields are rejected, not silently kept.

| `kind` | Keys |
|---|---|
| `container` | nested `children`; only containers may have children |
| `text` | `text`, `text_locs`, `font`, `format` (`left`/`right`/`center`); text becomes a localisation key and may contain `[?var\|fmt]` runtime tokens |
| `button` | `sprite`, `text`, `text_locs`, `click_effect` (**raw effect**), `click_enabled_trigger` (**raw trigger**), `font`, `tooltip`, `tooltip_locs` |
| `icon` / `progressbar` | `sprite` (registered GFX), `frame` (default 1), and **one** of `frame_var` or `image_expr` for a dynamic property binding |

The button's `click_effect` / `click_enabled_trigger` / `visible_trigger` hold raw HOI4 script — the program
wraps them into the scripted_gui's `effects` / `triggers` blocks for you.

### Progress indicators and variables

The supported `progressbar` control is exported as an `iconType` whose sprite
frame can change. It is **not** a continuous fill `progressBarType`. Choose a
resolved multi-frame sprite, verify its frame count, and keep the bound frame in
range. Reusing an unrelated icon does not establish the requested visual design.

`frame_var` names a HOI4 variable/expression token; allowed characters are letters,
digits, `_`, `.`, `@`, `^`. It is emitted as `properties = { <token>_<name> = {
frame = <frame_var> } }`. Do not add a `var:` prefix. Alternatively, `image_expr`
is a single-line image expression without quotes, braces or newlines. Set only
one binding. An unbound progressbar raises `panel.progressbar_unbound`.

A button in the same panel can define/change its variable, for example:

```json
{"kind":"button","name":"advance","text":"Advance",
 "click_effect":"add_political_power = 1\nset_variable = { example_frame = 2 }"}
```

The corresponding indicator uses `frame_var: "example_frame"`. This proves the
saved effect and binding refer to the same variable; it does not prove the
variable is initialised before the first click. The panel API has no dedicated
initialisation hook. Read the project's existing variable producer or report that
initialisation/runtime gap; do not invent an `on_open` field or create unrelated
events/decisions to hide it.

## Sub-resource
- `GET /ui-panels/{panel_db_id}/files` — the **compiled** `.gui` text + `.txt` text + per-language loc.
  This is byte-identical to what the export ships (preview == export) — use it to eyeball the result.
- `get_ui_panel` returns the complete persisted tree and `updated_at`.
  `update_ui_panel` edits panel-level fields only; it never accepts replacement
  controls. Use `apply_ui_panel_operations` for add/update/delete/move/reorder,
  passing the exact `expected_updated_at` from the content you inspected. A stale
  guard is a conflict, not permission to replay old changes against a newer tree.
- After a write, call `get_ui_panel`, `validate_ui_panel`, and
  `get_ui_panel_files`. Validation checks strict fields, script snippets, actual
  GFX references and emitted control/binding names.
- `export_valid` and `runtime_verified` are separate. The current tools return
  `runtime_verified: false`: they do not establish that HOI4 mounted the panel,
  displayed it in the intended context, or executed its button. Report the
  returned `capability_gaps` and do not call a successful compile an in-game test.

### Common quirks
- **`width`/`height` on a control are canvas-preview only for buttons.** The program deliberately omits
  the button size block from the emitted GUI — writing a `size = { … }` on a HOI4 `buttonType` crashes the
  parser and the button renders but can't be clicked. The program handles this correctly; you just fill
  w/h for your own preview.
- An empty button `sprite` falls back to a default visible sprite (an invisible-but-clickable button is a
  trap the program avoids by construction).
- `token` is the naming seed — renaming it re-derives every name; the program cleans up the old snapshot.
- Save is a live compile: after POST/PUT, GET `/files` to see the exact compiled output.
