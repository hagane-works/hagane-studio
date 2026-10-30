---
contentId: check-and-troubleshoot
kind: tutorial
slug: check-and-troubleshoot
locale: en
status: published
title: "Step 4: Check your Mod and find what did not take effect"
summary: "Check loading, spirit assignment, display text and the actual modifier separately. File checks and in-game verification are different steps."
primaryTopic: research
topics: [research]
scenarios: [ideas, focus]
gameTerms: [National spirit, Research Speed, Game logs]
aliases: [Mod not working, national spirit missing, troubleshooting, error.log, checklist]
relatedIds: [first-mod, first-content, localisation, "wiki:modifier:research_speed_factor", focus-research-bonus, idea-supply-consumption]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "HOI4 modding step four: check the selected Mod, paths and extensions, braces, spirit assignment, English localisation and Research Speed value. Search this launch's logs/error.log for your files and IDs. No logged errors does not prove the gameplay result is correct."
sources:
  - label: "HOI4 Wiki: Mod structure and testing"
    url: "https://hoi4.paradoxwikis.com/Mod_structure"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
  - label: "HOI4 Wiki: Console commands"
    url: "https://hoi4.paradoxwikis.com/Console_commands"
---
You now have descriptors, a spirit definition and English display text. The final step is to check the whole small example, from loading to its actual effect—not just whether a tool shows an error.

## Check the files before starting the game

- The structure matches Step 1 (`first-mod`), and the launcher has selected the current exercise Mod rather than an older copy with the same name.
- The braces in `hw_learning_ideas.txt` are balanced, script tokens are unchanged, and the value is `research_speed_factor = 0.05`.
- `HW_learning_spirit` is spelled consistently in the definition, localisation and test command, without duplicate definitions.
- `hw_learning_l_english.yml` really uses UTF-8 with BOM and the `l_english:` header, and is not named `.yml.txt`.

These checks can find file problems. They do not prove that the game loaded the Mod, granted the spirit or applied the modifier.

## Use a test game with a known starting state

Keep your regular saves separate. Start a local test game with only the Mods this exercise needs. Use the checked game version and note the country's existing spirits and research modifiers before the test.

This series uses the console command in Step 2 (`first-content`) to grant the spirit manually, isolating whether the definition works. If you later grant it through a focus or another gameplay trigger, test that path again. A successful console test does not prove the focus is connected correctly.

## Work backwards from what you see

| Observed problem | Check first |
| --- | --- |
| The launcher cannot find the Mod | External `.mod` file, content directory and actual path |
| The test cannot find `HW_learning_spirit` | Whether the current Mod is enabled; definition path, real extension, braces and unique ID |
| The definition exists, but the country has no spirit | Whether the granting command or effect actually ran; saving a definition is not granting it |
| The spirit's name is still a raw ID | English filename, language header, encoding, key, quotes and the game's selected language |
| The spirit exists, but its modifier is wrong | `modifier` nesting, modifier name, number and sign |
| Research days do not change as expected | Research Speed first, then other spirits, technology year and ahead-of-time penalties; do not compare days alone |

Change one kind of problem at a time. After each edit, repeat the relevant check and confirm that you are reading this launch's result, not an old log or save state.

## How should I read the logs?

Find `logs/error.log` in the game's actual user-data directory and inspect information from this launch. Start by searching for your filename, `HW_learning_spirit` or `research_speed_factor`. When a log reports a line number, also check preceding lines for a missing quote or closing brace; the original mistake can be earlier.

If the log also contains messages about vanilla or other Mods, distinguish their source before editing your own files. Do not delete unrelated content just to remove every message.

**No relevant error does not prove the result is correct.** Also check which country received the spirit, its name, the modifier's direction and value, and whether the intended gameplay trigger ran.

## Record a result someone can check again

Keep four things in your test notes:

1. Game version, enabled Mods, test country and starting state.
2. Files changed and the method used to grant the spirit.
3. Expected result—Research Training with a +5% Research Speed modifier—and what you actually observed.
4. Screenshots and relevant log excerpts from that run. Remove unnecessary personal paths or account information before sharing.

Keep “checked against script and vanilla documentation” separate from “reproduced in the game”. This series currently has the former evidence; the four-step exercise has not been playtested.

## What can I try next?

Once the difference between defining and granting content is clear, try a supply consumption spirit (`idea-supply-consumption`). If you already have a focus tree, try a one-use industry research bonus (`focus-research-bonus`).

Previous: English localisation (`localisation`). Back to the start: Mod file structure (`first-mod`).
