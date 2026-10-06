package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"hordefall/internal/audio"
)

// ===== Types =====

type Intro struct {
	Seconds       float32
	FirePresses   int
	SkipSeconds   float32
	IsSkipping    bool
	HasImpacted   bool
	Logo          *VoxelLogo
	Glass         *ShatteredGlass
	ShakeX        float32
	ShakeY        float32
	FadeInSeconds float32
}

// ===== Constants =====

const (
	introBlackSeconds    = 0.35
	introFallSeconds     = audio.BombWhistleSeconds
	introImpactAt        = introBlackSeconds + introFallSeconds
	introHoldSeconds     = 1.7
	introFadeSeconds     = 0.8
	introEndAt           = introImpactAt + introHoldSeconds + introFadeSeconds
	introSkipPresses     = 3
	introSkipFadeSeconds = 0.4
	introTumbleTurns     = 2
	introSpinTurns       = 1.25
	introRollTurns       = 0.75
	introChaos           = 0.9
	introStartDistance   = 14
	introFinalWidth      = 0.72
	introStartLift       = 1.6
	introImpactBounce    = 0.06
	introBounceSeconds   = 0.18
	introShakeStrength   = 16
	introShakeSeconds    = 0.45
	introFlashSeconds    = 0.14
	introMenuFadeSeconds = 0.6
	introGlassSeed       = 0x6E55
)

// ===== Internal =====

func (g *Game) startIntro() {
	g.intro = Intro{Logo: NewVoxelLogo()}
	g.audio.SetMusicOn(false)
	g.state = StateIntro
}

func (g *Game) updateIntro() {
	intro := &g.intro
	intro.Seconds += deltaSeconds
	intro.FirePresses += boolToIndex(g.controls.JustPressed&ActionFire != 0)
	g.startIntroSkipIf(intro.FirePresses >= introSkipPresses && !intro.IsSkipping)
	g.playSoundIf(audio.SoundBombWhistle, intro.Seconds >= introBlackSeconds && intro.Seconds-deltaSeconds < introBlackSeconds && !intro.IsSkipping)
	g.impactIf(intro.Seconds >= introImpactAt && !intro.HasImpacted && !intro.IsSkipping)
	shake := introShakeStrength * max(0, 1-(intro.Seconds-introImpactAt)/introShakeSeconds) * boolToFloat(intro.HasImpacted)
	intro.ShakeX, intro.ShakeY = g.random.Between(-shake, shake), g.random.Between(-shake, shake)
	isOver := intro.Seconds >= introEndAt || intro.IsSkipping && intro.Seconds-intro.SkipSeconds >= introSkipFadeSeconds
	if isOver {
		g.finishIntro()
	}
}

func (g *Game) startIntroSkipIf(shouldSkip bool) {
	if !shouldSkip {
		return
	}
	g.intro.IsSkipping = true
	g.intro.SkipSeconds = g.intro.Seconds
}

func (g *Game) impactIf(shouldImpact bool) {
	if !shouldImpact {
		return
	}
	g.intro.HasImpacted = true
	g.intro.Glass = NewShatteredGlass(screenWidth/2, screenHeight/2, introGlassSeed)
	g.playSound(audio.SoundBigImpact)
	g.playSound(audio.SoundGlassShatter)
}

func (g *Game) finishIntro() {
	g.audio.SetMusicOn(g.settings.IsMusicOn)
	g.intro.FadeInSeconds = introMenuFadeSeconds
	g.switchState(StateMainMenu)
	g.openLanguageMenuIfFirstRun()
}

func (g *Game) drawIntro(screen *ebiten.Image) {
	screen.Fill(color.Black)
	intro := &g.intro
	fall := clamp((intro.Seconds-introBlackSeconds)/introFallSeconds, 0, 1)
	if intro.Seconds >= introBlackSeconds {
		intro.Logo.Draw(screen, g.introLogoPose(fall))
	}
	if intro.Glass != nil {
		intro.Glass.Draw(screen, 1)
	}
	g.post.ApplyBloom(bloomLevels[g.settings.BloomLevel].Factor)
	flash := max(0, 1-(intro.Seconds-introImpactAt)/introFlashSeconds) * boolToFloat(intro.HasImpacted)
	fillRect(screen, 0, 0, screenWidth, screenHeight, toColor(textColor, flash*0.8))
	fillRect(screen, 0, 0, screenWidth, screenHeight, toColor([3]float32{}, g.introFadeOut()))
}

func (g *Game) introLogoPose(fall float32) LogoPose {
	intro := &g.intro
	finalDistance := logoFocal * intro.Logo.Width / (screenWidth * introFinalWidth)
	distance := finalDistance * (introStartDistance + (1-introStartDistance)*fall*fall)
	sinceImpact := intro.Seconds - introImpactAt
	bounce := introImpactBounce * max(0, 1-sinceImpact/introBounceSeconds) * boolToFloat(intro.HasImpacted)
	remaining := 1 - fall
	return LogoPose{
		CenterX: screenWidth/2 + intro.ShakeX, CenterY: screenHeight/2 + intro.ShakeY,
		Distance: distance * (1 - bounce),
		Tumble:   (introTumbleTurns*6.2831853 + introChaos*sine(fall*7.3)) * remaining,
		Spin:     (introSpinTurns*6.2831853 + introChaos*sine(fall*5.1+1.7)) * remaining * remaining,
		Roll:     (introRollTurns*6.2831853 + introChaos*sine(fall*9.7+0.6)) * remaining,
		Lift:     -intro.Logo.Height * introStartLift * remaining,
	}
}

func (g *Game) introFadeOut() float32 {
	intro := &g.intro
	normalFade := (intro.Seconds - (introImpactAt + introHoldSeconds)) / introFadeSeconds
	skipFade := (intro.Seconds - intro.SkipSeconds) / introSkipFadeSeconds
	fades := [2]float32{normalFade, max(normalFade, skipFade)}
	return clamp(fades[boolToIndex(intro.IsSkipping)], 0, 1)
}

func (g *Game) drawMenuFadeIn(screen *ebiten.Image) {
	alpha := g.intro.FadeInSeconds / introMenuFadeSeconds
	if alpha <= 0 {
		return
	}
	fillRect(screen, 0, 0, screenWidth, screenHeight, toColor([3]float32{}, alpha))
}
