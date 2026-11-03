package game

import (
	"image/color"

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
	Logos       [offersPerLevel]UpgradeLogo
	Count       int
	HasBeenLost bool
}

// ===== Constants =====

const (
	maximumVolleys          = 2
	upgradeLogoRadius       = 27
	volleyDistance          = screenHeight / 3
	volleyArenaMargin       = 40
	volleyFlightSeconds     = 0.7
	volleyHopHeight         = 34
	volleyColorSeconds      = 0.3
	volleyPickupOverlap     = 2
	volleyHaloAlpha         = 0.4
	volleyBurstFlashSeconds = 0.25
	rollBonusSeconds        = 5
	logoImageMargin         = 6
	logoOutlineWidth        = 1.2
	logoOutlineAlphaGain    = 2
	beaconPeriodSeconds     = 1.6
	beaconRingCount         = 1
	beaconStartRadius       = upgradeLogoRadius
	beaconEndRadius         = 61
	beaconAlpha             = 0.9
	beaconStrokeWidth       = 10
	beaconSegments          = 72
	beaconEdgeSoftness      = 1
	beaconPixelSize         = 3
	pointerEdgeMargin       = 30
	pointerLength           = 22
	pointerHalfWidth        = 13
	pointerRimWidth         = 3
)

var (
	beaconLayerOffsets = [...]float32{-1, -1, 1, 1}
	beaconLayerInset   = [...]float32{0, -1, 1, 0}
	beaconLayerAlphas  = [...]float32{0, 1, 1, 0}
	pointerColor       = [3]float32{0.992, 0.961, 0}
	pointerRimColor    = [3]float32{0.443, 0, 0}
)

var logoRenderers = [2]func(r *Renderer, target *ebiten.Image, logo *UpgradeLogo, center, radius float32){(*Renderer).renderWeaponLogo, (*Renderer).renderRollLogo}

var rollAppliers = [rollBonusCount]func(g *Game){
	RollHeal: (*Game).applyRollHeal,
	RollNuke: (*Game).detonateNuke,
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
	for g.volleyCount < maximumVolleys && g.pendingVolleys > 0 && !g.isNextVolleyARollInWaiting() {
		g.pendingVolleys--
		g.burstVolley()
	}
}

func (g *Game) isNextVolleyARollInWaiting() bool {
	return len(g.upgradeOfferPool()) == 0 && g.hasRollOnGround()
}

func (g *Game) hasRollOnGround() bool {
	for index := range g.volleyCount {
		if g.volleys[index].Logos[0].IsRoll {
			return true
		}
	}
	return false
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
	pool := g.upgradeOfferPool()
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

func (g *Game) upgradeOfferPool() []UpgradeOffer {
	g.offerPool = g.offerPool[:0]
	for kind := range weaponKindCount {
		g.offerPool = g.appendWeaponOffer(g.offerPool, kind)
	}
	g.offerPool = g.appendLifestealOffer(g.offerPool)
	return g.offerPool
}

func (g *Game) updateVolleys(deltaSeconds float32) {
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		volley.HasBeenLost = volley.HasBeenLost || !g.renderer.isVolleyInSight(volley)
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
	g.playSound(SoundLogoPickup)
	if logo.IsRoll {
		bonus, _ := logo.RollBonus()
		rollAppliers[bonus](g)
		return
	}
	offerAppliers[logo.Offer.Kind](g, logo.Offer)
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
	pool := g.upgradeOfferPool()
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
	current, isStillOffered := findOffer(pool, logo.Offer)
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

func findOffer(pool []UpgradeOffer, wanted UpgradeOffer) (UpgradeOffer, bool) {
	for _, offer := range pool {
		if offer.isSameChoice(wanted) {
			return offer, true
		}
	}
	return UpgradeOffer{}, false
}

func (g *Game) unusedOffer(pool []UpgradeOffer) (UpgradeOffer, bool) {
	for _, offer := range pool {
		if !g.isOfferOnGround(offer) {
			return offer, true
		}
	}
	return UpgradeOffer{}, false
}

func (g *Game) isOfferOnGround(offer UpgradeOffer) bool {
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		for slot := range volley.Count {
			logo := &volley.Logos[slot]
			if !logo.IsRoll && logo.Offer.isSameChoice(offer) {
				return true
			}
		}
	}
	return false
}

func (g *Game) applyRollHeal() {
	g.applyHealUpgrade(UpgradeOffer{Kind: UpgradeHeal})
}

func logoColorOf(logo *UpgradeLogo) [3]float32 {
	if logo.IsRoll {
		bonus, _ := logo.RollBonus()
		return rollColors[bonus]
	}
	return OfferLogoColor(logo.Offer)
}

func (r *Renderer) isVolleyInSight(volley *Volley) bool {
	for slot := range volley.Count {
		screenX, screenY := r.camera.ToScreen(volley.Logos[slot].ToX, volley.Logos[slot].ToY)
		if screenX >= 0 && screenY >= 0 && screenX <= screenWidth && screenY <= screenHeight {
			return true
		}
	}
	return false
}

func (r *Renderer) drawUpgradeLogos(g *Game, screen *ebiten.Image) {
	r.drawLogoBeacons(g, screen)
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		for slot := range volley.Count {
			r.drawUpgradeLogo(screen, &volley.Logos[slot])
		}
		r.drawVolleyPointerIf(g, screen, volley, !r.isVolleyInSight(volley))
	}
}

