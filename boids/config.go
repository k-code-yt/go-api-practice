package main

// ── Global ───────────────────────────────────────────────────────────────────
const (
	// for debug
	isDebugMode         = true
	strokeWidth float32 = 3.0
)

// ── Sheep/Boid ───────────────────────────────────────────────────────────────────
const (
	boidsCount     = 50
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
	barnSheetPath = "./assets/tiles/barn25d.png"
	barnSizeX     = 400
	barnSizeY     = 500
	barnOffsetX   = 10

	// -for collision drawing
	barnBodyOffX = -60.0 // computed from pixel scan: (54+315)/2 - 251) * (400/502)
	barnBodyOffY = -40.0 // wall base is in lower half of image
	barnBodyW    = 200.0
	barnBodyH    = 280.0

	// Fence pen
	barnFenceOffX = +130.0
	barnFenceOffY = +54.0
	barnFenceW    = 95.0
	barnFenceH    = 318.0

	// Gate opening (entry point for sheep)
	barnGateOffX = +111.0
	barnGateOffY = +164.0
	barnGateW    = 32.0
	barnGateH    = 44.0
)
