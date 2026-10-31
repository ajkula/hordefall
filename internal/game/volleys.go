package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"github.com/ajkula/ajkurig/draw"
)

// ===== Types =====

type UpgradeLogo struct {
	Offer        UpgradeOffer
	IsRoll       bool
	FirstBonus   RollBonus
	FromX        float32
	FromY        float32
	ToX          float32
	ToY          float32
	Seconds      float32
	image        *ebiten.Image
	renderedAt   float32
	isRenderDirt bool
}

type Volley struct {
	Logos [offersPerLevel]UpgradeLogo
	Count int
}

// ===== Constants =====

const (
	maximumVolleys          = 2
	upgradeLogoRadius       = 27
	volleyDistance          = screenHeight / 3
	volleyArenaMargin       = 40
	volleyFlightSeconds     = 0.7
	volleyHopHeight         = 34
	volleyColorSeconds      = 0.6
	volleyPickupOverlap     = 2
	volleyHaloAlpha         = 0.85
	volleyBurstFlashSeconds = 0.25
	rollBonusSeconds        = 5
	nukeRadius              = screenWidth / 2
	nukeBossDamageFraction  = 0.3
	invincibilitySeconds    = 5
	logoImageMargin         = 2
)

var logoRenderers = [2]func(r *Renderer, logo *UpgradeLogo, center, radius float32){(*Renderer).renderWeaponLogo, (*Renderer).renderRollLogo}

var rollAppliers = [rollBonusCount]func(g *Game){
	RollHeal:          (*Game).applyRollHeal,
	RollNuke:          (*Game).detonateNuke,
	RollInvincibility: (*Game).grantInvincibility,
}

// ===== Public API =====

func (logo *UpgradeLogo) IsSettled() bool {
	return logo.Seconds >= volleyFlightSeconds
}

func (logo *UpgradeLogo) Position() (x, y, hop float32) {
	progress := clamp(logo.Seconds/volleyFlightSeconds, 0, 1)
	glide := progress * (2 - progress)
	x = logo.FromX + (logo.ToX-logo.FromX)*glide
	y = logo.FromY + (logo.ToY-logo.FromY)*glide
	return x, y, volleyHopHeight * (1 - progress) * abs(sine(progress*fullTurn))
}

func (logo *UpgradeLogo) RollBonus() (bonus RollBonus, remaining float32) {
	cycles := logo.Seconds / rollBonusSeconds
	completed := int(cycles)
	return RollBonus((int(logo.FirstBonus) + completed) % int(rollBonusCount)), 1 - (cycles - float32(completed))
}

// ===== Internal =====

func (g *Game) resolveLevelUps() {
	for ; g.player.PendingLevelUps > 0; g.player.PendingLevelUps-- {
		g.grantNextPassive()
		g.pendingVolleys++
	}
	for g.volleyCount < maximumVolleys && g.pendingVolleys > 0 {
		g.pendingVolleys--
		g.burstVolley()
	}
}

func (g *Game) grantNextPassive() {
	kind := g.nextRotatingPassive()
	player := g.player
	hasPassiveLeft := player.PassiveLevels[kind] < passiveTable[kind].MaximumLevel
	if !hasPassiveLeft {
		return
	}
	g.lastRotatedPassive = kind
	g.applyPassiveUpgrade(UpgradeOffer{Kind: UpgradePassive, Passive: kind, NextLevel: player.PassiveLevels[kind] + 1})
	bannerPrefix := [2]string{"LEVEL UP! ", g.levelUpBanner}[boolToIndex(g.levelUpBannerSeconds > 0)]
	g.levelUpBanner = bannerPrefix + "  + " + passiveTable[kind].DisplayName()
	g.levelUpBannerSeconds = levelUpBannerDuration
}

func (g *Game) burstVolley() {
	volley := &g.volleys[g.volleyCount]
	g.volleyCount++
	*volley = Volley{}
	pool := g.weaponOfferPool()
	for volley.Count < offersPerLevel && len(pool) > 0 {
		pick := g.random.Below(len(pool))
		volley.Logos[volley.Count] = UpgradeLogo{Offer: pool[pick]}
		volley.Count++
		pool[pick] = pool[len(pool)-1]
		pool = pool[:len(pool)-1]
	}
	g.fillRollIfEmpty(volley)
	g.launchLogos(volley)
	g.player.LevelFlashSeconds = volleyBurstFlashSeconds
	g.playSound(SoundLevelUp)
	g.say(ChatterLevelUp)
}

