package guirender

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"haganeworks/internal/hoi4/gfx"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/hoi4/loc"
	"haganeworks/internal/hoi4/paradox"
)

type windowRecord struct {
	File   string
	Source string
	Node   map[string]any
}

type spriteRecord struct {
	File         string
	TextureFiles []string
	Frames       int

	FramesDeclared bool
	Effect         string
	BlockType      string
	Procedural     bool
	Cornered       bool
	BorderSize     map[string]any
	NaturalSize    map[string]any
}

type Index struct {
	ownedDLC    map[string]bool
	files       map[string][]byte
	filesFolded map[string]string
	windows     map[string]windowRecord
	sprites     map[string]spriteRecord
	loc         map[string]string

	lang        string
	langLoc     map[string]string
	fonts       map[string]fontRecord
	bitmapFonts map[string]bitmapFontDef
	fnts        map[string]*fntData

	focusTrees    map[string]*focusTreeRecord
	sharedFocuses map[string]*Focus

	continuousPalettes map[string]*continuousPalette

	scriptedTriggers map[string]*paradox.Block

	focusIcons map[string]string

	focusDefs         map[string]*Focus
	focusRewardBlocks map[string][]*paradox.Entry

	focusRewards map[string]map[string]float64

	defines map[string]float64

	alertCategories map[string]string

	characters        map[string]*characterRecord
	charactersByToken map[string]*Character

	ideaCategories []IdeaCategory

	bookmarks []Bookmark

	bookmarkFiles []string
	countryTags   map[string]struct{}

	countryHistories map[string][]historyFile
	ideologyParty    map[string]string

	ideologyGroups []ideologyGroup

	ideaPictures map[string]string
	ideaGroups   map[string]string

	ideaGroupOrder []string
	ideaTypes      map[string]string

	ideaCosts map[string]float64

	ideaMinExport map[string]float64

	ideaOrder   map[string][]string
	ideaChoices map[string]ideaChoice

	autonomyTrade map[string]float64
	subjectOf     map[string]subjectRecord

	autonomyLevels map[string]*autonomyRecord
	autonomySeq    int

	countryTagSeq  map[string]int
	countryTagNext int

	wargoals   map[string]*wargoalRecord
	wargoalSeq int

	intelAgencies  map[string][]*intelAgencyRecord
	intelBranches  map[string]*intelBranchRecord
	intelBranchSeq int

	specialProjects   map[string]*specialProjectRecord
	specialProjectSeq int
	specializations   map[string]*specializationRecord
	specializationSeq int

	scientistTraitIcons map[string]string

	strategicRegions map[int]*strategicRegion
	provinceRegions  map[int]int

	unitNames map[string]map[string][]string

	factionTemplates map[string]factionTemplate

	factionGoals  map[string]factionGoal
	factionLed    map[string][]datedFaction
	factionMember map[string][]datedFaction

	autonomyDefault string

	autonomyIDs []string

	ideaDefaults map[string]string

	ideaConscription map[string]float64

	oobs map[string]*oobRecord

	oobSources map[string][]byte

	divisionNames map[string]string

	namesGroups   map[string]*namesGroup
	namesGroupSeq int

	subUnits map[string]*subUnitRecord

	decisionCategories []DecisionCategory
	decisions          map[string][]Decision
	decisionConstants  map[string]float64

	definedTexts map[string]*definedText

	events map[string]*eventRecord

	technologies    map[string]*technologyRecord
	techFolders     map[string]*techFolderRecord
	techFolderOrder []string
	equipment       map[string]*equipmentRecord

	resourceFrames map[string]int

	countryCultures map[string]string
	countryFiles    map[string]string

	countryColors map[string][3]uint8

	tagColors map[string][3]uint8

	projectFlags map[string]string

	projectSprites map[string][2]int

	projectSpriteVersions map[string]string

	projectFileSources map[string]string

	spriteOverrides map[string]SpriteOverride

	projectRawTextures map[string]ProjectRawTexture

	dynamicModifiers map[string]string

	continents    []string
	stateProvince map[int]int

	states            map[int]*stateRecord
	provinceContinent map[int]int

	provinceDefs []provinceDef

	buildings map[string]*buildingRecord

	industrialOrgs     map[string]*industrialOrg
	industrialOrgOrder []string
	equipmentGroups    map[string][]string

	equipmentDuplicates map[string]equipmentDuplicate

	leaderTraits     map[string]*leaderTrait
	leaderTraitOrder []string
	characterSeq     int

	techSeq int

	subUnitSeq int

	hqTemplateRules []hqTemplateRule

	modifierDecimals map[string]int

	equipmentModules  map[string]*equipmentModule
	equipmentUpgrades map[string]*equipmentUpgrade

	equipmentSeq int

	countryLeaderTraits map[string]*paradox.Block

	grandDoctrines     map[string]grandDoctrine
	grandDoctrineOrder []string

	doctrines *doctrineIndex

	ideaModifierBlocks    map[string]*paradox.Block
	dynamicModifierBlocks map[string]*paradox.Block

	staticModifiers map[string]*paradox.Block

	source string

	fallback *Index

	external      map[string]ExternalTexture
	externalLoose map[string]string
	version       string

	baseURL string

	textureURL func(rel string) string

	sourceArchiveKey      string
	sourceArchiveRevision string

	masking *maskPlan
}

