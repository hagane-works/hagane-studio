package exportengine

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	xdraw "golang.org/x/image/draw"
	"haganeworks/internal/hoi4/cosmetic"
	"haganeworks/internal/hoi4/emit/locout"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/emit/moddesc"
	emitpreview "haganeworks/internal/hoi4/emit/preview"
	emitscripts "haganeworks/internal/hoi4/emit/scripts"
	"haganeworks/internal/hoi4/emit/worldout"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/exportplan/history"
	"haganeworks/internal/hoi4/exportplan/timeline"
	"haganeworks/internal/hoi4/imgcodec"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/hoi4/paradox"
)

type NativeArtifact struct {
	Filename string
	Entries  []Entry
	Coverage []string

	Timeline *timeline.ArtifactMetadata

	Warnings []string

	stageReuse StageReuse
}

type BlobResolver func(key string, maxBytes int64) ([]byte, error)

type ScenarioHistoryPlan = history.ScenarioHistoryPlan

type NativeExportPlan struct {
	Timeline        timeline.Plan
	ScenarioHistory ScenarioHistoryPlan
}

type imageSink struct {
	msg      exportmsg.Printer
	warnings []string

	worldMeta worldout.Meta

	ideaPictureDrops []IdeaPictureDrop

	sharedPlacementPaths map[string]bool
}

func (s *imageSink) skipped(label string) {
	s.say("image.skipped", exportmsg.Args{"item": label})
}

func (s *imageSink) say(key string, args exportmsg.Args) {
	if s == nil {
		return
	}
	s.note(s.msg.Text(key, args))
}

func (s *imageSink) note(text string) {
	if s == nil {
		return
	}
	s.warnings = append(s.warnings, text)
}

func (s *imageSink) lang() exportmsg.Printer {
	if s == nil {
		return exportmsg.Printer{}
	}
	return s.msg
}

func commentLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = "#"
		} else {
			lines[i] = "# " + line
		}
	}
	return lines
}

func joinCapped(items []string, max int, msg exportmsg.Printer) string {
	if len(items) <= max {
		return strings.Join(items, ", ")
	}
	return msg.Text("list.more", exportmsg.Args{"items": strings.Join(items[:max], ", "), "count": len(items) - max})
}

func entriesContainSource(entries []Entry, source string) bool {
	for _, entry := range entries {
		if entry.SourceEntity == source {
			return true
		}
	}
	return false
}

func entriesContainPathSuffix(entries []Entry, suffix string) bool {
	for _, entry := range entries {
		if strings.HasSuffix(entry.Path, suffix) {
			return true
		}
	}
	return false
}

func containsCanonicalString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func BuildNativeShadow(snapshot Snapshot, replacePaths []string, includeImported bool) (NativeArtifact, error) {
	return BuildNativeShadowWithBlobs(snapshot, replacePaths, includeImported, nil)
}

func BuildNativeShadowWithBlobs(snapshot Snapshot, replacePaths []string, includeImported bool, resolve BlobResolver) (NativeArtifact, error) {
	return buildNativeShadowWithBlobs(snapshot, replacePaths, includeImported, resolve, nativeBuildOptions{})
}

type nativeBuildOptions struct {
	skipAdvisory bool

	gameFilesOnly bool

	msg exportmsg.Printer
}

func buildNativeShadowWithBlobs(snapshot Snapshot, replacePaths []string, includeImported bool, resolve BlobResolver, opts nativeBuildOptions) (NativeArtifact, error) {

	if !opts.gameFilesOnly {
		resolve = newBlobPrefetch(context.Background(), resolve, collectPrefetchKeys(snapshot, includeImported), blobConcurrency())
	}
	rawEntries, rawReplacePaths, err := nativeInlineRawFiles(snapshot, includeImported, resolve)
	if err != nil {
		return NativeArtifact{}, err
	}
	plan, err := BuildTimelinePlanFromSnapshot(
		snapshot, replacePaths, rawReplacePaths, rawEntries, includeImported,
	)
	if err != nil {
		return NativeArtifact{}, err
	}
	return buildNativeShadowWithTimelinePlan(snapshot, plan, includeImported, resolve, rawEntries, opts)
}

func BuildNativeShadowWithTimelinePlan(snapshot Snapshot, plan timeline.Plan, includeImported bool, resolve BlobResolver, msg exportmsg.Printer) (NativeArtifact, error) {
	resolve = newBlobPrefetch(context.Background(), resolve, collectPrefetchKeys(snapshot, includeImported), blobConcurrency())
	rawEntries, _, err := nativeInlineRawFiles(snapshot, includeImported, resolve)
	if err != nil {
		return NativeArtifact{}, err
	}
	return buildNativeShadowWithTimelinePlan(snapshot, plan, includeImported, resolve, rawEntries, nativeBuildOptions{msg: msg})
}

func BuildNativeShadowWithPlan(snapshot Snapshot, plan NativeExportPlan, includeImported bool, resolve BlobResolver, msg exportmsg.Printer) (NativeArtifact, error) {
	return BuildNativeShadowWithTimelinePlan(snapshot, plan.Timeline, includeImported, resolve, msg)
}

func buildNativeShadowWithTimelinePlan(snapshot Snapshot, plan timeline.Plan, includeImported bool, resolve BlobResolver, rawEntries []Entry, opts nativeBuildOptions) (NativeArtifact, error) {

	projectRow, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return NativeArtifact{}, err
	}
	cover, err := resolveProjectCover(snapshot, projectRow.Str("picture"), resolve, opts.msg)
	if err != nil {
		return NativeArtifact{}, err
	}
	ideologyPlan, err := nativeIdeologyExportPlan(snapshot, rawEntries)
	if err != nil {
		return NativeArtifact{}, err
	}
	artifact, err := buildNativeDescriptor(snapshot, plan.EffectiveReplacePaths, &cover.Filename)
	if err != nil {
		return NativeArtifact{}, err
	}
	sink := &imageSink{msg: opts.msg}
	memo := newStageMemo(snapshot, includeImported, opts)
	baselineEntries := nativeImportedUIBaseline(
		snapshot, artifactRoot(artifact), includeImported, resolve, sink,
	)
	if len(baselineEntries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, baselineEntries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "imported_ui_baseline")
	}
	artifact, err = applyNativeInheritedIdeologySource(snapshot, plan.EffectiveReplacePaths, rawEntries, resolve, artifact, opts.msg)
	if err != nil {
		return NativeArtifact{}, err
	}
	persistedFindings := ExportFindingsFromTimelinePlan(plan, false)
	timelineMetadata := plan.ArtifactMetadata()
	artifact.Timeline = &timelineMetadata
	if cover.Filename != "" {
		artifact.Entries = append(artifact.Entries, Entry{
			Path: artifactRoot(artifact) + "/" + cover.Filename, Bytes: cover.Bytes,
			Origin: "private_asset", SourceEntity: "project_cover",
		})
		artifact.Coverage = append(artifact.Coverage, "project_cover")
	}
	out, err := memo.run(stageCharacters, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeCharacters(snapshot, artifactRoot(artifact), includeImported, resolve, sink)
		return stageOutput{entries: entries}, err
	})
	entries := out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "character_scripts", "character_localisation")
		if len(snapshot.Tables["entity_obs_paths"]) > 0 {
			artifact.Coverage = append(artifact.Coverage, "character_assets")
		}
	}
	characterEntries := entries
	out, err = memo.run(stageTraitOverlays, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		entries, namespaces, err := nativeTraitOverlays(snapshot, artifactRoot(artifact), includeImported, characterEntries)
		return stageOutput{entries: entries, traitNamespaces: namespaces}, err
	})
	traitOverlayEntries, traitOverlayNamespaces := out.entries, out.traitNamespaces
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(traitOverlayEntries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, traitOverlayEntries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "character_trait_overlays")
	}
	out, err = memo.run(stageAnimatedPortrait, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeAnimatedPortraits(snapshot, artifactRoot(artifact), includeImported, resolve, sink)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "animated_portraits")
	}
	out, err = memo.run(stageTechnologies, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeTechnologies(snapshot, artifactRoot(artifact), includeImported, resolve, sink)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "technology_scripts", "technology_localisation")
		if len(snapshot.Tables["equipments"]) > 0 {
			artifact.Coverage = append(artifact.Coverage, "equipment_localisation")
		}
		if len(snapshot.Tables["sub_units"]) > 0 {
			artifact.Coverage = append(artifact.Coverage, "sub_unit_localisation")
		}
		if len(snapshot.Tables["division_templates"]) > 0 || len(snapshot.Tables["division_deployments"]) > 0 ||
			len(snapshot.Tables["oob_air_wing_groups"]) > 0 || len(snapshot.Tables["oob_fleets"]) > 0 {
			artifact.Coverage = append(artifact.Coverage, "military_oob")
		}
		if entriesContainSource(entries, "technology_icons") {
			artifact.Coverage = append(artifact.Coverage, "technology_assets")
		}
	}
	entries, err = nativeTechnologySurface(snapshot, artifactRoot(artifact), includeImported, resolve)
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "technology_surface")
	}
	out, err = memo.run(stageDecisions, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeDecisions(snapshot, artifactRoot(artifact), includeImported, resolve, sink)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "decision_scripts", "decision_localisation")
		if len(snapshot.Tables["super_events"]) > 0 {
			artifact.Coverage = append(artifact.Coverage, "super_event_scripts", "super_event_localisation")
		}
	}
	out, err = memo.run(stageFocus, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeFocus(snapshot, artifactRoot(artifact), includeImported, resolve, sink)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "focus_scripts", "focus_localisation", "focus_file_policy")
		if entriesContainSource(entries, "focus_icons") {
			artifact.Coverage = append(artifact.Coverage, "focus_assets")
		}
	}
	out, err = memo.run(stageIdeasEvents, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeIdeasEvents(snapshot, artifactRoot(artifact), includeImported, resolve, sink)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "idea_scripts", "idea_localisation", "event_scripts", "event_localisation")
	}
	out, err = memo.run(stageModifiers, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		entries, err := nativeModifiers(snapshot, artifactRoot(artifact), includeImported)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "opinion_modifiers", "dynamic_modifiers", "autonomy_states", "modifier_localisation")
		if len(snapshot.Tables["modifier_definitions"]) > 0 {
			artifact.Coverage = append(artifact.Coverage, "modifier_definitions")
		}
	}
	out, err = memo.run(stageAdvanced, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeAdvancedScripts(snapshot, artifactRoot(artifact), includeImported, resolve, sink)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "custom_trait_scripts", "custom_trait_localisation", "mio_scripts", "mio_localisation", "doctrine_scripts", "doctrine_localisation", "intelligence_agency_scripts", "intelligence_agency_localisation", "special_project_scripts")
	}
	out, err = memo.run(stageAIPlanning, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		entries, err := nativeAIPlanning(snapshot, artifactRoot(artifact), includeImported)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "ai_strategy_plans", "ai_templates")
	}
	out, err = memo.run(stageWargoals, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		entries, err := nativeWargoals(snapshot, artifactRoot(artifact), includeImported)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "wargoal_types")
	}
	out, err = memo.run(stagePanels, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		entries, err := nativePanels(snapshot, artifactRoot(artifact), includeImported)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "ui_panels")
	}
	entries, err = nativeGUIOverlays(snapshot, artifactRoot(artifact), resolve, sink, false)
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "gui_overlays")
	}
	out, err = memo.run(stageScriptedFiles, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		entries, err := nativeScriptedFiles(snapshot, artifactRoot(artifact), includeImported)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "scripted_snippets", "scripted_localisations")
	}

	out, err = memo.run(stagePreservedLocRows, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		rows, err := decodePreservedLocRows(snapshot.Tables["project_loc_entries"])
		return stageOutput{preservedLocRows: rows}, err
	})
	if err != nil {
		return NativeArtifact{}, err
	}
	preservedLocRows := out.preservedLocRows
	out, err = memo.run(stageWorld, sink, func(snapshot Snapshot, sink *imageSink) (stageOutput, error) {
		entries, err := nativeWorld(snapshot, artifactRoot(artifact), includeImported, resolve, sink, &ideologyPlan, preservedLocRows)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "countries", "states", "history_files", "bookmarks", "world_localisation")
	}

	out, err = memo.run(stageOnActions, sink, func(snapshot Snapshot, _ *imageSink) (stageOutput, error) {
		entries, err := NativeOnActionsWithTraitOverlays(snapshot, artifactRoot(artifact), includeImported, traitOverlayNamespaces)
		return stageOutput{entries: entries}, err
	})
	entries = out.entries
	if err != nil {
		return NativeArtifact{}, err
	}
	if len(entries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, entries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "on_actions")
	}
	if plan.EmitDefines && plan.Defines.Path != "" && plan.Defines.Content != "" {
		artifact.Entries, err = overlayEntries(artifact.Entries, []Entry{{
			Path:  artifactRoot(artifact) + "/" + plan.Defines.Path,
			Bytes: []byte(plan.Defines.Content), Origin: "generated",
			SourceEntity: "timeline_defines:" + plan.PlanSHA256,
		}})
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "timeline_defines")
	}

	preservedEntries, preservedWarnings, err := nativePreservedLoc(
		preservedLocRows, artifactRoot(artifact), includeImported, artifact.Entries)
	if err != nil {
		return NativeArtifact{}, err
	}
	for _, warning := range preservedWarnings {
		artifact.Warnings = append(artifact.Warnings, opts.msg.Render(warning))
	}
	if len(preservedEntries) > 0 {
		artifact.Entries, err = overlayEntries(artifact.Entries, preservedEntries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "preserved_localisation")
	}

	if len(rawEntries) > 0 {
		sink.warnRawSharedPlacement(rawEntries)
		artifact.Entries, err = overlayEntries(artifact.Entries, rawEntries)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Coverage = append(artifact.Coverage, "raw_inline_files")
	}
	if err := validateCosmeticGeneratedCollisions(snapshot, includeImported, artifact.Entries); err != nil {
		return NativeArtifact{}, err
	}

	if len(rawEntries) > 0 {
		modRoot := artifactRoot(artifact)
		lines := make([]string, 0, len(rawEntries))
		for _, entry := range rawEntries {
			lines = append(lines, "WROTE: "+strings.TrimPrefix(entry.Path, modRoot+"/"))
		}

		sort.Strings(lines)
		if logEntry := RawOverrideLogEntry(modRoot, lines, opts.msg); logEntry.Path != "" {
			artifact.Entries = append(artifact.Entries, logEntry)
			artifact.Coverage = append(artifact.Coverage, "raw_override_log")
		}
	}
	if opts.gameFilesOnly {

		artifact.Warnings = append(artifact.Warnings, sink.warnings...)
		artifact.stageReuse = memo.reuse()
		return artifact, nil
	}

	if collisions := VanillaCollisions(artifact.Entries, opts.msg); len(collisions) > 0 {
		artifact.Warnings = append(artifact.Warnings, collisions...)
		if logEntry := VanillaCollisionLogEntry(artifactRoot(artifact), collisions, opts.msg); logEntry.Path != "" {
			artifact.Entries = append(artifact.Entries, logEntry)
			artifact.Coverage = append(artifact.Coverage, "vanilla_collision_lint")
		}
	}

	if !opts.skipAdvisory {
		if locReport := ScanLocFallback(artifact.Entries, opts.msg); locReport.FallbackKeys() > 0 {
			artifact.Warnings = append(artifact.Warnings, locReport.Headline())
			if logEntry := LocFallbackLogEntry(artifactRoot(artifact), locReport); logEntry.Path != "" {
				artifact.Entries = append(artifact.Entries, logEntry)
				artifact.Coverage = append(artifact.Coverage, "loc_fallback_lint")
			}
		}
	}

	ApplyWorldLints(&artifact, sink.worldMeta, opts.msg)

	if phReport := ScanPlaceholderIDs(snapshot, opts.msg); phReport.Count() > 0 {
		artifact.Warnings = append(artifact.Warnings, phReport.Headline())
	}

	if dlcReport := ScanDLCConditionals(artifact.Entries, opts.msg); dlcReport.Total() > 0 {
		artifact.Warnings = append(artifact.Warnings, dlcReport.Headline())
		if logEntry := DLCConditionalLogEntry(artifactRoot(artifact), dlcReport); logEntry.Path != "" {
			artifact.Entries = append(artifact.Entries, logEntry)
			artifact.Coverage = append(artifact.Coverage, "dlc_conditional_lint")
		}
	}
	literalReport := ScanLiteralNewlines(artifact.Entries, opts.msg)
	persistedFindings = append(persistedFindings, ExportFindingsFromLiteralNewlines(literalReport, false)...)
	if literalReport.Count() > 0 {
		artifact.Warnings = append(artifact.Warnings, literalReport.Headline())
		artifact.Entries = append(artifact.Entries, LiteralNewlineLogEntry(artifactRoot(artifact), literalReport))
		artifact.Coverage = append(artifact.Coverage, "literal_newline_lint")
	}

	if cover.Warning != "" {
		artifact.Warnings = append(artifact.Warnings, cover.Warning)
		if logEntry := coverWarningEntry(artifactRoot(artifact), cover.Warning); logEntry.Path != "" {
			artifact.Entries = append(artifact.Entries, logEntry)
			artifact.Coverage = append(artifact.Coverage, "project_cover_warning")
		}
	}
	if !opts.skipAdvisory {
		var err error
		if persistedFindings, err = appendAdvisoryReports(&artifact, snapshot, persistedFindings, opts.msg); err != nil {
			return NativeArtifact{}, err
		}
	}

	artifact.Warnings = append(artifact.Warnings, DuplicatePrivateAssets(artifact.Entries, opts.msg)...)
	artifact.Warnings = append(artifact.Warnings, sink.warnings...)

	persistedFindings = append(persistedFindings, ExportFindingsFromIdeaPictureDrops(sink.ideaPictureDrops, false, opts.msg)...)
	if len(persistedFindings) > 0 {
		findingEntry, err := ExportFindingsEntry(artifact.Entries, persistedFindings)
		if err != nil {
			return NativeArtifact{}, err
		}
		artifact.Entries = append(artifact.Entries, findingEntry)
		artifact.Coverage = appendCoverageOnce(artifact.Coverage, "export_findings_v1")
	}
	return artifact, nil
}

