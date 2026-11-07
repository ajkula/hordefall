package game

import (
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/store"
)

// ===== Types =====

type RobotSlotKind uint8

type RobotCounter uint8

type RobotSlot struct {
	NameKey string
	Count   int
	Part    func(look *RobotLook) int
	SetPart func(look *RobotLook, part int)
	Paint   func(c *ArtCanvas, pose ArtPose, part int, color RobotColor)
}

type RobotUnlock struct {
	Slot    RobotSlotKind
	Part    int
	Counter RobotCounter
	Goal    int
}

type RobotGarage struct {
	Look   RobotLook              `json:"look"`
	Counts [robotCounterCount]int `json:"counts"`
}

// ===== Constants =====

const (
	RobotSlotHead RobotSlotKind = iota
	RobotSlotEyes
	RobotSlotAbdomen
	RobotSlotLegs
	RobotSlotArms
	RobotSlotColor
	robotSlotCount
)

const (
	RobotCounterKills RobotCounter = iota
	RobotCounterBosses
	robotCounterCount
)

const (
	robotGarageFileName = "robot.json"
	robotSlotPartMax    = 8
)

var defaultRobotLook = lanternRobotLook

var robotSlots = [robotSlotCount]RobotSlot{
	RobotSlotHead: {
		NameKey: "garage.head", Count: int(robotHeadCount),
		Part:    func(look *RobotLook) int { return int(look.Head) },
		SetPart: func(look *RobotLook, part int) { look.Head = RobotHead(part) },
		Paint: func(c *ArtCanvas, pose ArtPose, part int, color RobotColor) {
			robotArtHeads[part](c, pose, &robotPaints[color])
		},
	},
	RobotSlotEyes: {
		NameKey: "garage.eyes", Count: int(robotEyesCount),
		Part:    func(look *RobotLook) int { return int(look.Eyes) },
		SetPart: func(look *RobotLook, part int) { look.Eyes = RobotEyes(part) },
		Paint: func(c *ArtCanvas, pose ArtPose, part int, color RobotColor) {
			robotArtEyes[part](c, pose, &robotPaints[color])
		},
	},
	RobotSlotAbdomen: {
		NameKey: "garage.abdomen", Count: int(robotAbdomenCount),
		Part:    func(look *RobotLook) int { return int(look.Abdomen) },
		SetPart: func(look *RobotLook, part int) { look.Abdomen = RobotAbdomen(part) },
		Paint: func(c *ArtCanvas, pose ArtPose, part int, color RobotColor) {
			robotArtAbdomens[part](c, pose, &robotPaints[color])
		},
	},
	RobotSlotLegs: {
		NameKey: "garage.legs", Count: int(robotLegsCount),
		Part:    func(look *RobotLook) int { return int(look.Legs) },
		SetPart: func(look *RobotLook, part int) { look.Legs = RobotLegs(part) },
		Paint: func(c *ArtCanvas, pose ArtPose, part int, color RobotColor) {
			robotArtRearLegs[part](c, pose, &robotPaints[color])
			robotArtFrontLegs[part](c, pose, &robotPaints[color])
		},
	},
	RobotSlotArms: {
		NameKey: "garage.arms", Count: int(robotArmsCount),
		Part:    func(look *RobotLook) int { return int(look.Arms) },
		SetPart: func(look *RobotLook, part int) { look.Arms = RobotArms(part) },
		Paint: func(c *ArtCanvas, pose ArtPose, part int, color RobotColor) {
			robotArtArms[part](c, pose, &robotPaints[color])
		},
	},
	RobotSlotColor: {
		NameKey: "garage.color", Count: int(robotColorCount),
		Part:    func(look *RobotLook) int { return int(look.Color) },
		SetPart: func(look *RobotLook, part int) { look.Color = RobotColor(part) },
		Paint: func(c *ArtCanvas, pose ArtPose, part int, _ RobotColor) {
			c.paintEllipsoid(pose, 0, -36, 24, 24, robotPaints[part].Shell)
		},
	},
}

