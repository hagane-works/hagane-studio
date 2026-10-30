# Command line and tools

The editing tools Hagane Works is built on can be called by scripts and other programs on this
computer. `hw.exe`, a small console program, calls them; underneath is a JSON API that the running
program serves on its own address.

## Command line

`hw.exe` sits beside `HaganeWorks.exe` (`bin\`). From source:
`go build -o bin/hw.exe ./cmd/hw`.

```text
hw [--config <settings.json>] [--port <n>] [--images <dir>] <command> [arguments]

  status [--json]                        the running program: version, address, folders, game index
  tools [--json]                         every tool, one line each (--json: with input schemas)
  describe <tool> [--json]               a tool's description, input schema and field reference
  call <tool> [<json> | @file.json | -]  call a tool; its result is printed as JSON
  fields [<name>]                        the field references: what each tool field expects
  --help                                 help (also: hw <command> --help)
```

**Finding the program.** `hw` reads the program's settings — `%APPDATA%\Hagane Works\settings.json`
on Windows, or the file `--config` names, the same option the program takes — for its port (the
`port` setting, else 37330; `--port` overrides it) and for the tools token. When Hagane Works is not
running there, `hw` starts `HaganeWorks.exe --no-window` from its own folder, waits for it, and leaves
it running: open `http://127.0.0.1:37330/` in a browser to see it, and quit it there (Settings → Quit
Hagane Works). Starting the program yourself first is just as good; `hw` then uses that copy.

**Input.** `call` takes the tool's input as a JSON object: the argument itself, a file (`@file.json`,
UTF-8 or the UTF-16 Windows PowerShell 5.1 writes), or stdin (`-`). No input means `{}`.
`hw describe <tool>` shows what the input may hold.

**Output.** The result is JSON on stdout, UTF-8. Pictures a tool returns (the world-from-a-picture
previews) are saved as files — in `--images <dir>`, else a `hagane-works-images` folder in the
temporary folder — and the result names them under `images[].file`.

**Exit codes.** `0` done · `1` the tool answered with an error (its message on stderr) · `2` a usage
error (a bad command line, input that is not a JSON object, an unknown tool) · `3` the program could
not be reached (not running and not startable, or a tools token that does not match).

### Examples

PowerShell 7:

```powershell
hw status
hw tools
hw describe create_idea
hw call list_projects
hw call search_knowledge '{"query": "focus reward political power", "locale": "en"}'
hw call get_knowledge_article '{"id": "wiki:effect:add_political_power", "locale": "en"}'
hw call create_idea '{"project_id": 1, "name": "Iron Will", "country_tag": "TST", "modifier": {"stability_factor": 0.05}}'
'{"project_id": 1}' | hw call get_project_overview -
hw call create_idea "@idea.json"          # quote it: @ starts splatting in PowerShell
```

Windows PowerShell 5.1 drops the double quotes inside an argument, so escape them, or use a file:

```powershell
hw call get_project_overview '{\"project_id\": 1}'
'{"project_id": 1}' | Set-Content -Encoding utf8 in.json; hw call get_project_overview "@in.json"
```

bash (Git Bash, WSL, macOS, Linux):

```bash
hw call create_idea '{"project_id": 1, "name": "Iron Will", "country_tag": "TST"}'
echo '{"project_id": 1}' | hw call get_project_overview -
hw call list_projects | jq -r '.projects[] | "\(.id) \(.name)"'
hw call export_project '{"project_id": 1, "save_to": "C:/Users/me/Desktop"}'
```

`cmd.exe`: `hw call get_project_overview "{\"project_id\": 1}"`.

### The tool API

`hw` is a thin client; a script can call the same API directly. The running program serves it on its
own address, for programs on this computer only:

| | |
| --- | --- |
| `GET /api/tools` | the program's status and every tool (name, description, input schema) |
| `GET /api/tools/{name}` | one tool, with its field reference |
| `POST /api/tools/{name}` | call a tool: the body is its input object, the answer its result object |
| `GET /api/tools/fields`, `GET /api/tools/fields/{name}` | the field references |

