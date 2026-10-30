---
contentId: state-file-basics
kind: guide
slug: state-file-basics
locale: en
status: published
title: "State files: where ownership, cores, provinces and buildings each live"
summary: "A state is one file: who owns it, who has a core, which provinces it is made of, what is built there. The first two are a few clicks in the workbench; changing which provinces a state contains touches the map layer, where five engine rules apply and breaking them ranges from a warning to a main menu that refuses to load."
primaryTopic: territory
topics: [territory]
scenarios: [history]
gameTerms: [state, province, ownership, core, claim, controller, manpower, victory point, building, strategic region, state id]
aliases: [how to write a state file, change state owner, add a core, state name not applied, map failed to load, state id, province in two states, history states, state file]
relatedIds: [focus-transfer-state-core, recut-states-without-redrawing-map, reference-image-to-states, "wiki:effect:transfer_state", "wiki:effect:add_state_core", "wiki:trigger:owns_state"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: State modding"
    url: "https://hoi4.paradoxwikis.com/State_modding"
  - label: "HOI4 Wiki: Map modding"
    url: "https://hoi4.paradoxwikis.com/Map_modding"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# State files: where ownership, cores, provinces and buildings each live

## What this system covers

Every owned piece of land on the map is a **state**. Which states a country owns, which are cores, how many factories a state has and how many people live there is all written in that state's own file under `history/states/`.

As an author there are three layers you can decide on a state:

1. **Whose it is**: owner, controller, cores, claims.
2. **What is in it**: manpower, category, resources, buildings, victory points.
3. **Which provinces it is made of.**

The first two layers only touch this one file; the engine does not check them against the map. The third is different: provinces are the smallest unit of the map, and when you change a state's provinces the engine applies five hard rules, covered one by one in the fifth section. That is also why the workbench keeps the first two layers in the main area and puts the province list under the extra fields.

## Core idea: changing attributes and changing provinces are different orders of magnitude

| What you change | How the engine treats it | If you get it wrong |
| --- | --- | --- |
| Owner, controller, cores, claims | Written and read as is, no validation | At worst a capital points at a state the country does not own; the engine refuses and falls back |
| Manpower, category, resources, buildings | Written and read as is | One error line when buildings exceed the category's slot limit |
| Name | Goes through localisation | A wrong key shows the raw key |
| **Province list** | **Five hard rules** | From a warning, to a main menu that reports the map failed to load, to a crash at startup |

"This state belongs to another country at start" or "give this country a core here" is layer one: change the owner or cores in the workbench and you are done. "Merge two states" or "cut a piece out of one" is layer three: read the fifth section first.

## File structure and how it runs

One state per file under `history/states/`. The base game's Luxembourg looks like this, with only two provinces:

```
state = {
    id = 8
    name = "STATE_8"
    state_category = town
    manpower = 294748
    resources = {
        steel = 14
    }
    history = {
        owner = LUX
        add_core_of = LUX
        buildings = {
            infrastructure = 3
            arms_factory = 1
            industrial_complex = 3
        }
        victory_points = { 6583 5 }
    }
    provinces = {
        6583 13375
    }
}
```

| Key | Meaning | Name in the workbench |
| --- | --- | --- |
| `id` | State id, unique across the map, must run from 1 to the maximum without gaps | State id, read-only |
| `name` | The localisation key of the state name, not display text | Name, read-only; rename via display name |
| `state_category` | Category, which decides building slots | State category |
| `manpower` | Population | Manpower |
| `history` block | Starting situation: owner, controller, cores, claims, buildings, victory points | The main-area fields |
| `buildings` block | State-level buildings directly; province-level ones (ports, bunkers) nested under a province id | State buildings and province buildings |
| `victory_points` | Province id plus value | Victory points |
| `provinces` | Which provinces make up the state | Provinces |

The `history` block may also contain dated sub-blocks such as `1939.1.1 = { ... }` that run only for that bookmark; the workbench preserves them as is without expanding them into the form.

### The filename decides between "override" and "a second copy"

A file with the same name replaces the base-game one. **A different filename with the same id inside** makes the engine report `State ID conflict`, list both files, and then **the later-loaded one wins**. So when overriding a base-game state the filename must match exactly, including the base game's spelling (Luxembourg's file is `8-Luxemburg.txt`, not Luxembourg). Base-game filenames are not all "id-name" either, there are ones like `560-TS 2.txt`; copy them from the base-game folder, do not derive them.

The workbench keeps the base-game or imported filename on export, so you do not need to manage this.

### The state name and the map label are two keys

The state name uses `STATE_<id>`; the text next to the city on the map uses `VICTORY_POINTS_<provinceId>`. Change only the former and the map still shows the base-game place name. Put both in the localisation file:

```
l_english:
 STATE_8: "Grand Duchy"
 VICTORY_POINTS_6583: "Luxembourg City"
```

