package main

// ── Global ───────────────────────────────────────────────────────────────────
const (
	// for debug
	isDebugMode         = false
	strokeWidth float32 = 3.0
)

// ── Game ───────────────────────────────────────────────────────────────────
const (
	screenHeight    = 640 * 1.5
	screenWidth     = 1080.00 * 1.5
	bgPath          = "./assets/bush_border/bush.png"
	eventSpawnTicks = 900
)

// ── Sheep/Boid ───────────────────────────────────────────────────────────────────
const (
	boidsCount     = 80
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

// ── Events ───────────────────────────────────────────────────────────────────
const (
	bananaEventPath          = "./assets/tiles/banana.png"
	energyEventPath          = "./assets/tiles/energy.png"
	bananaPeelPath           = "./assets/tiles/banana_peel.png"
	ramPath                  = "./assets/tiles/ram.png"
	eventSize                = 100
	bananaCollisionW float64 = 0.38
	bananaCollisionH float64 = 0.43

	bananaPeelCollisionW float64 = 0.3
	bananaPeelCollisionH float64 = 0.2

	bananaPeelCount = 5
)

// ── Player ───────────────────────────────────────────────────────────────────
const (
	playerFrameDelay             = 10
	playerDefaultSpeed           = 5.0
	playerEnergyMult             = 2.0
	playerSizeX                  = 140
	playerSizeY                  = 140
	collisionOffsetY     float64 = 0.18
	playerSlipDuration           = 45
	playerEnergyDuration         = 600

	// collision box
	playerCollisionW float64 = 0.22
	playerCollisionH float64 = 0.36
)

// ── Ram ───────────────────────────────────────────────────────────────────
const (
	ramFrameDelay     = 10
	ramSpeed          = 10.0
	ramSizeX          = 140
	ramSizeY          = 140
	ramChargeDuration = 30
	ramSleepDuration  = 10
	ramHitDuration    = 32
	ramHitRadius      = 120

	// collision box
	ramCollisionW float64 = 0.22
	ramCollisionH float64 = 0.36
)

// ── Barn ───────────────────────────────────────────────────────────────────
const (
	barnSheetPath = "./assets/tiles/barn.png"
	barnSizeX     = 550
	barnSizeY     = 530
	barnOffsetX   = -100

	barnGateOffX = -0.21 // X offset from center as fraction of drawnW
	barnGateOffY = 0.11  // Y offset from center as fraction of drawnH
	barnGateW    = 0.35  // half-width  as fraction of drawnW (full = *2)
	barnGateH    = 0.21  // half-height as fraction of drawnH

	barnEntryForce     = 1.0
	barnEntryTolerance = 4.0
)