func (r *Renderer) drawVolleyPointerIf(g *Game, screen *ebiten.Image, volley *Volley, shouldDraw bool) {
	if !shouldDraw {
		return
	}
	targetX, targetY := nearestLogoTarget(volley, g.player.X, g.player.Y)
	directionX, directionY := normalize(targetX-g.player.X, targetY-g.player.Y)
	screenX, screenY := r.camera.ToScreen(targetX, targetY)
	tipX := clamp(screenX, pointerEdgeMargin, screenWidth-pointerEdgeMargin)
	tipY := clamp(screenY, pointerEdgeMargin, screenHeight-pointerEdgeMargin)
	draw.FillTriangle(screen, pointerTriangle(tipX, tipY, directionX, directionY, pointerRimWidth), pointerRimColor, 1)
	draw.FillTriangle(screen, pointerTriangle(tipX, tipY, directionX, directionY, 0), pointerColor, 1)
}

func nearestLogoTarget(volley *Volley, x, y float32) (float32, float32) {
	nearest := &volley.Logos[0]
	for slot := 1; slot < volley.Count; slot++ {
		logo := &volley.Logos[slot]
		if distanceSquared(logo.ToX, logo.ToY, x, y) < distanceSquared(nearest.ToX, nearest.ToY, x, y) {
			nearest = logo
		}
	}
	return nearest.ToX, nearest.ToY
}

func pointerTriangle(tipX, tipY, directionX, directionY, growth float32) [3][2]float32 {
	length := float32(pointerLength + 2*growth)
	halfWidth := float32(pointerHalfWidth + growth)
	baseX, baseY := tipX-directionX*(length-growth), tipY-directionY*(length-growth)
	return [3][2]float32{
		{tipX + directionX*growth, tipY + directionY*growth},
		{baseX - directionY*halfWidth, baseY + directionX*halfWidth},
		{baseX + directionY*halfWidth, baseY - directionX*halfWidth},
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
}

func (r *Renderer) queueUpgradeHalos(g *Game) {
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		for slot := range volley.Count {
			r.queueUpgradeHalo(&volley.Logos[slot])
		}
	}
}

func (r *Renderer) queueUpgradeHalo(logo *UpgradeLogo) {
	x, y, hop := logo.Position()
	screenX, screenY := r.camera.ToScreen(x, y-hop)
	colorProgress := clamp((logo.Seconds-volleyFlightSeconds)/volleyColorSeconds, 0, 1)
	r.batches.Glow.AddCircle(screenX, screenY, upgradeLogoRadius, logoColorOf(logo), volleyHaloAlpha*colorProgress)
}

func (r *Renderer) drawLogoBeacons(g *Game, screen *ebiten.Image) {
	r.beaconVertices, r.beaconIndices = r.beaconVertices[:0], r.beaconIndices[:0]
	for index := range g.volleyCount {
		volley := &g.volleys[index]
		for slot := range volley.Count * boolToIndex(volley.HasBeenLost) {
			r.appendLogoBeacon(&volley.Logos[slot])
		}
	}
	if len(r.beaconIndices) == 0 {
		return
	}
	if r.beaconPixel == nil {
		r.beaconPixel = ebiten.NewImage(beaconPixelSize, beaconPixelSize)
		r.beaconPixel.Fill(color.White)
	}
	screen.DrawTriangles32(r.beaconVertices, r.beaconIndices, r.beaconPixel, &ebiten.DrawTrianglesOptions{})
}