To override base-game keys, the `localisation/<language>/replace/` folder is the safe place. A plain folder also works, but the engine records a duplicate key in `logs/text.log` and adds a "loc key collisions" line to `error.log`. The workbench's display-name field writes exactly this kind of override; the internal key never changes.

## Common goals and where to start

| What you want to see in game | Start here | The distinction that matters |
| --- | --- | --- |
| This state belongs to another country at start | Owner | Owner without a core means that country holds it as non-core land |
| A country has a core or a claim here | Cores, claims | Cores and claims are different things and can coexist |
| Occupied at start: one owns, another controls | Controller | Empty means same as the owner |
| Rename the state | Display name | The map label is a separate key, `VICTORY_POINTS_<province>` |
| More factories, better infrastructure | State buildings | The cap comes from the state category |
| Ports, bunkers | Province buildings, placed by province id | A port building and a port site are different things, see section five |
| Transfer a state or add a core during the story | Focus or event effects | See a focus that transfers a state and adds a core (`focus-transfer-state-core`) |
| Regroup provinces into new states | The provinces field, or write the file directly | Layer three; see section five and re-cutting states without redrawing the map (`recut-states-without-redrawing-map`) |

## Easy things to get wrong

### 🔴 State ids must run from 1 to the maximum; one gap and the game will not start

The engine counts from 1 to the highest id. Each missing one logs `Missing State ID`, then `Definition for state id N is either missing or invalid`, and then **the main menu declares the map failed to load**; singleplayer cannot be started. Tested: the base game's highest id is 1081; creating 1083 while skipping 1082 blocks the main menu, changing it to 1082 fixes it.

So new states go one by one right after the current maximum. The intuition of "leaving gaps per region" does not work here.

### 🔴 All provinces of a state must belong to one strategic region

Strategic regions, defined under `map/strategicregions/`, are hard borders. Put provinces from two regions into one state and the main menu again fails to load the map, with both region names in `error.log`:

```
MAP_ERROR: State "..."(#1082) has provinces (#13375, #3305) belonging to different strategic areas: "..." (#5), "..." (#275)
```

This rule means re-cutting states has to follow strategic regions, not historical divisions.

### 🔴 A province belongs to exactly one state

The same province in two states crashes the game at startup. This is the most severe case. The workbench's pre-export map check lists it as "must fix before download" and will not let such output through.

### 🔴 State files must not carry a BOM

A `.txt` with a BOM is unreadable from its first character: `Unexpected token: ï»¿state`, and the whole file is discarded. That state then does not exist, its id is missing, and the previous "map failed to load" rule fires. Localisation files are the opposite: they must have a BOM. The two rules point in opposite directions, so check the encoding when saving from an editor.

### After moving provinces, `map/buildings.txt` must follow

That file has more than sixty thousand lines and its first column is a state id. Move a province from state 8 to state 1082 and the dozen building-site lines pointing at it no longer match. The engine does not crash, it discards those lines, and **prints the correct answer in the error**:

```
map/buildings.txt error at line 50927: map building(civilian factory) location is not within specified state. Supposed to be '8 - ...' but was '1082 - ...'for province 13375. BUILDING IGNORED!
```

After "Supposed to be" is what the file says; after "but was" is what the engine worked out itself. The fix is to change the first column to what the engine says, not to guess which state a province falls in. If the discarded line is a port site, another author's tests saw the game crash; this page only tested an inland province, with no crash.

### The capital must point at a state the country owns

A capital in a state the country does not own logs `Attempting to set capital state #N for ..., they dont own it!`; the engine refuses and picks one itself. No crash, though another author saw one direct crash from this. The workbench's pre-export check warns about it.

### A land province without a state: no crash, a popup in game

A land province that belongs to no state loads fine; once in game a MAP_ERROR popup appears and `error.log` records `The land province N has no state`.

### A state referencing a non-existent province: that one reference is dropped

`error.log` records one `Malformed token`; the state's other provinces are fine.

### Version and scope

This page was compiled against the base-game files labelled 1.19.2.0. The base game has 1081 states and 13414 provinces: 10155 land, 126 lake, 3133 sea. Lakes belong to states too; when cutting states filter by "not sea", not by "land only", or lakes get left behind.

## Where to go next

- Transferring states, adding cores, occupying without owning, in a focus or event: a focus that transfers a state and adds a core (`focus-transfer-state-core`).
- Tearing down the base game's thousand-plus states without redrawing the map: re-cutting states without redrawing the map (`recut-states-without-redrawing-map`).
- An extreme example done purely for looks: turning a reference image into states (`reference-image-to-states`).
- Effects and triggers: transfer a state (`wiki:effect:transfer_state`), add a core (`wiki:effect:add_state_core`), check ownership of a state (`wiki:trigger:owns_state`).
