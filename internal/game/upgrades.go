package game

import ()

// ===== Types =====

type PassiveKind uint8

type PassiveDefinition struct {
	Key          string
	MaximumLevel int
	Apply        func(player *Player)
}

type UpgradeKind uint8

type UpgradeOffer struct {
	Kind        UpgradeKind
	Weapon      WeaponKind
	Passive     PassiveKind
	NextLevel   int
	IsEvolution bool
}

// ===== Constants =====

const (
	PassiveVitality PassiveKind = iota
	PassiveSwiftness
	PassiveMagnetism
	PassiveHaste
	PassiveMight
	PassiveReach
	PassiveLifesteal
	passiveKindCount
)

const rotatingPassiveCount = PassiveLifesteal

const (
	UpgradeWeapon UpgradeKind = iota
	UpgradePassive
	UpgradeHeal
	upgradeKindCount
)

const (
	lifestealKillHealFraction = 0.01
	offersPerLevel            = 3
	healUpgradeAmount         = 35
)

var passiveTable = [passiveKindCount]PassiveDefinition{
	PassiveVitality: {"vitality", 5, func(player *Player) {
		player.MaximumHealth += 20
		player.Health = min(player.MaximumHealth, player.Health+20)
	}},
	PassiveSwiftness: {"swiftness", 5, func(player *Player) { player.MoveSpeed *= 1.1 }},
	PassiveMagnetism: {"magnetism", 5, func(player *Player) { player.MagnetRadius *= 1.4 }},
	PassiveHaste:     {"haste", 5, func(player *Player) { player.CooldownMultiplier *= 0.92 }},
	PassiveMight:     {"might", 5, func(player *Player) { player.DamageMultiplier *= 1.12 }},
	PassiveReach:     {"reach", 5, func(player *Player) { player.AreaMultiplier *= 1.12 }},
	PassiveLifesteal: {"lifesteal", 1, func(player *Player) {
		player.KillHealFraction = lifestealKillHealFraction
	}},
}

var offerAppliers = [upgradeKindCount]func(g *Game, offer UpgradeOffer){
	UpgradeWeapon:  (*Game).applyWeaponUpgrade,
	UpgradePassive: (*Game).applyPassiveUpgrade,
	UpgradeHeal:    (*Game).applyHealUpgrade,
}

// ===== Public API =====

// ===== Internal =====

func (g *Game) nextRotatingPassive() PassiveKind {
	for step := PassiveKind(1); step <= rotatingPassiveCount; step++ {
		candidate := (g.lastRotatedPassive + step) % rotatingPassiveCount
		if g.player.PassiveLevels[candidate] < passiveTable[candidate].MaximumLevel {
			return candidate
		}
	}
	return g.lastRotatedPassive
}

func (offer UpgradeOffer) isSameChoice(other UpgradeOffer) bool {
	return offer.Kind == other.Kind && offer.Weapon == other.Weapon && offer.Passive == other.Passive
}

func (g *Game) appendLifestealOffer(pool []UpgradeOffer) []UpgradeOffer {
	if g.player.PassiveLevels[PassiveLifesteal] >= passiveTable[PassiveLifesteal].MaximumLevel {
		return pool
	}
	return append(pool, UpgradeOffer{Kind: UpgradePassive, Passive: PassiveLifesteal, NextLevel: 1})
}

func (g *Game) appendWeaponOffer(pool []UpgradeOffer, kind WeaponKind) []UpgradeOffer {
	definition := &weaponTable[kind]
	level := g.player.WeaponLevel(kind)
	familyKind, familyLevel := g.player.FamilyWeapon(definition.Family)
	isFamilyTaken := familyLevel > 0 && familyKind != kind
	isEvolution := isFamilyTaken && weaponTable[familyKind].IsFamilyBase && !definition.IsFamilyBase
	isUpgradable := level > 0 && level < definition.MaximumLevel
	isAcquirable := level == 0 && !isFamilyTaken && len(g.player.Weapons) < maximumWeaponSlots
	if !isUpgradable && !isAcquirable && !isEvolution {
		return pool
	}
	nextLevels := [2]int{level + 1, familyLevel}
	return append(pool, UpgradeOffer{Kind: UpgradeWeapon, Weapon: kind, NextLevel: nextLevels[boolToIndex(isEvolution)], IsEvolution: isEvolution})
}

func (g *Game) applyWeaponUpgrade(offer UpgradeOffer) {
	player := g.player
	for slot := range player.Weapons {
		weapon := &player.Weapons[slot]
		isEvolving := offer.IsEvolution && weaponTable[weapon.Kind].Family == weaponTable[offer.Weapon].Family
		if weapon.Kind == offer.Weapon || isEvolving {
			weapon.Kind, weapon.Level = offer.Weapon, offer.NextLevel
			g.sayIf(ChatterEvolution, offer.IsEvolution)
			return
		}
	}
	player.Weapons = append(player.Weapons, WeaponState{Kind: offer.Weapon, Level: 1})
}

func (g *Game) applyPassiveUpgrade(offer UpgradeOffer) {
	g.player.PassiveLevels[offer.Passive] = offer.NextLevel
	passiveTable[offer.Passive].Apply(g.player)
}

func (g *Game) applyHealUpgrade(UpgradeOffer) {
	g.player.Health = min(g.player.MaximumHealth, g.player.Health+healUpgradeAmount)
}
