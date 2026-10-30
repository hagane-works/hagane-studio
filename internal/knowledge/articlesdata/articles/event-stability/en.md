---
contentId: event-stability
kind: case
slug: event-stability
locale: en
status: published
title: "Make an event option that grants stability"
summary: "Open an event from a one-off decision and grant 5 percentage points of base stability when its option executes. Includes the event, caller, English localisation and checks to perform."
primaryTopic: politics
topics: [politics]
scenarios: [event, decision]
gameTerms: ["Stability","Base Stability","Event","Decision"]
aliases: ["event stability reward","stability +5% after an option","decision fires an event"]
relatedIds: [stability, "wiki:effect:add_stability", "wiki:trigger:has_stability", "wiki:modifier:stability_weekly"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "event option gain stability once 5 percentage points base stability +5% positive reward complete event decision caller country_event add_stability English localisation not weekly gain"
sources:
  - label: "HOI4 Wiki: Event modding"
    url: "https://hoi4.paradoxwikis.com/Event_modding"
  - label: "HOI4 Wiki: Decision modding"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
  - label: "HOI4 Wiki: Stability effect"
    url: "https://hoi4.paradoxwikis.com/Effects#add_stability"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

This case follows one clear sequence: select a free one-off decision, receive an event, choose “Proceed as planned”, and gain 5 percentage points of base stability.

The example is source-checked but has not been run in the game. The game checks below are still to perform.

## Before you start

- Use a mod that already loads and reaches gameplay. This case adds content; it does not create a country, map or focus tree.
- Add the files inside your mod, not the installed game. All file paths below are relative to the mod root.
- Use the English game interface. The `hag_knowledge_stability` prefix must not conflict with content in your mod or dependencies.
- Use an environment that retains the vanilla resources. The event refers to the existing `GFX_report_event_generic_parliament`; no new image is supplied. If a total conversion replaces it, choose a registered event picture valid in that mod.

The caller has no country-tag restriction, but AI selection is disabled. Each country may use this ordinary decision once. Add a verified country condition if your final story applies only to one country.

## 1. Define the event

Create `events/hag_knowledge_stability.txt`. This is the complete file:

```pdx
add_namespace = hag_knowledge_stability

country_event = {
    id = hag_knowledge_stability.1
    title = hag_knowledge_stability.1.t
    desc = hag_knowledge_stability.1.d
    picture = GFX_report_event_generic_parliament

    is_triggered_only = yes

    option = {
        name = hag_knowledge_stability.1.a
        add_stability = 0.05
    }
}
```

Put `add_namespace` outside the event. `is_triggered_only` makes the event depend on a caller instead of firing automatically; do not add `mean_time_to_happen` here.

The reward goes directly inside `option`, not in `immediate` or an extra `effect` block. It executes with the option, not when the window first opens. An ordinary event that times out can also auto-select an option and execute its effects; this is not a window that must wait indefinitely for a click.

## 2. Add a selectable caller

Create this complete category file at `common/decisions/categories/hag_knowledge_stability.txt`:

```pdx
hag_knowledge_stability_category = {
    allowed = {
        always = yes
    }
}
```

Then create this complete decision file at `common/decisions/hag_knowledge_stability.txt`:

```pdx
hag_knowledge_stability_category = {
    hag_knowledge_open_stability_event = {
        cost = 0
        fire_only_once = yes

        available = {
            has_stability < 0.95
        }

        complete_effect = {
            country_event = {
                id = hag_knowledge_stability.1
            }
        }

        ai_will_do = {
            factor = 0
        }
    }
}
```

The category key must match the previous file. Selecting the decision runs `complete_effect` and sends the event to the country using it. The sequence does not switch to another country or state.

`has_stability < 0.95` is a demonstration requirement of strictly less than 95%, intended to avoid testing near the cap. Not every stability reward needs this condition. It does not spend stability.

The one-time restriction belongs to the decision. It does not stop another script from calling the same event again, so do not add duplicate callers unintentionally.

## 3. Add English text

Create `localisation/english/hag_knowledge_stability_l_english.yml` and save it as UTF-8 with BOM. This block gives the complete text; still check your editor's save encoding after pasting:

```yaml
l_english:
 hag_knowledge_stability_category:0 "Local Affairs"
 hag_knowledge_stability_category_desc:0 "Coordinate local affairs and restore public confidence in everyday administration."
 hag_knowledge_open_stability_event:0 "Convene a Local Affairs Meeting"
 hag_knowledge_open_stability_event_desc:0 "Hear from local representatives and agree on the next steps."
 hag_knowledge_stability.1.t:0 "A Common Plan"
 hag_knowledge_stability.1.d:0 "Local representatives have reached an agreement on the immediate administrative priorities. The relevant departments will now put the plan into practice."
 hag_knowledge_stability.1.a:0 "Proceed as planned."
```

The filename suffix and language header must match the game language. Title, description and option use distinct keys; do not reuse the title's `.t` key for the button. Other game languages need their own localisation files.

## 4. Check it in the game

These steps have not been performed. Use a disposable new test save, not an important campaign.

1. Enable the mod and play a country below 95% stability. Check that the decision name is readable. At 95%, inspect the unmet requirement instead of treating the disabled decision as a loading failure.
2. Pause and record base stability and other sources in its tooltip. Select the decision and check the event's title, description and option.
3. Before the option executes, this event should not already have added stability. Hover over the option and inspect the generated effect tooltip.
4. Choose “Proceed as planned” and compare base stability. With no other changes or limits, 50% plus 5 percentage points is 55%, not 52.5%.
5. Check that the same country cannot reuse the decision and that other countries did not automatically receive the reward. Inspect this run's `error.log` for new errors tied to these IDs, namespace or localisation.

If only the displayed total appears unchanged, inspect base stability, other sources and limits. If no event appears, inspect the caller's ID and namespace. If raw keys appear, inspect the localisation path, header and BOM. These are separate problems.

## Adapt the effect

- For a one-off penalty, see negative add_stability values (`wiki:effect:add_stability`) and update the narrative. This case itself is a positive reward.
- For weekly increases, do not repeatedly call this event; see weekly stability (`wiki:modifier:stability_weekly`).
- For a requirement only, see stability thresholds (`wiki:trigger:has_stability`).
