package exportengine

import (
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/exportmsg"
)

func noteVanillaCharacterIDCollision(sink *imageSink, entity *emitmisc.NS) {
	tag := strings.TrimSpace(entity.Str("tag"))
	if tag == "" {
		return
	}
	if !loadCrossRefVanillaIDs()["character"][tag] {
		return
	}

	key := "character.vanilla_id_recruited_twice"
	if entity.Bool("replaces_vanilla_id", false) {
		key = "character.vanilla_id"
	}
	sink.say(key, exportmsg.Args{"tag": tag})
}