func appendAdvisoryReports(artifactPtr *NativeArtifact, snapshot Snapshot, persistedFindings []ExportFinding, msg exportmsg.Printer) ([]ExportFinding, error) {
	artifact := *artifactPtr
	defer func() { *artifactPtr = artifact }()

	syntaxReport := ApplyFinalArtifactSyntax(&artifact, msg)
	persistedFindings = append(persistedFindings, ExportFindingsFromFinalSyntax(syntaxReport, false)...)
	registryReport := ScanRegistryValidation(artifact.Entries, msg)
	persistedFindings = append(persistedFindings, ExportFindingsFromRegistryValidation(registryReport, false)...)
	artifact.Coverage = append(artifact.Coverage, RegistryValidationCoverage)
	if registryReport.Count() > 0 {
		artifact.Warnings = append(artifact.Warnings, registryReport.Headline())
		if logEntry := RegistryValidationLogEntry(artifactRoot(artifact), registryReport); logEntry.Path != "" {
			artifact.Entries = append(artifact.Entries, logEntry)
		}
	}

	lint, err := BuildLintReport(snapshot, artifact.Entries, msg)
	if err != nil {
		return persistedFindings, err
	}
	if logEntries := lint.Entries(artifactRoot(artifact)); len(logEntries) > 0 {
		artifact.Entries = append(artifact.Entries, logEntries...)
		artifact.Coverage = append(artifact.Coverage, "cross_ref_lint")
	}
	refs := BuildReferenceIntegrityReport(snapshot, artifact.Entries, msg)
	persistedFindings = append(persistedFindings, ExportFindingsFromReferenceIntegrity(refs, false)...)
	if entries := refs.Entries(artifactRoot(artifact)); len(entries) > 0 {
		artifact.Entries = append(artifact.Entries, entries...)
		artifact.Coverage = append(artifact.Coverage, "reference_integrity")
	}
	if headline := refs.Headline(); headline != "" {
		artifact.Warnings = append(artifact.Warnings, headline)
	}

	if h := lint.ScriptedHeadline(); h != "" {
		artifact.Warnings = append(artifact.Warnings, h)
	}
	return persistedFindings, nil
}

func nativePanels(snapshot Snapshot, modDir string, includeImported bool) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	var entries []Entry
	for _, raw := range snapshot.Tables["ui_panels"] {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		if !includeImported && textValue(row["source"]) == "imported" {
			continue
		}
		token := panelToken(row["token"])
		if token == "" {
			return nil, fmt.Errorf("UI panel token is required")
		}
		gui, err := compilePanelGUI(row)
		if err != nil {
			return nil, err
		}
		txt, err := compilePanelTXT(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries,
			Entry{Path: modDir + "/interface/zz_uipanel_" + token + ".gui", Bytes: []byte(gui), Origin: "generated", SourceEntity: "ui_panels"},
			Entry{Path: modDir + "/common/scripted_guis/zz_uipanel_" + token + ".txt", Bytes: []byte(txt), Origin: "generated", SourceEntity: "ui_panels"},
		)
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	tag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if tag == "" {
		tag = "mod"
	}
	for _, lang := range emitmisc.BuildExportLangTriples(stringList(project, "export_languages")) {
		text := panelLocalisation(rows, lang.Code)
		if strings.Count(text, "\n") > 0 {
			entries = append(entries, Entry{Path: modDir + "/localisation/" + lang.Subdir + "/" + tag + "_ui_panels_" + lang.Suffix + ".yml", Bytes: []byte("\uFEFF" + text), Origin: "generated", SourceEntity: "ui_panel_localisation"})
		}
	}
	return entries, nil
}

var wargoalBlockFields = []string{"allowed", "available", "take_states", "puppet", "liberate"}
var wargoalScalarFields = []string{"generate_base_cost", "generate_per_state_cost", "take_states_limit", "take_states_cost", "puppet_cost", "force_government_cost", "expire", "threat", "take_states_threat_factor"}

func nativeWargoals(snapshot Snapshot, modDir string, includeImported bool) ([]Entry, error) {
	groups := map[string][]map[string]any{}
	var paths []string
	defaultPath := "common/wargoals/zz_" + SanitizeModDirName(modDir, "hagane") + "_wargoals.txt"
	for _, raw := range snapshot.Tables["wargoal_types"] {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		if !includeImported && textValue(row["source"]) == "imported" {
			continue
		}
		path := strings.Trim(strings.ReplaceAll(textValue(row["source_file"]), "\\", "/"), "/")
		if !strings.HasPrefix(path, "common/wargoals/") || !strings.HasSuffix(strings.ToLower(path), ".txt") {
			path = defaultPath
		}
		if _, err := SafePath(path); err != nil {
			path = defaultPath
		}
		if _, ok := groups[path]; !ok {
			paths = append(paths, path)
		}
		groups[path] = append(groups[path], row)
	}
	sort.Strings(paths)
	entries := make([]Entry, 0, len(paths))
	for _, path := range paths {
		definitions := make([]string, 0, len(groups[path]))
		for _, row := range groups[path] {
			definition, err := renderWargoal(row)
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		trivia, _ := groups[path][0]["file_trivia"].(map[string]any)
		lines, err := renderCommentLeading(nil, trivia["wrapper_leading"], 0)
		if err != nil {
			return nil, err
		}
		lines = append(lines, "wargoal_types = {")
		lines, err = renderCommentLeading(lines, trivia["block_leading"], 1)
		if err != nil {
			return nil, err
		}
		lines = append(lines, strings.Join(definitions, "\n\n"))
		lines, err = renderCommentLeading(lines, trivia["block_trailing"], 1)
		if err != nil {
			return nil, err
		}
		lines = append(lines, "}")
		lines, err = renderCommentsAfterLast(lines, appendCommentLists(trivia["wrapper_trailing"], trivia["root_trailing"]), 0)
		if err != nil {
			return nil, err
		}
		text := strings.Join(lines, "\n") + "\n"
		entries = append(entries, Entry{Path: modDir + "/" + path, Bytes: []byte(text), Origin: "generated", SourceEntity: "wargoal_types"})
	}
	return entries, nil
}

func renderWargoal(row map[string]any) (string, error) {
	id, err := safeScriptToken(row["wargoal_key"])
	if err != nil {
		return "", err
	}
	desired := map[string]string{"war_name": strings.TrimSpace(textValue(row["war_name"]))}
	if scalars, ok := row["safe_scalars"].(map[string]any); ok {
		for _, key := range wargoalScalarFields {
			desired[key] = strings.TrimSpace(textValue(scalars[key]))
		}
	}
	for _, key := range wargoalBlockFields {
		desired[key] = strings.Trim(textValue(row[key]), "\n")
	}
	known := map[string]bool{"war_name": true}
	for _, key := range append(append([]string{}, wargoalBlockFields...), wargoalScalarFields...) {
		known[key] = true
	}
	seen := map[string]bool{}
	var pieces []string
	records, _ := row["ordered_entries"].([]any)
	for _, item := range records {
		record, ok := item.(map[string]any)
		if !ok {
			return "", fmt.Errorf("wargoal %q has invalid ordered entry", id)
		}
		key := textValue(record["key"])
		script := strings.Trim(textValue(record["script"]), "\n")
		if key == "" || script == "" {
			return "", fmt.Errorf("wargoal %q has empty ordered entry", id)
		}
		if _, recoveries, err := paradox.Parse(script); err != nil || len(recoveries) > 0 {
			return "", fmt.Errorf("wargoal %q has invalid ordered entry %q", id, key)
		}
		if known[key] {
			if seen[key] {
				return "", fmt.Errorf("wargoal %q repeats modeled key %q", id, key)
			}
			seen[key] = true
			if value := desired[key]; value != "" || slicesContains(wargoalBlockFields, key) {
				piece, err := renderWargoalField(key, value)
				if err != nil {
					return "", err
				}
				pieces = append(pieces, piece)
			}
		} else {
			pieces = append(pieces, script)
		}
	}
	order := append([]string{"war_name"}, wargoalBlockFields...)
	order = append(order, wargoalScalarFields...)
	for _, key := range order {
		if seen[key] || desired[key] == "" {
			continue
		}
		piece, err := renderWargoalField(key, desired[key])
		if err != nil {
			return "", err
		}
		pieces = append(pieces, piece)
	}
	lines, err := renderCommentLeading(nil, row["definition_leading_comments"], 1)
	if err != nil {
		return "", err
	}
	lines = append(lines, "\t"+id+" = {")
	for _, piece := range pieces {
		for _, line := range strings.Split(piece, "\n") {
			lines = append(lines, "\t\t"+line)
		}
	}
	lines = append(lines, "\t}")
	lines, err = renderCommentsAfterLast(lines, row["definition_trailing_comments"], 1)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func appendCommentLists(values ...any) []any {
	var result []any
	for _, value := range values {
		if items, ok := value.([]any); ok {
			result = append(result, items...)
		}
	}
	return result
}

func commentRecords(value any) ([]map[string]any, error) {
	items, _ := value.([]any)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid stored comment record")
		}
		text := textValue(record["text"])
		if !strings.HasPrefix(text, "#") || strings.ContainsAny(text, "\r\n") {
			return nil, fmt.Errorf("invalid stored comment record")
		}
		result = append(result, record)
	}
	return result, nil
}

func renderCommentLeading(lines []string, value any, indent int) ([]string, error) {
	records, err := commentRecords(value)
	if err != nil {
		return nil, err
	}
	prefix := strings.Repeat("\t", indent)
	for _, record := range records {
		lines = append(lines, prefix+textValue(record["text"]))
	}
	return lines, nil
}

func renderCommentsAfterLast(lines []string, value any, indent int) ([]string, error) {
	records, err := commentRecords(value)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return renderCommentLeading(lines, value, indent)
	}
	prefix := strings.Repeat("\t", indent)
	attachedInline := false
	for _, record := range records {
		inline, _ := record["inline"].(bool)
		if inline && !attachedInline {
			gap := textValue(record["whitespace"])
			if gap == "" {
				gap = " "
			}
			lines[len(lines)-1] += gap + textValue(record["text"])
			attachedInline = true
		} else {
			lines = append(lines, prefix+textValue(record["text"]))
		}
	}
	return lines, nil
}

func renderWargoalField(key, value string) (string, error) {
	if slicesContains(wargoalBlockFields, key) {
		if value != "" {
			if _, recoveries, err := paradox.Parse(value); err != nil || len(recoveries) > 0 {
				return "", fmt.Errorf("invalid wargoal block %q", key)
			}
		}
		if value == "" {
			return key + " = {\n}", nil
		}
		lines := []string{key + " = {"}
		for _, line := range strings.Split(value, "\n") {
			lines = append(lines, "\t"+line)
		}
		return strings.Join(append(lines, "}"), "\n"), nil
	}
	if strings.ContainsAny(value, "\r\n{}=#") {
		return "", fmt.Errorf("invalid wargoal scalar %q", key)
	}
	return key + " = " + value, nil
}

func slicesContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func nativeAIPlanning(snapshot Snapshot, modDir string, includeImported bool) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	tag := strings.ToLower(strings.TrimSpace(project.Str("country_tag")))
	if tag == "" {
		tag = "mod"
	}
	type aiFamily struct {
		table, dir, defaultName, idField string
		template                         bool
	}
	families := []aiFamily{
		{"ai_strategy_plans", "ai_strategy_plans", tag + "_ai_strategy_plans.txt", "plan_id", false},
		{"ai_templates", "ai_templates", tag + "_ai_templates.txt", "template_id", true},
	}
	var entries []Entry
	for _, family := range families {
		groups := map[string][]map[string]any{}
		var names []string
		for _, raw := range snapshot.Tables[family.table] {
			var row map[string]any
			if err := json.Unmarshal(raw, &row); err != nil {
				return nil, fmt.Errorf("decode %s: %w", family.table, err)
			}
			if !includeImported && textValue(row["source"]) == "imported" {
				continue
			}
			name := safeAIPlanningFilename(textValue(row["source_file"]), family.defaultName)
			if _, ok := groups[name]; !ok {
				names = append(names, name)
			}
			groups[name] = append(groups[name], row)
		}
		sort.Strings(names)
		for _, name := range names {
			blocks := make([]string, 0, len(groups[name]))
			for _, row := range groups[name] {
				var block string
				if family.template {
					block, err = renderAITemplate(row)
				} else {
					block, err = renderAIStrategyPlan(row)
				}
				if err != nil {
					return nil, fmt.Errorf("emit %s %q: %w", family.table, textValue(row[family.idField]), err)
				}
				blocks = append(blocks, block)
			}
			if len(blocks) > 0 {
				entries = append(entries, Entry{Path: modDir + "/common/" + family.dir + "/" + name, Bytes: []byte(strings.Join(blocks, "\n\n")), Origin: "generated", SourceEntity: family.table})
			}
		}
	}
	return entries, nil
}

func safeAIPlanningFilename(source, fallback string) string {
	source = strings.ReplaceAll(strings.TrimSpace(source), "\\", "/")
	if slash := strings.LastIndex(source, "/"); slash >= 0 {
		source = source[slash+1:]
	}
	if source == "" || source == "." || source == ".." || !strings.HasSuffix(strings.ToLower(source), ".txt") {
		return fallback
	}
	return source
}

func textValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func safeScriptToken(value any) (string, error) {
	token := strings.TrimSpace(textValue(value))
	if token == "" || strings.ContainsAny(token, " \t\r\n{}=#\"'") {
		return "", fmt.Errorf("unsafe HOI4 token %q", token)
	}
	return token, nil
}

func quoteScript(value any) string {
	text := strings.ReplaceAll(textValue(value), "\\", "\\\\")
	text = strings.ReplaceAll(text, "\"", "\\\"")
	return "\"" + text + "\""
}

func scriptBlock(value any, label string) (string, error) {
	text := strings.TrimSpace(textValue(value))
	if !strings.HasPrefix(text, "{") || !strings.HasSuffix(text, "}") {
		return "", fmt.Errorf("%s must be one brace block", label)
	}
	return text, nil
}

func appendAssignment(lines []string, key, value string) []string {
	parts := strings.Split(strings.TrimSpace(value), "\n")
	if len(parts) == 0 {
		parts = []string{"{\n}"}
	}
	lines = append(lines, "\t"+key+" = "+parts[0])
	for _, part := range parts[1:] {
		lines = append(lines, "\t"+part)
	}
	return lines
}

func stringSliceAny(value any) []string {
	items, _ := value.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, textValue(item))
	}
	return out
}

