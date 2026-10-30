package locout

import (
	"strings"
	"testing"

	"haganeworks/internal/hoi4/loc"
)

type corpusCase struct {
	Raw     string
	Decoded string
	Src     string
	Key     string
}

var locCorpus = []corpusCase{
	{Raw: "\\n", Decoded: "\n", Src: "mun_ideas_l_english.yml", Key: "CZE_skoda_fast_optimization_tt_off_dynamic"},
	{Raw: "§S社会民主主义者§!赢得了选举，他们将在下个月上台\\n", Decoded: "§S社会民主主义者§!赢得了选举，他们将在下个月上台\n", Src: "BRA - Brazil l_simp_chinese.yml", Key: "BRA_socdem_takes_power_jan"},
	{Raw: "一、毁其希望。\\n\\n二、运筹帷幄。\\n\\n三、攻其弱点。\\n\\n四、奇袭制胜。", Decoded: "一、毁其希望。\n\n二、运筹帷幄。\n\n三、攻其弱点。\n\n四、奇袭制胜。", Src: "TFR_country_localisation_SOV_l_simp_chinese.yml", Key: "SOV_navalnys_list_desc"},
	{Raw: "\\nWe can now deploy §YMountaineer Divisions§!.\\n\\n", Decoded: "\nWe can now deploy §YMountaineer Divisions§!.\n\n", Src: "NOR_l_english.yml", Key: "NOR_mountaineers_tt"},
	{Raw: "§Y人口贩运§!\\n\\n美墨边境数处拘留设施中至少有十余名在押人员离奇失踪。深入调查显示，这些人员均遭掳掠挪作私用。", Decoded: "§Y人口贩运§!\n\n美墨边境数处拘留设施中至少有十余名在押人员离奇失踪。深入调查显示，这些人员均遭掳掠挪作私用。", Src: "TFR_country_localisation_USB_l_simp_chinese.yml", Key: "USB_evidence5_loc7"},
	{Raw: "The §BOrganization of Free Nations§! grows much more §Gunified§!.\\n", Decoded: "The §BOrganization of Free Nations§! grows much more §Gunified§!.\n", Src: "TNO_USA_loc_l_english.yml", Key: "USA_add_OFN_unity_large"},
	{Raw: "我们现在保障$COUNT|Y$个国家独立，保障$NAME|Y$独立将会花费$COST|Y$£pol_power£，因为每一个保障独立都会增加花费。\\n", Decoded: "我们现在保障$COUNT|Y$个国家独立，保障$NAME|Y$独立将会花费$COST|Y$£pol_power£，因为每一个保障独立都会增加花费。\n", Src: "diplomacy_l_simp_chinese.yml", Key: "GUARANTEEE_COST2"},
	{Raw: "Mannstärke pro Unterdrückung: $MANPOWER|*H$\\nMannstärke pro Division: $MANPOWERDIV|*H$", Decoded: "Mannstärke pro Unterdrückung: $MANPOWER|*H$\nMannstärke pro Division: $MANPOWERDIV|*H$", Src: "resistance_and_occupation_l_german.yml", Key: "GARRISON_ENTRY_MANPOWER_TOOLTIP"},
	{Raw: "\\nSe §Rreducirá§! la influencia de la §OOTDP§! en un §Y[?global.JAP_influence_CAM_var_temp|=+1%]§!", Decoded: "\nSe §Rreducirá§! la influencia de la §OOTDP§! en un §Y[?global.JAP_influence_CAM_var_temp|=+1%]§!", Src: "TFR_country_localisation_JAP_l_spanish.yml", Key: "JAP_subtract_CAM_influence_tt"},
	{Raw: "\\nPour chaque État sous contrôle où la §Yrésistance§! est active : §G-15%§! sur l'objectif de la résistance.\\n\\n", Decoded: "\nPour chaque État sous contrôle où la §Yrésistance§! est active : §G-15%§! sur l'objectif de la résistance.\n\n", Src: "aat_focus_l_french.yml", Key: "FIN_red_finland_resistance_tt"},
	{Raw: "$BRA_senate_state_0$\\n   $BRA_party_psp_c$ aligned candidates' popularity §Rfalls considerably§![BRA_senado_next_election].\\n", Decoded: "$BRA_senate_state_0$\n   $BRA_party_psp_c$ aligned candidates' popularity §Rfalls considerably§![BRA_senado_next_election].\n", Src: "TNO_brazil_electoral_effects_l_english.yml", Key: "BRA_harm_PSP_aligned_candidates_acre_high"},
	{Raw: "[SOV.GetAdjective] §YAttitude:§! §gNeutral§!\\nRange: §YOpinion:§! §R-10§! - §YOpinion:§! §G+10§!\\n§YModifiers:§!\\n\\n§YActivation effect:§!\\n", Decoded: "[SOV.GetAdjective] §YAttitude:§! §gNeutral§!\nRange: §YOpinion:§! §R-10§! - §YOpinion:§! §G+10§!\n§YModifiers:§!\n\n§YActivation effect:§!\n", Src: "AFG_l_english.yml", Key: "AFG_SOV_INDICATOR_NEUTRAL"},
	{Raw: "\\\"\\\"", Decoded: "\"\"", Src: "TFR_country_localisation_ENG_l_spanish.yml", Key: "ENG_release_tfr_desc"},
	{Raw: "Es hora de que la prepa - la uni...\\\"", Decoded: "Es hora de que la prepa - la uni...\"", Src: "TFR_country_localisation_USB_l_spanish.yml", Key: "bdnspch.8.b"},
	{Raw: "我本人根本没有兴趣去美国，在美国也没有一分一厘的资产。所谓的\\\"制裁\\\"，不过是美国政客自取其辱的丑陋闹剧和令人作呕的拙劣把戏，根本吓唬不了谁。\\n- 陈全国", Decoded: "我本人根本没有兴趣去美国，在美国也没有一分一厘的资产。所谓的\"制裁\"，不过是美国政客自取其辱的丑陋闹剧和令人作呕的拙劣把戏，根本吓唬不了谁。\n- 陈全国", Src: "TFR_replace_loading_tips_l_simp_chinese.yml", Key: "LOADING_TIP_235"},
	{Raw: "\\t- $PERSON$: $VALUE|+=$", Decoded: "\t- $PERSON$: $VALUE|+=$", Src: "core_l_english.yml", Key: "XP_CHAR_ENTRY"},
	{Raw: "\\t- $AMOUNT|H0$ additional $EQUIPMENT|H$ needed to fill divisions and garrisons ( §R$STOCK_AMOUNT|H0^$ available§! )", Decoded: "\t- $AMOUNT|H0$ additional $EQUIPMENT|H$ needed to fill divisions and garrisons ( §R$STOCK_AMOUNT|H0^$ available§! )", Src: "designer_l_english.yml", Key: "DIVISION_MODIFICATION_NEED_NOT_FILLED_GARRISON"},
	{Raw: "\\\\t- $PERSON$: $VALUE|+=$", Decoded: "\\t- $PERSON$: $VALUE|+=$", Src: "core_l_braz_por.yml", Key: "XP_CHAR_ENTRY"},
	{Raw: "\\\\t- $EQUIPMENT|H$: potrzeba $AMOUNT|H0$ mniej, by zaspokoić potrzeby dywizji i garnizonów.", Decoded: "\\t- $EQUIPMENT|H$: potrzeba $AMOUNT|H0$ mniej, by zaspokoić potrzeby dywizji i garnizonów.", Src: "designer_l_polish.yml", Key: "DIVISION_MODIFICATION_LESS_EQUIPMENT_NEEDED_GARRISON"},
	{Raw: "¿Elecciones? ¿Qué son las Elecciones?\\", Decoded: "¿Elecciones? ¿Qué son las Elecciones?\\", Src: "TFR_country_localisation_CHI_l_spanish.yml", Key: "taiwan.12.e"},
	{Raw: "Gain £GFX_GER_green_economy_points_texticon §7Green Economy Points§!\\: [?GER_green_economy_points_var_temp|+0]", Decoded: "Gain £GFX_GER_green_economy_points_texticon §7Green Economy Points§!\\: [?GER_green_economy_points_var_temp|+0]", Src: "TFR_country_localisation_GER_l_english.yml", Key: "GER_add_green_economy_points_tooltip"},
	{Raw: "Bekomme £GFX_GER_green_economy_points_texticon §7Grüne Wirtschaftspunkte§!\\: [?GER_green_economy_points_var_temp|+0]", Decoded: "Bekomme £GFX_GER_green_economy_points_texticon §7Grüne Wirtschaftspunkte§!\\: [?GER_green_economy_points_var_temp|+0]", Src: "TFR_country_localisation_GER_l_german.yml", Key: "GER_add_green_economy_points_tooltip"},
}