func (g *Game) fillRollIfEmpty(volley *Volley) {
	if volley.Count > 0 {
		return
	}
	volley.Logos[0] = UpgradeLogo{IsRoll: true, FirstBonus: RollBonus(g.random.Below(int(rollBonusCount)))}
	volley.Count = 1
}

func (g *Game) launchLogos(volley *Volley) {
	player := g.player
	firstAngle := g.random.Angle()
	for index := range volley.Count {
		angle := firstAngle + float32(index)*fullTurn/offersPerLevel
		logo := &volley.Logos[index]
		logo.FromX, logo.FromY = player.X, player.Y
		logo.ToX = clamp(player.X+cosine(angle)*volleyDistance, volleyArenaMargin, arenaSize-volleyArenaMargin)
		logo.ToY = clamp(player.Y+sine(angle)*volleyDistance, volleyArenaMargin, arenaSize-volleyArenaMargin)
		logo.isRenderDirt = true
	}
}

func (g *Game) weaponOfferPool() []UpgradeOffer {
	g.offerPool = g.offerPool[:0]
	for kind := range weaponKindCount {
		g.offerPool = g.appendWeaponOffer(g.offerPool, kind)
	}
	return g.offerPool
}

func (g *Game) updateVolleys(deltaSeconds float32) {
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		for logo := range volley.Count {
			volley.Logos[logo].Seconds += deltaSeconds
		}
	}
	for index := range g.volleyCount {
		if g.pickUpFromVolley(index) {
			return
		}
	}
}

func (g *Game) pickUpFromVolley(index int) bool {
	volley := &g.volleys[index]
	for slot := range volley.Count {
		logo := &volley.Logos[slot]
		if !g.isTouchingLogo(logo) {
			continue
		}
		g.applyLogo(logo)
		g.removeVolley(index)
		g.refreshVolleys()
		g.resolveLevelUps()
		return true
	}
	return false
}

func (g *Game) isTouchingLogo(logo *UpgradeLogo) bool {
	reach := float32(playerRadius + upgradeLogoRadius - volleyPickupOverlap)
	return logo.IsSettled() && distanceSquared(logo.ToX, logo.ToY, g.player.X, g.player.Y) < reach*reach
}

func (g *Game) applyLogo(logo *UpgradeLogo) {
	g.effects.AddRing(logo.ToX, logo.ToY, upgradeLogoRadius*2, logoColorOf(logo))
	if logo.IsRoll {
		bonus, _ := logo.RollBonus()
		rollAppliers[bonus](g)
		return
	}
	g.applyWeaponUpgrade(logo.Offer)
}

func (g *Game) removeVolley(index int) {
	g.volleyCount--
	g.volleys[index], g.volleys[g.volleyCount] = g.volleys[g.volleyCount], g.volleys[index]
}

func (g *Game) refreshVolleys() {
	for index := g.volleyCount - 1; index >= 0; index-- {
		g.refreshVolley(&g.volleys[index])
		if g.volleys[index].Count == 0 {
			g.removeVolley(index)
		}
	}
}

func (g *Game) refreshVolley(volley *Volley) {
	pool := g.weaponOfferPool()
	for slot := volley.Count - 1; slot >= 0; slot-- {
		logo := &volley.Logos[slot]
		if logo.IsRoll {
			continue
		}
		g.refreshLogo(volley, slot, pool)
	}
}

func (g *Game) refreshLogo(volley *Volley, slot int, pool []UpgradeOffer) {
	logo := &volley.Logos[slot]
	current, isStillOffered := findOffer(pool, logo.Offer.Weapon)
	replacement, hasReplacement := g.unusedOffer(pool)
	logo.isRenderDirt = true
	if isStillOffered {
		logo.Offer = current
		return
	}
	if hasReplacement {
		logo.Offer = replacement
		g.effects.SpawnSparks(logo.ToX, logo.ToY, 10, logoColorOf(logo), 120)
		return
	}
	g.effects.SpawnSparks(logo.ToX, logo.ToY, 16, logoColorOf(logo), 200)
	g.effects.AddShake(2)
	volley.Count--
	volley.Logos[slot], volley.Logos[volley.Count] = volley.Logos[volley.Count], volley.Logos[slot]
}

func findOffer(pool []UpgradeOffer, kind WeaponKind) (UpgradeOffer, bool) {
	for _, offer := range pool {
		if offer.Weapon == kind {
			return offer, true
		}
	}
	return UpgradeOffer{}, false
}

func (g *Game) unusedOffer(pool []UpgradeOffer) (UpgradeOffer, bool) {
	for _, offer := range pool {
		if !g.isOfferOnGround(offer.Weapon) {
			return offer, true
		}
	}
	return UpgradeOffer{}, false
}