type fontRecord struct {
	Outline bool
	Line    int
	Size    int
}

var spriteBlockKeys = map[string]bool{
	"spritetype": true, "textspritetype": true, "corneredtilespritetype": true,
	"frameanimatedspritetype": true, "maskedshieldtype": true,
	"progressbartype": true, "circularprogressbartype": true,
	"piecharttype": true, "linecharttype": true,
}

func (idx *Index) HasOwnFile(name string) bool {
	if idx == nil {
		return false
	}
	_, ok := idx.files[name]
	return ok
}

func (idx *Index) SourceBytes() int64 {
	var total int64
	for _, body := range idx.files {
		total += int64(len(body))
	}
	return total
}

func NewIndex(files map[string][]byte) (*Index, error) {
	idx, err := newIndex(files, "mod")
	if err != nil {
		return nil, err
	}
	idx.expandDuplicateArchetypes()
	return idx, nil
}

func NewIndexWithFallback(files map[string][]byte, fallback *Index) (*Index, error) {
	idx, err := newIndex(files, "mod")
	if err != nil {
		return nil, err
	}
	idx.fallback = fallback
	idx.expandDuplicateArchetypes()
	return idx, nil
}

func newIndex(files map[string][]byte, source string) (*Index, error) {
	idx := &Index{
		files: map[string][]byte{}, filesFolded: map[string]string{},
		windows: map[string]windowRecord{}, sprites: map[string]spriteRecord{},
		loc: map[string]string{}, fonts: map[string]fontRecord{},
		bitmapFonts: map[string]bitmapFontDef{}, fnts: map[string]*fntData{},
		focusTrees: map[string]*focusTreeRecord{}, sharedFocuses: map[string]*Focus{},
		continuousPalettes:    map[string]*continuousPalette{},
		scriptedTriggers:      map[string]*paradox.Block{},
		focusIcons:            map[string]string{},
		focusDefs:             map[string]*Focus{},
		focusRewardBlocks:     map[string][]*paradox.Entry{},
		focusRewards:          map[string]map[string]float64{},
		defines:               map[string]float64{},
		alertCategories:       map[string]string{},
		characters:            map[string]*characterRecord{},
		charactersByToken:     map[string]*Character{},
		countryTags:           map[string]struct{}{},
		definedTexts:          map[string]*definedText{},
		countryHistories:      map[string][]historyFile{},
		ideologyParty:         map[string]string{},
		ideaPictures:          map[string]string{},
		ideaGroups:            map[string]string{},
		ideaTypes:             map[string]string{},
		ideaCosts:             map[string]float64{},
		ideaMinExport:         map[string]float64{},
		ideaOrder:             map[string][]string{},
		ideaChoices:           map[string]ideaChoice{},
		autonomyTrade:         map[string]float64{},
		subjectOf:             map[string]subjectRecord{},
		factionTemplates:      map[string]factionTemplate{},
		factionGoals:          map[string]factionGoal{},
		factionLed:            map[string][]datedFaction{},
		factionMember:         map[string][]datedFaction{},
		ideaDefaults:          map[string]string{},
		ideaConscription:      map[string]float64{},
		oobs:                  map[string]*oobRecord{},
		divisionNames:         map[string]string{},
		namesGroups:           map[string]*namesGroup{},
		subUnits:              map[string]*subUnitRecord{},
		decisions:             map[string][]Decision{},
		decisionConstants:     map[string]float64{},
		events:                map[string]*eventRecord{},
		technologies:          map[string]*technologyRecord{},
		resourceFrames:        map[string]int{},
		techFolders:           map[string]*techFolderRecord{},
		equipment:             map[string]*equipmentRecord{},
		countryCultures:       map[string]string{},
		countryColors:         map[string][3]uint8{},
		tagColors:             map[string][3]uint8{},
		countryFiles:          map[string]string{},
		dynamicModifiers:      map[string]string{},
		stateProvince:         map[int]int{},
		states:                map[int]*stateRecord{},
		provinceContinent:     map[int]int{},
		buildings:             map[string]*buildingRecord{},
		industrialOrgs:        map[string]*industrialOrg{},
		equipmentGroups:       map[string][]string{},
		equipmentDuplicates:   map[string]equipmentDuplicate{},
		leaderTraits:          map[string]*leaderTrait{},
		countryLeaderTraits:   map[string]*paradox.Block{},
		grandDoctrines:        map[string]grandDoctrine{},
		ideaModifierBlocks:    map[string]*paradox.Block{},
		dynamicModifierBlocks: map[string]*paradox.Block{},
		staticModifiers:       map[string]*paradox.Block{},
		source:                source,
	}
	paths := make([]string, 0, len(files))
	for name, body := range files {
		clean := strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/")
		idx.files[clean] = body
		idx.filesFolded[strings.ToLower(clean)] = clean
		paths = append(paths, clean)
	}
	sort.Strings(paths)
	for _, name := range paths {
		body := idx.files[name]
		lower := strings.ToLower(name)
		switch {
		case strings.HasPrefix(lower, "interface/") && strings.HasSuffix(lower, ".gui"):
			text, ok := gfx.DecodeScriptText(body)
			if !ok {
				continue
			}
			windows, err := ParseGUI(text)
			if err != nil {
				continue
			}
			for _, win := range windows {
				windowName, _ := win["name"].(string)
				if windowName == "" {
					continue
				}
				idx.windows[windowName] = windowRecord{
					File: strings.TrimPrefix(name, "interface/"), Source: idx.source, Node: win,
				}
			}
		case strings.HasPrefix(lower, "interface/") && strings.HasSuffix(lower, ".gfx"):
			text, ok := gfx.DecodeScriptText(body)
			if ok {
				idx.indexGFX(name, text)
			}
		case strings.HasPrefix(lower, "localisation/") && strings.HasSuffix(lower, ".yml"):
			idx.indexLocalisation(name, body)
		case strings.HasPrefix(lower, "gfx/fonts/") && strings.HasSuffix(lower, ".fnt"):
			idx.indexFont(name, body)
		case strings.HasPrefix(lower, "common/national_focus/") && strings.HasSuffix(lower, ".txt"):
			idx.indexFocusTreeFile(body)
		case strings.HasPrefix(lower, "common/continuous_focus/") && strings.HasSuffix(lower, ".txt"):
			idx.indexContinuousFocusFile(body)
		case strings.HasPrefix(lower, "common/scripted_triggers/") && strings.HasSuffix(lower, ".txt"):
			idx.indexScriptedTriggerFile(body)
		case strings.HasPrefix(lower, "common/characters/") && strings.HasSuffix(lower, ".txt"):
			idx.indexCharacterFile(name, body)
		case strings.HasPrefix(lower, "common/dynamic_modifiers/") && strings.HasSuffix(lower, ".txt"):
			idx.indexDynamicModifierFile(body)
		case strings.HasPrefix(lower, "common/ideas/") && strings.HasSuffix(lower, ".txt"):
			idx.indexIdeaFile(body)
		case strings.HasPrefix(lower, "common/idea_tags/") && strings.HasSuffix(lower, ".txt"):
			idx.indexIdeaTagFile(body)
		case strings.HasPrefix(lower, "common/ideologies/") && strings.HasSuffix(lower, ".txt"):
			idx.indexIdeologyFile(body)
		case strings.HasPrefix(lower, "common/bookmarks/") && strings.HasSuffix(lower, ".txt"):
			idx.indexBookmarkFile(lower, body)
		case strings.HasPrefix(lower, "common/country_tags/") && strings.HasSuffix(lower, ".txt"):
			idx.indexCountryTagFile(body)
		case strings.HasPrefix(lower, "history/countries/") && strings.HasSuffix(lower, ".txt"):
			idx.indexCountryHistoryFile(name, body)
		case strings.HasPrefix(lower, "common/countries/") && strings.HasSuffix(lower, ".txt"):
			idx.indexCountryFile(name, body)
		case strings.HasPrefix(lower, "common/technologies/") && strings.HasSuffix(lower, ".txt"):
			idx.indexTechnologyFile(body)
		case strings.HasPrefix(lower, "common/technology_tags/") && strings.HasSuffix(lower, ".txt"):
			idx.indexTechnologyTagFile(body)
		case strings.HasPrefix(lower, "common/buildings/") && strings.HasSuffix(lower, ".txt"):
			idx.indexBuildingFile(body)
		case strings.HasPrefix(lower, "common/resources/") && strings.HasSuffix(lower, ".txt"):
			idx.indexResourceFile(body)
		case strings.HasPrefix(lower, "common/wargoals/") && strings.HasSuffix(lower, ".txt"):
			idx.indexWargoalFile(body)
		case strings.HasPrefix(lower, "common/intelligence_agencies/") && strings.HasSuffix(lower, ".txt"):
			idx.indexIntelAgencyFile(name, body)
		case strings.HasPrefix(lower, "common/special_projects/projects/") && strings.HasSuffix(lower, ".txt"):
			idx.indexSpecialProjectFile(name, body)
		case strings.HasPrefix(lower, "common/special_projects/specialization/") && strings.HasSuffix(lower, ".txt"):
			idx.indexSpecializationFile(body)
		case strings.HasPrefix(lower, "common/scientist_traits/") && strings.HasSuffix(lower, ".txt"):
			idx.indexScientistTraitFile(body)
		case strings.HasPrefix(lower, "map/strategicregions/") && strings.HasSuffix(lower, ".txt"):
			idx.indexStrategicRegionFile(name, body)
		case strings.HasPrefix(lower, "common/intelligence_agency_upgrades/") && strings.HasSuffix(lower, ".txt"):
			idx.indexIntelUpgradeFile(name, body)
		case strings.HasPrefix(lower, "common/autonomous_states/") && strings.HasSuffix(lower, ".txt"):
			idx.indexAutonomyFile(name, body)
		case strings.HasPrefix(lower, "common/factions/") && strings.HasSuffix(lower, ".txt"):
			idx.indexFactionTemplateFile(body)
		case strings.HasPrefix(lower, "common/defines/") && strings.HasSuffix(lower, ".lua"):
			idx.indexDefinesFile(body)
		case lower == "common/alerts.txt":
			idx.indexAlertFile(body)
		case strings.HasPrefix(lower, "common/military_industrial_organization/organizations/") && strings.HasSuffix(lower, ".txt"):
			idx.indexIndustrialOrgFile(name, body)
		case strings.HasPrefix(lower, "common/unit_leader/") && strings.HasSuffix(lower, ".txt"):
			idx.indexLeaderTraitFile(body)
		case strings.HasPrefix(lower, "common/country_leader/") && strings.HasSuffix(lower, ".txt"):
			idx.indexCountryLeaderTraitFile(body)
		case strings.HasPrefix(lower, "common/doctrines/grand_doctrines/") && strings.HasSuffix(lower, ".txt"):
			idx.indexGrandDoctrineFile(body)
		case strings.HasPrefix(lower, "common/doctrines/folders/") && strings.HasSuffix(lower, ".txt"):
			idx.indexDoctrineFolderFile(body)
		case strings.HasPrefix(lower, "common/doctrines/tracks/") && strings.HasSuffix(lower, ".txt"):
			idx.indexDoctrineTrackFile(body)
		case strings.HasPrefix(lower, "common/doctrines/subdoctrines/") && strings.HasSuffix(lower, ".txt"):
			idx.indexSubDoctrineFile(body)
		case strings.HasPrefix(lower, "common/modifiers/") && strings.HasSuffix(lower, ".txt"):
			idx.indexStaticModifierFile(body)
		case strings.HasPrefix(lower, "common/equipment_groups/") && strings.HasSuffix(lower, ".txt"):
			idx.indexEquipmentGroupFile(body)
		case strings.HasPrefix(lower, "common/units/equipment/") && strings.HasSuffix(lower, ".txt"):
			idx.indexEquipmentFile(body)

		case strings.HasPrefix(lower, "common/units/names/") && strings.HasSuffix(lower, ".txt"):
			idx.indexUnitNamesFile(body)
		case strings.HasPrefix(lower, "common/units/names_divisions/") && strings.HasSuffix(lower, ".txt"):
			idx.indexDivisionNamesFile(body)
		case strings.HasPrefix(lower, "common/units/") && strings.HasSuffix(lower, ".txt"):
			idx.indexSubUnitFile(name, body)
		case strings.HasPrefix(lower, "common/scripted_localisation/") && strings.HasSuffix(lower, ".txt"):
			idx.indexScriptedLocFile(body)
		case strings.HasPrefix(lower, "common/decisions/categories/") && strings.HasSuffix(lower, ".txt"):
			idx.indexDecisionCategoryFile(body)
		case strings.HasPrefix(lower, "common/decisions/") && strings.HasSuffix(lower, ".txt"):
			idx.indexDecisionFile(name, body)
		case strings.HasPrefix(lower, "events/") && strings.HasSuffix(lower, ".txt"):
			idx.indexEventFile(body)
		case strings.HasPrefix(lower, "history/units/") && strings.HasSuffix(lower, ".txt"):
			idx.indexOOBFile(lower, body)
		case strings.HasPrefix(lower, "history/general/") && strings.HasSuffix(lower, ".txt"):
			idx.indexGeneralHistoryFile(name, body)
		case strings.HasPrefix(lower, "history/states/") && strings.HasSuffix(lower, ".txt"):
			idx.indexStateFile(body)
		case lower == "documentation/modifiers_documentation.md":
			idx.indexModifierDocs(body)
		case lower == "map/continent.txt":
			idx.indexContinentFile(body)
		case lower == "map/definition.csv":
			idx.indexProvinceDefinitions(body)
		}
	}
	return idx, nil
}

