package game

import "fmt"

// ===== Types =====

type PassiveKind uint8

type PassiveDefinition struct {
	Name         string
	Description  string
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

type upgradeApplier func(game *Game, offer UpgradeOffer)

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
	PassiveVitality: {"Vitality", "+20 max health and heal 20.", 5, func(player *Player) {
		player.MaximumHealth += 20
		player.Health = min(player.MaximumHealth, player.Health+20)
	}},
	PassiveSwiftness: {"Swiftness", "+10% move speed.", 5, func(player *Player) { player.MoveSpeed *= 1.1 }},
	PassiveMagnetism: {"Magnetism", "+40% pickup radius.", 5, func(player *Player) { player.MagnetRadius *= 1.4 }},
	PassiveHaste:     {"Haste", "-8% weapon cooldowns.", 5, func(player *Player) { player.CooldownMultiplier *= 0.92 }},
	PassiveMight:     {"Might", "+12% damage, reactions included.", 5, func(player *Player) { player.DamageMultiplier *= 1.12 }},
	PassiveReach:     {"Reach", "+12% area for novas, rain, flasks, blades, slams, mines.", 5, func(player *Player) { player.AreaMultiplier *= 1.12 }},
	PassiveLifesteal: {"Lifesteal", "Each kill restores 1% of max health. One time only.", 1, func(player *Player) {
		player.KillHealFraction = lifestealKillHealFraction
	}},
}

var offerDrawers = [2]func(game *Game){(*Game).drawRandomOffers, (*Game).drawRotatingPassiveOffer}

var upgradeAppliers = [upgradeKindCount]upgradeApplier{
	UpgradeWeapon:  (*Game).applyWeaponUpgrade,
	UpgradePassive: (*Game).applyPassiveUpgrade,
	UpgradeHeal:    (*Game).applyHealUpgrade,
}

// ===== Public API =====

func (offer UpgradeOffer) Title() string {
	titles := [upgradeKindCount]string{
		UpgradeWeapon:  weaponTable[offer.Weapon].Name,
		UpgradePassive: passiveTable[offer.Passive].Name,
		UpgradeHeal:    "Second Wind",
	}
	return titles[offer.Kind]
}

func (offer UpgradeOffer) Subtitle() string {
	isNew := offer.NextLevel == 1 && offer.Kind != UpgradeHeal && !offer.IsEvolution
	levelLabels := [2]string{fmt.Sprintf("Level %d", offer.NextLevel), "NEW"}
	weaponLabels := [2]string{levelLabels[boolToIndex(isNew)], "EVOLVE YOUR SHOT"}
	kindLabels := [2]string{weaponLabels[boolToIndex(offer.IsEvolution)], fmt.Sprintf("+%d health", healUpgradeAmount)}
	return kindLabels[boolToIndex(offer.Kind == UpgradeHeal)]
}

func (offer UpgradeOffer) Description() string {
	descriptions := [upgradeKindCount]string{
		UpgradeWeapon:  weaponTable[offer.Weapon].Description,
		UpgradePassive: passiveTable[offer.Passive].Description,
		UpgradeHeal:    fmt.Sprintf("Restore %d health.", healUpgradeAmount),
	}
	return descriptions[offer.Kind]
}

func (offer UpgradeOffer) Color() [3]float32 {
	colors := [upgradeKindCount][3]float32{
		UpgradeWeapon:  weaponTable[offer.Weapon].Color,
		UpgradePassive: {0.75, 0.75, 0.85},
		UpgradeHeal:    {0.4, 1, 0.5},
	}
	return colors[offer.Kind]
}

// ===== Internal =====

func (g *Game) buildUpgradeOffers() {
	g.offerPool = g.offerPool[:0]
	for kind := range weaponKindCount {
		g.offerPool = g.appendWeaponOffer(g.offerPool, kind)
	}
	isOnlyPassivesLeft := len(g.offerPool) == 0
	for kind := range passiveKindCount {
		g.offerPool = g.appendPassiveOffer(g.offerPool, kind)
	}
	g.offers = g.offers[:0]
	offerDrawers[boolToIndex(isOnlyPassivesLeft)](g)
	g.offers = appendHealIfEmpty(g.offers)
	g.selectedOffer = 0
}

func (g *Game) drawRandomOffers() {
	for len(g.offers) < offersPerLevel && len(g.offerPool) > 0 {
		pick := g.random.Below(len(g.offerPool))
		g.offers = append(g.offers, g.offerPool[pick])
		g.offerPool[pick] = g.offerPool[len(g.offerPool)-1]
		g.offerPool = g.offerPool[:len(g.offerPool)-1]
	}
}

func (g *Game) drawRotatingPassiveOffer() {
	if len(g.offerPool) == 0 {
		return
	}
	kind := g.nextRotatingPassive()
	g.lastRotatedPassive = kind
	g.offers = append(g.offers, UpgradeOffer{Kind: UpgradePassive, Passive: kind, NextLevel: g.player.PassiveLevels[kind] + 1})
}

func (g *Game) nextRotatingPassive() PassiveKind {
	for step := PassiveKind(1); step <= passiveKindCount; step++ {
		candidate := (g.lastRotatedPassive + step) % passiveKindCount
		if g.player.PassiveLevels[candidate] < passiveTable[candidate].MaximumLevel {
			return candidate
		}
	}
	return g.lastRotatedPassive
}

func appendHealIfEmpty(offers []UpgradeOffer) []UpgradeOffer {
	if len(offers) > 0 {
		return offers
	}
	return append(offers, UpgradeOffer{Kind: UpgradeHeal})
}

func (g *Game) appendWeaponOffer(pool []UpgradeOffer, kind WeaponKind) []UpgradeOffer {
	definition := &weaponTable[kind]
	level := g.player.WeaponLevel(kind)
	familyKind, familyLevel := g.player.FamilyWeapon(definition.Family)
	isFamilyTaken := familyLevel > 0 && familyKind != kind
	isEvolution := isFamilyTaken && weaponTable[familyKind].IsFamilyBase && !definition.IsFamilyBase
	isUpgradable := level > 0 && level < maximumWeaponLevel
	isAcquirable := level == 0 && !isFamilyTaken && len(g.player.Weapons) < maximumWeaponSlots
	if !isUpgradable && !isAcquirable && !isEvolution {
		return pool
	}
	nextLevels := [2]int{level + 1, familyLevel}
	return append(pool, UpgradeOffer{Kind: UpgradeWeapon, Weapon: kind, NextLevel: nextLevels[boolToIndex(isEvolution)], IsEvolution: isEvolution})
}

func (g *Game) appendPassiveOffer(pool []UpgradeOffer, kind PassiveKind) []UpgradeOffer {
	level := g.player.PassiveLevels[kind]
	if level >= passiveTable[kind].MaximumLevel {
		return pool
	}
	return append(pool, UpgradeOffer{Kind: UpgradePassive, Passive: kind, NextLevel: level + 1})
}

func (g *Game) applyOffer(offer UpgradeOffer) {
	upgradeAppliers[offer.Kind](g, offer)
	g.player.PendingLevelUps--
}

func (g *Game) applyWeaponUpgrade(offer UpgradeOffer) {
	player := g.player
	for slot := range player.Weapons {
		weapon := &player.Weapons[slot]
		isEvolving := offer.IsEvolution && weaponTable[weapon.Kind].Family == weaponTable[offer.Weapon].Family
		if weapon.Kind == offer.Weapon || isEvolving {
			weapon.Kind, weapon.Level = offer.Weapon, offer.NextLevel
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
