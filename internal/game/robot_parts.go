package game

// ===== Types =====

type RobotHead uint8
type RobotEyes uint8
type RobotAbdomen uint8
type RobotLegs uint8
type RobotArms uint8
type RobotColor uint8

type RobotLook struct {
	Head    RobotHead
	Eyes    RobotEyes
	Abdomen RobotAbdomen
	Legs    RobotLegs
	Arms    RobotArms
	Color   RobotColor
}

type RobotPaint struct {
	Body      [3]float32
	Highlight [3]float32
	Outline   [3]float32
	Shell     Material
	Head      Material
	Dark      Material
}

type RobotFrame struct {
	IsOnWater bool
	Paint     *RobotPaint
	Body      [3]float32
	Breath    float32
	Alpha     float32
}

type RobotScreenPart func(r *Renderer, player *Player)
type RobotScreenLeg func(r *Renderer, player *Player, leg int)
type RobotArtPart func(c *ArtCanvas, pose ArtPose, paint *RobotPaint)

// ===== Constants =====

const (
	RobotHeadRound RobotHead = iota
	RobotHeadTurret
	RobotHeadCap
	RobotHeadCrook
	RobotHeadLid
	RobotHeadScreen
	RobotHeadSeat
	RobotHeadHelmet
	robotHeadCount
)

const (
	RobotEyesTrio RobotEyes = iota
	RobotEyesVisor
	RobotEyesCyclops
	RobotEyesDuo
	RobotEyesPixel
	RobotEyesHeadlight
	RobotEyesPorthole
	robotEyesCount
)

const (
	RobotAbdomenBulb RobotAbdomen = iota
	RobotAbdomenCrate
	RobotAbdomenPaper
	RobotAbdomenCanopy
	RobotAbdomenIron
	RobotAbdomenTape
	RobotAbdomenBasket
	RobotAbdomenTank
	robotAbdomenCount
)

const (
	RobotLegsDroplet RobotLegs = iota
	RobotLegsStrut
	RobotLegsBamboo
	RobotLegsRib
	RobotLegsStub
	RobotLegsCable
	RobotLegsTube
	RobotLegsHose
	robotLegsCount
)

const (
	RobotArmsManipulator RobotArms = iota
	RobotArmsLamps
	RobotArmsHook
	RobotArmsRib
	RobotArmsLoop
	RobotArmsAntenna
	RobotArmsHandle
	RobotArmsPincer
	robotArmsCount
)

const (
	RobotColorBlue RobotColor = iota
	RobotColorYellow
	RobotColorBloodRed
	RobotColorViolet
	RobotColorRose
	RobotColorGreen
	robotColorCount
)

var robotPaints = [robotColorCount]RobotPaint{
	RobotColorBlue: {
		Body:      tsukumoBlue,
		Highlight: tsukumoHighlight,
		Outline:   tsukumoDeepBlue,
		Shell:     Material{Base: tsukumoBlue, Rim: redRim, RimSide: 1, IsShaded: true, IsHighlight: true},
		Head:      Material{Base: [3]float32{0.12, 0.3, 0.66}, Rim: redRim, RimSide: 1, IsShaded: true, IsHighlight: true},
		Dark:      Material{Base: [3]float32{0.08, 0.14, 0.32}, Rim: redRim, RimSide: 1, IsShaded: true},
	},
	RobotColorYellow: {
		Body:      robotPaletteYellow,
		Highlight: [3]float32{1, 1, 0.65},
		Outline:   robotPaletteRaisin,
		Shell:     Material{Base: robotPaletteYellow, Rim: redRim, RimSide: 1, IsShaded: true, IsHighlight: true},
		Head:      Material{Base: [3]float32{0.86, 0.83, 0}, Rim: redRim, RimSide: 1, IsShaded: true, IsHighlight: true},
		Dark:      Material{Base: robotPaletteRaisin, Rim: redRim, RimSide: 1, IsShaded: true},
	},
	RobotColorBloodRed: robotPaintOf([3]float32{0.443, 0, 0}, [3]float32{0.34, 0, 0}, robotPaletteSilver),
	RobotColorViolet:   robotPaintOf([3]float32{0.576, 0.439, 0.859}, [3]float32{0.46, 0.35, 0.72}, robotPaletteRaisin),
	RobotColorRose:     robotPaintOf([3]float32{0.894, 0.333, 0.682}, [3]float32{0.74, 0.26, 0.56}, robotPaletteRaisin),
	RobotColorGreen:    robotPaintOf([3]float32{0.102, 0.773, 0.69}, [3]float32{0.08, 0.62, 0.55}, robotPaletteRaisin),
}