func corpusInputs() []struct {
	Label string
	Value string
} {
	out := []struct {
		Label string
		Value string
	}{}
	for _, c := range locCorpus {
		out = append(out,
			struct {
				Label string
				Value string
			}{c.Src + ":" + c.Key + " [workbench/raw]", c.Raw},
			struct {
				Label string
				Value string
			}{c.Src + ":" + c.Key + " [import/decoded]", c.Decoded})
	}
	return out
}

func TestCorpusIsRepresentative(t *testing.T) {
	if len(locCorpus) < 20 {
		t.Fatalf("corpus too thin: %d", len(locCorpus))
	}
	n := 0
	for _, c := range locCorpus {
		if strings.Contains(c.Raw, `\n`) {
			n++
		}
	}
	if n < 10 {
		t.Fatalf("only %d fixtures carry a two-char \\n — headline bug untested", n)
	}
}

func TestCorpusEncodeIsIdempotent(t *testing.T) {
	for _, in := range corpusInputs() {
		once := YmlEscapeValue(in.Value)
		if twice := YmlEscapeValue(once); twice != once {
			t.Errorf("%s: %q -> %q -> %q", in.Label, in.Value, once, twice)
		}
	}
}

func TestCorpusSemanticSymmetry(t *testing.T) {
	for _, in := range corpusInputs() {
		got := loc.DecodeLocEscapes(YmlEscapeValue(in.Value))
		want := normaliseNewlines(loc.DecodeLocEscapes(in.Value))
		if got != want {
			t.Errorf("%s: decode(encode(x))=%q decode(x)=%q", in.Label, got, want)
		}
	}
}