func (idx *Index) WindowCount() int { return len(idx.windows) }
func (idx *Index) SpriteCount() int { return len(idx.sprites) }

func (idx *Index) SpritePNG(name string) ([]byte, int, int, bool) {

	record, ok := idx.spriteRecordFor(name)
	if !ok {
		return nil, 0, 0, false
	}
	for _, texture := range record.TextureFiles {
		actual, body, found := idx.localTexture(texture)
		if !found {
			continue
		}
		img, err := importassets.DecodeImage(body, actual)
		if err != nil {
			continue
		}
		var out bytes.Buffer
		if err := png.Encode(&out, img); err != nil {
			continue
		}
		bounds := img.Bounds()
		return out.Bytes(), bounds.Dx(), bounds.Dy(), true
	}
	return nil, 0, 0, false
}

func (idx *Index) SpriteTextureMember(name string) (string, bool) {
	record, ok := idx.spriteRecordFor(name)
	if ok {
		for _, texture := range record.TextureFiles {
			if actual, _, found := idx.localTexture(texture); found {
				return actual, true
			}
		}
		return "", false
	}
	for layer := idx; layer != nil; layer = layer.fallback {
		if actual, _, found := layer.resolveLooseTexture(name); found {
			return actual, true
		}
	}
	return "", false
}