var lanternRobotLook = RobotLook{
	Head: RobotHeadCap, Eyes: RobotEyesCyclops, Abdomen: RobotAbdomenPaper,
	Legs: RobotLegsBamboo, Arms: RobotArmsHook, Color: RobotColorBloodRed,
}

var parasolRobotLook = RobotLook{
	Head: RobotHeadCrook, Eyes: RobotEyesCyclops, Abdomen: RobotAbdomenCanopy,
	Legs: RobotLegsRib, Arms: RobotArmsRib, Color: RobotColorRose,
}

var kettleRobotLook = RobotLook{
	Head: RobotHeadLid, Eyes: RobotEyesDuo, Abdomen: RobotAbdomenIron,
	Legs: RobotLegsStub, Arms: RobotArmsLoop, Color: RobotColorGreen,
}

var televisionRobotLook = RobotLook{
	Head: RobotHeadScreen, Eyes: RobotEyesPixel, Abdomen: RobotAbdomenTape,
	Legs: RobotLegsCable, Arms: RobotArmsAntenna, Color: RobotColorViolet,
}

var cartRobotLook = RobotLook{
	Head: RobotHeadSeat, Eyes: RobotEyesHeadlight, Abdomen: RobotAbdomenBasket,
	Legs: RobotLegsTube, Arms: RobotArmsHandle, Color: RobotColorBlue,
}

var diverRobotLook = RobotLook{
	Head: RobotHeadHelmet, Eyes: RobotEyesPorthole, Abdomen: RobotAbdomenTank,
	Legs: RobotLegsHose, Arms: RobotArmsPincer, Color: RobotColorYellow,
}

var worksiteRobotLook = RobotLook{
	Head: RobotHeadTurret, Eyes: RobotEyesVisor, Abdomen: RobotAbdomenCrate,
	Legs: RobotLegsStrut, Arms: RobotArmsLamps, Color: RobotColorYellow,
}

