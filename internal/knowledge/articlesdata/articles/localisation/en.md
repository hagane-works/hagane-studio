---
contentId: localisation
kind: tutorial
slug: localisation
locale: en
status: published
title: "Step 3: Add in-game names and descriptions"
summary: "Turn script IDs into readable English text. Check the language filename, language header and UTF-8 with BOM encoding."
primaryTopic: research
topics: [research]
scenarios: [ideas, focus]
gameTerms: [National spirit name, National spirit description, Localisation]
aliases: [name does not appear, raw ID displayed, localisation, l_english, UTF-8-BOM]
relatedIds: [first-mod, first-content, check-and-troubleshoot, focus-research-bonus]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "HOI4 modding step three: hw_learning_l_english.yml uses l_english and UTF-8 with BOM. HW_learning_spirit provides the name and HW_learning_spirit_desc the description. Check raw IDs, file extensions, straight quotes and line breaks."
sources:
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
  - label: "HOI4 Wiki: National spirit display text"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
The previous page defined `HW_learning_spirit`. That ID is useful for script references, but it is not a player-facing name. Now give it the name “Research Training” and a description.

## 1. Create an English localisation file

Create `localisation/english/hw_learning_l_english.yml` inside the Mod. This is the **complete file**:

```yaml
l_english:
 HW_learning_spirit: "Research Training"
 HW_learning_spirit_desc: "Improve the organisation of research work to increase research speed."
```

Save it as **UTF-8 with BOM**. A code block supplies text only; copying it does not set your editor's file encoding.

## 2. Keep the three language markers consistent

| Location | This example |
| --- | --- |
| Directory | `localisation/english/` |
| Filename | `hw_learning_l_english.yml` |
| First-line header | `l_english:` |

The first character of `l_` is a lowercase L, not the number 1. The extension must be `.yml`, not `.yaml` or an accidental `.yml.txt` with a hidden extension.

Check this file with the game set to English. Changing Hagane Works' interface language does not create another language's Mod text or change the game's selected language.

## 3. What belongs in a key, and what belongs in a value?

The key before the colon is what the script looks up. `HW_learning_spirit` supplies the name; `HW_learning_spirit_desc` supplies the description. These must match the spirit ID from the previous page.

The quoted value after the colon is what the player reads. You can rename “Research Training” without renaming the script ID.

Keep these rules in mind:

- Use a normal ASCII colon and straight double quotes `"`, not typographic quotes.
- Keep script-compatible keys. Display text, including non-English text in other language files, belongs in the values rather than the keys.
- Keep each value on one physical line. For an in-game line break, write `\n` instead of pressing Enter inside the quoted value.
- The number in vanilla entries such as `key:0 "Text"` is an optional version marker. This tutorial omits it; its presence alone is not a localisation error.

This is a **partial example replacing the existing description line**, not an extra entry with a duplicate key:

```yaml
 HW_learning_spirit_desc: "Improve the organisation of research work.\nIncrease research speed."
```

## 4. New content does not need an entire vanilla file override

This exercise defines its own `HW_learning_spirit` key, so an ordinary localisation file is sufficient. Do not copy an entire vanilla file or define the same key in several places to make the new name appear.

If you later need to change an existing vanilla name, read about the priority rules of the localisation `replace/` directory. That is different from `replace_path` in a descriptor. This exercise needs neither.

## 5. Check the same small example

Once the test country has the spirit, confirm that it is named Research Training and shows your English description. Keep checking the Research Speed +5% modifier from the previous page too. Correct text does not prove the modifier is correct, and a correct modifier does not prove the name file loaded.

If you still see the raw ID, check the real extension, language in the filename, first-line header, encoding, key and quotes, in that order. Look for relevant errors in this launch's logs.

The rules were checked against the local guide and vanilla English files for 1.19.2.0. This exercise has not been tested in the game.

Previous: Define the national spirit (`first-content`). Next: Check and troubleshoot (`check-and-troubleshoot`).
