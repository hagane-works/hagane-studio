---
contentId: "wiki:effect:set_technology"
kind: reference
slug: set_technology
locale: en
status: published
title: "Unlock a specific technology immediately"
summary: "Give a country a specific technology directly. This sets technology ownership, rather than research speed or a limited-use bonus."
primaryTopic: research
topics: [research]
scenarios: [focus, event, decision, history]
gameTerms: [Unlock technology, Basic Machine Tools, Technology]
aliases: [grant technology, skip research, instantly unlock technology, give technology]
relatedIds: [research-speed, "wiki:effect:add_tech_bonus", "wiki:modifier:research_speed_factor"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/set_technology
searchText: "set_technology sets specific technologies for the current country. basic_machine_tools = 1 grants Basic Machine Tools; popup = no disables this effect's technology pop-up. It takes real technology IDs, not equipment IDs, category IDs or research-speed values."
sources:
  - label: "HOI4 Wiki: Set technology"
    url: "https://hoi4.paradoxwikis.com/Effects#set_technology"
  - label: "HOI4 Wiki: Technology definitions"
    url: "https://hoi4.paradoxwikis.com/Technology_modding"
---
## 1. What changes in the game, and where can I see it?

The country receives a specific technology without going through the normal research process. This example grants Basic Machine Tools. Check that technology's status in the research interface.

If the test country already has Basic Machine Tools, setting it again will not provide a useful before-and-after comparison. Check the country's starting technology status first.

## 2. Who receives it, and when does it apply or end?

This is a country-scoped effect. In a focus completion reward, event option or another country effect block, it sets the current country's technology when the effect runs. Merely saving the text in a file does not execute it.

Country history files can also use it to set starting technologies. Editing starting history is not the same as applying an effect to an existing save. This page does not cover removing technologies or reversing subsequent unlocks.

## 3. What do the numbers and parameters mean?

In `basic_machine_tools = 1`, the left side is a technology ID and `1` is the value used to grant that technology. It does not mean research speed +1, nor does it increase every technology by one level.

`popup = no` controls the technology pop-up, without changing which technology this example grants. The bundled documentation gives `yes` as the default.

Confirm technology IDs in `common/technologies`. Do not substitute the `industry` category, the display name “Basic Machine Tools”, or an equipment ID. This page documents the verified `= 1` granting pattern; it does not assume `0` is a generally safe way to undo technology changes.

## 4. Where does it go, and how do I write it?

This is a **complete snippet for an existing country-scoped effect block**, not a standalone file. It can go inside a focus's `completion_reward`, provided that the target country already loads the focus tree.

```hoi4
set_technology = {
    basic_machine_tools = 1
    popup = no
}
```

The example uses a vanilla technology and its existing localisation. You do not need to create another display name for Basic Machine Tools. If you substitute a custom technology, define that technology first.

## 5. What might I need next?

- Make research faster while retaining the research process: Ongoing Research Speed modifier (`wiki:modifier:research_speed_factor`).
- Reward just one or a few research projects: Limited-use research bonus (`wiki:effect:add_tech_bonus`).
- Choose the right kind of reward: Research guide (article `research-speed`).

Checked against the bundled effect documentation and vanilla technology definitions for 1.19.2.0. This example has not been tested in the game.