func (idx *Index) localTexture(value string) (string, []byte, bool) {
	variants := textureVariants(value)
	for layer := idx; layer != nil; layer = layer.fallback {
		for _, variant := range variants {
			if actual, found := layer.filesFolded[variant]; found {
				return actual, layer.files[actual], true
			}
		}
	}
	return "", nil, false
}

func (idx *Index) Source(file string) ([]byte, bool) {
	clean := strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	if actual, ok := idx.filesFolded[strings.ToLower(clean)]; ok {
		body := idx.files[actual]
		return append([]byte(nil), body...), true
	}
	if idx.fallback != nil {
		return idx.fallback.Source(file)
	}
	return nil, false
}

func (idx *Index) ListWindows() []map[string]any {
	rows := make([]map[string]any, 0, len(idx.windows))
	appendRows := func(layer *Index, shadowedBy map[string]windowRecord) {
		for name, record := range layer.windows {
			if _, shadowed := shadowedBy[name]; shadowed {
				continue
			}
			children, _ := record.Node["children"].([]map[string]any)
			rows = append(rows, map[string]any{
				"name": name, "file": record.File, "source": record.Source,
				"children": len(children), "category": classifyWindow(name, record.Node),
			})
		}
	}

	seen := map[string]windowRecord{}
	for layer := idx; layer != nil; layer = layer.fallback {
		appendRows(layer, seen)
		for name, record := range layer.windows {
			if _, ok := seen[name]; !ok {
				seen[name] = record
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i]["name"].(string)) < strings.ToLower(rows[j]["name"].(string))
	})
	return rows
}

