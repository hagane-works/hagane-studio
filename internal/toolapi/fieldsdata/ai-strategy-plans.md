# AI strategy plan fields — `/ai-strategy-plans`

An AI strategy plan selects behaviour for an AI-controlled country. It does not
create the referenced countries, focuses or divisions. Before writing, inspect
the current entity and resolve every referenced game identifier.

| Tool field | Meaning |
|---|---|
| `plan_id` | Unique safe HOI4 script token; not the numeric database id |
| `name` / `desc` | Plan title and description strings |
| `allowed` | List of complete trigger blocks gating which countries can use the plan |
| `abort` | List of complete trigger blocks that end the plan |
| `ai_national_focuses` | List of complete blocks containing existing focus script identifiers |
| `ai_strategy` | List of complete strategy directive blocks |
| `weight` | List of complete factor blocks; the default factor is 1 |

Each list entry must include its outer braces. For example:

```json
{
  "plan_id": "example_ger_plan",
  "allowed": ["{ tag = GER }"],
  "ai_strategy": ["{ type = befriend id = FRA value = 10 }"],
  "weight": ["{ factor = 1 }"]
}
```

`"allowed": ["tag = GER"]` is invalid. Do not pass a dictionary where the
schema requires a block string. A patch replaces each supplied list in full;
omit fields to preserve them. Explicit `null` is rejected. The workbench
also stores advanced fields such as `enable` and `field_order`; use only fields
actually exposed by the current tool schema.

Read back with `get_entity(kind="ai_strategy_plan", id=<database id>)`. Its
underlying API is list-only, so the tool selects that exact id from the complete
project list; do not construct a nonexistent detail read. `list_entities`
returns summaries and does not prove the complete trigger/strategy blocks were
saved. Read the complete entity, validate its references and inspect the exported
script before claiming that the plan is active in the game.
