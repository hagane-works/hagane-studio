# State fields — `/states`

Edit the territorial content of **existing** states — ownership, cores/claims, manpower, resources,
buildings, victory points.

> ⚠️ **States are EDIT-only — there is no "create a state".** A fresh project has no states. First bring
> vanilla states in with `POST /states/seed-from-vanilla` (`{state_ids: [...]}` or `{all: true}`), then
> `PUT` the ones you want to change. (Imported-mod projects already have their states.)

| Field | Meaning |
|---|---|
| `owner` | owning country TAG |
| `controller` | controlling country TAG (nullable = no distinct controller) |
| `state_category` | free-text size category (`city` / `town` / `wasteland` / `megalopolis` / …) — **not an enum** |
| `manpower` | state population/manpower (int) |
| `resources` | dict `{strategic_resource: amount}` |
| `add_core_of` | `list[TAG]` — countries that core this state |
| `add_claim_by` | `list[TAG]` — countries claiming it (parallel to `add_core_of`) |
| `remove_core_of` | `list[TAG]` — cores to remove |
| `victory_points` | `list[dict]` — VP placements |
| `set_province_controller` | `list[dict]` `{province_id, controller}` |
| `buildings` | state-level building levels dict |
| `province_buildings` | per-province building placement (serialized under the hood) |
| `impassable` | `impassable = yes` — **blocks troop entry and provincial building construction**. In game the state renders as impassable terrain, which reads to players as the state having vanished. Vanilla pairs this ONLY with `state_category = wasteland` (19 states) or `enclave` (2); **all 21** of those also carry `force_link_ownership_to` below — set that too, or the state ends up an ownerless wasteland. |
| `impassable_ignored_links` | 1.19+ · **raw inner list** of neighbor state ids an impassable state keeps its connection to (send `"1024"` or `"1024 1025"`; export wraps it as `impassable_ignored_links = { … }`) |
| `force_link_ownership_to` | `int` or `null` — the `state_id` of the neighbour state this impassable state follows for ownership. Measured on vanilla 1.19.2 (not in the wiki guides): **21/21** impassable states carry it. Pair it with every `impassable = true` you set — without it the wasteland has no owner to inherit. Send `null` to clear. |
| `display_name_locs` | `{"<lang>": str}` — **any** content language, partial ok — rename the state via localisation override. `edit_state` merges per language. Read side also returns derived `display_name` = the resolved `{lang: name}` map (your override first, the built-in vanilla name as fallback) |

No raw-script fields — states are structured territorial data.

### Common quirks
- **No POST** — you edit an existing state (seeded or imported), you don't create a new one. `PUT` to
  edit, `DELETE` to remove (see *Removing states* below). Any other verb on `/states/{id}` answers
  `405 GET, PUT or DELETE only`.
- `state_category` is free text, not an enum — custom categories are allowed.
- Don't hand a country a state it shouldn't own at start (esp. another country's capital) → in-game
  `capital … they don't own it` noise.
- Renaming = `display_name_locs` only. The state definition's `name = "STATE_<id>"` loc key never
  changes (see below).

## Workflow
The full territory chain:
1. **Seed** — `seed_states_from_vanilla(state_ids=[...])`. Copies the real vanilla definitions
   (provinces, manpower, buildings, exact vanilla filename) from your game's files into your project.
2. **Locate** — `get_states(name_query="Borneo")`. Each row carries `display_name` as a
   `{lang: name}` map — every language that state has a name in, the author's rename winning over the
   built-in vanilla name. `name_query` is a case-insensitive contains-match against **any** of those
   languages, the raw `STATE_<id>` loc key, or an exact state id (applied before the row cap).
3. **Edit** — `edit_state(...)`: `owner`, `add_core_of`, `buildings`, … plus `display_name_locs` —
   `{"ru": "Корсика"}` — to rename in any language (`display_name_en` / `display_name_zh` are the
   two-language shorthand for the same field). `edit_state` merges per language — send `{"ru": …}` and
   en/zh keep their values.
4. **Export** — the program writes each edited state under its **vanilla original filename**
   (filename-shadowing) and emits the rename as a `STATE_<id>` localisation override.

### Removing states
`preview_delete_entity` then `delete_entity` with `kind: "state"` and the state's **row id** (`id`,
not the HOI4 `state_id`) remove one state whatever its `source` — a state you edited goes too. The
workbench's bulk delete removes only imported states.

### "The filename does not determine how the file is handled" — what the wiki means
The game reads the state **id from the file's contents**, not from the filename — any filename parses.
But to **override** a vanilla state your file must ship under the **exact same filename** as vanilla's
(irregular: `333-British Borneo.txt` vs `1023 - Brunei.txt` — record verbatim, never synthesize):
same-name files shadow, exactly one definition wins. A different filename means BOTH definitions load →
double-defined provinces → tug-of-war. The program export does the same-name shadowing for you.

### Renaming a state = localisation override
The state file keeps `name = "STATE_613"` forever; the rename is a loc line the export emits for you:

```
l_english:
 STATE_613: "New Name"
```

⚠️ No `:0` version suffix on an override line — tested in-game, `STATE_613:0 "…"` triggers a
loc-key collision that vanilla wins (the rename silently fails). Bare `KEY: "text"` overrides cleanly.

Set it via `display_name_locs` (`display_name_en` / `display_name_zh` remain as shorthand) — never
edit the `name` loc key itself. The language list is **not** en/zh: whatever languages you store come
back out. One gap to know: states nobody renamed — the program's built-in vanilla name table
currently only carries en/zh, so a state you never touched
answers in English to a Russian-speaking author.