func (r *Renderer) appendLogoBeacon(logo *UpgradeLogo) {
	screenX, screenY := r.camera.ToScreen(logo.ToX, logo.ToY)
	for ring := range beaconRingCount {
		cycles := logo.Seconds/beaconPeriodSeconds + float32(ring)/beaconRingCount
		progress := cycles - float32(int(cycles))
		radius := beaconStartRadius + (beaconEndRadius-beaconStartRadius)*progress
		r.appendBeaconRing(screenX*draw.Scale, screenY*draw.Scale, radius*draw.Scale, beaconAlpha*(1-progress))
	}
}

func (r *Renderer) appendBeaconRing(x, y, radius, alpha float32) {
	base := uint32(len(r.beaconVertices))
	halfWidth := beaconStrokeWidth * draw.Scale / 2
	for segment := range beaconSegments {
		angle := float32(segment) * fullTurn / beaconSegments
		directionX, directionY := cosine(angle), sine(angle)
		for layer, offset := range beaconLayerOffsets {
			reach := radius + offset*(halfWidth+beaconEdgeSoftness) - beaconLayerInset[layer]*beaconEdgeSoftness
			r.beaconVertices = append(r.beaconVertices, beaconVertex(x+directionX*reach, y+directionY*reach, alpha*beaconLayerAlphas[layer]))
		}
		r.appendBeaconBands(base, uint32(segment))
	}
}

func beaconVertex(x, y, alpha float32) ebiten.Vertex {
	return ebiten.Vertex{DstX: x, DstY: y, SrcX: beaconPixelSize / 2, SrcY: beaconPixelSize / 2, ColorR: 1, ColorA: alpha}
}

func (r *Renderer) appendBeaconBands(base, segment uint32) {
	layers := uint32(len(beaconLayerOffsets))
	current := base + segment*layers
	next := base + (segment+1)%beaconSegments*layers
	for band := range layers - 1 {
		r.beaconIndices = append(r.beaconIndices, current+band, next+band, current+band+1, next+band, next+band+1, current+band+1)
	}
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
	art := r.logoArtCanvas(size)
	art.Clear()
	center := float32(size) / 2
	logoRenderers[boolToIndex(logo.IsRoll)](r, art, logo, center, radius)
	logo.image.Clear()
	drawOutlinedLogoArt(logo.image, art)
	logo.renderedAt, logo.isRenderDirt = draw.Scale, false
}

func (r *Renderer) logoArtCanvas(size int) *ebiten.Image {
	if r.logoArt == nil || r.logoArt.Bounds().Dx() != size {
		r.logoArt = ebiten.NewImage(size, size)
	}
	return r.logoArt
}

func drawOutlinedLogoArt(target, art *ebiten.Image) {
	var silhouette colorm.ColorM
	silhouette.Scale(0, 0, 0, logoOutlineAlphaGain)
	silhouette.Translate(float64(logoBackground[0]), float64(logoBackground[1]), float64(logoBackground[2]), 1-logoOutlineAlphaGain)
	width := float64(logoOutlineWidth * draw.Scale)
	for _, offset := range badgeOutlineOffsets {
		options := &colorm.DrawImageOptions{}
		options.GeoM.Translate(float64(offset[0])*width, float64(offset[1])*width)
		colorm.DrawImage(target, art, silhouette, options)
	}
	target.DrawImage(art, nil)
}

func (r *Renderer) renderWeaponLogo(target *ebiten.Image, logo *UpgradeLogo, center, radius float32) {
	DrawOfferLogo(target, center, center, radius, logo.Offer)
	DrawLogoBadge(target, center, center, radius, logo.Offer, r.badgeFace)
}

func (r *Renderer) renderRollLogo(target *ebiten.Image, logo *UpgradeLogo, center, radius float32) {
	bonus, remaining := logo.RollBonus()
	DrawRollLogo(target, center, center, radius, bonus, remaining)
}
