package main

// ===== Types =====

type Player struct {
	X                   float32
	Y                   float32
	FacingX             float32
	FacingY             float32
	VelocityX           float32
	VelocityY           float32
	Rig                 TachikomaRig
	AimX                float32
	AimY                float32
	IsAimLocked         bool
	IsFiring            bool
	Health              float32
	MaximumHealth       float32
	Level               int
	Experience          int
	ExperienceToNext    int
	PendingLevelUps     int
	InvulnerableSeconds float32
	DamageFlashSeconds  float32
	DashSeconds         float32
	DashCooldown        float32
	DashDirectionX      float32
	DashDirectionY      float32
	MoveSpeed           float32
	MagnetRadius        float32
	CooldownMultiplier  float32
	DamageMultiplier    float32
	AreaMultiplier      float32
	KillHealFraction    float32
	KillHealBudget      float32
	Weapons             []WeaponState
	PassiveLevels       [passiveKindCount]int
	BladeCount          int
	BladeRadius         float32
	BladeAngle          float32
}

// ===== Constants =====

const (
	playerRadius               = 11
	playerStartHealth          = 100
	playerStartSpeed           = 175
	playerStartMagnet          = 85
	dashDurationSeconds        = 0.18
	dashCooldownSeconds        = 1.1
	dashSpeedMultiplier        = 3.6
	hurtInvulnerabilitySeconds = 0.55
	groundFireDamagePerSecond  = 14
	maximumKillHealPerSecond   = 0.12
)

var groundPlayerSpeed = [groundKindCount]float32{
	GroundGrass: 1, GroundDirt: 1, GroundAsh: 1, GroundOil: 0.85, GroundFire: 1, GroundIce: 1.2, GroundWater: 0.7,
}

// ===== Public API =====

func NewPlayer() *Player {
	return &Player{
		X: arenaSize / 2, Y: arenaSize / 2, FacingX: 1, AimX: 1,
		Health: playerStartHealth, MaximumHealth: playerStartHealth,
		Level: 1, ExperienceToNext: experienceForLevel(1),
		MoveSpeed: playerStartSpeed, MagnetRadius: playerStartMagnet,
		CooldownMultiplier: 1, DamageMultiplier: 1, AreaMultiplier: 1,
		Weapons: []WeaponState{{Kind: WeaponEmberBolt, Level: 1}},
	}
}

func (p *Player) BladePosition(blade int) (float32, float32) {
	angle := p.BladeAngle + float32(blade)/float32(max(1, p.BladeCount))*2*3.14159265
	return p.X + cosine(angle)*p.BladeRadius, p.Y + sine(angle)*p.BladeRadius
}

func (p *Player) WeaponLevel(kind WeaponKind) int {
	for _, weapon := range p.Weapons {
		if weapon.Kind == kind {
			return weapon.Level
		}
	}
	return 0
}

func (p *Player) GainExperience(amount int) {
	p.Experience += amount
	for p.Experience >= p.ExperienceToNext {
		p.Experience -= p.ExperienceToNext
		p.Level++
		p.PendingLevelUps++
		p.ExperienceToNext = experienceForLevel(p.Level)
	}
}

func (g *Game) DamagePlayer(amount float32) {
	player := g.player
	if player.InvulnerableSeconds > 0 || amount <= 0 {
		return
	}
	player.Health -= amount
	player.InvulnerableSeconds = hurtInvulnerabilitySeconds
	player.DamageFlashSeconds = 0.25
	g.effects.AddShake(4)
	g.playSound(SoundHurt)
	g.effects.SpawnSparks(player.X, player.Y, 8, [3]float32{1, 0.2, 0.2}, 160)
}

func (g *Game) HealFromKill() {
	player := g.player
	healing := min(player.KillHealFraction*player.MaximumHealth, player.KillHealBudget, player.MaximumHealth-player.Health)
	if healing <= 0 {
		return
	}
	player.Health += healing
	player.KillHealBudget -= healing
	g.effects.SpawnTrail(player.X, player.Y, [3]float32{0.35, 1, 0.45})
}