func (g *Game) isOfferOnGround(kind WeaponKind) bool {
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		for slot := range volley.Count {
			logo := &volley.Logos[slot]
			if !logo.IsRoll && logo.Offer.Weapon == kind {
				return true
			}
		}
	}
	return false
}

func (g *Game) applyRollHeal() {
	g.applyHealUpgrade(UpgradeOffer{Kind: UpgradeHeal})
}

func (g *Game) detonateNuke() {
	player := g.player
	enemies := g.enemies
	for index := range enemies.Count {
		isInside := distanceSquared(enemies.PositionX[index], enemies.PositionY[index], player.X, player.Y) < nukeRadius*nukeRadius
		if !isInside {
			continue
		}
		kind := enemies.Kind[index]
		isBoss := kind >= EnemySpiderTank
		bossDamage := nukeBossDamageFraction * enemyTable[kind].Health * enemies.Power[index]
		g.HitEnemy(index, bossDamage+(enemies.Health[index]-bossDamage)*boolToFloat(!isBoss), ElementNone)
	}
	g.effects.AddRing(player.X, player.Y, nukeRadius, logoNukeColor)
	g.effects.AddShake(14)
	g.effects.SpawnSparks(player.X, player.Y, 40, logoNukeColor, 420)
	g.playSound(SoundExplosion)
}

func (g *Game) grantInvincibility() {
	player := g.player
	player.InvincibleSeconds = invincibilitySeconds
	player.InvulnerableSeconds = max(player.InvulnerableSeconds, invincibilitySeconds)
}

func logoColorOf(logo *UpgradeLogo) [3]float32 {
	if logo.IsRoll {
		bonus, _ := logo.RollBonus()
		return rollColors[bonus]
	}
	return liftedLogoColor(weaponTable[logo.Offer.Weapon].Color)
}

func (r *Renderer) drawUpgradeLogos(g *Game, screen *ebiten.Image) {
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		for slot := range volley.Count {
			r.drawUpgradeLogo(screen, &volley.Logos[slot])
		}
	}
}

func (r *Renderer) drawUpgradeLogo(screen *ebiten.Image, logo *UpgradeLogo) {
	r.renderLogoIfNeeded(logo)
	x, y, hop := logo.Position()
	screenX, screenY := r.camera.ToScreen(x, y-hop)
	whiteness := clamp(1-(logo.Seconds-volleyFlightSeconds)/volleyColorSeconds, 0, 1)
	size := float64(logo.image.Bounds().Dx())
	var colorMatrix colorm.ColorM
	colorMatrix.Scale(float64(1-whiteness), float64(1-whiteness), float64(1-whiteness), 1)
	colorMatrix.Translate(float64(whiteness), float64(whiteness), float64(whiteness), 0)
	options := &colorm.DrawImageOptions{Filter: ebiten.FilterLinear}
	options.GeoM.Translate(-size/2, -size/2)
	options.GeoM.Translate(float64(screenX*draw.Scale), float64(screenY*draw.Scale))
	colorm.DrawImage(screen, logo.image, colorMatrix, options)
	r.batches.Glow.AddCircle(screenX, screenY, upgradeLogoRadius, logoColorOf(logo), volleyHaloAlpha*(1-whiteness))
}

func (r *Renderer) renderLogoIfNeeded(logo *UpgradeLogo) {
	isRescaled := logo.renderedAt != draw.Scale
	if !logo.isRenderDirt && !isRescaled && !logo.IsRoll {
		return
	}
	radius := upgradeLogoRadius * draw.Scale
	size := int(2*radius) + 2*logoImageMargin
	if logo.image == nil || isRescaled {
		logo.image = ebiten.NewImage(size, size)
	}
	logo.image.Clear()
	center := float32(size) / 2
	logoRenderers[boolToIndex(logo.IsRoll)](r, logo, center, radius)
	logo.renderedAt, logo.isRenderDirt = draw.Scale, false
}

func (r *Renderer) renderWeaponLogo(logo *UpgradeLogo, center, radius float32) {
	DrawWeaponLogo(logo.image, center, center, radius, logo.Offer.Weapon)
	DrawLogoBadge(logo.image, center, center, radius, logo.Offer, r.badgeFace)
}

func (r *Renderer) renderRollLogo(logo *UpgradeLogo, center, radius float32) {
	bonus, remaining := logo.RollBonus()
	DrawRollLogo(logo.image, center, center, radius, bonus, remaining)
}
