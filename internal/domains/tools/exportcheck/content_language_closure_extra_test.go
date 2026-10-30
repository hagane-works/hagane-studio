package exportcheck

func (a ContentLanguageClosureAudit) IsClosed() bool {
	return len(a.MissingLanguages) == 0
}

func AuditContentLanguageClosureGeneric(
	exportLanguages []string, entities []map[string]any,
) ContentLanguageClosureAudit {
	fieldsByLanguage := map[string]map[string]bool{}
	for _, entity := range entities {
		scanEntity(entity, "entities[]", fieldsByLanguage, "generic")
	}
	return finishClosureAudit(exportLanguages, fieldsByLanguage)
}