// ===== Internal =====

func experienceForLevel(level int) int {
	return 8 + 7*level + level*level/2
}

func (g *Game) updatePlayer(deltaSeconds float32) {
	player := g.player
	player.InvulnerableSeconds = max(0, player.InvulnerableSeconds-deltaSeconds)
	player.DamageFlashSeconds = max(0, player.DamageFlashSeconds-deltaSeconds)
	player.DashCooldown = max(0, player.DashCooldown-deltaSeconds)
	healBudgetCap := maximumKillHealPerSecond * player.MaximumHealth
	player.KillHealBudget = min(healBudgetCap, player.KillHealBudget+healBudgetCap*deltaSeconds)
	player.IsFiring = g.controls.Held&ActionFire != 0 || g.controls.HasAimStick
	player.IsAimLocked = g.controls.Held&ActionAimLock != 0
	g.startDashIfRequested()
	isDashing := player.DashSeconds > 0
	player.DashSeconds = max(0, player.DashSeconds-deltaSeconds)
	g.updateFacing()
	g.applyAimStick()
	dashWeight := boolToFloat(isDashing)
	directionX := g.controls.MoveX + (player.DashDirectionX-g.controls.MoveX)*dashWeight
	directionY := g.controls.MoveY + (player.DashDirectionY-g.controls.MoveY)*dashWeight
	groundKind := g.ground.KindAt(player.X, player.Y)
	speed := player.MoveSpeed * groundPlayerSpeed[groundKind] * (1 + (dashSpeedMultiplier-1)*dashWeight)
	previousX, previousY := player.X, player.Y
	player.X = clamp(player.X+directionX*speed*deltaSeconds, playerRadius, arenaSize-playerRadius)
	player.Y = clamp(player.Y+directionY*speed*deltaSeconds, playerRadius, arenaSize-playerRadius)
	player.VelocityX, player.VelocityY = (player.X-previousX)/deltaSeconds, (player.Y-previousY)/deltaSeconds
	player.Health -= groundFireDamagePerSecond * deltaSeconds * boolToFloat(groundKind == GroundFire && !isDashing)
	g.spawnDashTrail(isDashing)
	collected := g.gems.Collect(player.X, player.Y, player.MagnetRadius, deltaSeconds)
	player.GainExperience(collected)
	g.playSoundIf(SoundGem, collected > 0)
	g.updateTachikoma(deltaSeconds)
}

func (g *Game) updateFacing() {
	isMoving := g.controls.MoveX != 0 || g.controls.MoveY != 0
	if !isMoving {
		return
	}
	player := g.player
	player.FacingX, player.FacingY = normalize(g.controls.MoveX, g.controls.MoveY)
	keepWeight := boolToFloat(player.IsAimLocked)
	player.AimX += (player.FacingX - player.AimX) * (1 - keepWeight)
	player.AimY += (player.FacingY - player.AimY) * (1 - keepWeight)
}

func (g *Game) applyAimStick() {
	if !g.controls.HasAimStick {
		return
	}
	g.player.AimX, g.player.AimY = g.controls.AimX, g.controls.AimY
}

func (g *Game) startDashIfRequested() {
	player := g.player
	canDash := g.controls.JustPressed&ActionDash != 0 && player.DashCooldown == 0
	if !canDash {
		return
	}
	player.DashSeconds = dashDurationSeconds
	player.DashCooldown = dashCooldownSeconds
	player.InvulnerableSeconds = max(player.InvulnerableSeconds, dashDurationSeconds+0.08)
	player.DashDirectionX, player.DashDirectionY = player.FacingX, player.FacingY
	g.effects.AddRing(player.X, player.Y, 26, [3]float32{0.7, 0.95, 1})
	g.playSound(SoundDash)
}

func (g *Game) spawnDashTrail(isDashing bool) {
	if !isDashing {
		return
	}
	g.effects.SpawnTrail(g.player.X, g.player.Y, [3]float32{0.5, 0.85, 1})
}