func renderAIStrategyPlan(row map[string]any) (string, error) {
	if textValue(row["source"]) == "imported" && strings.TrimSpace(textValue(row["raw_block"])) != "" {
		return strings.TrimSpace(textValue(row["raw_block"])), nil
	}
	id, err := safeScriptToken(row["plan_id"])
	if err != nil {
		return "", err
	}
	blockFields := []string{"allowed", "enable", "abort", "ai_national_focuses", "focus_factors", "research", "ideas", "traits", "ai_strategy", "weight"}
	queues := map[string][]string{}
	for _, field := range blockFields {
		queues[field] = stringSliceAny(row[field])
	}
	scalars := map[string]string{"name": textValue(row["name"]), "desc": textValue(row["desc"])}
	emitted := map[string]bool{}
	lines := []string{id + " = {"}
	for _, field := range stringSliceAny(row["field_order"]) {
		if value, ok := scalars[field]; ok && !emitted[field] {
			lines = append(lines, "\t"+field+" = "+quoteScript(value))
			emitted[field] = true
		} else if len(queues[field]) > 0 {
			value, err := scriptBlock(queues[field][0], field)
			if err != nil {
				return "", err
			}
			lines = appendAssignment(lines, field, value)
			queues[field] = queues[field][1:]
		}
	}
	for _, field := range []string{"name", "desc"} {
		if !emitted[field] && scalars[field] != "" {
			lines = append(lines, "\t"+field+" = "+quoteScript(scalars[field]))
		}
	}
	for _, field := range blockFields {
		for _, raw := range queues[field] {
			value, err := scriptBlock(raw, field)
			if err != nil {
				return "", err
			}
			lines = appendAssignment(lines, field, value)
		}
	}
	return strings.Join(append(lines, "}"), "\n"), nil
}

func renderAITemplate(row map[string]any) (string, error) {
	if textValue(row["source"]) == "imported" && strings.TrimSpace(textValue(row["raw_block"])) != "" {
		return strings.TrimSpace(textValue(row["raw_block"])), nil
	}
	id, err := safeScriptToken(row["template_id"])
	if err != nil {
		return "", err
	}
	role, err := safeScriptToken(row["role"])
	if err != nil {
		return "", err
	}
	lines := []string{id + " = {", "\trole = " + role}
	if override := strings.TrimSpace(textValue(row["front_role_override"])); override != "" {
		token, err := safeScriptToken(override)
		if err != nil {
			return "", err
		}
		lines = append(lines, "\tfront_role_override = "+token)
	}
	for _, field := range []string{"blocked_for", "available_for", "upgrade_prio", "enable"} {
		if raw := strings.TrimSpace(textValue(row[field])); raw != "" {
			value, err := scriptBlock(raw, field)
			if err != nil {
				return "", err
			}
			lines = appendAssignment(lines, field, value)
		}
	}
	targets, _ := row["target_templates"].([]any)
	if len(targets) == 0 {
		return "", fmt.Errorf("at least one target template is required")
	}
	for _, item := range targets {
		target, ok := item.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid target template")
		}
		targetID, err := safeScriptToken(target["id"])
		if err != nil {
			return "", err
		}
		body, err := scriptBlock(target["body"], targetID)
		if err != nil {
			return "", err
		}
		lines = appendAssignment(lines, targetID, body)
	}
	return strings.Join(append(lines, "}"), "\n"), nil
}

func nativeCharacters(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	projectTag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if projectTag == "" {
		projectTag = "UNK"
	}
	type row struct {
		raw               json.RawMessage
		misc              *emitmisc.NS
		loc               *locout.Obj
		ctag              string
		portrait, advisor []byte
	}
	obsKeys := map[string]map[int64]string{}
	for _, raw := range snapshot.Tables["entity_obs_paths"] {
		mapping, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, err
		}
		kind := mapping.Str("entity_type")
		if obsKeys[kind] == nil {
			obsKeys[kind] = map[int64]string{}
		}
		obsKeys[kind][mapping.Int("entity_id", 0)] = mapping.Str("obs_key")
	}
	var assetBytes int64
	resolveAsset := func(kind string, id int64) ([]byte, error) {
		key := strings.TrimSpace(obsKeys[kind][id])
		if key == "" {
			return nil, nil
		}
		label := fmt.Sprintf("character asset %s:%d", kind, id)
		data, ok, err := ResolveBlobTolerant(resolve, key, 32<<20, label)
		if err != nil {
			return nil, err
		}
		if !ok {

			sink.skipped(label)
			return nil, nil
		}
		assetBytes += int64(len(data))
		if assetBytes > 1<<30 {
			return nil, fmt.Errorf("character assets exceed byte budget")
		}
		return data, nil
	}
	var rows []row
	for _, raw := range snapshot.Tables["characters"] {
		entity, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("decode character: %w", err)
		}

		if !includeImported && entity.Str("source") == "imported" {
			continue
		}
		value, err := locout.ParseJSON(raw)
		if err != nil || value.Kind != locout.KObj {
			return nil, fmt.Errorf("decode character localisation")
		}
		ctag := strings.ToUpper(strings.TrimSpace(entity.Str("country_tag")))
		if ctag == "" {
			ctag = projectTag
		}
		portrait, err := resolveAsset("character_portrait", entity.Int("id", 0))
		if err != nil {
			return nil, err
		}
		var advisor []byte
		if entity.Str("role_type") == "advisor" || entity.Bool("has_advisor_role", false) {
			advisor, err = resolveAsset("character_advisor", entity.Int("id", 0))
			if err != nil {
				return nil, err
			}
		}
		noteVanillaCharacterIDCollision(sink, entity)
		rows = append(rows, row{raw: raw, misc: entity, loc: value.Obj, ctag: ctag, portrait: portrait, advisor: advisor})
	}
	if len(rows) == 0 {
		return nil, nil
	}

	scripts := make(map[int64]string, len(rows))
	for _, item := range rows {
		request, err := json.Marshal(map[string]any{
			"char": json.RawMessage(item.raw), "country_tag": item.ctag,
			"has_portrait_image": len(item.portrait) > 0, "has_advisor_image": len(item.advisor) > 0,

			"portrait_path_override": passthroughPortraitRef(item.misc.Str("large_portrait"), len(item.portrait) > 0),
			"advisor_path_override":  passthroughPortraitRef(item.misc.Str("small_portrait"), len(item.advisor) > 0),
		})
		if err != nil {
			return nil, err
		}
		script, err := emitpreview.EmitCharacterJSON(request)
		if err != nil {
			return nil, fmt.Errorf("emit character %q: %w", item.misc.Str("tag"), err)
		}
		scripts[item.misc.Int("id", 0)] = script
	}
	type scriptGroup struct {
		path  string
		parts []string
	}
	groupByPath := map[string]*scriptGroup{}
	var groups []*scriptGroup
	for _, item := range rows {
		path := strings.Trim(strings.ReplaceAll(item.misc.Str("source_file"), "\\", "/"), "/")
		if !strings.HasPrefix(path, "common/characters/") {
			path = "common/characters/" + projectTag + "_" + item.ctag + ".txt"
		}
		if _, err := SafePath(path); err != nil {
			path = "common/characters/" + projectTag + "_" + item.ctag + ".txt"
		}
		group := groupByPath[path]
		if group == nil {
			group = &scriptGroup{path: path}
			groupByPath[path] = group
			groups = append(groups, group)
		}
		group.parts = append(group.parts, scripts[item.misc.Int("id", 0)])
	}
	entries := make([]Entry, 0, len(groups)+len(rows)*3)
	for _, group := range groups {
		entries = append(entries, Entry{Path: modDir + "/" + group.path, Bytes: []byte(strings.Join(group.parts, "\n\n")), Origin: "generated", SourceEntity: "characters"})
	}
	gfxByCountry := map[string][]string{}
	for _, item := range rows {
		slug := emitpreview.ExtractSlug(item.misc.Str("tag"), item.ctag)
		if len(item.portrait) > 0 {
			path := emitpreview.ExportPortraitPath(item.ctag, slug)
			entries = append(entries, Entry{Path: modDir + "/" + path, Bytes: item.portrait, Origin: "private_asset", SourceEntity: fmt.Sprintf("character_portrait:%d", item.misc.Int("id", 0))})
			gfxByCountry[item.ctag] = append(gfxByCountry[item.ctag], "\tspriteType = {\n\t\tname = GFX_portrait_"+item.ctag+"_"+slug+"\n\t\ttexturefile = \""+path+"\"\n\t}")
		}
		if len(item.advisor) > 0 {
			path := emitpreview.ExportAdvisorIconPath(item.ctag, slug)
			entries = append(entries, Entry{Path: modDir + "/" + path, Bytes: item.advisor, Origin: "private_asset", SourceEntity: fmt.Sprintf("character_advisor:%d", item.misc.Int("id", 0))})
			gfxByCountry[item.ctag] = append(gfxByCountry[item.ctag], "\tspriteType = {\n\t\tname = GFX_advisor_"+item.ctag+"_"+slug+"\n\t\ttexturefile = \""+path+"\"\n\t}")
		}
	}
	for ctag, blocks := range gfxByCountry {
		entries = append(entries, Entry{Path: modDir + "/interface/" + ctag + "_characters.gfx", Bytes: []byte("spriteTypes = {\n" + strings.Join(blocks, "\n") + "\n}"), Origin: "generated", SourceEntity: "character_assets"})
	}

	buckets := map[string][]*locout.Obj{}
	var bucketOrder []string
	for _, item := range rows {
		if _, ok := buckets[item.ctag]; !ok {
			bucketOrder = append(bucketOrder, item.ctag)
		}
		buckets[item.ctag] = append(buckets[item.ctag], item.loc)
	}
	for _, ctag := range bucketOrder {
		for _, lang := range emitmisc.BuildExportLangTriples(stringList(project, "export_languages")) {
			text := locout.GenerateCharacterLocalisation(buckets[ctag], ctag, lang.Code)
			entries = append(entries, Entry{
				Path:  modDir + "/localisation/" + lang.Subdir + "/" + ctag + "_characters_" + lang.Suffix + ".yml",
				Bytes: []byte("\uFEFF" + text), Origin: "generated", SourceEntity: "character_localisation",
			})
		}
	}
	return entries, nil
}

func artifactRoot(artifact NativeArtifact) string {
	if len(artifact.Entries) == 0 {
		return "mod"
	}
	return strings.TrimSuffix(artifact.Entries[0].Path, ".mod")
}

func nativeTechnologies(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink) ([]Entry, error) {
	rows := snapshot.Tables["technologies"]
	techs := make([]*emitmisc.NS, 0, len(rows))
	for _, raw := range rows {
		entity, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("decode technology snapshot row: %w", err)
		}
		if includeImported || entity.Str("source") == "" || entity.Str("source") == "created" {
			techs = append(techs, entity)
		}
	}
	equipment, err := decodeEntities(snapshot.Tables["equipments"], func(entity *emitmisc.NS) bool {
		return includeImported || entity.Str("source") == "" || entity.Str("source") == "created"
	})
	if err != nil {
		return nil, err
	}
	subUnits, err := decodeEntities(snapshot.Tables["sub_units"], func(entity *emitmisc.NS) bool {
		return includeImported || entity.Str("source") == "" || entity.Str("source") == "created"
	})
	if err != nil {
		return nil, err
	}
	created := func(entity *emitmisc.NS) bool {
		return includeImported || entity.Str("source") == "" || entity.Str("source") == "created"
	}
	templates, err := decodeEntities(snapshot.Tables["division_templates"], created)
	if err != nil {
		return nil, err
	}
	deployments, err := decodeEntities(snapshot.Tables["division_deployments"], created)
	if err != nil {
		return nil, err
	}
	airGroups, err := decodeEntities(snapshot.Tables["oob_air_wing_groups"], created)
	if err != nil {
		return nil, err
	}
	airWings, err := decodeEntities(snapshot.Tables["oob_air_wings"], created)
	if err != nil {
		return nil, err
	}
	wingsByGroup := map[int64][]any{}
	for _, wing := range airWings {
		wingsByGroup[wing.Int("group_id", 0)] = append(wingsByGroup[wing.Int("group_id", 0)], wing)
	}
	for _, group := range airGroups {
		group.Set("wings", wingsByGroup[group.Int("id", 0)])
	}
	fleets, err := decodeEntities(snapshot.Tables["oob_fleets"], created)
	if err != nil {
		return nil, err
	}
	taskForces, err := decodeEntities(snapshot.Tables["oob_task_forces"], created)
	if err != nil {
		return nil, err
	}
	ships, err := decodeEntities(snapshot.Tables["oob_ships"], created)
	if err != nil {
		return nil, err
	}
	shipsByTaskForce := map[int64][]any{}
	for _, ship := range ships {
		shipsByTaskForce[ship.Int("task_force_id", 0)] = append(shipsByTaskForce[ship.Int("task_force_id", 0)], ship)
	}
	taskForcesByFleet := map[int64][]any{}
	for _, taskForce := range taskForces {
		taskForce.Set("ships", shipsByTaskForce[taskForce.Int("id", 0)])
		taskForcesByFleet[taskForce.Int("fleet_id", 0)] = append(taskForcesByFleet[taskForce.Int("fleet_id", 0)], taskForce)
	}
	for _, fleet := range fleets {
		fleet.Set("task_forces", taskForcesByFleet[fleet.Int("id", 0)])
	}
	rawFiles, err := decodeEntities(snapshot.Tables["project_raw_files"], nil)
	if err != nil {
		return nil, err
	}
	if len(techs) == 0 && len(equipment) == 0 && len(subUnits) == 0 && len(templates) == 0 &&
		len(deployments) == 0 && len(airGroups) == 0 && len(fleets) == 0 {
		return nil, nil
	}
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, fmt.Errorf("decode technology project context: %w", err)
	}
	tag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if tag == "" {
		tag = "UNK"
	}
	env := nativeMiscEnv(modDir, tag, project)
	emitmisc.EmitTreesTechMilitary(env, nil, techs, equipment, subUnits, templates, deployments, airGroups, fleets, rawFiles)
	entries := make([]Entry, 0, len(env.Writes))
	for _, write := range env.Writes {

		source := "technologies"
		if strings.HasPrefix(strings.TrimPrefix(write.Path, modDir+"/"), "history/units/") {
			source = "military_oob"
		}
		entries = append(entries, Entry{
			Path: write.Path, Bytes: []byte(write.Text), Origin: "generated", SourceEntity: source,
		})
	}
	obsByEntity, err := entityOBSPaths(snapshot, "tech_icon")
	if err != nil {
		return nil, err
	}
	var techGFX []emitmisc.SpritePair
	var total int64
	for _, tech := range techs {
		key := obsByEntity[tech.Int("id", 0)]
		if key == "" {
			continue
		}
		label := "technology " + tech.Str("tech_id") + " icon"
		raw, ok, err := ResolveBlobTolerant(resolve, key, 32<<20, label)
		if err != nil {
			return nil, err
		}
		if !ok {
			sink.skipped(label)
			continue
		}
		total += int64(len(raw))
		if total > 1<<30 {

			return nil, fmt.Errorf("technology icon assets exceed 1 GiB")
		}
		dds, err := imageBytesToDDS(raw, tech.Str("tech_id")+".png")
		if err != nil {

			sink.skipped(label)
			continue
		}
		path := emitmisc.ExportTechIconPath(tech.Str("tech_id"))
		techGFX = append(techGFX, emitmisc.SpritePair{Name: emitmisc.TechIconSpriteName(tech.Str("tech_id")), Path: path})
		entries = append(entries, Entry{Path: modDir + "/" + path, Bytes: dds, Origin: "generated", SourceEntity: "technology_icons"})
	}
	if len(techGFX) > 0 {
		entries = append(entries, Entry{Path: modDir + "/interface/" + tag + "_technologies.gfx", Bytes: []byte(emitmisc.EmitSpriteBlock(techGFX)), Origin: "generated", SourceEntity: "technology_icons"})
	}
	return entries, nil
}