var (
	robotScreenHeads    = [robotHeadCount]RobotScreenPart{RobotHeadRound: (*Renderer).queueRoundHead, RobotHeadTurret: (*Renderer).queueTurretHead, RobotHeadCap: (*Renderer).queueCapHead, RobotHeadCrook: (*Renderer).queueCrookHead, RobotHeadLid: (*Renderer).queueLidHead, RobotHeadScreen: (*Renderer).queueScreenHead, RobotHeadSeat: (*Renderer).queueSeatHead, RobotHeadHelmet: (*Renderer).queueHelmetHead}
	robotScreenEyes     = [robotEyesCount]RobotScreenPart{RobotEyesTrio: (*Renderer).queueTrioEyes, RobotEyesVisor: (*Renderer).queueVisorEyes, RobotEyesCyclops: (*Renderer).queueCyclopsEyes, RobotEyesDuo: (*Renderer).queueDuoEyes, RobotEyesPixel: (*Renderer).queuePixelEyes, RobotEyesHeadlight: (*Renderer).queueHeadlightEyes, RobotEyesPorthole: (*Renderer).queuePortholeEyes}
	robotScreenAbdomens = [robotAbdomenCount]RobotScreenPart{RobotAbdomenBulb: (*Renderer).queueBulbAbdomen, RobotAbdomenCrate: (*Renderer).queueCrateAbdomen, RobotAbdomenPaper: (*Renderer).queuePaperAbdomen, RobotAbdomenCanopy: (*Renderer).queueCanopyAbdomen, RobotAbdomenIron: (*Renderer).queueIronAbdomen, RobotAbdomenTape: (*Renderer).queueTapeAbdomen, RobotAbdomenBasket: (*Renderer).queueBasketAbdomen, RobotAbdomenTank: (*Renderer).queueTankAbdomen}
	robotScreenLegs     = [robotLegsCount]RobotScreenLeg{RobotLegsDroplet: (*Renderer).queueDropletLeg, RobotLegsStrut: (*Renderer).queueStrutLeg, RobotLegsBamboo: (*Renderer).queueBambooLeg, RobotLegsRib: (*Renderer).queueRibLeg, RobotLegsStub: (*Renderer).queueStubLeg, RobotLegsCable: (*Renderer).queueCableLeg, RobotLegsTube: (*Renderer).queueTubeLeg, RobotLegsHose: (*Renderer).queueHoseLeg}
	robotScreenArms     = [robotArmsCount]RobotScreenPart{RobotArmsManipulator: (*Renderer).queueManipulatorArms, RobotArmsLamps: (*Renderer).queueLampArms, RobotArmsHook: (*Renderer).queueHookArms, RobotArmsRib: (*Renderer).queueRibArms, RobotArmsLoop: (*Renderer).queueLoopArms, RobotArmsAntenna: (*Renderer).queueAntennaArms, RobotArmsHandle: (*Renderer).queueHandleArms, RobotArmsPincer: (*Renderer).queuePincerArms}

	robotArtHeads     = [robotHeadCount]RobotArtPart{RobotHeadRound: (*ArtCanvas).paintRoundHead, RobotHeadTurret: (*ArtCanvas).paintTurretHead, RobotHeadCap: (*ArtCanvas).paintCapHead, RobotHeadCrook: (*ArtCanvas).paintCrookHead, RobotHeadLid: (*ArtCanvas).paintLidHead, RobotHeadScreen: (*ArtCanvas).paintScreenHead, RobotHeadSeat: (*ArtCanvas).paintSeatHead, RobotHeadHelmet: (*ArtCanvas).paintHelmetHead}
	robotArtEyes      = [robotEyesCount]RobotArtPart{RobotEyesTrio: (*ArtCanvas).paintTrioEyes, RobotEyesVisor: (*ArtCanvas).paintVisorEyes, RobotEyesCyclops: (*ArtCanvas).paintCyclopsEyes, RobotEyesDuo: (*ArtCanvas).paintDuoEyes, RobotEyesPixel: (*ArtCanvas).paintPixelEyes, RobotEyesHeadlight: (*ArtCanvas).paintHeadlightEyes, RobotEyesPorthole: (*ArtCanvas).paintPortholeEyes}
	robotArtAbdomens  = [robotAbdomenCount]RobotArtPart{RobotAbdomenBulb: (*ArtCanvas).paintBulbAbdomen, RobotAbdomenCrate: (*ArtCanvas).paintCrateAbdomen, RobotAbdomenPaper: (*ArtCanvas).paintPaperAbdomen, RobotAbdomenCanopy: (*ArtCanvas).paintCanopyAbdomen, RobotAbdomenIron: (*ArtCanvas).paintIronAbdomen, RobotAbdomenTape: (*ArtCanvas).paintTapeAbdomen, RobotAbdomenBasket: (*ArtCanvas).paintBasketAbdomen, RobotAbdomenTank: (*ArtCanvas).paintTankAbdomen}
	robotArtRearLegs  = [robotLegsCount]RobotArtPart{RobotLegsDroplet: (*ArtCanvas).paintDropletRearLegs, RobotLegsStrut: (*ArtCanvas).paintStrutRearLegs, RobotLegsBamboo: (*ArtCanvas).paintBambooRearLegs, RobotLegsRib: (*ArtCanvas).paintRibRearLegs, RobotLegsStub: (*ArtCanvas).paintStubRearLegs, RobotLegsCable: (*ArtCanvas).paintCableRearLegs, RobotLegsTube: (*ArtCanvas).paintTubeRearLegs, RobotLegsHose: (*ArtCanvas).paintHoseRearLegs}
	robotArtFrontLegs = [robotLegsCount]RobotArtPart{RobotLegsDroplet: (*ArtCanvas).paintDropletFrontLegs, RobotLegsStrut: (*ArtCanvas).paintStrutFrontLegs, RobotLegsBamboo: (*ArtCanvas).paintBambooFrontLegs, RobotLegsRib: (*ArtCanvas).paintRibFrontLegs, RobotLegsStub: (*ArtCanvas).paintStubFrontLegs, RobotLegsCable: (*ArtCanvas).paintCableFrontLegs, RobotLegsTube: (*ArtCanvas).paintTubeFrontLegs, RobotLegsHose: (*ArtCanvas).paintHoseFrontLegs}
	robotArtArms      = [robotArmsCount]RobotArtPart{RobotArmsManipulator: (*ArtCanvas).paintManipulatorArms, RobotArmsLamps: (*ArtCanvas).paintLampArms, RobotArmsHook: (*ArtCanvas).paintHookArms, RobotArmsRib: (*ArtCanvas).paintRibArms, RobotArmsLoop: (*ArtCanvas).paintLoopArms, RobotArmsAntenna: (*ArtCanvas).paintAntennaArms, RobotArmsHandle: (*ArtCanvas).paintHandleArms, RobotArmsPincer: (*ArtCanvas).paintPincerArms}
)

// ===== Internal =====

func robotPaintOf(body, head, outline [3]float32) RobotPaint {
	return RobotPaint{
		Body:      body,
		Highlight: mixColor(body, [3]float32{1, 1, 1}, 0.55),
		Outline:   outline,
		Shell:     Material{Base: body, Rim: redRim, RimSide: 1, IsShaded: true, IsHighlight: true},
		Head:      Material{Base: head, Rim: redRim, RimSide: 1, IsShaded: true, IsHighlight: true},
		Dark:      Material{Base: robotPaletteRaisin, Rim: redRim, RimSide: 1, IsShaded: true},
	}
}
