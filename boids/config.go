package main

// ── Global ───────────────────────────────────────────────────────────────────
const (
	// for debug
	isDebugMode         = true
	strokeWidth float32 = 3.0
)

// ── Sheep/Boid ───────────────────────────────────────────────────────────────────
const (
	boidsCount     = 500
	targetBoidSize = 65
	sheepImgPath   = "./assets/sheep/sheep_run.png"

	// boid simulation params
	alignRadius         = targetBoidSize * 1.5
	alightForce         = 0.05
	cohRadius           = targetBoidSize * 2.5
	cohForce            = 0.0025
	sepRadius           = targetBoidSize / 5
	sepForce            = 2
	minSpeed    float64 = 1
	maxSpeed    float64 = 4

	// bg collision
	wallSepDistance = screenWidth / 15
	wallSepForce    = 0.75

	// player collision
	fleeRadius  = targetBoidSize * 2
	catchRadius = targetBoidSize * 1.2
	fleeSpeed   = maxSpeed * 2
	caughtTicks = 180
	fleeForce   = 1
)

// ── Game ───────────────────────────────────────────────────────────────────
const (
	screenHeight = 640 * 1.5
	screenWidth  = 1080.00 * 1.5
	bgPath       = "./assets/bush_border/bush.png"
)

// ── Player ───────────────────────────────────────────────────────────────────
const (
	playerSheetPath  = "./assets/character/knight_sprite.png"
	playerSheetCols  = 2
	playerSheetRows  = 3
	playerFrameCount = playerSheetCols * playerSheetRows
	playerFrameDelay = 10
	playerSpeed      = 5.0
	playerSize       = targetBoidSize * 2

	// collision box
	collisionW       float64 = 0.22
	collisionH       float64 = 0.36
	collisionOffsetY float64 = 0.18
)

// ── Barn ───────────────────────────────────────────────────────────────────
const (
	barnSheetPath = "./assets/tiles/barn.png"
	barnSizeX     = 500
	barnSizeY     = 550
	barnOffsetX   = -40

	// -for collision drawing
	// barnBodyOffX = -60.0 // computed from pixel scan: (54+315)/2 - 251) * (400/502)
	// barnBodyOffY = 43.0  // wall base is in lower half of image
	// barnBodyW    = 180.0
	// barnBodyH    = 100.0

	barnGateOffX = -0.14 // X offset from center as fraction of drawnW
	barnGateOffY = 0.09  // Y offset from center as fraction of drawnH
	barnGateW    = 0.40  // half-width  as fraction of drawnW (full = *2)
	barnGateH    = 0.18  // half-height as fraction of drawnH
)
