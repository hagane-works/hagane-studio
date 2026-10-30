package exportcheck

type AirWingGroup struct {
	ID         int64            `json:"id"`
	ProjectID  int64            `json:"project_id"`
	OobFile    string           `json:"oob_file"`
	SourceFile string           `json:"source_file"`
	CountryTag string           `json:"country_tag"`
	Location   string           `json:"location"`
	Ordering   int              `json:"ordering"`
	Layout     []map[string]any `json:"layout"`
	Source     string           `json:"source"`

	Warnings []string `json:"warnings"`
}

type AirWing struct {
	ID            int64            `json:"id"`
	ProjectID     int64            `json:"project_id"`
	GroupID       int64            `json:"group_id"`
	EquipmentType string           `json:"equipment_type"`
	Name          string           `json:"name"`
	Owner         string           `json:"owner"`
	Amount        string           `json:"amount"`
	Ordering      int              `json:"ordering"`
	BodyAst       []map[string]any `json:"body_ast"`
	Source        string           `json:"source"`
}
