package game

import "github.com/ajkula/ajkurig/i18n"

// ===== Internal =====

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