func TestCorpusTrailingBackslashRunIsEven(t *testing.T) {
	for _, in := range corpusInputs() {
		encoded := YmlEscapeValue(in.Value)
		if run := trailingBackslashRun(encoded); run%2 != 0 {
			t.Errorf("H-35 REGRESSION · %s: %q -> %q trailing run %d",
				in.Label, in.Value, encoded, run)
		}
	}
}

func TestCorpusEmittedLinesParseStrictly(t *testing.T) {
	for _, in := range corpusInputs() {
		text := "l_english:\n" +
			YmlLine("probe_key", Text{S: in.Value}) + "\n" +
			YmlLine("sentinel_key", Text{S: "still here"})
		_, entries, diags := loc.ParseLocText(text)
		if len(diags) != 0 {
			t.Errorf("H-35 REGRESSION · %s: emitted line not strictly parseable · "+
				"emitted=%q diags=%v", in.Label, text, diags)
			continue
		}
		if entries["sentinel_key"] != "still here" {
			t.Errorf("H-35 REGRESSION · %s: next line swallowed · %q", in.Label, text)
		}
		want := normaliseNewlines(loc.DecodeLocEscapes(in.Value))
		if entries["probe_key"] != want {
			t.Errorf("%s: value drifted · got %q want %q",
				in.Label, entries["probe_key"], want)
		}
	}
}

func TestCorpusAuthoredTwoCharNewlineNeverDoubled(t *testing.T) {
	checked := 0
	for _, c := range locCorpus {
		if !strings.Contains(c.Raw, `\n`) || strings.Contains(c.Raw, `\\`) {
			continue
		}
		encoded := YmlEscapeValue(c.Raw)
		if strings.Contains(encoded, `\\n`) {
			t.Errorf("%s:%s — author's \\n doubled back into a literal: %q -> %q",
				c.Src, c.Key, c.Raw, encoded)
		}
		if !strings.Contains(loc.DecodeLocEscapes(encoded), "\n") {
			t.Errorf("%s:%s — \\n did not survive as a real line break: %q",
				c.Src, c.Key, encoded)
		}
		checked++
	}
	if checked < 10 {
		t.Fatalf("only %d fixtures exercised the headline bug", checked)
	}
}