func nativeMiscEnv(modDir, tag string, project *emitmisc.NS) *emitmisc.Env {
	langs := stringList(project, "export_languages")
	return &emitmisc.Env{
		ModDir: modDir + "/", Tag: tag, Project: project,
		LangTriples: emitmisc.BuildExportLangTriples(langs), Gate: func(text string) string { return text },
		VanillaFocusIcons:    map[string]bool{},
		BuildFocusGfxEntries: func([]*emitmisc.NS) []emitmisc.FocusGfxEntry { return nil },
		BuildTechGfxEntries:  func([]*emitmisc.NS) []emitmisc.SpritePair { return nil },
		BuildCatGfxEntries:   func([]*emitmisc.NS) []emitmisc.SpritePair { return nil },
		BuildDecGfxEntries:   func([]*emitmisc.NS) []emitmisc.SpritePair { return nil },
		PngToDDS:             func(string) string { return "" }, MakePlaceholderDDS: func(string) string { return "" },
		SEPictureObs: map[int64]string{}, ResolveSEPicture: func(string) bool { return false },
		SkipCounter: emitmisc.NewOD(), StructuredUnitPaths: map[string]bool{},
	}
}

func stringList(entity *emitmisc.NS, field string) []string {
	values := entity.List(field)
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func rawFileScanChunks(snapshot Snapshot, includeImported bool) ([]string, error) {
	var chunks []string
	for _, raw := range snapshot.Tables["project_raw_files"] {
		row, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("decode raw file for reference scan: %w", err)
		}
		if !includeImported && row.Str("source") == "imported" {
			continue
		}
		if IsTombstoned(row) || row.Bool("is_binary", false) {
			continue
		}
		if text := row.Str("content"); text != "" {
			chunks = append(chunks, text)
		}
	}
	return chunks, nil
}

func noteDormantSuperEvents(env *emitmisc.Env, sink *imageSink) {
	if env == nil || env.SkipCounter == nil {
		return
	}
	const prefix = "super_event_dormant:"
	for _, key := range env.SkipCounter.Keys() {
		id, ok := strings.CutPrefix(key, prefix)
		if !ok || id == "" {
			continue
		}
		sink.say("super_event.dormant", exportmsg.Args{"id": id})
	}
}

func nativeDecisions(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink) ([]Entry, error) {
	decisions, err := decodeEntities(snapshot.Tables["decisions"], func(entity *emitmisc.NS) bool {
		return includeImported || entity.Str("source") == "" || entity.Str("source") == "created"
	})
	if err != nil {
		return nil, err
	}
	superEvents, err := decodeEntities(snapshot.Tables["super_events"], func(entity *emitmisc.NS) bool {
		return includeImported || entity.Str("source") == "" || entity.Str("source") == "created"
	})
	if err != nil {
		return nil, err
	}
	if len(decisions) == 0 && len(superEvents) == 0 {
		return nil, nil
	}
	categories, err := decodeEntities(snapshot.Tables["decision_categories"], nil)
	if err != nil {
		return nil, err
	}
	events, err := decodeEntities(snapshot.Tables["events"], nil)
	if err != nil {
		return nil, err
	}
	trees, err := decodeEntities(snapshot.Tables["focus_trees"], nil)
	if err != nil {
		return nil, err
	}
	nodes, err := decodeEntities(snapshot.Tables["focus_nodes"], nil)
	if err != nil {
		return nil, err
	}
	nodesByTree := map[int64][]any{}
	for _, node := range nodes {
		nodesByTree[node.Int("focus_tree_id", 0)] = append(nodesByTree[node.Int("focus_tree_id", 0)], node)
	}
	for _, tree := range trees {
		tree.Set("nodes", nodesByTree[tree.Int("id", 0)])
	}
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	tag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if tag == "" {
		tag = "UNK"
	}
	env := nativeMiscEnv(modDir, tag, project)
	decOBS, err := entityOBSPaths(snapshot, "decision_icon")
	if err != nil {
		return nil, err
	}
	catOBS, err := entityOBSPaths(snapshot, "decision_category_icon")
	if err != nil {
		return nil, err
	}
	ddsBySprite := map[string]string{}
	for _, group := range []struct {
		items   []*emitmisc.NS
		mapping map[int64]string
		sprite  func(string) string
		field   string
	}{
		{decisions, decOBS, emitmisc.DecisionIconSpriteName, "decision_id"},
		{categories, catOBS, emitmisc.DecisionCategoryIconSpriteName, "category_id"},
	} {
		for _, item := range group.items {
			key := group.mapping[item.Int("id", 0)]
			if key == "" {
				continue
			}
			id := item.Str(group.field)
			dds, ok, err := ResolveImageDDSTolerant(resolve, key, id)
			if err != nil {
				return nil, err
			}
			if !ok {
				sink.skipped(id)
				continue
			}
			ddsBySprite[group.sprite(id)] = string(dds)
		}
	}
	buildIcons := func(items []*emitmisc.NS, mapping map[int64]string, spriteFn func(string) string, pathFn func(string) string, idField string) []emitmisc.SpritePair {
		var pairs []emitmisc.SpritePair
		for _, item := range items {
			key := mapping[item.Int("id", 0)]
			if key == "" {
				continue
			}
			id := item.Str(idField)
			sprite, assetPath := spriteFn(id), pathFn(id)
			if ddsBySprite[sprite] != "" {
				pairs = append(pairs, emitmisc.SpritePair{Name: sprite, Path: assetPath})
			}
		}
		return pairs
	}
	env.BuildDecGfxEntries = func(items []*emitmisc.NS) []emitmisc.SpritePair {
		return buildIcons(items, decOBS, emitmisc.DecisionIconSpriteName, emitmisc.ExportDecisionIconPath, "decision_id")
	}
	env.BuildCatGfxEntries = func(items []*emitmisc.NS) []emitmisc.SpritePair {
		return buildIcons(items, catOBS, emitmisc.DecisionCategoryIconSpriteName, emitmisc.ExportDecisionCategoryIconPath, "category_id")
	}
	seOBS, err := entityOBSPaths(snapshot, "super_event_picture")
	if err != nil {
		return nil, err
	}
	env.SEPictureObs = seOBS
	resolvedSEKeys := map[string]bool{}
	for _, se := range superEvents {
		key := seOBS[se.Int("id", 0)]
		if key == "" {
			continue
		}
		ctag := strings.ToUpper(strings.TrimSpace(se.Str("country_tag")))
		if ctag == "" {
			ctag = tag
		}
		sprite := fmt.Sprintf("GFX_%s_SE_%04d", ctag, se.Int("number", 0))
		dds, ok, err := ResolveImageDDSTolerant(resolve, key, "super event "+sprite)
		if err != nil {
			return nil, err
		}
		if !ok {
			sink.skipped("super event " + sprite)
			continue
		}
		ddsBySprite[sprite] = string(dds)
		resolvedSEKeys[key] = true
	}
	env.ResolveSEPicture = func(key string) bool { return resolvedSEKeys[key] }
	env.MakePlaceholderDDS = func(name string) string {
		spec, ok := sePlaceholderFrames[name]
		if !ok {
			return ""
		}
		return string(makePlaceholderDDS(spec))
	}
	env.PngToDDS = func(sprite string) string { return ddsBySprite[sprite] }
	env.RawFileScanChunks, err = rawFileScanChunks(snapshot, includeImported)
	if err != nil {
		return nil, err
	}
	emitmisc.EmitDecisionsSuperevents(env, decisions, categories, superEvents, events, trees)
	noteDormantSuperEvents(env, sink)
	entries := make([]Entry, 0, len(env.Writes))
	for _, write := range env.Writes {
		entries = append(entries, Entry{
			Path: write.Path, Bytes: []byte(write.Text), Origin: "generated", SourceEntity: "decisions",
		})
	}
	return entries, nil
}

func nativeFocus(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink) ([]Entry, error) {
	allTrees, err := decodeEntities(snapshot.Tables["focus_trees"], nil)
	if err != nil {
		return nil, err
	}
	if len(allTrees) == 0 {
		return nil, nil
	}
	nodes, err := decodeEntities(snapshot.Tables["focus_nodes"], nil)
	if err != nil {
		return nil, err
	}
	nodesByTree := map[int64][]any{}
	for _, node := range nodes {
		treeID := node.Int("focus_tree_id", 0)
		nodesByTree[treeID] = append(nodesByTree[treeID], node)
	}
	treesByID := map[string]*emitmisc.NS{}
	var trees []*emitmisc.NS
	for _, tree := range allTrees {
		tree.Set("nodes", nodesByTree[tree.Int("id", 0)])
		treesByID[tree.Str("tree_id")] = tree
		if includeImported || tree.Str("source") == "" || tree.Str("source") == "created" {
			trees = append(trees, tree)
		}
	}
	if len(trees) == 0 {
		return nil, nil
	}
	originalTrees := trees
	trees, offsetWarnings := emitmisc.CompileSharedReferenceOffsets(trees)
	compiledSharedPools := compiledSharedPoolViews(originalTrees, trees)
	for _, warning := range offsetWarnings {
		sink.say(warning.Key, warning.Args)
	}
	for _, tree := range trees {
		treesByID[tree.Str("tree_id")] = tree
	}
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}

	projectCountryTag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	tag := projectCountryTag
	if tag == "" {
		tag = "UNK"
	}
	obsIcons, err := projectOBSIcons(snapshot)
	if err != nil {
		return nil, err
	}
	resolvedIcons := map[string]bool{}
	iconEntries := map[string]Entry{}
	type focusGFX struct {
		ctag, name, path string
		shine            bool
	}
	var focusGFXEntries []focusGFX
	seenGFX := map[string]bool{}
	registerIcon := func(ctag, name, path string, shine bool) error {
		seenKey := ctag + "\x00" + name
		if name == "" || seenGFX[seenKey] {
			return nil
		}
		key := obsIcons[name]
		if key == "" {
			return nil
		}
		label := "focus icon " + name
		raw, ok, err := ResolveBlobTolerant(resolve, key, 32<<20, label)
		if err != nil {
			return err
		}
		if !ok {
			sink.skipped(label)
			return nil
		}
		dds, err := imageBytesToDDS(raw, name+".png")
		if err != nil {

			sink.skipped(label)
			return nil
		}
		seenGFX[seenKey], resolvedIcons[name] = true, true
		focusGFXEntries = append(focusGFXEntries, focusGFX{ctag, name, path, shine})
		iconEntries[path] = Entry{Path: modDir + "/" + path, Bytes: dds, Origin: "generated", SourceEntity: "focus_icons"}
		return nil
	}
	treeTags := map[int64]string{}
	for _, tree := range trees {
		ctag := strings.ToUpper(strings.TrimSpace(tree.Str("country_tag")))
		if ctag == "" {
			ctag = tag
		}
		treeTags[tree.Int("id", 0)] = ctag
	}
	for _, node := range nodes {
		ctag := treeTags[node.Int("focus_tree_id", 0)]
		if ctag == "" {
			continue
		}
		focusID := node.Str("focus_id")
		if err := registerIcon(ctag, emitmisc.FocusIconSpriteName(focusID), emitmisc.ExportFocusIconPath(focusID), true); err != nil {
			return nil, err
		}
		for _, value := range node.List("dynamic_icon") {
			if obj, ok := value.(map[string]any); ok {
				name, _ := obj["icon"].(string)
				name = strings.TrimSpace(name)
				if err := registerIcon(ctag, name, emitmisc.LibraryIconZipPath(name), true); err != nil {
					return nil, err
				}
			}
		}
		name := strings.TrimSpace(node.Str("text_icon"))
		if err := registerIcon(ctag, name, emitmisc.LibraryIconZipPath(name), false); err != nil {
			return nil, err
		}
	}
	type focusGroup struct {
		path  string
		trees []*emitmisc.NS
	}
	groupByPath := map[string]*focusGroup{}
	var groups []*focusGroup
	assigned := map[string]bool{}
	policies, err := decodeEntities(snapshot.Tables["focus_file_policies"], nil)
	if err != nil {
		return nil, err
	}
	selected := map[string]bool{}
	for _, tree := range trees {
		selected[tree.Str("tree_id")] = true
	}
	for _, policy := range policies {
		target := strings.Trim(strings.ReplaceAll(policy.Str("target_path"), "\\", "/"), "/")
		if target == "" {
			continue
		}
		if _, err := SafePath(target); err != nil {
			return nil, fmt.Errorf("focus policy %d: %w", policy.Int("id", 0), err)
		}
		for _, value := range policy.List("tree_ids") {
			treeID, _ := value.(string)
			tree := treesByID[treeID]
			if tree == nil || !selected[treeID] {
				continue
			}
			group := groupByPath[target]
			if group == nil {
				group = &focusGroup{path: target}
				groupByPath[target] = group
				groups = append(groups, group)
			}
			group.trees = append(group.trees, tree)
			assigned[treeID] = true
		}
	}
	for _, tree := range trees {
		if assigned[tree.Str("tree_id")] {
			continue
		}

		if tree.Str("source") == "imported" && normaliseFocusPath(tree.Str("source_file")) == "" {
			return nil, fmt.Errorf("imported focus tree %q has no usable source_file and no persisted file policy", tree.Str("tree_id"))
		}

		target := FocusFallbackPath(tree.Str("source_file"), tree.Str("tree_id"), tag)
		if _, err := SafePath(target); err != nil {
			return nil, err
		}
		group := groupByPath[target]
		if group == nil {
			group = &focusGroup{path: target}
			groupByPath[target] = group
			groups = append(groups, group)
		}
		group.trees = append(group.trees, tree)
	}

	iconGate := FocusIconGate(len(project.List("dependencies")) > 0, resolvedIcons)
	var entries []Entry
	for _, group := range groups {
		sort.SliceStable(group.trees, func(i, j int) bool {
			ik, jk := group.trees[i].Str("tree_kind"), group.trees[j].Str("tree_kind")
			if ik != jk {
				return ik == "primary"
			}
			return group.trees[i].Str("tree_id") < group.trees[j].Str("tree_id")
		})
		parts := make([]string, 0, len(group.trees))
		for _, tree := range group.trees {

			plan := emitmisc.PlanFocusTreeSelection(
				tree.StrOr("scope_kind", "own"), tree.Str("country_tag"),
				projectCountryTag, stringList(tree, "shared_country_tags"),
			)

			if plan.Note.Key != "" && len(tree.List("nodes")) > 0 {
				sink.say("focus.tree_note", exportmsg.Args{"tree": tree.Str("tree_id"), "note": plan.Note})
			}
			parts = append(parts, emitmisc.GenerateFocusTreeScriptWithFallback(tree, iconGate, projectCountryTag))
			if sink != nil && compiledSharedPools[tree] {
				if sink.sharedPlacementPaths == nil {
					sink.sharedPlacementPaths = map[string]bool{}
				}
				sink.sharedPlacementPaths[modDir+"/"+group.path] = true
			}
		}
		entries = append(entries, Entry{
			Path: modDir + "/" + group.path, Bytes: []byte(strings.TrimRight(strings.Join(parts, "\n\n"), " \t\r\n") + "\n"),
			Origin: "generated", SourceEntity: "focus_trees",
		})
	}
	buckets := map[string][]*emitmisc.NS{}
	var bucketOrder []string
	for _, tree := range trees {
		ctag := strings.ToUpper(strings.TrimSpace(tree.Str("country_tag")))
		if ctag == "" {
			ctag = tag
		}
		if _, exists := buckets[ctag]; !exists {
			bucketOrder = append(bucketOrder, ctag)
		}
		buckets[ctag] = append(buckets[ctag], tree)
	}
	for _, ctag := range bucketOrder {
		for _, lang := range emitmisc.BuildExportLangTriples(stringList(project, "export_languages")) {
			text := emitmisc.GenerateFocusLocalisation(buckets[ctag], lang.Code)
			entries = append(entries, Entry{
				Path:  modDir + "/localisation/" + lang.Subdir + "/" + ctag + "_focus_" + lang.Suffix + ".yml",
				Bytes: []byte("\uFEFF" + text), Origin: "generated", SourceEntity: "focus_localisation",
			})
		}
	}
	if len(focusGFXEntries) > 0 {
		pairsByTag := map[string][]emitmisc.SpritePair{}
		shineByTag := map[string][]emitmisc.SpritePair{}
		for _, item := range focusGFXEntries {
			pair := emitmisc.SpritePair{Name: item.name, Path: item.path}
			pairsByTag[item.ctag] = append(pairsByTag[item.ctag], pair)
			if item.shine {
				shineByTag[item.ctag] = append(shineByTag[item.ctag], pair)
			}
		}
		gfxTags := make([]string, 0, len(pairsByTag))
		for ctag := range pairsByTag {
			gfxTags = append(gfxTags, ctag)
		}
		sort.Strings(gfxTags)
		for _, ctag := range gfxTags {
			entries = append(entries, Entry{Path: modDir + "/interface/" + ctag + "_focus.gfx", Bytes: []byte(emitmisc.EmitSpriteBlock(pairsByTag[ctag])), Origin: "generated", SourceEntity: "focus_icons"})
			if len(shineByTag[ctag]) > 0 {
				entries = append(entries, Entry{Path: modDir + "/interface/" + ctag + "_focus_shine.gfx", Bytes: []byte(emitmisc.EmitFocusShineBlock(shineByTag[ctag])), Origin: "generated", SourceEntity: "focus_icons"})
			}
		}
		paths := make([]string, 0, len(iconEntries))
		for path := range iconEntries {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			entries = append(entries, iconEntries[path])
		}
	}
	return entries, nil
}

