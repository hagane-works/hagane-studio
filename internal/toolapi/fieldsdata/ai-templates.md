# AI division template fields — `/ai-templates`

This workbench defines the unit composition the AI builds for a role. It is
separate from a deployed division and from an ordinary division-template entity.

| Tool field | Meaning |
|---|---|
| `template_id` | Unique safe HOI4 script token; not the numeric database id |
| `role` | AI role, for example `infantry`; default `infantry` |
| `upgrade_prio` | Complete factor block ranking this template against sibling templates |
| `enable` | Complete trigger block gating when the template is usable |
| `target_templates` | Full list of `{id, body}` entries; at least one target is required |

Each target `id` is a unique safe script token. Its `body` is one complete HOI4
block string, including outer braces, for example:

```json
{
  "template_id": "example_infantry_role",
  "role": "infantry",
  "upgrade_prio": "{ factor = 1 }",
  "enable": "{ tag = GER }",
  "target_templates": [{
    "id": "infantry_default",
    "body": "{ upgrade_prio = { factor = 1 } target_template = { regiments = { infantry = 6 } support = { } } }"
  }]
}
```

Verify all regiment/support keys before writing. Do not pass a dictionary in
place of a block string or omit its braces. `update_ai_template` replaces the
complete `target_templates` list: include every entry that must survive. Omit
unchanged fields; explicit `null` is rejected. Advanced workbench-only fields are not
extra tool parameters; follow the current tool schema.

Read back with `get_entity(kind="ai_template", id=<database id>)`. Its underlying
API is list-only, so the tool selects the exact row from the project list. Do not
invent a detail read. `list_entities` is a summary, not a full composition
verification. Inspect the saved blocks and exported script, and distinguish
authoring an AI template from creating or deploying a playable division.
