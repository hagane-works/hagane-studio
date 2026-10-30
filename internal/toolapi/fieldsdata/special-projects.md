# Special-project fields — `/special-projects`

Author DAG "special project" research nodes and their specialization tracks.
**DLC: Götterdämmerung** — only applies for players who own that DLC.

Two nested sub-resources: `specializations` (research tracks / facility types) and `projects` (the DAG
nodes). Plus `validate`, which checks required specialization, scalar and block syntax, missing parents,
and cycles. Invalid values are rejected before a create or update is saved.

## Project node (`POST /special-projects/projects`)
| Field | Meaning |
|---|---|
| `project_token` | unique string id of the DAG node |
| `specialization` | required specialization/track this node belongs to (**string token ref**, not an FK) |
| `country_tag` | owner (blank inherits the project's country) |
| `special_project_parent` | `list` of parent `project_token`s = the DAG prerequisite edges |
| `allowed` | **raw trigger** — who may access the project |
| `visible` | **raw trigger** — node visibility |
| `available` | **raw trigger** — availability gate |
| `resource_cost` | validated block body containing `resources = { steel = 10 }` |
| `project_output` | validated HOI4 output/reward block body |
| `complexity` | one positive integer or script constant such as `sp_complexity.medium` |
| `prototype_time` | one positive integer or script constant such as `sp_time.prototype.medium` |
| `generic_prototype_rewards` | `list` of generic prototype-reward tokens |

Raw-script trigger fields → `_raw-script-fields.md`.

### Common quirks
- `specialization` and `special_project_parent` reference other nodes by **string token**, not int FK —
  the DAG edges are token strings.
- `complexity` and `prototype_time` are scalar fields, not arbitrary script blocks.
- `resource_cost` must contain a `resources = { ... }` block. A bare value such as `steel = 10` is rejected.
- Run `/special-projects/validate` to catch required-field, script-token, missing-parent and cycle errors before export.
