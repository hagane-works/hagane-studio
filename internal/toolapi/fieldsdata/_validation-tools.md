# Validation tools — let the program check correctness for you

Read-only checks turn "did I wire this right?" into a call, instead of you grepping files. They are
not authoring surfaces — they're how you catch mistakes before the game does.

## Every build
- `validate_project` — entity-level validation: cross-references closed, loc complete, assets shipped,
  token legality and structure. Compact pages with `total` / `next_offset`; read them all for a
  release.
- `export_project` — the artifact-level findings (dropped content, skipped images, unreferenced super
  events) that only building the ZIP produces; the zip's `_*.log` files carry them.

## While writing
- Every create/update tool reports `reference_check` and `unresolved_references` when you wire
  references (a focus that fires an event, a decision that adds an idea, a character a focus
  unlocks…): a reference that does not resolve is a typo'd id or content that does not exist yet,
  which the export would silently drop.
- `preview_delete_entity` lists everything that references an entity — the project's content and its
  raw script files — before you remove it.

Run all of these in the program; your own game's `logs/error.log` is the final gate for the rare token
none of them can know about.
