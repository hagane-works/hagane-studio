package military

import "haganeworks/internal/project"

type writer func(tables []string, change func(*project.Tx) error) error

func inTx(tx *project.Tx) writer {
	return func(_ []string, change func(*project.Tx) error) error { return change(tx) }
}

func CreateDivisionTemplate(tx *project.Tx, in DivisionTemplateCreateInput) (DivisionTemplate, error) {
	return createDivisionTemplate(inTx(tx), in)
}

func UpdateDivisionTemplate(tx *project.Tx, id int64, patch DivisionTemplatePatch) (DivisionTemplate, error) {
	return updateDivisionTemplate(inTx(tx), id, patch)
}

func ImportDivisionTemplatesText(tx *project.Tx, body, oobFile, policy string) (DivisionTemplateImportResult, error) {
	return importDivisionTemplatesText(inTx(tx), body, oobFile, policy)
}

func InsertMIO(tx *project.Tx, in MIOCreateInput) (MIO, error) {
	return createMIO(inTx(tx), in)
}

func ValidateSubUnitClassification(unitTypes, categories []string) error {
	if _, err := validateSubUnitTypes(unitTypes); err != nil {
		return err
	}
	_, err := validateSubUnitCategories(categories)
	return err
}