func (idx *Index) Window(name string) (map[string]any, bool) {
	var record windowRecord
	ok := false
	for layer := idx; layer != nil && !ok; layer = layer.fallback {
		record, ok = layer.windows[name]
	}
	if !ok {
		return nil, false
	}

	return idx.enrich(name, record), true
}

func (idx *Index) WindowFromSource(name, file string, source []byte) (map[string]any, bool, error) {
	text, ok := gfx.DecodeScriptText(source)
	if !ok {
		return nil, false, fmt.Errorf("GUI source is not UTF-8 or Windows-1252 text")
	}
	windows, err := ParseGUI(text)
	if err != nil {
		return nil, false, err
	}
	for _, node := range windows {
		if nodeName, _ := node["name"].(string); nodeName == name {
			return idx.enrich(name, windowRecord{File: strings.TrimPrefix(file, "interface/"), Source: idx.sourceOfFile(file), Node: node}), true, nil
		}
	}
	return nil, false, nil
}

func (idx *Index) enrich(name string, record windowRecord) map[string]any {
	record.Node = withLanguageVariants(record.Node, idx.lang)
	sprites := map[string]any{}
	texts := map[string]any{}
	fonts := map[string]any{}
	var collect func(map[string]any)
	collect = func(node map[string]any) {
		refs := []any{node["sprite"]}
		if bg, ok := node["background"].(map[string]any); ok {
			refs = append(refs, bg["sprite"])
		}
		for _, raw := range refs {
			if ref, ok := raw.(string); ok && ref != "" {
				if _, seen := sprites[ref]; !seen {
					sprites[ref] = idx.spritePayload(ref)
				}
			}
		}
		if raw, ok := node["text"].(string); ok && raw != "" {
			if _, seen := texts[raw]; !seen {
				texts[raw] = idx.resolveText(raw)
			}
		}

		for _, key := range tooltipKeys(node) {
			if _, seen := texts[key]; !seen {
				texts[key] = idx.tooltipText(key)
			}
		}
		if raw, ok := node["font"].(string); ok && raw != "" {
			if _, seen := fonts[raw]; !seen {
				fonts[raw] = idx.fontMeta(raw)
			}
		}
		if bg, ok := node["background"].(map[string]any); ok {
			collect(bg)
		}
		for _, child := range nodeChildren(node) {
			collect(child)
		}
	}
	collect(record.Node)
	return map[string]any{
		"name": name, "file": record.File, "source": record.Source,
		"node": record.Node, "sprites": sprites, "texts": texts,
		"fonts": fonts, "overlap_groups": detectOverlapGroups(record.Node),
	}
}