func projectOBSIcons(snapshot Snapshot) (map[string]string, error) {
	out := map[string]string{}
	for _, raw := range snapshot.Tables["project_obs_icons"] {
		var row struct {
			GFXName string `json:"gfx_name"`
			OBSKey  string `json:"obs_key"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("decode project_obs_icons: %w", err)
		}
		if row.GFXName != "" && row.OBSKey != "" {
			out[row.GFXName] = row.OBSKey
		}
	}
	return out, nil
}

func entityOBSPaths(snapshot Snapshot, entityType string) (map[int64]string, error) {
	out := map[int64]string{}
	for _, raw := range snapshot.Tables["entity_obs_paths"] {
		var row struct {
			EntityType string `json:"entity_type"`
			EntityID   int64  `json:"entity_id"`
			OBSKey     string `json:"obs_key"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("decode entity_obs_paths: %w", err)
		}
		if row.EntityType == entityType && row.EntityID != 0 && row.OBSKey != "" {
			out[row.EntityID] = row.OBSKey
		}
	}
	return out, nil
}

func imageBytesToDDS(raw []byte, filename string) ([]byte, error) {

	cacheKey := ddsCacheKey(raw, filename)
	if dds, ok := processDDSCache.get(cacheKey); ok {
		return dds, nil
	}
	dds, err := imageBytesToDDSUncached(raw, filename)
	if err != nil {
		return nil, err
	}
	processDDSCache.put(cacheKey, dds)
	return dds, nil
}

func imageBytesToDDSUncached(raw []byte, filename string) ([]byte, error) {
	decoded, err := importassets.DecodeImage(raw, filename)
	if err != nil {
		return nil, err
	}
	b := decoded.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("empty image")
	}
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), decoded, b.Min, draw.Src)
	dds := imgcodec.EncodeDXT5DDS(nrgba.Pix, b.Dx(), b.Dy())
	if dds == nil {
		return nil, fmt.Errorf("DDS encode failed")
	}
	return dds, nil
}

type placeholderFrame struct {
	Width, Height int
	Fill          [4]byte
	Border        [4]byte
}

var sePlaceholderFrames = map[string]placeholderFrame{

	"SE_bg.dds": {500, 700, [4]byte{22, 26, 34, 240}, [4]byte{70, 78, 92, 255}},

	"SE_frame.dds": {500, 700, [4]byte{0, 0, 0, 0}, [4]byte{150, 156, 170, 255}},

	"SE_pic.dds": {460, 252, [4]byte{52, 58, 70, 255}, [4]byte{96, 104, 120, 255}},

	"SE_btn.dds": {120, 32, [4]byte{60, 72, 96, 255}, [4]byte{120, 136, 164, 255}},
}

func SuperEventPlaceholderDDS(name string) []byte {
	spec, ok := sePlaceholderFrames[name]
	if !ok {
		return nil
	}
	return makePlaceholderDDS(spec)
}

func makePlaceholderDDS(spec placeholderFrame) []byte {
	pix := make([]byte, spec.Width*spec.Height*4)
	for y := 0; y < spec.Height; y++ {
		for x := 0; x < spec.Width; x++ {
			at := (y*spec.Width + x) * 4
			colour := spec.Fill
			if x < 2 || y < 2 || x >= spec.Width-2 || y >= spec.Height-2 {
				colour = spec.Border
			}
			pix[at], pix[at+1], pix[at+2], pix[at+3] = colour[0], colour[1], colour[2], colour[3]
		}
	}
	return imgcodec.EncodeDXT5DDS(pix, spec.Width, spec.Height)
}

type nativeRow struct {
	raw  json.RawMessage
	misc *emitmisc.NS
	loc  *locout.Obj
}

func nativeRows(rows []json.RawMessage, includeImported bool) ([]nativeRow, error) {
	out := make([]nativeRow, 0, len(rows))
	for _, raw := range rows {
		misc, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, err
		}
		if !includeImported && misc.Str("source") == "imported" {
			continue
		}
		value, err := locout.ParseJSON(raw)
		if err != nil || value.Kind != locout.KObj {
			return nil, fmt.Errorf("decode localisation entity")
		}
		out = append(out, nativeRow{raw: raw, misc: misc, loc: value.Obj})
	}
	return out, nil
}

func nativeIdeasEvents(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	tag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if tag == "" {
		tag = "UNK"
	}
	langs := emitmisc.BuildExportLangTriples(stringList(project, "export_languages"))
	var entries []Entry

	ideas, err := nativeRows(snapshot.Tables["ideas"], includeImported)
	if err != nil {
		return nil, err
	}
	ideaOBS, err := entityOBSPaths(snapshot, "idea_picture")
	if err != nil {
		return nil, err
	}
	ideaRefs := map[int64]string{}
	ideaGFX := map[string][]emitmisc.SpritePair{}
	for _, row := range ideas {
		key := ideaOBS[row.misc.Int("id", 0)]
		if key == "" {
			continue
		}
		ctag := strings.ToUpper(strings.TrimSpace(row.misc.Str("country_tag")))
		if ctag == "" {
			ctag = tag
		}
		slug := emitmisc.Slugify(row.misc.Str("idea_id"))
		sprite := "GFX_idea_" + ctag + "_" + slug
		assetPath := "gfx/interface/ideas/idea_" + ctag + "_" + slug + ".dds"
		dds, ok, err := ResolveImageDDSTolerant(resolve, key, "idea "+row.misc.Str("idea_id"))
		if err != nil {
			return nil, err
		}
		if !ok {
			sink.skipped("idea " + row.misc.Str("idea_id"))
			continue
		}
		ideaRefs[row.misc.Int("id", 0)] = strings.TrimPrefix(sprite, "GFX_idea_")
		ideaGFX[ctag] = append(ideaGFX[ctag], emitmisc.SpritePair{Name: sprite, Path: assetPath})
		entries = append(entries, Entry{Path: modDir + "/" + assetPath, Bytes: dds, Origin: "generated", SourceEntity: "idea_pictures"})
	}

	resolveIdeaPictureRefs(ideas, ideaRefs, len(project.List("dependencies")) > 0, projectSpriteNames(snapshot, includeImported), sink)
	ideaBuckets := map[string][]nativeRow{}
	var ideaOrder []string
	for _, row := range ideas {
		ctag := strings.ToUpper(strings.TrimSpace(row.misc.Str("country_tag")))
		if ctag == "" {
			ctag = tag
		}
		if _, ok := ideaBuckets[ctag]; !ok {
			ideaOrder = append(ideaOrder, ctag)
		}
		ideaBuckets[ctag] = append(ideaBuckets[ctag], row)
	}
	for _, ctag := range ideaOrder {
		rows := ideaBuckets[ctag]
		groups := groupNativeRows(rows, func(row nativeRow) string {
			return "common/ideas/" + strings.ToLower(tag) + "_" + strings.ToLower(ctag) + ".txt"
		}, "common/ideas")
		for _, group := range groups {
			raw := make([]json.RawMessage, 0, len(group.rows))
			for _, row := range group.rows {
				raw = append(raw, row.raw)
			}
			body, _ := json.Marshal(map[string]any{"ideas": raw, "picture_refs": ideaRefs})
			script, err := emitpreview.EmitIdeasJSON(body)
			if err != nil {
				return nil, err
			}
			entries = append(entries, Entry{Path: modDir + "/" + group.path, Bytes: []byte(script), Origin: "generated", SourceEntity: "ideas"})
		}
		locObjects := make([]*locout.Obj, 0, len(rows))
		for _, row := range rows {
			locObjects = append(locObjects, row.loc)
		}
		for _, lang := range langs {
			text := locout.GenerateIdeasLocalisation(locObjects, lang.Code)
			entries = append(entries, Entry{
				Path:  modDir + "/localisation/" + lang.Subdir + "/" + ctag + "_ideas_" + lang.Suffix + ".yml",
				Bytes: []byte("\uFEFF" + text), Origin: "generated", SourceEntity: "idea_localisation",
			})
		}
	}
	for ctag, pairs := range ideaGFX {
		entries = append(entries, Entry{Path: modDir + "/interface/" + ctag + "_ideas.gfx", Bytes: []byte(emitmisc.EmitSpriteBlock(pairs)), Origin: "generated", SourceEntity: "idea_pictures"})
	}

	namespaces, err := nativeRows(snapshot.Tables["event_namespaces"], true)
	if err != nil {
		return nil, err
	}
	events, err := nativeRows(snapshot.Tables["events"], includeImported)
	if err != nil {
		return nil, err
	}
	nsByID := map[int64]nativeRow{}
	for _, namespace := range namespaces {
		nsByID[namespace.misc.Int("id", 0)] = namespace
	}
	eventOBS, err := entityOBSPaths(snapshot, "event_picture")
	if err != nil {
		return nil, err
	}
	eventPictures := map[string]bool{}
	eventPictureRefs := map[int64]string{}
	var eventGFX []emitmisc.SpritePair
	for _, row := range events {
		key := eventOBS[row.misc.Int("id", 0)]
		if key == "" {
			continue
		}
		ctag := tag
		if ns := nsByID[row.misc.Int("namespace_id", 0)].misc; ns != nil && strings.TrimSpace(ns.Str("country_tag")) != "" {
			ctag = strings.ToUpper(strings.TrimSpace(ns.Str("country_tag")))
		}
		slug := emitmisc.Slugify(row.misc.Str("event_id"))
		sprite := "GFX_report_event_" + ctag + "_" + slug
		assetPath := "gfx/event_pictures/event_" + ctag + "_" + slug + ".dds"
		dds, ok, err := ResolveImageDDSTolerant(resolve, key, "event "+row.misc.Str("event_id"))
		if err != nil {
			return nil, err
		}
		if !ok {
			sink.skipped("event " + row.misc.Str("event_id"))
			continue
		}
		row.misc.Set("picture", sprite)
		eventPictures[sprite] = true
		eventPictureRefs[row.misc.Int("id", 0)] = sprite
		eventGFX = append(eventGFX, emitmisc.SpritePair{Name: sprite, Path: assetPath})
		entries = append(entries, Entry{Path: modDir + "/" + assetPath, Bytes: dds, Origin: "generated", SourceEntity: "event_pictures"})
	}
	eventGroups := groupNativeRows(events, func(row nativeRow) string {
		namespace := nsByID[row.misc.Int("namespace_id", 0)].misc
		name := "events"
		if namespace != nil && namespace.Str("namespace") != "" {
			name = namespace.Str("namespace")
		}
		return "events/" + tag + "_" + name + ".txt"
	}, "events")
	for _, group := range eventGroups {
		byNamespace := map[int64][]json.RawMessage{}
		var nsOrder []int64
		for _, row := range group.rows {
			id := row.misc.Int("namespace_id", 0)
			if _, ok := byNamespace[id]; !ok {
				nsOrder = append(nsOrder, id)
			}
			raw := row.raw
			if picture := eventPictureRefs[row.misc.Int("id", 0)]; picture != "" {
				var obj map[string]any
				if err := json.Unmarshal(row.raw, &obj); err != nil {
					return nil, err
				}
				obj["picture"] = picture
				raw, _ = json.Marshal(obj)
			}
			byNamespace[id] = append(byNamespace[id], raw)
		}
		var segments []string
		for _, id := range nsOrder {
			namespace := nsByID[id].misc
			if namespace == nil {
				continue
			}
			body, _ := json.Marshal(map[string]any{
				"namespace": namespace.Str("namespace"), "events": byNamespace[id], "valid_pictures": eventPictures,
			})
			script, err := emitpreview.EmitEventsJSON(body)
			if err != nil {
				return nil, err
			}
			segments = append(segments, script)
		}
		if len(segments) > 0 {
			entries = append(entries, Entry{Path: modDir + "/" + group.path, Bytes: []byte(strings.Join(segments, "\n")), Origin: "generated", SourceEntity: "events"})
		}
	}
	if len(eventGFX) > 0 {
		entries = append(entries, Entry{Path: modDir + "/interface/" + tag + "_eventpictures.gfx", Bytes: []byte(emitmisc.EmitSpriteBlock(eventGFX)), Origin: "generated", SourceEntity: "event_pictures"})
	}
	nsBuckets := map[string][]*locout.Obj{}
	eventBuckets := map[string][]*locout.Obj{}
	var eventBucketOrder []string
	ensureBucket := func(ctag string) {
		if _, ok := eventBuckets[ctag]; !ok {
			eventBucketOrder = append(eventBucketOrder, ctag)
			eventBuckets[ctag] = nil
		}
	}
	for _, namespace := range namespaces {
		ctag := strings.ToUpper(strings.TrimSpace(namespace.misc.Str("country_tag")))
		if ctag == "" {
			ctag = tag
		}
		ensureBucket(ctag)
		nsBuckets[ctag] = append(nsBuckets[ctag], namespace.loc)
	}
	for _, event := range events {
		namespace := nsByID[event.misc.Int("namespace_id", 0)]
		ctag := tag
		if namespace.misc != nil && strings.TrimSpace(namespace.misc.Str("country_tag")) != "" {
			ctag = strings.ToUpper(strings.TrimSpace(namespace.misc.Str("country_tag")))
		}
		ensureBucket(ctag)
		eventBuckets[ctag] = append(eventBuckets[ctag], event.loc)
	}

	for _, ctag := range eventBucketOrder {
		if len(events) == 0 {
			break
		}
		for _, lang := range langs {
			text := locout.GenerateEventsLocalisation(nsBuckets[ctag], eventBuckets[ctag], lang.Code)
			entries = append(entries, Entry{
				Path:  modDir + "/localisation/" + lang.Subdir + "/" + ctag + "_events_" + lang.Suffix + ".yml",
				Bytes: []byte("\uFEFF" + text), Origin: "generated", SourceEntity: "event_localisation",
			})
		}
	}
	return entries, nil
}

func resolveImageDDS(resolve BlobResolver, key, label string) ([]byte, error) {
	if resolve == nil {
		return nil, fmt.Errorf("%s requires private object resolver", label)
	}
	raw, err := resolve(key, 32<<20)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", label, err)
	}
	dds, err := imageBytesToDDS(raw, label+".png")
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", label, err)
	}
	return dds, nil
}

type nativeRowGroup struct {
	path string
	rows []nativeRow
}

func groupNativeRows(rows []nativeRow, defaultPath func(nativeRow) string, expectedDir string) []nativeRowGroup {
	var groups []nativeRowGroup
	index := map[string]int{}
	for _, row := range rows {
		path := groupedSourcePath(row.misc.Str("source_file"), expectedDir)
		if path == "" {
			path = defaultPath(row)
		}
		at, ok := index[path]
		if !ok {
			at = len(groups)
			index[path] = at
			groups = append(groups, nativeRowGroup{path: path})
		}
		groups[at].rows = append(groups[at].rows, row)
	}
	return groups
}

func groupedSourcePath(sourceFile, expectedDir string) string {
	raw := strings.TrimSpace(strings.ReplaceAll(sourceFile, "\\", "/"))
	if raw == "" || strings.Contains(raw, ":") {
		return ""
	}
	var parts []string
	for _, part := range strings.Split(raw, "/") {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return ""
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return ""
	}
	candidate := strings.Join(parts, "/")
	prefix := strings.Trim(expectedDir, "/")
	if strings.HasPrefix(candidate, prefix+"/") {
		return candidate
	}
	return prefix + "/" + parts[len(parts)-1]
}

