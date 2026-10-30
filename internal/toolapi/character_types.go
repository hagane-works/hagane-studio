package toolapi

type CharacterCreateInput struct {
	Name                       string
	CountryTag                 string
	RoleType                   string
	Description                string
	Tag                        string
	Skill                      int
	Traits                     []string
	Ideology                   string
	Gender                     string
	HasAdvisorRole             bool
	AdvisorSlot                string
	AdvisorTraits              []string
	AdvisorCost                int
	CanBeCaptured              bool
	AlwaysShowOnActionsTooltip *bool

	DisplayNameLocs map[string]string
	DescriptionLocs map[string]string
}
