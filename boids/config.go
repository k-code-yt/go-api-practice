package main

// ── Global ───────────────────────────────────────────────────────────────────
const (
	// for debug
	isDebugMode         = true
	strokeWidth float32 = 3.0
)

// ── Sheep/Boid ───────────────────────────────────────────────────────────────────
const (
	boidsCount     = 150
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
