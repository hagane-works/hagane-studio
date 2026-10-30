package hoi4dlc

import "strings"

var Selectable = []string{
	"No Compromise, No Surrender",
	"Arms Against Tyranny",
	"La Resistance",
	"By Blood Alone",
	"No Step Back",
	"Gotterdammerung",
	"Battle for the Bosporus",
	"Graveyard of Empires",
	"Trial of Allegiance",
	"Poland: United and Ready",
}

var Integrated = []string{"Together for Victory", "Death or Dishonor", "Waking the Tiger", "Man the Guns"}

func Canonical(name string) (string, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, known := range Selectable {
		if strings.ToLower(known) == want {
			return known, true
		}
	}
	return "", false
}