func (idx *Index) indexGFX(file, text string) {
	root, _, err := paradox.Parse(text)
	if err != nil {
		return
	}
	expandConstants(root)
	var scan func([]*paradox.Entry)
	scan = func(entries []*paradox.Entry) {
		for _, entry := range entries {
			if entry == nil || entry.Key == nil || entry.Block == nil {
				continue
			}
			key := strings.ToLower(*entry.Key)
			if key == "spritetypes" || key == "bitmapfonts" {
				scan(entry.Block.Entries)
				continue
			}
			if key == "bitmapfont" {
				idx.indexBitmapFont(entry.Block.Entries)
				continue
			}
			if !spriteBlockKeys[key] {
				continue
			}
			body := entry.Block.Entries
			name := ciScalar(body, "name")
			if name == "" {
				continue
			}
			textures := make([]string, 0, 4)
			for _, field := range []string{"texturefile", "texturefile1", "texturefile2", "texturefile3"} {
				if value := ciScalar(body, field); value != "" && !contains(textures, value) {
					textures = append(textures, strings.ReplaceAll(value, "\\", "/"))
				}
			}
			frames := toInt(ciScalarValue(body, "noofframes"), 1)
			if frames < 1 {
				frames = 1
			}
			natural := ciPoint(body, "size")
			if natural == nil {
				if value, ok := toFloat(ciScalarValue(body, "size")); ok {
					n := int(value)
					natural = map[string]any{"x": n, "y": n}
				}
			}
			idx.sprites[name] = spriteRecord{
				File:         file,
				TextureFiles: textures, Frames: frames, FramesDeclared: ciScalarValue(body, "noofframes") != nil,
				Effect:    ciScalar(body, "effectfile"),
				BlockType: key, Procedural: (key == "piecharttype" || key == "linecharttype") && len(textures) == 0,
				Cornered: key == "corneredtilespritetype", BorderSize: ciPoint(body, "bordersize"), NaturalSize: natural,
			}
		}
	}
	scan(root.Entries)
}

func ciScalarValue(entries []*paradox.Entry, wanted ...string) any {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block != nil {
			continue
		}
		for _, key := range wanted {
			if strings.EqualFold(*entry.Key, key) {
				return scalarValue(entry.Scalar)
			}
		}
	}
	return nil
}

func ciScalar(entries []*paradox.Entry, wanted ...string) string {
	return toString(ciScalarValue(entries, wanted...))
}

func ciPoint(entries []*paradox.Entry, wanted ...string) map[string]any {
	for _, entry := range entries {
		if entry == nil || entry.Key == nil || entry.Block == nil {
			continue
		}
		match := false
		for _, key := range wanted {
			match = match || strings.EqualFold(*entry.Key, key)
		}
		if !match {
			continue
		}
		x, xok := strictInt(ciScalarValue(entry.Block.Entries, "x"))
		y, yok := strictInt(ciScalarValue(entry.Block.Entries, "y"))
		if xok && yok {
			return map[string]any{"x": x, "y": y}
		}
		return nil
	}
	return nil
}

