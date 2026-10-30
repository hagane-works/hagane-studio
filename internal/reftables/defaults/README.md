# Reference tables

Small tables of facts the program needs and cannot read from the player's Hearts of Iron IV files.
They are built into the program from this folder (`internal/reftables`).

## Replacing or emptying a table

Put a file with the same name in the `tables` folder beside `settings.json`
(`%APPDATA%\Hagane Works\tables\` on Windows; with `--config <file>`, beside that file) and restart
the program. The file replaces the built-in table whole; nothing is merged.

To turn a table off, give it empty lists and objects, for example `{"groups": [], "sub_ideologies": []}`.
Everything that reads a table keeps working without it: it just shows or checks less (see each table).
A file that cannot be read or does not parse is reported in the program's log and the built-in table
is used instead.

## ideologies.json

The base game's four ROOT ideology groups and its 22 sub-ideologies, in the game's order, with each
sub-ideology's group, its English and Chinese display names and `random_disabled` (the game's
`can_be_randomly_selected = no`).

Source: `common/ideologies/00_ideologies.txt` of the base game, using the ideology
registry (`VANILLA_IDEOLOGY_REGISTRY`) listed it; the display names are that registry's.

Read by: the ideology lists in the editors (served to the UI by `GET /api/app/reference-tables`),
the export of a project's own sub-ideologies and ROOT groups (which ideologies are the base game's
and need no definition, which groups a new sub-ideology can join), the export checks and the
workbench's validation of the ideology registry, and the world-map generator's neutral leaders
(the `neutrality` group and its sub-ideologies).

Emptied: no ideology counts as the base game's. The editors list only the project's own
ideologies, a new sub-ideology needs one of the project's own ROOT groups, and a country whose
ruling party is a base-game ideology gets no `set_politics` from the export.

## alerts.json

The alerts of the topbar's alert row in the game view.

- `engine`: every alert type the row can draw, in type order: the engine's type number, its name
  and the frame (1-based) of `GFX_global_alert_icons` it draws.
- `text_keys`: the alerts whose tooltip is not `<name>_instant` / `<name>_delayed`, with the
  localisation keys of the instant and the delayed part (`""` when there is none).

Source: read out of `hoi4.exe` of game 1.19.3; nothing in it is a guess from screenshots.

- The alert manager registers each alert type with its name, in type order: `alert_no_research` is
  type 8, `alert_faction_can_select_goal` type 81 (81 registrations; type 68 has no name, see
  below).
- `CGlobalAlertIcon` picks the icon frame with a switch on the type (`frame = f(type)`, 1-based).
  Every type from 8 to 81 lands on its own frame of the 70-frame strip, except 19 (expeditionary
  force), 43 (battle plans with no divisions) and 45/46 (dangerous naval invasion), which fall to
  the default like types 0-7 (hostile troops, invasions, supply, air superiority). Those are
  theatre alerts, drawn on `theater_alerts_strip.dds` by the theatre selector, not in this row, and
  are left out. That leaves exactly 70 types for 70 frames.
- The switch has no case of its own for type 41 (`alert_naval_convoy_raiding_results`): it jumps
  straight to the `SetFrame` call with the register still holding 0x20 from the allocation above
  it, so the frame is 32 — the globe-and-convoy icon, which fits.
- Type 68 is never registered by name. Its frame (a government building with a clock, beside
  decision timeout's gavel with a clock) and the one key that fits, `alert_bop_timeout_delayed`
  ("These Balance of Power events or decisions are about to time out"), make it the Balance of
  Power timeout; its name here is that key's stem. The game view marks this one name as inferred.
- Types 70 and 71 share the name `ui_alert_special_project_available` and differ only in their
  icon.
- Registration builds the tooltip from `<name>_instant` and `<name>_delayed`. Ten alerts build
  their text from other keys instead (the cryptology, operative and special-project ones among
  them); `text_keys` lists the keys their own code references.

Screenshots of a 1939 Germany agree with the table slot for slot (research frame 1, civilian
factories 6, faction invite 18, through faction goal 70).

Emptied: the alert row shows no alerts.

## text_colors.json

The colours of the game's text colour codes (`§G…§!` in localisation).

- `global`: the codes every font knows, from the `textcolors` block of `interface/core.gfx`.
- `fonts`: codes a font redefines, by font name: `hoi_18mbs` (the topbar's numbers) has its own
  `textcolors` in `interface/core.gfx`.

Read by: the game view's tooltips, resource and trade numbers and topbar (served to the UI by
`GET /api/app/reference-tables`).

Emptied: text with a colour code is drawn in its normal colour.
