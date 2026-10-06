package game

import (
	"strings"

	"hordefall/internal/i18n"
)

// ===== Internal =====

func translatedLabel(key string) func(game *Game) string {
	return func(*Game) string { return i18n.T(key) }
}

func translateValue(label string) string {
	return i18n.TOr("value."+strings.ToLower(label), label)
}

func actionName(action RemapAction) string {
	return i18n.T("action." + remapTable[action].Key)
}

func remapVerb(device InputDevice) string {
	return i18n.T(remapInputVerbKeys[device])
}

func (d *WeaponDefinition) DisplayName() string {
	return i18n.T("weapon." + d.Key)
}

func (d *WeaponDefinition) DisplayDescription() string {
	return i18n.T("weapon." + d.Key + ".description")
}

func (d *PassiveDefinition) DisplayName() string {
	return i18n.T("passive." + d.Key)
}

func (d *PassiveDefinition) DisplayDescription() string {
	return i18n.T("passive." + d.Key + ".description")
}
