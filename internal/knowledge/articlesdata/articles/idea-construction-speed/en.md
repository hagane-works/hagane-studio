---
contentId: idea-construction-speed
kind: case
slug: idea-construction-speed
locale: en
status: published
title: "Grant a construction-speed national spirit for 180 days"
summary: "Use a one-off decision to grant +10% construction speed for 180 days. Includes the spirit, a factory-count requirement, its caller and English localisation."
primaryTopic: industry
topics: [industry]
scenarios: [ideas, decision]
gameTerms: ["Construction Speed","National Spirit","Civilian Factory","Decision"]
aliases: ["construction speed +10%","timed construction bonus","180-day national spirit","build factories faster with a spirit"]
relatedIds: [construction-speed, "wiki:modifier:production_speed_buildings_factor", "wiki:trigger:num_of_civilian_factories", "wiki:effect:add_building_construction", "wiki:effect:add_timed_idea", "wiki:effect:remove_ideas"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "national spirit construction speed increase 10% 180 days timed construction bonus positive reward one-off decision add_timed_idea production_speed_buildings_factor English localisation not immediate factory not slowdown case"
sources:
  - label: "HOI4 Wiki: Idea modding"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki: Decision modding"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
  - label: "HOI4 Wiki: Timed national spirits"
    url: "https://hoi4.paradoxwikis.com/Effects#add_timed_idea"
  - label: "HOI4 Wiki: Construction speed modifier"
    url: "https://hoi4.paradoxwikis.com/Modifiers#production_speed_buildings_factor"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

The goal is a period of faster construction, not an immediate factory reward. A country with at least ten civilian factories can take a free one-off decision to receive “Construction Coordination”, contributing +10% construction speed for 180 days.

This case is source-checked, not run in HOI4. Below are the complete new content files and checks still to perform.

## Before you start

- Use an already working mod and the English game interface. All paths are relative to your mod root, not the installed vanilla game.
- Check that the `hag_knowledge_construction` and `hag_knowledge_construction_plan` IDs do not conflict with your mod or dependencies.
- Keep the vanilla `GFX_idea_generic_production_bonus` resource, or use a valid spirit picture from your total conversion. No new image is supplied.
- Prepare a test country that meets the factory requirement and can undertake construction. This checks `num_of_civilian_factories`, not idle factories; it does not reserve ten factories.

There is no country-tag restriction, the AI will not choose the decision, and each country can use it once. It is a demonstration caller; connect a finished focus reward or story event to your own content flow.

## 1. Define the national spirit

Create `common/ideas/hag_knowledge_construction.txt`. This is the complete file:

```pdx
ideas = {
    country = {
        hag_knowledge_construction_plan = {
            picture = generic_production_bonus
            removal_cost = -1

            modifier = {
                production_speed_buildings_factor = 0.10
            }
        }
    }
}
```

`country` is the national-spirit category. The game adds the `GFX_idea_` prefix to `picture`; do not repeat it by writing `GFX_idea_generic_production_bonus` there.

Put the construction-speed modifier inside the spirit's `modifier` block. Defining the spirit does not grant it. `removal_cost = -1` prevents manual removal, not expiry of a timed spirit.

## 2. Grant it from a decision and set its duration

Create this complete category file at `common/decisions/categories/hag_knowledge_construction.txt`:

```pdx
hag_knowledge_construction_category = {
    allowed = {
        always = yes
    }
}
```

Create this complete decision file at `common/decisions/hag_knowledge_construction.txt`:

```pdx
hag_knowledge_construction_category = {
    hag_knowledge_begin_construction_plan = {
        cost = 0
        fire_only_once = yes

        available = {
            NOT = {
                num_of_civilian_factories < 10
            }
        }

        complete_effect = {
            add_timed_idea = {
                idea = hag_knowledge_construction_plan
                days = 180
            }
        }

        ai_will_do = {
            factor = 0
        }
    }
}
```

The `available` condition means “at least ten”, including exactly ten. It checks the count without deducting factories. Below ten, the decision cannot be selected; it is not hidden through `visible`.

`add_timed_idea` executes for the country taking the decision. Its `idea` must match the definition ID exactly. The 180 days belong in the granting effect, not inside `modifier`.

This caller grants the spirit only once, so repeated clicks do not obscure duration checks. This article does not define refresh behaviour for repeated grants or promise inheritance through civil wars.

## 3. Add English names and descriptions

Create `localisation/english/hag_knowledge_construction_l_english.yml` and save it as UTF-8 with BOM:

```yaml
l_english:
 hag_knowledge_construction_category:0 "Construction Affairs"
 hag_knowledge_construction_category_desc:0 "Coordinate building work and construction schedules."
 hag_knowledge_begin_construction_plan:0 "Begin Construction Coordination"
 hag_knowledge_begin_construction_plan_desc:0 "Coordinate construction work to improve building speed for a limited period."
 hag_knowledge_construction_plan:0 "Construction Coordination"
 hag_knowledge_construction_plan_desc:0 "Coordinate construction schedules and reduce delays between projects."
```

Description text does not create a modifier. The actual values come from the script and its generated effect tooltip. After pasting, verify the encoding and language header. This file covers English game text; other game languages need their own files.

## 4. Check it in the game

These steps have not been performed. Use a disposable new test save and record the game version and enabled mods.

1. Check countries with fewer than ten, exactly ten and more than ten civilian factories. Read the condition tooltip before assuming a loading failure. Verify unusual trade or occupation cases separately.
2. Before taking the decision, record national spirits, the construction tooltip for one project and its assigned factories.
3. After taking it, inspect the “Construction Coordination” name, icon, +10% construction-speed contribution and remaining duration. Confirm that it did not immediately grant a factory.
4. Compare construction data while keeping other conditions as constant as possible. “Remaining time fell by exactly 10%” is not a sufficient test; speed and time are different quantities.
5. Advance the test save until expiry and check that the source disappears and stops contributing. Expiry should not remove completed buildings. The country should still be unable to reuse the one-off decision.
6. Inspect new `error.log` messages tied to these IDs, modifier or localisation, not unrelated errors from an older run.

If the spirit appears but the speed contribution does not, check the `modifier` nesting, key and value. If the decision appears but the spirit is not granted, check the grant ID. For raw text keys or a missing icon, inspect localisation and picture references separately.

## Adapt the design

- Change the strength or introduce a penalty using positive and negative construction-speed values (`wiki:modifier:production_speed_buildings_factor`), and update the narrative. This case only demonstrates a positive bonus.
- For an immediate civilian factory, use the building effect (`wiki:effect:add_building_construction`), not a factory count as a modifier value.
- To end the source, see the base remove_ideas reference (`wiki:effect:remove_ideas`). Removing a spirit and reopening a one-off decision are separate actions.
- Return to construction speed (article `construction-speed`).