func nativeModifiers(snapshot Snapshot, modDir string, includeImported bool) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	tag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if tag == "" {
		tag = "UNK"
	}
	langs := emitmisc.BuildExportLangTriples(stringList(project, "export_languages"))
	var entries []Entry
	type family struct {
		table, requestKey, path string
		emit                    func([]byte) (string, error)
		loc                     func([]*locout.Obj, string) string
		locName                 string
	}
	families := []family{
		{
			table: "opinion_modifiers", requestKey: "modifiers",
			path: "common/opinion_modifiers/" + tag + "_opinion_modifiers.txt",
			emit: emitscripts.EmitOpinionModifiersJSON,
		},
		{
			table: "dynamic_modifiers", requestKey: "modifiers",
			path: "common/dynamic_modifiers/" + tag + "_dynamic_modifiers.txt",
			emit: emitscripts.EmitDynamicModifiersJSON, loc: locout.GenerateDynamicModifiersLocalisation,
			locName: tag + "_dynamic_modifiers_",
		},
		{
			table: "autonomy_states", requestKey: "states",
			path: "common/autonomous_states/" + tag + "_autonomous_states.txt",
			emit: emitscripts.EmitAutonomyStatesJSON, loc: locout.GenerateAutonomyStatesLocalisation,
			locName: tag + "_autonomous_states_",
		},
	}
	for _, current := range families {
		rows, err := nativeRows(snapshot.Tables[current.table], includeImported)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			continue
		}
		raw := make([]json.RawMessage, 0, len(rows))
		locObjects := make([]*locout.Obj, 0, len(rows))
		for _, row := range rows {
			raw = append(raw, row.raw)
			locObjects = append(locObjects, row.loc)
		}
		body, _ := json.Marshal(map[string]any{current.requestKey: raw})
		script, err := current.emit(body)
		if err != nil {
			return nil, fmt.Errorf("emit %s: %w", current.table, err)
		}
		if script != "" {
			entries = append(entries, Entry{Path: modDir + "/" + current.path, Bytes: []byte(script), Origin: "generated", SourceEntity: current.table})
		}
		if current.loc != nil {
			for _, lang := range langs {
				text := current.loc(locObjects, lang.Code)
				if strings.Count(text, "\n") < 1 {
					continue
				}
				entries = append(entries, Entry{
					Path:  modDir + "/localisation/" + lang.Subdir + "/" + current.locName + lang.Suffix + ".yml",
					Bytes: []byte("\uFEFF" + text), Origin: "generated", SourceEntity: current.table + "_localisation",
				})
			}
		}
	}
	definitionGroups := map[string][]*emitmisc.NS{}
	var definitionNames []string
	for _, raw := range snapshot.Tables["modifier_definitions"] {
		row, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, err
		}
		if !includeImported && row.Str("source") == "imported" {
			continue
		}
		name := tag + "_modifier_definitions.txt"
		if row.Str("source") == "imported" {
			name = safeAIPlanningFilename(row.Str("source_file"), name)
		}
		if _, ok := definitionGroups[name]; !ok {
			definitionNames = append(definitionNames, name)
		}
		definitionGroups[name] = append(definitionGroups[name], row)
	}
	sort.Strings(definitionNames)
	for _, name := range definitionNames {
		blocks := make([]string, 0, len(definitionGroups[name]))
		for _, row := range definitionGroups[name] {
			block, err := renderModifierDefinition(row)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
		entries = append(entries, Entry{Path: modDir + "/common/modifier_definitions/" + name, Bytes: []byte(strings.Join(blocks, "\n\n")), Origin: "generated", SourceEntity: "modifier_definitions"})
	}
	return entries, nil
}

func renderModifierDefinition(row *emitmisc.NS) (string, error) {
	id := row.Str("modifier_id")
	if id == "" || !(id[0] == '_' || id[0] >= 'A' && id[0] <= 'Z' || id[0] >= 'a' && id[0] <= 'z') {
		return "", fmt.Errorf("invalid modifier definition id %q", id)
	}
	for _, char := range id[1:] {
		if !(char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9') {
			return "", fmt.Errorf("invalid modifier definition id %q", id)
		}
	}
	allowed := func(value string, values ...string) bool {
		for _, candidate := range values {
			if value == candidate {
				return true
			}
		}
		return false
	}
	color := row.StrOr("color_type", "bad")
	valueType := row.StrOr("value_type", "number")
	postfix := row.StrOr("postfix", "none")
	precision := row.Int("precision", 2)
	if !allowed(color, "good", "bad", "neutral") || !allowed(valueType, "number", "percentage", "percentage_in_hundred", "yes_no") || !allowed(postfix, "none", "days", "hours", "daily") || precision < 0 || precision > 9 {
		return "", fmt.Errorf("invalid modifier definition %q", id)
	}
	explicit := map[string]bool{}
	for _, value := range row.List("explicit_fields") {
		explicit[textValue(value)] = true
	}
	emitAll := row.Str("source") != "imported"
	lines := []string{id + " = {"}
	if emitAll || explicit["color_type"] {
		lines = append(lines, "\tcolor_type = "+color)
	}
	if emitAll || explicit["value_type"] {
		lines = append(lines, "\tvalue_type = "+valueType)
	}
	if emitAll || explicit["precision"] {
		lines = append(lines, fmt.Sprintf("\tprecision = %d", precision))
	}
	if emitAll || explicit["postfix"] {
		lines = append(lines, "\tpostfix = "+postfix)
	}
	if emitAll || explicit["category"] {
		for _, value := range row.List("categories") {
			category := textValue(value)
			if category != "" {
				lines = append(lines, "\tcategory = "+category)
			}
		}
	}
	return strings.Join(append(lines, "}"), "\n"), nil
}

func nativeAdvancedScripts(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	tag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	if tag == "" {
		tag = "UNK"
	}
	langs := emitmisc.BuildExportLangTriples(stringList(project, "export_languages"))
	var entries []Entry
	entityMaps := map[string]map[int64]string{}
	for _, kind := range []string{"mio_icon", "intel_agency_logo", "custom_trait_icon"} {
		m, err := entityOBSPaths(snapshot, kind)
		if err != nil {
			return nil, err
		}
		entityMaps[kind] = m
	}
	projectIcons, err := projectOBSIcons(snapshot)
	if err != nil {
		return nil, err
	}
	resolvedMIO, resolvedMIOTraits, resolvedAgency := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var advancedSprites []emitmisc.SpritePair
	addAsset := func(key, sprite, assetPath, label string) error {
		if key == "" || sprite == "" {
			return nil
		}
		dds, ok, err := ResolveImageDDSTolerant(resolve, key, label)
		if err != nil {
			return err
		}
		if !ok {
			sink.skipped(label)
			return nil
		}
		entries = append(entries, Entry{Path: modDir + "/" + assetPath, Bytes: dds, Origin: "generated", SourceEntity: "advanced_images"})
		advancedSprites = append(advancedSprites, emitmisc.SpritePair{Name: sprite, Path: assetPath})
		return nil
	}
	preMIOs, err := decodeEntities(snapshot.Tables["military_industrial_organizations"], nil)
	if err != nil {
		return nil, err
	}
	for _, mio := range preMIOs {
		id := mio.Str("mio_id")
		sprite := "GFX_idea_mio_" + emitmisc.Slugify(id)
		if key := entityMaps["mio_icon"][mio.Int("id", 0)]; key != "" {
			if err := addAsset(key, sprite, "gfx/interface/ideas/mio_"+emitmisc.Slugify(id)+".dds", "MIO "+id); err != nil {
				return nil, err
			}
			resolvedMIO[sprite] = true
		}
		for _, rawTrait := range mio.List("traits") {
			var trait map[string]any
			switch v := rawTrait.(type) {
			case map[string]any:
				trait = v
			case *emitmisc.NS:
				continue
			}
			token, _ := trait["token"].(string)
			icon, _ := trait["icon"].(string)
			if token == "" || icon == "" {
				continue
			}
			if key := projectIcons[icon]; key != "" {
				if err := addAsset(key, icon, "gfx/interface/ideas/mio_trait_"+emitmisc.Slugify(id)+"_"+emitmisc.Slugify(token)+".dds", "MIO trait "+token); err != nil {
					return nil, err
				}
				resolvedMIOTraits[icon] = true
			}
		}
	}
	preAgencies, err := decodeEntities(snapshot.Tables["intelligence_agencies"], nil)
	if err != nil {
		return nil, err
	}
	for _, agency := range preAgencies {
		id := agency.Str("agency_id")
		sprite := "GFX_intelligence_agency_logo_" + emitmisc.Slugify(id)
		if key := entityMaps["intel_agency_logo"][agency.Int("id", 0)]; key != "" {

			label := "agency " + id
			raw, ok, err := ResolveBlobTolerant(resolve, key, 32<<20, label)
			if err != nil {
				return nil, err
			}
			var dds []byte
			if ok {
				if dds, err = agencyLogoStripDDS(raw, label); err != nil {
					ok = false
				}
			}
			if !ok {
				sink.skipped(label)
				continue
			}
			assetPath := "gfx/interface/intelligence_agencies/intel_agency_" + emitmisc.Slugify(id) + ".dds"
			entries = append(entries, Entry{Path: modDir + "/" + assetPath, Bytes: dds, Origin: "generated", SourceEntity: "advanced_images"})
			advancedSprites = append(advancedSprites, emitmisc.SpritePair{Name: sprite, Path: assetPath, Frames: agencyLogoFrames})
			resolvedAgency[sprite] = true
		}
	}
	appendScript := func(path, source, script string) {
		if strings.TrimSpace(script) != "" {
			entries = append(entries, Entry{Path: modDir + "/" + path, Bytes: []byte(script), Origin: "generated", SourceEntity: source})
		}
	}
	emitRows := func(table, key string, emit func([]byte) (string, error), extra map[string]any) (string, []nativeRow, error) {
		rows, err := nativeRows(snapshot.Tables[table], includeImported)
		if err != nil || len(rows) == 0 {
			return "", rows, err
		}
		raw := make([]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			raw = append(raw, row.raw)
		}
		body := map[string]any{key: raw}
		for k, v := range extra {
			body[k] = v
		}
		encoded, _ := json.Marshal(body)
		script, err := emit(encoded)
		return script, rows, err
	}

	traits, err := nativeRows(snapshot.Tables["custom_traits"], true)
	if err != nil {
		return nil, err
	}
	type traitFamily struct {
		kind, dir, file string
		emit            func([]byte) (string, error)
	}
	traitFamilies := []traitFamily{
		{"country_leader", "common/country_leader", "custom_traits", emitscripts.EmitCustomCountryLeaderTraitsJSON},
		{"unit_leader", "common/unit_leader", "custom_unit_traits", emitscripts.EmitCustomUnitLeaderTraitsJSON},
		{"scientist", "common/scientist_traits", "custom_scientist_traits", emitscripts.EmitCustomScientistTraitsJSON},
	}
	for _, family := range traitFamilies {
		buckets := map[string][]nativeRow{}
		var order []string
		for _, row := range traits {
			if row.misc.Str("trait_type") != family.kind {
				continue
			}
			ctag := strings.ToUpper(strings.TrimSpace(row.misc.Str("country_tag")))
			if ctag == "" {
				ctag = tag
			}
			if _, ok := buckets[ctag]; !ok {
				order = append(order, ctag)
			}
			buckets[ctag] = append(buckets[ctag], row)
		}
		for _, ctag := range order {
			raw := make([]json.RawMessage, 0, len(buckets[ctag]))
			locs := make([]*locout.Obj, 0, len(buckets[ctag]))
			for _, row := range buckets[ctag] {
				raw = append(raw, row.raw)
				locs = append(locs, row.loc)
			}
			body, _ := json.Marshal(map[string]any{"traits": raw})
			script, err := family.emit(body)
			if err != nil {
				return nil, fmt.Errorf("emit %s traits: %w", family.kind, err)
			}
			appendScript(family.dir+"/"+ctag+"_"+family.file+".txt", "custom_traits", script)
			for _, lang := range langs {
				text := locout.GenerateCustomTraitsLocalisation(locs, lang.Code)
				if strings.Count(text, "\n") >= 1 {
					appendScript("localisation/"+lang.Subdir+"/"+ctag+"_"+family.file+"_"+lang.Suffix+".yml", "custom_trait_localisation", "\uFEFF"+text)
				}
			}
		}
	}

	mios, err := nativeRows(snapshot.Tables["military_industrial_organizations"], includeImported)
	if err != nil {
		return nil, err
	}
	ownMIOFile := "common/military_industrial_organization/organizations/" + tag + "_organizations.txt"
	mioFiles := map[string][]nativeRow{}
	var mioFileOrder []string
	for _, row := range mios {
		file := MIOSourceFile(row.misc)
		if file == "" {
			file = ownMIOFile
		}
		if _, seen := mioFiles[file]; !seen {
			mioFileOrder = append(mioFileOrder, file)
		}
		mioFiles[file] = append(mioFiles[file], row)
	}
	for _, file := range mioFileOrder {
		raw := make([]json.RawMessage, 0, len(mioFiles[file]))
		for _, row := range mioFiles[file] {
			raw = append(raw, row.raw)
		}
		encoded, _ := json.Marshal(map[string]any{
			"mios": raw, "resolved_managed_icons": resolvedMIO, "resolved_managed_trait_icons": resolvedMIOTraits,
		})
		mioScript, err := emitscripts.EmitMIOJSON(encoded)
		if err != nil {
			return nil, err
		}
		appendScript(file, "military_industrial_organizations", mioScript)
	}

	mioLocRows := make([]nativeRow, 0, len(mios))
	for _, row := range mios {
		if MIOSourceFile(row.misc) != "" {
			row.loc = withoutMIOTraits(row.loc)
		}
		mioLocRows = append(mioLocRows, row)
	}
	appendAdvancedLoc := func(rows []nativeRow, name, source string, generate func([]*locout.Obj, string) string) {
		objects := make([]*locout.Obj, 0, len(rows))
		for _, row := range rows {
			objects = append(objects, row.loc)
		}
		for _, lang := range langs {
			text := generate(objects, lang.Code)
			if strings.Count(text, "\n") >= 1 {
				appendScript("localisation/"+lang.Subdir+"/"+tag+"_"+name+"_"+lang.Suffix+".yml", source, "\uFEFF"+text)
			}
		}
	}
	appendAdvancedLoc(mioLocRows, "mio", "mio_localisation", locout.GenerateMIOLocalisation)

	doctrines, err := nativeRows(snapshot.Tables["doctrines"], includeImported)
	if err != nil {
		return nil, err
	}
	doctrineGroups := map[string][]json.RawMessage{}
	var doctrineOrder []string
	for _, row := range doctrines {
		rel := strings.Trim(strings.ReplaceAll(row.misc.Str("source_file"), "\\", "/"), "/")
		rel = strings.TrimPrefix(rel, "common/doctrines/")
		if rel == "" {
			folder := strings.TrimSpace(row.misc.Str("folder_key"))
			if folder == "" {
				folder = "land"
			}
			switch row.misc.Str("entity_type") {
			case "folder":
				rel = "folders/zz_" + modDir + "_doctrine_folders.txt"
			case "grand_doctrine":
				rel = "grand_doctrines/zz_" + modDir + "_" + folder + "_grand_doctrines.txt"
			case "track":
				rel = "tracks/zz_" + modDir + "_" + folder + "_doctrine_tracks.txt"
			case "subdoctrine":
				track := strings.TrimSpace(row.misc.Str("track_key"))
				if track == "" {
					track = "infantry"
				}
				rel = "subdoctrines/" + folder + "/zz_" + modDir + "_" + track + "_subdoctrines.txt"
			default:
				rel = "misc/zz_" + modDir + "_doctrines.txt"
			}
		}
		if _, err := SafePath(rel); err != nil {
			return nil, fmt.Errorf("doctrine source_file: %w", err)
		}
		if _, ok := doctrineGroups[rel]; !ok {
			doctrineOrder = append(doctrineOrder, rel)
		}
		doctrineGroups[rel] = append(doctrineGroups[rel], row.raw)
	}
	for _, rel := range doctrineOrder {
		body, _ := json.Marshal(map[string]any{"doctrines": doctrineGroups[rel]})
		script, err := emitscripts.EmitDoctrineJSON(body)
		if err != nil {
			return nil, err
		}
		appendScript("common/doctrines/"+rel, "doctrines", script)
	}
	if len(doctrines) > 0 {
		doctrineLocs := make([]*locout.Obj, 0, len(doctrines))
		for _, row := range doctrines {
			doctrineLocs = append(doctrineLocs, row.loc)
		}
		for _, lang := range langs {
			text := locout.GenerateDoctrineLocalisation(doctrineLocs, lang.Code)
			if strings.Count(text, "\n") >= 1 {
				appendScript("localisation/"+lang.Subdir+"/doctrines_"+lang.Suffix+".yml", "doctrine_localisation", "\uFEFF"+text)
			}
		}
	}

	agencyScript, agencies, err := emitRows("intelligence_agencies", "agencies", emitscripts.EmitIntelAgenciesJSON, map[string]any{"resolved_managed_logos": resolvedAgency})
	if err != nil {
		return nil, err
	}
	appendScript("common/intelligence_agencies/"+tag+"_intelligence_agencies.txt", "intelligence_agencies", agencyScript)
	upgradeScript, _, err := emitRows("intelligence_agencies", "agencies", emitscripts.EmitIntelAgencyUpgradesJSON, nil)
	if err != nil {
		return nil, err
	}
	appendScript("common/intelligence_agency_upgrades/"+tag+"_agency_upgrades.txt", "intelligence_agencies", upgradeScript)
	appendAdvancedLoc(agencies, "intelligence_agencies", "intelligence_agency_localisation", locout.GenerateIntelAgencyLocalisation)

	specScript, _, err := emitRows("special_project_specializations", "specs", emitscripts.EmitSpecializationsJSON, nil)
	if err != nil {
		return nil, err
	}
	appendScript("common/special_projects/specialization/"+tag+"_specializations.txt", "special_project_specializations", specScript)
	projectScript, projects, err := emitRows("special_projects", "projects", emitscripts.EmitSpecialProjectsJSON, nil)
	if err != nil {
		return nil, err
	}
	appendScript("common/special_projects/projects/"+tag+"_projects.txt", "special_projects", projectScript)
	if len(projects) > 0 {
		tagScript, _, err := emitRows("special_projects", "projects", emitscripts.EmitProjectTagsJSON, nil)
		if err != nil {
			return nil, err
		}
		appendScript("common/special_projects/project_tags/"+tag+"_project_tags.txt", "special_projects", tagScript)
		appendScript("common/special_projects/prototype_rewards/"+tag+"_prototype_rewards.txt", "special_projects", "# Prototype rewards: unique rewards are authored inline on each project.\n")
	}

	for _, row := range doctrines {
		icon := strings.TrimSpace(row.misc.Str("icon"))
		if key := projectIcons[icon]; key != "" {
			base := strings.TrimPrefix(icon, "GFX_")
			if err := addAsset(key, icon, "gfx/interface/doctrines/"+base+".dds", "doctrine "+row.misc.Str("doctrine_id")); err != nil {
				return nil, err
			}
		}
	}
	for _, row := range traits {
		kind, id := row.misc.Str("trait_type"), row.misc.Str("trait_id")
		sprite := ""
		if kind == "scientist" {
			sprite = "GFX_" + id
		} else if kind == "unit_leader" {
			sprite = "GFX_trait_" + id
		}
		if key := entityMaps["custom_trait_icon"][row.misc.Int("id", 0)]; key != "" && sprite != "" {
			if err := addAsset(key, sprite, "gfx/interface/traits/customtrait_"+kind+"_"+emitmisc.Slugify(id)+".dds", "trait "+id); err != nil {
				return nil, err
			}
		}
	}
	if len(advancedSprites) > 0 {
		entries = append(entries, Entry{Path: modDir + "/interface/" + tag + "_advanced_assets.gfx", Bytes: []byte(emitmisc.EmitSpriteBlock(dedupeSpritePairs(advancedSprites))), Origin: "generated", SourceEntity: "advanced_images"})
	}
	return entries, nil
}

