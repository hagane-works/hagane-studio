# Editing an existing mod — import it, modify it, save it back

The journey for "I have a finished mod and want to change or continue it" (修补 / 续作). It is a
distinct project identity with its own export semantics — not a submod.

## Loaded-existing vs submod — two different projects

| | Loaded-existing mod | Submod |
|---|---|---|
| Project shape | `dependencies: []` + `is_loaded_existing_mod: true` (set at creation by the load-existing import flow) | `dependencies: ["<base mod>"]` |
| What the project holds | the WHOLE mod's entities (`source: "imported"`) + your changes | ONLY your content — the base mod loads at runtime |
| Main export operation | **save back into the original mod** (append / diff channels below) | standalone additive zip that loads AFTER the base — **never append** |

Extending a mod you don't own, that players keep running alongside your addon, is a submod. This page
is for when the original mod itself is the thing being edited.

## The flow
1. **Import the whole package** with the program's **Load existing mod** (scan-and-parse). Entities land
   with `source: "imported"`.
2. **Edit** through the normal workbenches. Touching an imported entity flips its `source` to
   `created` — it now counts as yours. New entities are `created` from birth.
3. **Export** — pick the artifact by intent:

| You want | Call | Semantics |
|---|---|---|
| a patch/addon (your changes + additions only) | `export_project`, `include_imported: false` (default) — see **Getting the ZIP** below | emits `source="created"` only — everything you touched or added |
| the full mod re-emitted | `export_project`, `include_imported: true` | everything, incl. untouched imported entities |
| the original mod UPDATED in place | the Export page's **append to existing mod** (the original mod's zip or folder + `include_imported` + `replace_paths`) | merges your output over the original, yours winning on path conflicts; it reports the overlap |
| conflict-by-conflict control | the Export page's **diff** export (a dry-run conflict report, then your per-path choices) | per-entity conflict reports for the diffable entity types; everything else falls back to the append merge |

The bottom two rows are merge operations the Export page runs. The plain "give me the ZIP" rows are
the export job below — the same artifact the Export page builds.

## Getting the ZIP

Call `export_project`. It starts the job, waits for it and returns the file list; with `save_to` (an
existing folder on this computer) it saves the ZIP there:

```bash
hw call export_project '{"project_id": 3, "include_imported": false, "save_to": "C:/Users/me/Desktop"}'
```

Read the export's notes before you ship: the zip carries `_*.log` files at its root that say what it
had to drop or fall back on — missing assets, English-fallback loc keys, scripted names it could not
model. **Install to game** on the Export page puts the mod straight into the game's mod folder.

## Notes that save a day
- ⚠️ **Don't re-package base content into an addon.** If the artifact loads after the original mod, the
  default created-only export is the right one; `include_imported=true` would duplicate the entire
  original inside your patch — a huge zip and double definitions.
- **Changing what vanilla already ships** (a vanilla character, event file, state) → prefer a same-name
  file override over replace_path — the two mechanisms are contrasted in
  `_timeline-and-replace-path.md`.
- 🚩 **25 MiB binary cap.** Binary raw files (added on the Export page's raw-files manager) cap at
  25 MiB per file and are a per-file override hatch, NOT a bulk asset pipe — the import already carries
  the mod's own assets, so don't re-funnel a big mod's binaries through raw one by one.