func (idx *Index) indexLocalisation(name string, body []byte) {
	lower := strings.ToLower(name)
	for _, lang := range []string{"braz_por", "french", "german", "japanese", "korean", "polish", "russian", "simp_chinese", "spanish"} {
		if strings.Contains(lower, "/"+lang+"/") {
			return
		}
	}
	text, ok := gfx.DecodeScriptText(body)
	if !ok {
		return
	}
	language, entries, _ := loc.ParseLocText(text)
	if language != "" && language != "l_english" {
		return
	}
	for key, value := range entries {
		idx.loc[key] = value
	}
}

var (
	fontOutline = regexp.MustCompile(`outline=(\d+)`)
	fontLine    = regexp.MustCompile(`lineHeight=(\d+)`)
	fontSize    = regexp.MustCompile(`\bsize=(-?\d+)`)
)

func (idx *Index) indexFont(name string, body []byte) {
	text := string(body)
	if len(text) > 2000 {
		text = text[:2000]
	}
	stem := strings.ToLower(strings.TrimSuffix(path.Base(name), path.Ext(name)))
	if data, ok := parseFnt(body); ok {
		idx.fnts[foldTexturePath(strings.TrimSuffix(name, path.Ext(name)))] = data
	}
	idx.fonts[stem] = fontRecord{
		Outline: regexInt(fontOutline, text) > 0,
		Line:    regexInt(fontLine, text), Size: abs(regexInt(fontSize, text)),
	}
}

func regexInt(re *regexp.Regexp, text string) int {
	match := re.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0
	}
	n, _ := strconv.Atoi(match[1])
	return n
}

func (idx *Index) spritePayload(ref string) map[string]any {
	meta, found := idx.spriteRecordFor(ref)
	width, height, resolved, externalRel := 0, 0, false, ""
	projectVersion := ""

	override, overridden := idx.spriteOverride(ref)
	if overridden {
		width, height, resolved = override.Width, override.Height, true
	}
	if !overridden {

		if size, ok := idx.projectSpriteSize(ref); ok {
			width, height, resolved = size[0], size[1], true
			projectVersion = idx.projectSpriteVersion(ref)
		}
	}
	if found && !resolved {

		for _, candidate := range meta.TextureFiles {
			if texture, ok := idx.projectRawTexture(candidate); ok {
				width, height, resolved = texture.Width, texture.Height, true
				projectVersion = texture.Version
				break
			}
		}
	}
	if found && !resolved {
		for _, candidate := range meta.TextureFiles {
			if w, h, rel, ok := idx.textureSize(candidate); ok {
				width, height, resolved, externalRel = w, h, true, rel
				break
			}
		}
	}
	if !found && !resolved {
		if w, h, rel, ok := idx.looseTextureSize(ref); ok {
			width, height, resolved, externalRel = w, h, true, rel
		}
	}
	if !resolved && meta.NaturalSize != nil {
		width = toInt(meta.NaturalSize["x"], 0)
		height = toInt(meta.NaturalSize["y"], 0)
	}
	hintW, hintH := spriteSizeHint(ref, meta.TextureFiles)
	if width == 0 {
		width = hintW * max(1, meta.Frames)
	}
	if height == 0 {
		height = hintH
	}
	frames := max(1, meta.Frames)
	if frames == 1 && !meta.FramesDeclared && hintW > 0 && width >= hintW*2 && width%hintW == 0 && width/hintW <= 16 && (hintH == 0 || height == 0 || abs(height-hintH) <= 4) {
		frames = width / hintW
	}
	textureFiles := append([]string(nil), meta.TextureFiles...)
	primary := ""
	if len(textureFiles) > 0 {
		primary = textureFiles[0]
	}
	payload := map[string]any{
		"resolved": resolved, "procedural": meta.Procedural, "frames": frames,
		"width": width, "height": height, "texturefile": primary,
		"texturefiles": textureFiles, "effect": nilIfEmpty(meta.Effect),
		"effect_kind": effectKind(meta.Effect), "block_type": nilIfEmpty(meta.BlockType),
		"cornered": meta.Cornered, "bordersize": nilMap(meta.BorderSize),
		"missing_file": !resolved && !meta.Procedural, "overridden": overridden,
		"override_path": nil,
	}
	if projectVersion != "" {

		payload["version"] = projectVersion
	}
	if overridden {
		payload["override_path"] = "gui_sprite_overrides/" + ref
		if override.URL != "" {
			payload["vanilla_url"] = override.URL
		}
	}
	if meta.BlockType == "progressbartype" || meta.BlockType == "circularprogressbartype" {

		if meta.NaturalSize != nil {
			payload["size"] = meta.NaturalSize
		}
		urls := make([]string, 0, len(textureFiles))
		for _, texture := range textureFiles {
			url := ""
			if _, _, rel, ok := idx.textureSize(texture); ok && rel != "" {
				url = idx.vanillaTextureURL(rel)
			}
			urls = append(urls, url)
		}
		payload["texture_urls"] = urls
	}
	if externalRel != "" {

		payload["vanilla"] = true
		if url := idx.vanillaTextureURL(externalRel); url != "" {
			payload["vanilla_url"] = url
		}
	}
	return payload
}