var robotUnlocks = [...]RobotUnlock{
	{RobotSlotLegs, int(RobotLegsRib), RobotCounterKills, 2500},
	{RobotSlotEyes, int(RobotEyesDuo), RobotCounterKills, 10000},
	{RobotSlotArms, int(RobotArmsAntenna), RobotCounterKills, 22500},
	{RobotSlotHead, int(RobotHeadCrook), RobotCounterKills, 40000},
	{RobotSlotAbdomen, int(RobotAbdomenTape), RobotCounterKills, 62500},
	{RobotSlotArms, int(RobotArmsLoop), RobotCounterBosses, 1},
	{RobotSlotLegs, int(RobotLegsStrut), RobotCounterKills, 90000},
	{RobotSlotEyes, int(RobotEyesVisor), RobotCounterKills, 122500},
	{RobotSlotHead, int(RobotHeadLid), RobotCounterKills, 160000},
	{RobotSlotEyes, int(RobotEyesPixel), RobotCounterBosses, 2},
	{RobotSlotAbdomen, int(RobotAbdomenCanopy), RobotCounterKills, 202500},
	{RobotSlotLegs, int(RobotLegsCable), RobotCounterKills, 250000},
	{RobotSlotLegs, int(RobotLegsStub), RobotCounterBosses, 3},
	{RobotSlotArms, int(RobotArmsRib), RobotCounterKills, 302500},
	{RobotSlotHead, int(RobotHeadScreen), RobotCounterKills, 360000},
	{RobotSlotAbdomen, int(RobotAbdomenIron), RobotCounterKills, 422500},
	{RobotSlotAbdomen, int(RobotAbdomenCrate), RobotCounterBosses, 5},
	{RobotSlotArms, int(RobotArmsLamps), RobotCounterKills, 490000},
	{RobotSlotHead, int(RobotHeadTurret), RobotCounterKills, 562500},
	{RobotSlotEyes, int(RobotEyesHeadlight), RobotCounterKills, 640000},
	{RobotSlotHead, int(RobotHeadSeat), RobotCounterBosses, 8},
	{RobotSlotEyes, int(RobotEyesTrio), RobotCounterKills, 722500},
	{RobotSlotLegs, int(RobotLegsTube), RobotCounterKills, 810000},
	{RobotSlotArms, int(RobotArmsHandle), RobotCounterKills, 902500},
	{RobotSlotLegs, int(RobotLegsHose), RobotCounterBosses, 12},
	{RobotSlotAbdomen, int(RobotAbdomenBasket), RobotCounterKills, 1000000},
	{RobotSlotArms, int(RobotArmsManipulator), RobotCounterBosses, 18},
	{RobotSlotEyes, int(RobotEyesPorthole), RobotCounterKills, 1102500},
	{RobotSlotArms, int(RobotArmsPincer), RobotCounterKills, 1210000},
	{RobotSlotLegs, int(RobotLegsDroplet), RobotCounterKills, 1322500},
	{RobotSlotHead, int(RobotHeadHelmet), RobotCounterBosses, 25},
	{RobotSlotHead, int(RobotHeadRound), RobotCounterBosses, 35},
	{RobotSlotAbdomen, int(RobotAbdomenTank), RobotCounterKills, 1440000},
	{RobotSlotAbdomen, int(RobotAbdomenBulb), RobotCounterBosses, 50},
}

var robotCounterKeys = [robotCounterCount]string{
	RobotCounterKills:  "garage.unlock_kills",
	RobotCounterBosses: "garage.unlock_bosses",
}

// ===== Public API =====

func LoadRobotGarage(saves store.Store) RobotGarage {
	garage := RobotGarage{Look: defaultRobotLook}
	if err := saves.Load(robotGarageFileName, &garage); err != nil {
		return RobotGarage{Look: defaultRobotLook}
	}
	garage.repairLook()
	return garage
}

func (g *RobotGarage) Owns(slot RobotSlotKind, part int) bool {
	unlock := findRobotUnlock(slot, part)
	return unlock == nil || g.Counts[unlock.Counter] >= unlock.Goal
}

func (g *RobotGarage) DescribeUnlock(slot RobotSlotKind, part int) string {
	unlock := findRobotUnlock(slot, part)
	if unlock == nil || g.Owns(slot, part) {
		return ""
	}
	progress := scores.FormatThousands(g.Counts[unlock.Counter])
	return i18n.F(robotCounterKeys[unlock.Counter], progress, scores.FormatThousands(unlock.Goal))
}

// ===== Internal =====

func findRobotUnlock(slot RobotSlotKind, part int) *RobotUnlock {
	for index := range robotUnlocks {
		unlock := &robotUnlocks[index]
		if unlock.Slot == slot && unlock.Part == part {
			return unlock
		}
	}
	return nil
}

func (g *RobotGarage) repairLook() {
	for kind := range robotSlotCount {
		slot := &robotSlots[kind]
		part := slot.Part(&g.Look)
		if part < slot.Count && g.Owns(kind, part) {
			continue
		}
		slot.SetPart(&g.Look, slot.Part(&defaultRobotLook))
	}
}

func (g *Game) countRobotProgress(definition *EnemyDefinition) {
	g.garage.Counts[RobotCounterKills]++
	g.garage.Counts[RobotCounterBosses] += boolToIndex(definition == &enemyTable[EnemySpiderTank])
	g.isGarageDirty = true
}

func (g *Game) countWormDefeat() {
	g.garage.Counts[RobotCounterBosses] += boolToIndex(!g.isDemo)
	g.isGarageDirty = g.isGarageDirty || !g.isDemo
}

func (g *Game) applyRobotLook(look RobotLook) {
	g.robotLook, g.ui.art.Look = look, look
}

func (g *Game) saveRobotGarageOnDeath() {
	if g.player.Health <= 0 {
		g.saveRobotGarage()
	}
}

func (g *Game) saveRobotGarage() {
	if !g.isGarageDirty || g.robotLab != nil {
		return
	}
	g.garage.Look = g.robotLook
	g.isGarageDirty = false
	if err := g.app.Store.Save(robotGarageFileName, &g.garage); err != nil {
		g.app.Message = i18n.F("settings.save_failed", err.Error())
	}
}
