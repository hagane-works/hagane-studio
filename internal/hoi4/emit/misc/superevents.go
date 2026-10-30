package emitmisc

import (
	"regexp"
	"strings"
)

func SeIsAutoFire(se *NS) bool {
	if se.Bool("auto_fire_on_startup", false) {
		return true
	}
	v := se.Attr("auto_fire_after_days", int64(0))
	if v == nil {
		v = int64(0)
	}
	if !Truthy(v) {
		return false
	}
	return asInt64(v) > 0
}

func ScanForEventRef(text, eventID string) bool {
	if text == "" || eventID == "" {
		return false
	}
	eid := regexp.QuoteMeta(eventID)
	bare := `(?:country_event|news_event|event)\s*=\s*` + eid + `\b`
	braced := `(?:country_event|news_event|event)\s*=\s*\{[^}]*\bid\s*=\s*` + eid + `\b`
	pattern := regexp.MustCompile(`(?:` + bare + `)|(?:` + braced + `)`)
	return pattern.MatchString(text)
}

func EmitSuperEventBoilerplate(write func(path, text string), modDir, ctag string) {
	upper := pyUpper(ctag)

	gfxText := `spriteTypes = {

	### Superevent frame assets (binary DDS shipped with ` + "`gfx/tvbigevent/`" + `).
	### These 4 sprite names are referenced by ` + "`interface/" + upper + "_superevent.gui`" + `.

	spriteType = {
		name = "GFX_` + upper + `_SE_bg"
		texturefile = "gfx//tvbigevent//SE_bg.dds"
		legacy_lazy_load = yes
	}
	spriteType = {
		name = "GFX_` + upper + `_SE_exit_btn"
		texturefile = "gfx//tvbigevent//SE_btn.dds"
		legacy_lazy_load = yes
	}
	spriteType = {
		name = "GFX_` + upper + `_SE_frame_overlay"
		texturefile = "gfx//tvbigevent//SE_frame.dds"
		legacy_lazy_load = yes
	}
	# Fallback picture · used when no SuperEvent row uploaded a custom image.
	spriteType = {
		name = "GFX_` + upper + `_SE_pic_entry"
		texturefile = "gfx//tvbigevent//SE_pic.dds"
		legacy_lazy_load = yes
	}

	### Per-event picture sprites are registered separately in
	### interface/` + upper + `_se_pictures.gfx (auto-emitted at export time).
}
`
	write(modDir+"interface/"+upper+"_superevent.gfx", gfxText)

	guiText := `guiTypes = {

	containerWindowType = {
		name = "` + upper + `_SE_main_container"
		orientation = center
		Origo = center
		size = { width = 500 height = 700 }
		position = { x = 0  y = 0 }
		moveable = yes

		background = {
			name = "Background"
			quadTextureSprite = "GFX_tiled_window_transparent"
		}

		iconType = {
			name = "` + upper + `_SE_bg"
			spriteType = "GFX_` + upper + `_SE_bg"
			position = { x = 0 y = 0 }
			Orientation = upper_left
			alwaystransparent = yes
		}

		iconType = {
			name = "` + upper + `_SE_picture_positioning_entry"
			spriteType = "GFX_` + upper + `_SE_pic_entry"
			position = { x = 20  y = 21 }
			Orientation = upper_left
			alwaystransparent = yes
		}

		iconType = {
			name = "` + upper + `_SE_frame_overlay"
			spriteType = "GFX_` + upper + `_SE_frame_overlay"
			position = { x = 0 y = 0 }
			Orientation = upper_left
			alwaystransparent = yes
		}

		# 关闭按钮 · clickable hitbox 必须来自 quadTextureSprite · buttonType 用裸
		# spriteType(iconType 的非交互属性)会渲染贴图但拿不到点击 hitbox → 整个
		# superevent「点哪里都没反应」。三角验证:TFR(GFX_spacebar)+ CCCW
		# (GFX_Super_event_bg)两个 working mod 的 close 按钮全用 quadTextureSprite。
		# 按钮名 + "_click" = scripted_gui effect 名(` + upper + `_SE_exit_btn_click)·
		# 这是 HOI4 scripted_gui 的绑定约定(TFR Super_Event_Close→_click 印证)。
		buttonType = {
			name = "` + upper + `_SE_exit_btn"
			position = { x = 184  y = 638 }
			quadTextureSprite = "GFX_` + upper + `_SE_exit_btn"
			orientation = upper_left
			scale = 1.1
			clicksound = click_ok
		}

		# 5 段化文案 · text 字段必须用 [NAME] 方括号触发 scripted_localisation dispatch.
		instantTextboxType = {
			name = "` + upper + `_SE_btn"
			position = { x = 0 y = 654  }
			font = "hoi_20bs"
			text = "[` + upper + `_SE_btn]"
			maxWidth = 500
			maxHeight = 30
			fixedsize = yes
			format = center
		}
		instantTextboxType = {
			name = "` + upper + `_SE_title"
			position = { x = 0 y = 278 }
			font = "hoi_30header"
			text = "[` + upper + `_SE_title]"
			maxWidth = 500
			maxHeight = 80
			fixedsize = yes
			format = center
		}
		instantTextboxType = {
			name = "` + upper + `_SE_desc_left"
			position = { x = 30 y = 342 }
			font = "hoi_18mbs"
			text = "[` + upper + `_SE_desc]"
			maxWidth = 440
			maxHeight = 200
			format = left
			fixedsize = yes
		}
		instantTextboxType = {
			name = "` + upper + `_SE_desc_center"
			position = { x = 30 y = 342 }
			font = "hoi_18mbs"
			text = "[` + upper + `_SE_desc]"
			maxWidth = 440
			maxHeight = 200
			format = center
			fixedsize = yes
		}
		instantTextboxType = {
			name = "` + upper + `_SE_quote"
			position = { x = 50  y = 560 }
			font = "hoi_16mbs"
			text = "[` + upper + `_SE_quote]"
			maxWidth = 400
			maxHeight = 50
			format = center
			fixedsize = yes
		}
		instantTextboxType = {
			name = "` + upper + `_SE_quote_extra"
			position = { x = 50  y = 560 }
			font = "hoi_16mbs"
			text = "[` + upper + `_SE_quote]"
			maxWidth = 400
			maxHeight = 50
			format = center
			fixedsize = yes
		}
	}

}
`
	write(modDir+"interface/"+upper+"_superevent.gui", guiText)

	seEffects := `SE_center = {
	set_global_flag = ` + upper + `_SE_center
}
# Toggle the SuperEvent layout to centered text mode (default is left-aligned).
`
	write(modDir+"common/scripted_effects/"+upper+"_big_event_scripted_effects.txt", seEffects)

	sguiText := `scripted_gui = {
	` + upper + `_SE_main_container = {
		window_name = ` + upper + `_SE_main_container
		context_type = player_context

		visible = {
			has_variable = global.` + upper + `_se_display
			is_ai = no
		}
		triggers = {
			# Toggle alignment by hiding one of the two desc textboxes
			` + upper + `_SE_desc_left_visible = { NOT = { has_global_flag = ` + upper + `_SE_center } }
			` + upper + `_SE_desc_center_visible = { has_global_flag = ` + upper + `_SE_center }
		}
		effects = {
			` + upper + `_SE_exit_btn_click = {
				clr_global_flag = ` + upper + `_SE_center
				clear_variable = global.` + upper + `_se_display
			}
		}
		properties = {
			# scripted_localisation [` + upper + `_SE_getpicture] returns the GFX sprite
			# matching global.` + upper + `_se_display value. Routed in
			# common/scripted_localisation/` + upper + `_SE_getpicture.txt (auto-emitted).
			` + upper + `_SE_picture_positioning_entry = { image = "[` + upper + `_SE_getpicture]" }
		}
	}
}
`
	write(modDir+"common/scripted_guis/"+upper+"_superevent_gui.txt", sguiText)
}

var sePlaceholderFrameNames = []string{"SE_bg.dds", "SE_frame.dds", "SE_pic.dds", "SE_btn.dds"}

var _ = strings.TrimSpace
