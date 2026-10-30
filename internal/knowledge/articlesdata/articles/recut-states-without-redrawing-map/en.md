---
contentId: recut-states-without-redrawing-map
kind: case
slug: recut-states-without-redrawing-map
locale: en
status: published
title: "Re-cutting states without redrawing the map: the engine's hard rules and a pipeline that works"
summary: "Leave every province alone and only regroup them into the states of your own era. Four of the five hard rules keep you out of the main menu entirely; the building table rule is the counter-intuitive one, because the engine prints the right answer in its error message and you should copy it rather than compute it."
primaryTopic: territory
topics: [territory]
scenarios: [history]
gameTerms: [re-cut states, state, province, strategic region, state id, building table, port site, capital, lake, total conversion]
aliases: [re-cut states, redivide states, total conversion map, alternate map without redrawing, contiguous state ids, cross strategic region, buildings.txt, Missing State ID, map failed to load]
relatedIds: [state-file-basics, reference-image-to-states, focus-transfer-state-core, "wiki:effect:set_capital"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: State modding"
    url: "https://hoi4.paradoxwikis.com/State_modding"
  - label: "HOI4 Wiki: Map modding"
    url: "https://hoi4.paradoxwikis.com/Map_modding"
  - label: "HOI4 Wiki: Strategic region modding"
    url: "https://hoi4.paradoxwikis.com/Strategic_region_modding"
---

# Re-cutting states without redrawing the map: the engine's hard rules and a pipeline that works

## Goal and observable result

When this page is done you should see three things in game:

1. In one strategic region of your choosing, the base game's states are replaced by states you divided yourself, with borders on base-game province borders.
2. The main menu loads the map normally, and after starting, `error.log` contains none of the six fatal patterns listed here.
3. No coastal province has lost its port site, and the game does not crash at start.

This route is for authors of **alternate-history or historical total conversions**: tearing down the base game's thousand-plus states and replacing them with the divisions of your own era, **without redrawing the province map**.

## What you need before starting

**First, how this route relates to the workbench.** The workbench's states page can edit a single state's province list, can move a province from one state to another on the map, and its pre-export check blocks "a province in two states", the error that crashes the game. It has no batch re-cut, no strategic-region check, no id-continuity check, and it does not rewrite the building table for you. This page describes the pure-script route: you generate the files and write the tooling; what you get here are the criteria and the order, not commands to copy.

- Four read-only things from the game's install folder: `map/definition.csv` (province types), `map/strategicregions/` (strategic regions), `map/buildings.txt` (the building table), `provinces.bmp` (the province image, only used to compute adjacency).
- The ability to launch with `-debug` and read `error.log`. Without it, the log may be empty when the game crashes.
- Read state files (`state-file-basics`) first.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| State files | `history/states/` | What you generate; override base-game ones with their exact filenames |
| Province definitions | `map/definition.csv` | Read-only; province type and coastal flag come from here |
| Strategic regions | `map/strategicregions/` | Read-only; the hard border for cutting |
| Building table | `map/buildings.txt` | Must be rewritten; its first column is a state id |
| Country history | `history/countries/` | Capitals must be redirected when stolen |
| State names | `localisation/` | State name and map label are two keys |

## 1. Decide first: leave the province image alone

Provinces are the smallest unit of the map. Redrawing them means owning shape, adjacency, terrain, coastline, rivers, supply nodes and building positions all at once, and they are entangled. Only re-cutting states is completely different: provinces stay put and you merely **regroup** them. You get all of this for free:

| For free | Note |
| --- | --- |
| Province adjacency | Who borders whom, already computed by the engine |
| Terrain and its bonuses | Mountains, hills, plains, urban |
| Coastline and port site coordinates | Port sites live in the building table; positions need no work |
| Rivers and crossings | River-crossing penalties |
| Victory point positions | Names can change; positions need no work |

The only cost: state borders can only fall on base-game province borders. In densely populated areas the base game's province density is a few dozen kilometres per cell, enough for the vast majority of historical divisions. East Asia as an example: the base game cuts 1172 provinces into only 109 states, nearly 11 per state; it is coarse because East Asia in 1936 did not matter much, not because provinces were short.

**Everything below is the "do not touch the province image" route.**

## 2. Five hard rules, ordered by danger

Each one corresponds to a real load failure or crash.

### 2.1 State ids must run from 1 to the maximum without gaps

The engine walks from 1 to the highest id; each missing one logs `Missing State ID`, and then the main menu declares **the map failed to load**, with singleplayer unavailable. We tested creating 1083 while skipping 1082: the main menu was blocked; 1082 restored it.

Corollary: **you cannot casually abandon base-game ids**. If your cut consumes every province of a base-game state, its id dangles and one of your new states must **claim** it: output your state under that base-game filename. Base-game filenames are not all "id-name", so list the folder on disk rather than deriving them.

### 2.2 All provinces of a state must belong to one strategic region

Breaking this fails the map load at the main menu, with both region names in the log:

```
MAP_ERROR: State "..."(#1082) has provinces (#13375, #3305) belonging to different strategic areas: "..." (#5), "..." (#275)
```

The base game has 304 strategic region files. This rule governs the whole batch plan: **cut by strategic region first, then group within a region by your own divisions**. Modern strategic regions do not match your era's geography, and that has to be accepted; a historically essential core area may share a region with a large unrelated area, and then there are only two options: take the whole region or leave the whole region alone.

### 2.3 A province belongs to exactly one state

Breaking this crashes at startup. The most common source is a leftover file from before a renumbering that still declares its provinces, so clear the output folder before generating state files.

### 2.4 The building table is keyed by state id and must be rewritten after a re-cut

In one line: losing one port site line means a coastal province without a port site, and the engine warns it "will likely crash". Section five is dedicated to this.

### 2.5 Reusing the id of a consumed base-game state can steal a base-game country's capital

A base-game country's capital points at state N; you consumed N's provinces and used the id for your own state; that country's capital is now a state it does not own:

```
Attempting to set capital state #N for ..., they dont own it!
```

In our test the engine refused and fell back, no crash; in another author's project it once crashed outright. Do not gamble: scan the base game's `history/countries/` and redirect every country whose capital falls in a state you now occupy to a state it still owns.

### Three more that are not fatal but cost hours

- **Lakes are state members too.** Of the base game's 13414 provinces, 10155 are land, 126 lake, 3133 sea. Filter by "not sea", never by "land only", or lakes stay in their old state and appear in game as someone else's water inside your territory.
- **`.txt` must not carry a BOM; localisation `.yml` must.** A state file with a BOM is discarded whole, logging `Unexpected token: ï»¿state`, and cascades into 2.1.
- **`buildings.txt` must not end with an extra newline.** The base-game file has no trailing newline on its last line.

## 3. Data model: two hand-maintained tables

Every tool starts from these two tables; they are the single source of truth.

**Table one: province to state**, one row per province, with name and owner repeated on every row of the same state so it can be scanned by eye:

```
province_id,state_key,name_zh,name_en,owner
7656,my_capital_prefecture,开封府,Kaifeng,MYT
4066,my_capital_prefecture,开封府,Kaifeng,MYT
```

**Table two: state attributes**, one row per state:

```
state_key,state_id,override_file,category,manpower,vp_province,vp_value,infrastructure
my_capital_prefecture,1098,,metropolis,2000000,7656,30,4
my_border_prefecture,615,615-Shanxi.txt,town,300000,1531,2,2
```

A non-empty `override_file` means "output under this base-game filename", which is how a dangling id from 2.1 is claimed. `vp_province` must really belong to the state.

Beyond that, build three read-only indexes once from the base game: a province master table (type, coastal flag, terrain, centroid, base-game state), province adjacency (scanned pixel by pixel from the province image, stored by colour rather than id), and province to strategic region. Do pixel-level scans in a compiled language; a scripted loop over eleven million pixels takes minutes.

## 4. The pipeline for one batch

```
① Survey      list candidate provinces, city anchors and base-game states, by strategic region
② Map         hand-write the two tables above                     ← the only step that needs a human
③ Validate    connectivity / same region / a province claimed by two batches / keys match 1:1
④ Holes       gaps completely enclosed by your territory (lakes included)
⑤ Emit states generate history/states/*.txt and state-name localisation
⑥ Overrides   trim reduced base-game states / report consumed ids / check id continuity
⑦ Capitals    redirect base-game countries whose capital was stolen
⑧ Buildings   rewrite buildings.txt, see section five
⑨ Install     run with -debug, read error.log, go back to ⑧ if anything remains
```

Steps ① to ⑧ never launch the game. ⑨ is the only step in the pipeline that needs a person at the keyboard; spend it sparingly.

**Do a pre-check during the survey**: which base-game states are involved; which lie wholly inside (consume one and you must claim its id) and which straddle the region (they become a reduced base-game state); are there lakes; how many coastal provinces. When a strategic region happens to equal the union of several whole base-game states, the batch is unusually clean.

**Find holes with connected components, not per-province scores.** A "hole" is a property of a connected component, not of a single province: take every unclaimed non-sea province, compute connected components, drop the largest one (that is the outside world), and every remaining component is a hole enclosed by your land. Per-province scoring never finds a hole that spans several provinces.

**Look at both pictures**: a cut map rendered from your tables shows shapes; an ownership map rendered from the merged `history/states/` shows final ownership, and only the latter shares its source with the game.

## 5. The building table: let the engine be the authority

This is the most counter-intuitive chapter of the whole method. **The conclusion first: do not try to simulate the engine; the engine prints the answer for you.**

### 5.1 The problem

`map/buildings.txt` has more than sixty thousand lines, first column a state id:

```
state;type;x;y;z;rot;sea_province
598;naval_base_spawn;4666.96;9.50;1228.61;-4.07;2757
```

The engine looks up the province from the coordinates, then checks whether that province really is in the declared state. On a mismatch it discards the line. In our test, after moving one province from state 8 to state 1082, all twelve lines pointing at it were dropped, and the log looked like this:

```
map/buildings.txt error at line 50927: map building(civilian factory) location is not within specified state. Supposed to be '8 - ...' but was '1082 - ...'for province 13375. BUILDING IGNORED!
```

If the dropped line is a port site, the engine goes on to warn that the province is coastal without a port building and will likely crash. In another author's project that corresponded to real crashes; we only tested an inland province, with no crash.

### 5.2 The wrong approach: sampling pixels to imitate the engine

The natural idea is "the engine resolves the province from the coordinates, so I sample the province image the same way". Another author's tests showed this damages the file in two ways: one pixel off near a state border picks the neighbouring state and the script overwrites unconditionally, so that in one round more than a hundred discarded lines were **all** lines the script had changed; and port coordinates sit at the water's edge, so sampling often lands on a sea province, which has no state, so the line is left "unresolved" untouched, and that is precisely the line that crashes.

### 5.3 The right approach

**Rule one: if the sampled province is outside your re-cut area, do not touch a single byte.** Outside the area, the base game was already right.

**Rule two: never locate ports by pixels.** The last field of each line is the sea province the port opens onto; the land province must be one of its neighbours; take the one nearest the coordinates, using real adjacency.

**Rule three: for a state whose borders did not move, the base-game keys are still authoritative.** If one of your states claimed a base-game id and its province set is identical to the base game's, every base-game line pointing at it must still hold; recomputing can only turn right into wrong. This can be made into a pre-flight gate: compare the base game and your output line by line, and any changed line pointing at an "unmoved" state is a regression.

**Rule four: split the engine's answer in two and keep only the invariant half.** In the error line above, "line number to province" is a property of the map and does not change however you cut; store it, it is valid forever. "Province to state" depends on your current cut; recompute it each time. Old logs then remain valid after you re-cut, accumulate across batches, and get more accurate every run.

### 5.4 Ten building types with a per-state quota

For ten types the base game has a fixed number of lines per state; the table's totals are exact multiples of 1081:

| Type | Lines per state |
| --- | --- |
| Air base, rocket site, stronghold network, synthetic refinery, nuclear reactor, radar station, fuel silo | 1 |
| Arms factory, industrial complex | 6 |
| Anti-air | 3 |

A re-cut scrambles this: some states end up with three air base sites, some with none. A state missing one logs `MAP_ERROR: no air base site defined for state N`. In our test, two states each missing one line logged the error without crashing; in another author's project, dozens of states missing sites went along with crashes, and when the last three types were missing **the engine wrote nothing to the log at all** and died on "start game". The safe approach is to restore the base-game quota: move surplus whole lines to the states that lack them, **changing the state key and the coordinates together**; the coordinates must fall inside the target state, or the engine judges "not within specified state" and ignores the line, as if nothing was moved.

### 5.5 A port site is not a port

Everything above only keeps the state key of port **sites** correct. Port **buildings** are written in the state file's `buildings` block, by province id:

```
buildings = {
    6583 = {
        naval_base = 1
    }
}
```

Two separate jobs.

### 5.6 The loop

```
run with -debug → read error.log → paste back the engine's answers → install → run again
```

It converges in one or two rounds. Line numbers stay stable because every step only changes the first column and never adds, removes or reorders lines. **Do not clear `error.log` before verifying**; if you did, retrieve the snapshot from the crash folder.

## 6. Check each item

Run this list at the end of every batch; the first seven need no game launch:

| Check | Expected |
| --- | --- |
| The two tables' keys match 1:1; ids and override filenames unique | Pass |
| No province claimed by two batches | Pass |
| Every state internally connected (known exceptions such as islands marked explicitly) | Pass |
| Every state's provinces in one strategic region | Pass |
| State ids 1 to max with no gaps | Pass |
| Every fully consumed base-game state claimed | Pass |
| No enclaves, or only intentional ones | Pass |
| Every piece of land on the ownership map belongs to the right country | By eye |
| Launch with `-debug`; all six fatal patterns at zero | See below |

The six fatal patterns: `BUILDING IGNORED`, `no port building in the nudger`, `Missing State ID`, `provinces belonging to different strategic areas`, `dont own it`, `invalid province building`.

On a crash, look in four places: `logs/error.log` for the cause; `logs/game.log` for how far it got, where a normal start shows the render-device restore lines after "Launching SINGLEPLAYER-game"; `crashes/<latest>/exception.txt` for the exception address, **same address means the same bug**; `crashes/<latest>/logs/error.log` for the snapshot at the moment of the crash.

## Common mistakes

- **Sampling pixels to imitate the engine's province resolution**: the engine prints its own answer in the error; imitating it is neither necessary nor reliable.
- **Finding holes by per-province scoring**: a hole spanning several provinces is never found; use connected components.
- **Three check scripts all written as "land only"**: the same wrong assumption copied three times gives three layers the same blind spot (lakes). Redundant checks are only redundant when their assumptions differ.
- **Recognising errors by keyword**: the `MAP_ERROR` prefix appears both in the fatal strategic-region conflict and in masses of harmless noise (missing air base sites for an era without aircraft can be nine tenths of the log). Keep your own list of benign noise and check only that the six fatal patterns are at zero.
- **Debugging a crash without `-debug`**: in normal mode `error.log` may be empty.
- **Changing several variables at once**: the efficiency of crash hunting depends entirely on one variable at a time.

## Want a different approach?

- **Keep non-playable neighbours 1:1 as base-game states**: the remaining strategic regions often contain only whole base-game states; adopt them whole and connectivity and region rules cannot break structurally, with ids naturally 1:1. This is the main tool for covering "the rest of the world", but it saves province-by-province research, not the question of who owns the land.
- **Split a base-game state that straddles a real border**: one half claims the base-game id under its filename, the other half takes a new id after the current maximum. No hard rule is broken.
- **Change the look without changing gameplay**: see turning a reference image into states (`reference-image-to-states`).
