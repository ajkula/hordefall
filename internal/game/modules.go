package game

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/rig"
)

// ===== Constants =====

const (
	moduleWorld       = worldModuleName
	modulePlayer      = "player"
	moduleWeapons     = "weapons"
	moduleMines       = "mines"
	moduleProjectiles = "projectiles"
	moduleEnemies     = "enemies"
	moduleSpiders     = "spiders"
	moduleWorm        = "worm"
	moduleDirector    = "director"
	moduleGround      = "ground"
	moduleWeather     = "weather"
	moduleChatter     = "chatter"
	moduleEffects     = "effects"
	moduleGems        = "gems"
	moduleHealOrbs    = "heal orbs"
	moduleHud         = "hud"
	moduleLevelUp     = "level up"
)

var demoPreludes = [2]func(g *Game){func(*Game) {}, (*Game).prepareDemoTick}

var demoEpilogues = [2]func(g *Game){func(*Game) {}, (*Game).finishDemoTick}

// ===== Public API =====

func (g *Game) Modules() []rig.Module {
	return append(append(g.simulationModules(), g.renderModules()...), g.hudModules()...)
}

// ===== Internal =====

func (g *Game) simulationModules() []rig.Module {
	return []rig.Module{
		g,
		simulationStep(modulePlayer, func() { g.updatePlayer(deltaSeconds) }),
		simulationStep(moduleWeapons, func() { g.updateWeapons(deltaSeconds) }),
		simulationStep(moduleMines, func() { g.updateMines(deltaSeconds) }),
		simulationStep(moduleProjectiles, func() { g.updateProjectiles(deltaSeconds) }),
		simulationStep(moduleEnemies, func() {
			g.updateEnemies(deltaSeconds)
			g.processBursts()
			g.removeDeadEnemies()
		}),
		simulationStep(moduleSpiders, func() { g.updateSpiders(deltaSeconds) }),
		simulationStep(moduleWorm, func() { g.updateWorm(deltaSeconds) }),
		simulationStep(moduleDirector, func() {
			g.updateSpawning(deltaSeconds)
			g.startBossEventIfDue()
		}),
		simulationStep(moduleGround, g.tickGroundIfDue),
		simulationStep(moduleWeather, func() { g.updateWeather(deltaSeconds) }),
		simulationStep(moduleChatter, func() { g.updateChatter(deltaSeconds) }),
		simulationStep(moduleEffects, func() { g.effects.Update(deltaSeconds) }),
		&rig.Func{ID: moduleWorld, OnUpdate: g.endTick},
	}
}

func (g *Game) renderModules() []rig.Module {
	r := g.renderer
	return []rig.Module{
		renderStep(moduleWorld, func(*ebiten.Image) { r.updateCamera(g) }),
		renderStep(moduleGround, func(screen *ebiten.Image) {
			r.drawGround(g, screen)
			r.queueGroundGlow(g)
		}),
		renderStep(moduleGems, func(*ebiten.Image) { r.queueGems(g) }),
		renderStep(moduleHealOrbs, func(*ebiten.Image) { r.queueHealOrbs(g) }),
		renderStep(moduleMines, func(*ebiten.Image) { r.queueMines(g) }),
		renderStep(moduleWorm, func(screen *ebiten.Image) {
			r.drawWormGroundSprites(g, screen)
			r.queueDrillWormGround(g)
		}),
		renderStep(moduleEnemies, func(*ebiten.Image) { r.queueEnemies(g) }),
		renderStep(modulePlayer, func(*ebiten.Image) { r.queuePlayer(g) }),
		renderStep(moduleSpiders, func(*ebiten.Image) { r.queueSpiders(g) }),
		renderStep(moduleWorm, func(*ebiten.Image) { r.queueDrillWorm(g) }),
		renderStep(moduleProjectiles, func(*ebiten.Image) { r.queueProjectiles(g) }),
		renderStep(moduleEffects, func(*ebiten.Image) { g.effects.Queue(r.batches, r.camera) }),
		renderStep(moduleSpiders, func(*ebiten.Image) { r.queueSpiderBeams(g) }),
		renderStep(moduleWeather, func(*ebiten.Image) { r.queuePrecipitation(g) }),
		renderStep(modulePlayer, func(*ebiten.Image) { r.queueMouseCrosshair(g) }),
		renderStep(moduleWorm, func(screen *ebiten.Image) {
			r.batches.Solid.Flush(screen)
			r.drawDrillWormSprites(g, screen)
		}),
		&rig.Func{ID: moduleWeather, OnDrawLate: func(_ *rig.Frame, screen *ebiten.Image) { r.applyWeatherGrade(g, screen) }},
	}
}

func (g *Game) hudModules() []rig.Module {
	u := g.ui
	return []rig.Module{
		hudStep(moduleHud, func(screen *ebiten.Image) { u.drawStatus(g, screen) }),
		hudStep(moduleChatter, func(screen *ebiten.Image) { u.drawChatter(g, screen) }),
		hudStep(moduleLevelUp, func(screen *ebiten.Image) { u.drawLevelUpBanner(g, screen) }),
		hudStep(moduleEffects, func(screen *ebiten.Image) { g.effects.DrawPopups(screen, u.kit, g.renderer.camera) }),
	}
}

func simulationStep(name string, step func()) rig.Module {
	return &rig.Func{ID: name, OnUpdate: func(*rig.Frame) { step() }}
}

func renderStep(name string, step func(screen *ebiten.Image)) rig.Module {
	return &rig.Func{ID: name, OnDraw: func(_ *rig.Frame, screen *ebiten.Image) { step(screen) }}
}

func hudStep(name string, step func(screen *ebiten.Image)) rig.Module {
	return &rig.Func{ID: name, OnDrawHUD: func(_ *rig.Frame, screen *ebiten.Image) { step(screen) }}
}

func (g *Game) beginTick(frame *rig.Frame) {
	g.controls = frame.Controls
	g.applyGraphicsSettings()
	demoPreludes[boolToIndex(frame.IsDemo)](g)
	g.frame++
	g.elapsedSeconds += deltaSeconds
	g.grid.Rebuild(g.enemies.PositionX, g.enemies.PositionY, g.enemies.Count)
}

func (g *Game) endTick(frame *rig.Frame) {
	g.levelUpBannerSeconds = max(0, g.levelUpBannerSeconds-deltaSeconds)
	demoEpilogues[boolToIndex(frame.IsDemo)](g)
	g.updateMusicSignals()
}