func (idx *Index) resolveTexture(value string) (string, []byte, bool) {
	for _, variant := range textureVariants(value) {
		if actual, ok := idx.filesFolded[variant]; ok {
			return actual, idx.files[actual], true
		}
	}
	return "", nil, false
}

func textureVariants(value string) []string {
	rel := foldTexturePath(value)
	variants := []string{rel}
	ext := path.Ext(rel)
	if ext == ".dds" || ext == ".png" || ext == ".tga" {
		stem := strings.TrimSuffix(rel, ext)
		for _, other := range []string{".dds", ".tga", ".png"} {
			if other != ext {
				variants = append(variants, stem+other)
			}
		}
	}
	return variants
}

func (idx *Index) resolveLooseTexture(ref string) (string, []byte, bool) {
	wanted := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ref)), "gfx_")
	keys := []string{wanted, "gfx_" + wanted}
	paths := make([]string, 0)
	for name := range idx.files {
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "gfx/interface/") || (path.Ext(lower) != ".dds" && path.Ext(lower) != ".png" && path.Ext(lower) != ".tga") {
			continue
		}
		candidates := looseTextureNames(lower)
		for _, key := range keys {
			if contains(candidates, key) {
				paths = append(paths, name)
			}
		}
	}
	if len(paths) == 0 {
		return "", nil, false
	}
	sort.Strings(paths)
	return paths[0], idx.files[paths[0]], true
}

func looseTextureNames(lowerPath string) []string {
	stem := strings.TrimSuffix(path.Base(lowerPath), path.Ext(lowerPath))
	parent := path.Base(path.Dir(lowerPath))
	singular := strings.TrimSuffix(parent, "s")
	if strings.HasSuffix(parent, "ies") {
		singular = strings.TrimSuffix(parent, "ies") + "y"
	}
	return []string{stem, "gfx_" + stem, parent + "_" + stem, "gfx_" + parent + "_" + stem, singular + "_" + stem, "gfx_" + singular + "_" + stem}
}

func imageSize(name string, body []byte) (int, int, bool) {
	if strings.EqualFold(path.Ext(name), ".dds") {
		if len(body) < 20 || string(body[:4]) != "DDS " {
			return 0, 0, false
		}
		height := int(binary.LittleEndian.Uint32(body[12:16]))
		width := int(binary.LittleEndian.Uint32(body[16:20]))
		return width, height, width > 0 && height > 0
	}
	if strings.EqualFold(path.Ext(name), ".tga") {
		if len(body) < 18 {
			return 0, 0, false
		}
		width := int(binary.LittleEndian.Uint16(body[12:14]))
		height := int(binary.LittleEndian.Uint16(body[14:16]))
		return width, height, width > 0 && height > 0
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(body))
	return config.Width, config.Height, err == nil && config.Width > 0 && config.Height > 0
}

var sizeHintRE = regexp.MustCompile(`(?i)(^|[^0-9])(\d{2,4})x(\d{2,4})([^0-9]|$)`)

func spriteSizeHint(ref string, textures []string) (int, int) {
	for _, value := range append([]string{ref}, textures...) {
		match := sizeHintRE.FindStringSubmatch(value)
		if len(match) == 5 {
			width, _ := strconv.Atoi(match[2])
			height, _ := strconv.Atoi(match[3])
			return width, height
		}
	}
	return 0, 0
}

func effectKind(effect string) any {
	if effect == "" {
		return nil
	}
	lower := strings.ToLower(effect)
	switch {
	case strings.Contains(lower, "buttonstate"):
		return "button_state"
	case strings.Contains(lower, "progress") || strings.Contains(lower, "line_graph"):
		return "progress"
	case strings.Contains(lower, "mask"):
		return "mask"
	case strings.HasSuffix(lower, ".shader") || strings.HasSuffix(lower, ".fx"):
		return "shader"
	default:
		return "other"
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nilMap(value map[string]any) any {
	if value == nil {
		return nil
	}
	return value
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