Each request needs `Authorization: Bearer <token>`, the token in the `tools-token` file beside
`settings.json` (the program makes it on its first start; delete the file and restart the program for
a new one). Use the address `127.0.0.1:<port>` exactly: like the program's other routes, the API
refuses other host names and requests from web pages (an `Origin` other than the program's own), so
no website open in a browser can call it.

A tool error answers 4xx with `{"detail", "code"}`: 400 for input that does not fit the tool's schema,
the project's own answer (404 project not found, 409 conflict, 422 refused, …) when the program
refused the change, 422 otherwise; 503 means wait and retry (the game index is still being built).

```bash
curl -s -H "Authorization: Bearer $(cat "$APPDATA/Hagane Works/tools-token")" \
     -d '{"project_id": 1}' http://127.0.0.1:37330/api/tools/get_project_overview
```

## Tools

The 149 tools include read-only knowledge and validation tools as well as editing tools that change the project folders
the program's workbenches show; a workbench open in the window picks the change up when it reloads.
Projects themselves are created, imported and deleted in the program's window; `list_projects` lists
them.

Conventions the tools share:

- **Knowledge first.** Before relying on unfamiliar HOI4 behavior, call `search_knowledge`, then
  `get_knowledge_article` for the relevant result. Preserve its verification level and game version;
  the current tool schema and game validation still decide what can be written.
- **ids and keys.** `id` is the row id of a thing in the project (use it to change or delete it); `key`
  is the identifier the game uses (use it to reference the thing from other content). `list_entities`
  and `get_entity` read every kind of content with both.
- **Field references.** Write tools point at their field reference — `hw describe <tool>` shows it,
  `hw fields <name>` shows any of them — for what each field expects, and which hold raw HOI4 script.
- **Write results.** A write answers with what was saved and a `mutation_receipt`: `write_status`
  says the change is saved; `verification_status` and `reference_check` say whether it was checked.
  When an id was taken and the program stored another, the result says so in `id_reassigned`.
- **Edit revisions.** Focus-tree and bookmark writes accept `edit_revisions`, the revision of the
  content you read (`focus_tree:<id>` / `bookmark:<id>`); a stale one is refused instead of
  overwriting newer work. The result's `edit_revisions` are the ones to send next.
- **Authoring defaults.** Before a write, the project's export languages gain the languages the
  content is written in, and a standalone project without a campaign end date gets one (shown under
  `authoring_defaults`); `ensure_project_settings` sets them explicitly.
- **The game.** Some tools read the base game (focus trees, states seeded from it, the map); they need
  the Hearts of Iron IV folder set in the program's Settings.

| Area | Tools |
| --- | --- |
| Offline knowledge | `search_knowledge`, `get_knowledge_article` |
| Projects | `list_projects`, `get_project_overview`, `ensure_project_settings`, `validate_project`, `export_project` (`save_to`: a folder to save the ZIP in) |
| Any content | `list_entities`, `get_entity`, `batch_update_entities`, `preview_delete_entity`, `delete_entity` |
| Focus trees | `create_focus_tree`, `update_focus_tree`, `get_focus_tree`, `create_focus`, `update_focus`, `connect_focus`, `set_focus_mutual_exclusion`, `delete_focus` |
| Characters | `create_character`, `update_character`, `assign_character_role`, `get_characters`, `create_custom_trait`, `update_custom_trait` |
| Events | `create_event_namespace`, `update_event_namespace`, `create_event`, `update_event`, `add_event_option`, `create_super_event`, `update_super_event` |
| Decisions and ideas | `create_decision_category`, `update_decision_category`, `create_decision`, `update_decision`, `create_idea`, `update_idea` |
| Countries and history | `create_country`, `update_country`, `set_country_history`, `create_bookmark`, `update_bookmark` |
| Politics | `get_politics_references`, `create_ideology_root`, `create_ideology_subtype`, `update_ideology`, `preview_delete_ideology`, `apply_delete_ideology`, `discard_ideology_conflict` |
| Technology and equipment | `create_technology`, `update_technology`, `create_equipment`, `create_equipment_variant`, `update_equipment`, `create_doctrine`, `update_doctrine`, `get_technology_surface_environment`, `confirm_technology_surface_environment`, `list_technology_surface_branches`, `get_technology_surface_branch`, `create_technology_surface_branch`, `update_technology_surface_branch`, `delete_technology_surface_branch`, `add_technology_surface_node`, `repair_technology_surface_timeline` |
| Military | `create_sub_unit`, `update_sub_unit`, `get_sub_units`, `create_division_template`, `update_division_template`, `add_division_deployment`, `update_division_deployment`, `create_air_wing_group`, `update_air_wing_group`, `add_air_wing`, `update_air_wing`, `create_fleet`, `update_fleet`, `add_task_force`, `update_task_force`, `add_ship`, `update_ship`, `get_oob_snapshot` |
| Organisations | `create_mio`, `update_mio`, `add_mio_trait`, `create_intelligence_agency`, `update_intelligence_agency`, `add_agency_upgrade`, `create_specialization`, `update_specialization`, `create_special_project`, `update_special_project`, `validate_special_projects`, `create_autonomy_state`, `update_autonomy_state` |
| States and the map | `get_states`, `seed_states_from_vanilla`, `edit_state`, `check_world_map`, `repair_world_map`, `list_world_map_records`, `undo_world_map_change`, `restore_world_map` |
| World from a picture | `create_world_image` (a picture as `data_base64` or `file_path`), `get_world_image`, `create_terrain_sketch`, `create_country_split`, `preview_world_from_images`, `create_world_from_images`, `get_world_state_layout`, `name_world_map` |
| Scripting | `create_scripted_localisation`, `update_scripted_localisation`, `create_scripted_snippet`, `update_scripted_snippet`, `create_dynamic_modifier`, `update_dynamic_modifier`, `create_modifier_definition`, `update_modifier_definition`, `create_opinion_modifier`, `update_opinion_modifier`, `create_wargoal`, `update_wargoal`, `create_ai_strategy_plan`, `update_ai_strategy_plan`, `create_ai_template`, `update_ai_template`, `list_raw_templates`, `create_raw_from_template`, `create_raw_file`, `update_raw_file` |
| Interface | `list_ui_panels`, `get_ui_panel`, `get_ui_panel_files`, `create_ui_panel`, `update_ui_panel`, `apply_ui_panel_operations`, `validate_ui_panel`, `get_gui_window`, `get_gui_overlay`, `preview_gui_overlay`, `apply_gui_overlay_edits`, `validate_gui_overlay`, `get_gui_sprites`, `get_gui_texts`, `update_gui_texts` |
