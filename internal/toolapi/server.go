package toolapi

func New(proxy *RESTProxy) *Server {
	server := NewServer()

	server.AddReceivingMiddleware(
		editRevisionsMiddleware(),
		mutationReceiptMiddleware(proxy),
		idReassignmentMiddleware(),
		authoringDefaultsMiddleware(proxy),

		fieldReferenceHintMiddleware(),
	)

	registerListProjects(server, proxy)
	registerCreateCharacter(server, proxy)
	registerEnsureProjectSettings(server, proxy)
	registerCreateCountry(server, proxy)

	registerUpdateCharacter(server, proxy, "/api")
	registerAssignCharacterRole(server, proxy, "/api")

	registerCreateEventNamespace(server, proxy)
	registerCreateEvent(server, proxy)
	registerUpdateEvent(server, proxy)
	registerAddEventOption(server, proxy)

	registerCreateFocusTree(server, proxy)
	registerCreateFocus(server, proxy)
	registerUpdateFocus(server, proxy)
	registerConnectFocus(server, proxy)
	registerSetFocusMutualExclusion(server, proxy)
	registerDeleteFocus(server, proxy)

	registerCreateDecisionCategory(server, proxy)
	registerCreateDecision(server, proxy)
	registerUpdateDecision(server, proxy)
	registerCreateIdea(server, proxy)
	registerUpdateIdea(server, proxy)
	registerCreateBookmark(server, proxy)
	registerSetCountryHistory(server, proxy)

	registerCreateTechnology(server, proxy)
	registerUpdateTechnology(server, proxy)
	registerCreateEquipment(server, proxy)
	registerCreateEquipmentVariant(server, proxy)
	registerCreateDoctrine(server, proxy)
	registerUpdateDoctrine(server, proxy)

	registerTechSurfaceProxyTools(server, proxy)
	registerPoliticsProxyTools(server, proxy)

	registerCreateSubUnit(server, proxy)
	registerCreateAirWingGroup(server, proxy)
	registerAddAirWing(server, proxy)
	registerCreateFleet(server, proxy)
	registerAddTaskForce(server, proxy)
	registerAddShip(server, proxy)
	registerCreateDivisionTemplate(server, proxy)
	registerAddDivisionDeployment(server, proxy)

	registerCreateMio(server, proxy)
	registerAddMioTrait(server, proxy)
	registerCreateIntelligenceAgency(server, proxy)
	registerAddAgencyUpgrade(server, proxy)
	registerCreateSpecialProject(server, proxy)
	registerCreateAutonomyState(server, proxy)
	registerEditState(server, proxy)
	registerSeedStatesFromVanilla(server, proxy)

	registerCreateCustomTrait(server, proxy)
	registerCreateDynamicModifier(server, proxy)
	registerCreateRawFromTemplate(server, proxy)
	registerCreateUiPanel(server, proxy)
	registerUIPanelTools(server, proxy)
	registerGUIOverlayTools(server, proxy)
	registerCreateSuperEvent(server, proxy)
	registerCreateScriptedLocalisation(server, proxy)
	registerCreateScriptedSnippet(server, proxy)
	registerGetGUISprites(server, proxy)
	registerGUITextTools(server, proxy)

	registerWorldImageTools(server, proxy)

	registerWorldMapCheckTools(server, proxy)

	registerWorldMapRecordTools(server, proxy)

	registerCreateOpinionModifier(server, proxy)
	registerUpdateOpinionModifier(server, proxy)
	registerCreateModifierDefinition(server, proxy)
	registerUpdateModifierDefinition(server, proxy)
	registerCreateWargoal(server, proxy)
	registerUpdateWargoal(server, proxy)
	registerCreateAiStrategyPlan(server, proxy)
	registerUpdateAiStrategyPlan(server, proxy)
	registerCreateAiTemplate(server, proxy)
	registerUpdateAiTemplate(server, proxy)

	registerGetProjectOverview(server, proxy)
	registerKnowledgeTools(server)
	registerGetFocusTree(server, proxy)
	registerGetCharacters(server, proxy)
	registerValidateProject(server, proxy)

	registerListEntities(server, proxy)
	registerGetEntity(server, proxy)

	registerUpdateGapTools(server, proxy)

	registerDailyEditTools(server, proxy)

	registerPreviewDeleteEntity(server, proxy)
	registerDeleteEntity(server, proxy)
	registerListRawTemplates(server, proxy)
	registerListUiPanels(server, proxy)
	registerGetSubUnits(server, proxy)
	registerGetOobSnapshot(server, proxy)
	registerGetStates(server, proxy)
	registerValidateSpecialProjects(server, proxy)

	registerExportProject(server, proxy)
	return server
}