func dedupeSpritePairs(in []emitmisc.SpritePair) []emitmisc.SpritePair {
	seen := map[string]bool{}
	out := make([]emitmisc.SpritePair, 0, len(in))
	for _, item := range in {
		if !seen[item.Name] {
			seen[item.Name] = true
			out = append(out, item)
		}
	}
	return out
}

func decodeEntities(rows []json.RawMessage, keep func(*emitmisc.NS) bool) ([]*emitmisc.NS, error) {
	entities := make([]*emitmisc.NS, 0, len(rows))
	for _, raw := range rows {
		entity, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, err
		}
		if keep == nil || keep(entity) {
			entities = append(entities, entity)
		}
	}
	return entities, nil
}

func nativeInlineRawFiles(snapshot Snapshot, includeImported bool, resolve BlobResolver) ([]Entry, []string, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, nil, err
	}
	tag := strings.ToUpper(strings.TrimSpace(project.Str("country_tag")))
	fallback := "mod_" + tag
	if tag == "" {
		fallback = "mod_" + strconv.FormatInt(snapshot.ProjectID, 10)
	}
	modDir := SanitizeModDirName(project.Str("name"), fallback)
	var entries []Entry
	var replacePaths []string
	var resolvedBytes int64
	const maxRawObjectBytes = int64(256 << 20)
	const maxRawTotalBytes = int64(2 << 30)
	for _, raw := range snapshot.Tables["project_raw_files"] {
		row, err := emitmisc.DecodeEntityJSON(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("decode raw file: %w", err)
		}
		if !includeImported && row.Str("source") == "imported" {
			continue
		}

		if IsTombstoned(row) {
			continue
		}
		destination := strings.Trim(strings.ReplaceAll(row.Str("dest_path"), "\\", "/"), "/")
		if _, err := SafePath(destination); err != nil {
			return nil, nil, fmt.Errorf("raw file %d: %w", row.Int("id", 0), err)
		}
		if strings.EqualFold(path.Base(destination), ExportFindingsBasename) ||
			strings.EqualFold(path.Base(destination), ReferenceIntegrityLogBasename) {

			continue
		}

		kind := row.StrOr("raw_kind", "verbatim_override")
		emptyShadow := kind == "empty_shadow"
		if kind != "verbatim_override" && !emptyShadow {
			return nil, nil, fmt.Errorf("raw file %d uses unsupported construction policy %q", row.Int("id", 0), kind)
		}
		if emptyShadow {

			entries = append(entries, Entry{
				Path: modDir + "/" + destination, Bytes: []byte{}, Origin: "raw_empty_shadow",
				SourceEntity: fmt.Sprintf("project_raw_files:%d", row.Int("id", 0)),
			})
			continue
		}
		var content []byte
		if row.Bool("is_binary", false) {
			if key := strings.TrimSpace(row.Str("content_obs_key")); key != "" {
				if resolve == nil {
					return nil, nil, fmt.Errorf("raw file %d requires private object %q", row.Int("id", 0), key)
				}
				content, err = resolve(key, maxRawObjectBytes)
				if err != nil {
					return nil, nil, fmt.Errorf("raw file %d private object: %w", row.Int("id", 0), err)
				}
				resolvedBytes += int64(len(content))
				if resolvedBytes > maxRawTotalBytes {
					return nil, nil, fmt.Errorf("raw private objects exceed %d byte budget", maxRawTotalBytes)
				}
			} else {
				content, err = decodePostgresByteaJSON(row.Str("content_binary"))
				if err != nil {
					return nil, nil, fmt.Errorf("raw file %d binary: %w", row.Int("id", 0), err)
				}
			}
		} else {
			content = []byte(row.Str("content"))
		}
		entries = append(entries, Entry{
			Path: modDir + "/" + destination, Bytes: content, Origin: "raw_override",
			SourceEntity: fmt.Sprintf("project_raw_files:%d", row.Int("id", 0)),
		})
		if row.Bool("generate_replace_path", false) {
			if slash := strings.LastIndex(destination, "/"); slash > 0 {
				replacePaths = append(replacePaths, destination[:slash])
			}
		}
	}
	return entries, replacePaths, nil
}

func decodePostgresByteaJSON(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	if strings.HasPrefix(value, `\x`) {
		return hex.DecodeString(value[2:])
	}
	return nil, fmt.Errorf("unexpected PostgreSQL bytea representation")
}

func canonicalStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.Trim(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"), "/")
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func rawEntriesOwnDirectory(entries []Entry, directory string) bool {
	directory = strings.Trim(strings.ReplaceAll(directory, "\\", "/"), "/")
	needle := "/" + directory + "/"
	for _, entry := range entries {
		value := "/" + strings.Trim(strings.ReplaceAll(entry.Path, "\\", "/"), "/") + "/"
		if strings.Contains(value, needle) && strings.HasPrefix(entry.Origin, "raw_") {
			return true
		}
	}
	return false
}

func overlayEntries(generated, overrides []Entry) ([]Entry, error) {
	result := append([]Entry(nil), generated...)
	index := make(map[string]int, len(result))
	spelling := make(map[string]string, len(result))
	for i, entry := range result {
		folded := strings.ToLower(entry.Path)
		index[folded] = i
		spelling[folded] = entry.Path
	}
	for _, entry := range overrides {
		folded := strings.ToLower(entry.Path)
		if previous, exists := spelling[folded]; exists && previous != entry.Path {
			return nil, fmt.Errorf("case-colliding artifact paths: %q and %q", previous, entry.Path)
		}
		if i, exists := index[folded]; exists {
			result[i] = entry
		} else {
			index[folded] = len(result)
			spelling[folded] = entry.Path
			result = append(result, entry)
		}
	}
	return result, nil
}

func nativeScriptedFiles(snapshot Snapshot, modDir string, includeImported bool) ([]Entry, error) {
	selectRows := func(table string) ([]json.RawMessage, error) {
		selected := make([]json.RawMessage, 0, len(snapshot.Tables[table]))
		for _, raw := range snapshot.Tables[table] {
			row, err := emitmisc.DecodeEntityJSON(raw)
			if err != nil {
				return nil, err
			}
			if includeImported || row.Str("source") == "" || row.Str("source") == "created" {
				selected = append(selected, raw)
			}
		}
		return selected, nil
	}
	var entries []Entry
	snippets, err := selectRows("scripted_snippets")
	if err != nil {
		return nil, err
	}
	if len(snippets) > 0 {
		body, _ := json.Marshal(map[string]any{"snippets": snippets, "mod_dir_name": modDir})
		files, err := emitscripts.EmitScriptedSnippetsJSON(body)
		if err != nil {
			return nil, err
		}
		entries = appendFilePairs(entries, modDir, files, "scripted_snippets")
	}
	locs, err := selectRows("scripted_localisations")
	if err != nil {
		return nil, err
	}
	if len(locs) > 0 {
		body, _ := json.Marshal(map[string]any{"locs": locs, "mod_dir_name": modDir})
		files, err := emitscripts.EmitScriptedLocalisationsJSON(body)
		if err != nil {
			return nil, err
		}
		entries = appendFilePairs(entries, modDir, files, "scripted_localisations")
	}
	return entries, nil
}

func appendFilePairs(entries []Entry, modDir string, files [][]string, source string) []Entry {
	for _, pair := range files {
		if len(pair) != 2 {
			continue
		}
		entries = append(entries, Entry{
			Path: modDir + "/" + strings.TrimLeft(pair[0], "/"), Bytes: []byte(pair[1]),
			Origin: "generated", SourceEntity: source,
		})
	}
	return entries
}

func countryFlagIdeologies(projectRaw json.RawMessage, assetMap map[string]string, ctag string) []string {
	ordered := []string{"democratic", "fascism", "neutrality", "communism"}
	seen := map[string]bool{}
	for _, ideology := range ordered {
		seen[ideology] = true
	}
	add := func(raw string) {
		ideology := strings.ToLower(strings.TrimSpace(raw))
		if ideology == "" || seen[ideology] {
			return
		}
		seen[ideology] = true
		ordered = append(ordered, ideology)
	}
	var projectDoc struct {
		IdeologyRegistry *struct {
			Groups          []string `json:"groups"`
			RootDefinitions []struct {
				ID string `json:"id"`
			} `json:"root_definitions"`
		} `json:"ideology_registry"`
	}
	if json.Unmarshal(projectRaw, &projectDoc) == nil && projectDoc.IdeologyRegistry != nil {
		for _, group := range projectDoc.IdeologyRegistry.Groups {
			add(group)
		}
		for _, root := range projectDoc.IdeologyRegistry.RootDefinitions {
			add(root.ID)
		}
	}
	prefix := "country_flag_ideology\x00" + ctag + "_"
	var assetGroups []string
	for key := range assetMap {
		if strings.HasPrefix(key, prefix) {
			assetGroups = append(assetGroups, strings.TrimPrefix(key, prefix))
		}
	}
	sort.Strings(assetGroups)
	for _, ideology := range assetGroups {
		add(ideology)
	}
	return ordered
}

var ideologyIconRootIDRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func configuredIdeologyIconGroups(projectRaw json.RawMessage) []string {
	var projectDoc struct {
		IdeologyIconPaths map[string]string `json:"ideology_icon_paths"`
		IdeologyRegistry  *struct {
			Groups          []string `json:"groups"`
			RootDefinitions []struct {
				ID string `json:"id"`
			} `json:"root_definitions"`
		} `json:"ideology_registry"`
	}
	if json.Unmarshal(projectRaw, &projectDoc) != nil {
		return nil
	}
	ordered := []string{"democratic", "communism", "fascism", "neutrality"}
	seen := map[string]bool{}
	for _, group := range ordered {
		seen[group] = true
	}
	add := func(raw string) {
		group := strings.TrimSpace(raw)
		if !ideologyIconRootIDRe.MatchString(group) || seen[group] {
			return
		}
		seen[group] = true
		ordered = append(ordered, group)
	}
	if projectDoc.IdeologyRegistry != nil {
		for _, group := range projectDoc.IdeologyRegistry.Groups {
			add(group)
		}
		for _, root := range projectDoc.IdeologyRegistry.RootDefinitions {
			add(root.ID)
		}
	}
	configured := make([]string, 0, len(ordered))
	for _, group := range ordered {
		if strings.TrimSpace(projectDoc.IdeologyIconPaths[group]) != "" {
			configured = append(configured, group)
		}
	}
	return configured
}

func projectRootsBesideInheritedSource(project *worldout.OMap, plan *worldout.IdeologyExportPlan) bool {
	if plan == nil || !plan.InheritedSource || plan.Facts.RawOverride || !plan.Facts.Standalone {
		return false
	}
	registry, _ := project.Get("ideology_registry")
	text, _ := worldout.CustomRootIdeologiesFile(registry, plan.SourceRoots)
	return text != ""
}

func ideologyIconGFX(groups []string) []byte {
	if len(groups) == 0 {
		return nil
	}
	var out strings.Builder
	out.WriteString("spriteTypes = {\n")
	for _, group := range groups {
		fmt.Fprintf(&out, "\tspriteType = { name = GFX_ideology_%s_group texturefile = gfx/interface/ideologies/%s_group.dds }\n", group, group)
	}
	out.WriteString("}\n")
	return []byte(out.String())
}

func nativeWorld(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink, ideologyPlan *worldout.IdeologyExportPlan, locRows []preservedLocRow) ([]Entry, error) {
	project := worldout.NewOMap()
	if err := project.UnmarshalJSON(snapshot.Project); err != nil {
		return nil, fmt.Errorf("decode world project: %w", err)
	}
	decodeRows := func(table string, filterSource bool) ([]*worldout.OMap, error) {
		out := make([]*worldout.OMap, 0, len(snapshot.Tables[table]))
		for _, raw := range snapshot.Tables[table] {
			row := worldout.NewOMap()
			if err := row.UnmarshalJSON(raw); err != nil {
				return nil, fmt.Errorf("decode world %s: %w", table, err)
			}
			if filterSource && !includeImported && worldString(row, "source") == "imported" {
				continue
			}
			out = append(out, row)
		}
		return out, nil
	}
	countries, err := decodeRows("countries", false)
	if err != nil {
		return nil, err
	}
	historyFiles, err := decodeRows("history_files", true)
	if err != nil {
		return nil, err
	}
	states, err := decodeRows("states", true)
	if err != nil {
		return nil, err
	}
	bookmarks, err := decodeRows("bookmarks", true)
	if err != nil {
		return nil, err
	}
	cosmeticDefinitions, err := nativeCosmeticDefinitions(snapshot, includeImported)
	if err != nil {
		return nil, err
	}
	cosmeticOwners, err := nativeCosmeticOwners(snapshot)
	if err != nil {
		return nil, err
	}
	if len(countries)+len(historyFiles)+len(states)+len(bookmarks)+len(cosmeticDefinitions) == 0 &&
		(ideologyPlan == nil || ideologyPlan.FullText == "") && !projectRootsBesideInheritedSource(project, ideologyPlan) {
		return nil, nil
	}
	tag := strings.ToUpper(strings.TrimSpace(worldString(project, "country_tag")))
	if tag == "" {
		tag = "UNK"
	}
	contentByTag := worldout.NewOMap()
	for _, rows := range [][]*worldout.OMap{countries, historyFiles} {
		for _, row := range rows {
			ctag := strings.ToUpper(strings.TrimSpace(firstWorldString(row, "tag", "country_tag")))
			if ctag != "" {
				contentByTag.Set(ctag, true)
			}
		}
	}
	existence := map[string]map[string]bool{"idea": {}, "focus": {}}
	for _, tableField := range [][2]string{{"ideas", "idea_id"}, {"focus_nodes", "focus_id"}} {
		for _, raw := range snapshot.Tables[tableField[0]] {
			row := worldout.NewOMap()
			if row.UnmarshalJSON(raw) == nil {
				if id := worldString(row, tableField[1]); id != "" {
					existence[map[string]string{"ideas": "idea", "focus_nodes": "focus"}[tableField[0]]][id] = true
				}
			}
		}
	}

	for id := range loadCrossRefVanillaIDs()["idea"] {
		existence["idea"][id] = true
	}

	for _, raw := range snapshot.Tables["characters"] {
		row := worldout.NewOMap()
		if row.UnmarshalJSON(raw) == nil {
			if token := worldString(row, "advisor_idea_token"); token != "" {
				existence["idea"][token] = true
			}
		}
	}
	for id := range loadCrossRefVanillaIDs()["character"] {
		existence["idea"][id] = true
	}
	dependencies, _ := project.Get("dependencies")
	depList, _ := dependencies.([]worldout.Value)

	recruitByCtag, clByCtag, err := BuildRecruitAndLeaderMaps(snapshot, includeImported)
	if err != nil {
		return nil, err
	}

	oobFiles, err := func() ([]worldout.OOBFile, error) {
		createdOOB := func(entity *emitmisc.NS) bool {
			return includeImported || entity.Str("source") == "" || entity.Str("source") == "created"
		}
		oobRawFiles, err := decodeEntities(snapshot.Tables["project_raw_files"], nil)
		if err != nil {
			return nil, err
		}
		overridePaths := emitmisc.AuthorRawOverridePaths(oobRawFiles)
		var order []string
		byPath := map[string]*worldout.OOBFile{}

		fleetPath := map[int64]string{}

		collect := func(table, kind string) error {
			rows, err := decodeEntities(snapshot.Tables[table], createdOOB)
			if err != nil {
				return err
			}
			for _, row := range rows {
				path := emitmisc.OOBDestination(row)
				file := byPath[path]
				if file == nil {
					file = &worldout.OOBFile{Path: path, RawOverridden: overridePaths[path]}
					byPath[path] = file
					order = append(order, path)
				}
				if ctag := strings.ToUpper(strings.TrimSpace(row.Str("country_tag"))); ctag != "" {
					file.Tags = append(file.Tags, ctag)
				}
				switch kind {
				case "deployment":
					if province := row.Int("location", 0); province != 0 {
						file.Locations = append(file.Locations, province)
					}
				case "fleet":
					if province := row.Int("naval_base", 0); province != 0 {
						file.NavalProvinces = append(file.NavalProvinces, province)
					}
					if id := row.Int("id", 0); id != 0 {
						fleetPath[id] = path
					}
				case "air":
					if state := row.Int("location", 0); state != 0 {
						file.AirStates = append(file.AirStates, state)
					}
				}
			}
			return nil
		}
		for _, table := range []struct{ name, kind string }{
			{"division_templates", ""}, {"division_deployments", "deployment"},
			{"oob_air_wing_groups", "air"}, {"oob_fleets", "fleet"},
		} {
			if err := collect(table.name, table.kind); err != nil {
				return nil, err
			}
		}

		taskForcesOOB, err := decodeEntities(snapshot.Tables["oob_task_forces"], createdOOB)
		if err != nil {
			return nil, err
		}
		for _, row := range taskForcesOOB {
			file := byPath[fleetPath[row.Int("fleet_id", 0)]]
			if file == nil {
				continue
			}
			if province := row.Int("location", 0); province != 0 {
				file.NavalProvinces = append(file.NavalProvinces, province)
			}
		}

		for _, row := range oobRawFiles {
			if !includeImported && row.Str("source") == "imported" {
				continue
			}
			if IsTombstoned(row) {
				continue
			}
			if kind := row.StrOr("raw_kind", "verbatim_override"); kind != "verbatim_override" && kind != "empty_shadow" {
				continue
			}
			dest := strings.Trim(strings.ReplaceAll(row.Str("dest_path"), "\\", "/"), "/")
			if !strings.HasPrefix(dest, "history/units/") || !strings.HasSuffix(dest, ".txt") {
				continue
			}
			if _, err := SafePath(dest); err != nil {
				continue
			}
			if byPath[dest] != nil {
				continue
			}
			byPath[dest] = &worldout.OOBFile{Path: dest, RawOverridden: true}
			order = append(order, dest)
		}
		out := make([]worldout.OOBFile, 0, len(order))
		for _, path := range order {
			out = append(out, *byPath[path])
		}
		return out, nil
	}()
	if err != nil {
		return nil, err
	}
	preservedLocKeys := eligiblePreservedLocKeys(locRows, includeImported)
	scenario := &worldout.Scenario{
		Project: project, Tag: tag, ModName: worldString(project, "name"), ModDirName: modDir,
		IsSubmod: len(depList) > 0, IncludeImported: includeImported, ExistenceSets: existence,
		ContentByTag: contentByTag, RecruitByCtag: recruitByCtag, ClByCtag: clByCtag,
		Countries: countries, HistoryFiles: historyFiles, States: states, Bookmarks: bookmarks,
		OOBFiles: oobFiles, IdeologyPlan: ideologyPlan,
		PreservedLocKeys:    preservedLocKeys,
		CosmeticDefinitions: cosmeticDefinitions,
		CosmeticOwners:      cosmeticOwners,
	}
	bookmarkOBS, err := entityOBSPaths(snapshot, "bookmark_picture")
	if err != nil {
		return nil, err
	}
	var imageEntries []Entry
	for _, bookmark := range bookmarks {
		id := worldString(bookmark, "bookmark_id")
		entityID := worldInt(bookmark, "id")
		key := bookmarkOBS[entityID]
		if id == "" || key == "" {
			continue
		}
		slug := emitmisc.Slugify(id)
		sprite := "GFX_hagane_select_date_" + slug
		assetPath := "gfx/loadingscreens/select_date_" + slug + ".dds"
		dds, ok, err := ResolveImageDDSTolerant(resolve, key, "bookmark "+id)
		if err != nil {
			return nil, err
		}
		if !ok {
			sink.skipped("bookmark " + id)
			continue
		}
		scenario.BookmarkGfx = append(scenario.BookmarkGfx, [2]string{sprite, assetPath})
		imageEntries = append(imageEntries, Entry{Path: modDir + "/" + assetPath, Bytes: dds, Origin: "generated", SourceEntity: "bookmark_pictures"})
	}
	assetMap, err := projectOBSAssets(snapshot)
	if err != nil {
		return nil, err
	}
	flagOBS, err := entityOBSPaths(snapshot, "country_flag")
	if err != nil {
		return nil, err
	}
	flagSizes := []struct {
		sub  string
		w, h int
	}{{"", 82, 52}, {"medium/", 41, 26}, {"small/", 10, 7}}

	var flagsMissing []string
	type flagTarget struct {
		tag      string
		entityID int64
		country  *worldout.OMap
	}
	flagTargets := map[string]flagTarget{}
	for _, country := range countries {
		ctag := strings.ToUpper(strings.TrimSpace(worldString(country, "tag")))
		if ctag != "" {
			flagTargets[ctag] = flagTarget{tag: ctag, entityID: worldInt(country, "id"), country: country}
		}
	}
	if ctag := strings.ToUpper(strings.TrimSpace(worldString(project, "country_tag"))); ctag != "" {
		if _, exists := flagTargets[ctag]; !exists {
			flagTargets[ctag] = flagTarget{tag: ctag}
		}
	}
	flagTags := make([]string, 0, len(flagTargets))
	for ctag := range flagTargets {
		flagTags = append(flagTags, ctag)
	}
	sort.Strings(flagTags)
	for _, ctag := range flagTags {
		target := flagTargets[ctag]
		var base image.Image
		if key := flagOBS[target.entityID]; key != "" {
			raw, err := resolvePrivateImage(resolve, key, "flag "+ctag)
			if err != nil {
				return nil, err
			}
			base, err = importassets.DecodeImage(raw, ctag+".png")
			if err != nil {
				return nil, fmt.Errorf("decode flag %s: %w", ctag, err)
			}
		}
		variants := map[string]image.Image{}
		ideologies := countryFlagIdeologies(snapshot.Project, assetMap, ctag)
		for _, ideology := range ideologies {
			key := assetMap["country_flag_ideology\x00"+ctag+"_"+ideology]
			if key == "" {
				continue
			}
			raw, err := resolvePrivateImage(resolve, key, "flag "+ctag+" "+ideology)
			if err != nil {
				return nil, err
			}
			img, err := importassets.DecodeImage(raw, ctag+"_"+ideology+".png")
			if err != nil {
				return nil, err
			}
			variants[ideology] = img
		}
		if base == nil && len(variants) == 0 {
			if countryFlagHasSomethingOnRecord(target.country) {
				flagsMissing = append(flagsMissing, ctag)
			}
			continue
		}
		for _, size := range flagSizes {
			var baseTGA []byte
			if base != nil {
				baseTGA = encodeFlagTGA(base, size.w, size.h)
			}
			for _, ideology := range ideologies {
				data := baseTGA
				if variants[ideology] != nil {
					data = encodeFlagTGA(variants[ideology], size.w, size.h)
				}
				if len(data) > 0 {
					imageEntries = append(imageEntries, Entry{Path: modDir + "/gfx/flags/" + size.sub + ctag + "_" + ideology + ".tga", Bytes: data, Origin: "generated", SourceEntity: "country_flags"})
				}
			}
			if len(baseTGA) > 0 {
				imageEntries = append(imageEntries, Entry{Path: modDir + "/gfx/flags/" + size.sub + ctag + ".tga", Bytes: baseTGA, Origin: "generated", SourceEntity: "country_flags"})
			}
		}
	}
	if note := flagsMissingWarning(flagsMissing, sink.lang()); note != "" {
		sink.note(note)
	}
	emittedIdeologyIcons := make([]string, 0)
	for _, group := range configuredIdeologyIconGroups(snapshot.Project) {
		key := assetMap["ideology_icon\x00"+group]
		if key == "" {
			continue
		}
		dds, ok, err := ResolveImageDDSTolerant(resolve, key, "ideology "+group)
		if err != nil {
			return nil, err
		}
		if !ok {
			sink.skipped("ideology " + group)
			continue
		}
		imageEntries = append(imageEntries, Entry{Path: modDir + "/gfx/interface/ideologies/" + group + "_group.dds", Bytes: dds, Origin: "generated", SourceEntity: "ideology_icons"})
		emittedIdeologyIcons = append(emittedIdeologyIcons, group)
	}
	if gfx := ideologyIconGFX(emittedIdeologyIcons); len(gfx) > 0 {
		imageEntries = append(imageEntries, Entry{Path: modDir + "/interface/hagane_ideology_icons.gfx", Bytes: gfx, Origin: "generated", SourceEntity: "ideology_icons"})
	}
	tables, err := worldout.LoadVanillaTables(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load the base game's world tables: %w", err)
	}
	records, meta := worldout.EmitScenario(tables, scenario)
	if sink != nil {
		sink.worldMeta = meta

		for _, notice := range meta.OOB {
			sink.say(notice.Key, notice.Args)
		}
	}
	entries := make([]Entry, 0, len(records))
	for _, record := range records {
		entry := Entry{
			Path: record.Path, Bytes: []byte(record.Text), Origin: "generated", SourceEntity: "world",
		}
		if strings.HasSuffix(record.Path, "/common/countries/cosmetic.txt") {
			entry.CosmeticColorTags, err = cosmetic.ColorFileTags(entry.Bytes)
			if err != nil {
				return nil, err
			}
			base, err := cosmetic.VanillaColorFile(context.Background())
			if err != nil {
				return nil, err
			}
			entry.Bytes, err = cosmetic.MergeColorFiles(base, entry.Bytes, entry.CosmeticColorTags)
			if err != nil {
				return nil, err
			}
		}
		entries = append(entries, entry)
	}
	entries = append(entries, imageEntries...)
	return entries, nil
}

func worldInt(row *worldout.OMap, field string) int64 {
	value, ok := row.Get(field)
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func resolvePrivateImage(resolve BlobResolver, key, label string) ([]byte, error) {
	if resolve == nil {
		return nil, fmt.Errorf("%s requires private object resolver", label)
	}
	raw, err := resolve(key, 32<<20)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", label, err)
	}
	return raw, nil
}

func encodeFlagTGA(src image.Image, w, h int) []byte {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	header := make([]byte, 18)
	header[2] = 2
	binary.LittleEndian.PutUint16(header[12:14], uint16(w))
	binary.LittleEndian.PutUint16(header[14:16], uint16(h))
	header[16] = 32

	header[17] = flagTGADescriptorBottomLeft
	out := make([]byte, 18+w*h*4)
	copy(out, header)
	at := 18
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := dst.NRGBAAt(x, y)
			out[at], out[at+1], out[at+2], out[at+3] = c.B, c.G, c.R, 255
			at += 4
		}
	}
	return out
}

func worldString(row *worldout.OMap, field string) string {
	value, ok := row.Get(field)
	if !ok || value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

func firstWorldString(row *worldout.OMap, fields ...string) string {
	for _, field := range fields {
		if value := worldString(row, field); value != "" {
			return value
		}
	}
	return ""
}

func BuildNativeDescriptor(snapshot Snapshot, replacePaths []string) (NativeArtifact, error) {
	return buildNativeDescriptor(snapshot, replacePaths, nil)
}

func buildNativeDescriptor(snapshot Snapshot, replacePaths []string, coverOverride *string) (NativeArtifact, error) {
	projectValue, err := locout.ParseJSON(snapshot.Project)
	if err != nil {
		return NativeArtifact{}, fmt.Errorf("decode snapshot project: %w", err)
	}
	if projectValue.Kind != locout.KObj || projectValue.Obj == nil {
		return NativeArtifact{}, fmt.Errorf("snapshot project must be an object")
	}
	project := projectValue.Obj
	name := stringAttr(project, "name")
	tag := strings.ToUpper(strings.TrimSpace(stringAttr(project, "country_tag")))
	fallback := "mod_" + tag
	if tag == "" {
		fallback = "mod_" + strconv.FormatInt(snapshot.ProjectID, 10)
	}
	modDir := SanitizeModDirName(name, fallback)
	if coverOverride != nil {
		project.M["picture"] = locout.Str(*coverOverride)
	}
	descriptor := moddesc.GenerateDescriptor(project, replacePaths)
	return NativeArtifact{
		Filename: modDir + ".zip",
		Coverage: []string{"descriptor"},
		Entries: []Entry{
			{
				Path: modDir + ".mod", Bytes: []byte(descriptor + "\npath=\"mod/" + modDir + "\""),
				Origin: "generated", SourceEntity: "project:" + strconv.FormatInt(snapshot.ProjectID, 10),
			},
			{
				Path: modDir + "/descriptor.mod", Bytes: []byte(descriptor),
				Origin: "generated", SourceEntity: "project:" + strconv.FormatInt(snapshot.ProjectID, 10),
			},
		},
	}, nil
}

func stringAttr(obj *locout.Obj, name string) string {
	v := obj.Attr(name)
	if v.Kind == locout.KStr {
		return v.Str
	}
	return ""
}

func SanitizeModDirName(name, fallback string) string {
	var out strings.Builder
	lastUnderscore := false
	for _, r := range name {
		replace := r > unicode.MaxASCII || strings.ContainsRune(`:\/*?"<>|`, r)
		if replace {
			r = '_'
		}
		if r == '_' {
			if lastUnderscore {
				continue
			}
			lastUnderscore = true
		} else {
			lastUnderscore = false
		}
		out.WriteRune(r)
	}
	base := strings.Trim(out.String(), "_ ")
	base = strings.TrimRight(base, ". ")
	if base == "" {
		base = fallback
	}
	if base == "" {
		base = "mod"
	}
	return base
}
