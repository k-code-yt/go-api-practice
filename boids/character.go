package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── Frame identity ────────────────────────────────────────────────────────────

// FrameID is a typed int so the compiler catches wrong values in direction maps.
type FrameID int

const (
	// Knight frames
	KnightFrontIdle FrameID = iota
	KnightFrontWalk
	KnightSideWalk1
	KnightSideWalk2
	KnightSlip1
	KnightSlip2

	// Girl frames
	GirlFrontIdle
	GirlFrontWalk
	GirlFrontAttack
	GirlSideWalk1
	GirlSideWalk2
	GirlSlip
	GirlExtra0
	GirlExtra1
	GirlExtra2
	GirlExtra3

	RamWalk0 // row0: walking right, step 1
	RamWalk1 // row0: walking right, step 2
	RamWalk2 // row0: walking right, step 3
	RamWalk3 // row0: walking right, step 4

	RamWalkLeft0 // row1: walking left, step 1
	RamWalkLeft1 // row1: walking left, step 2
	RamWalkLeft2 // row1: walking left, step 3

	RamAttack0 // row2: charge/attack, step 1
	RamAttack1 // row2: charge/attack, step 2
	RamAttack2 // row2: charge/attack, step 3
	RamAttack3 // row2: charge/attack, step 4

	RamHit0 // row3: hit flash
	RamHit1 // row3: hit flash 2
	RamHit2 // row3: recovering
	RamHit3 // row3: recovering 2

	RamExplode0 // row4: explosion frame 1
	RamExplode1 // row4: explosion frame 2

	RamSmall // row5: small/squished single frame

	RamWalkDown0 // row6: walking down, step 1
	RamWalkDown1 // row6: walking down, step 2
	RamWalkDown2 // row6: walking down, step 3
	RamWalkDown3 // row6: walking down, step 4
)

type FrameRect struct{ X, Y, W, H int }

// ── Per-character rect tables (measured) ─────────────────────────────────────

// Knight: 470x531, 2 sprites per row, margin cols stripped
// row1 y=6-192   row2 y=192-364   row3 y=364-526
var knightRects = map[FrameID]FrameRect{
	KnightFrontIdle: {X: 6, Y: 6, W: 198, H: 186},
	KnightFrontWalk: {X: 230, Y: 6, W: 210, H: 186},
	KnightSideWalk1: {X: 18, Y: 192, W: 225, H: 172},
	KnightSideWalk2: {X: 243, Y: 192, W: 198, H: 172},
	KnightSlip1:     {X: 22, Y: 364, W: 204, H: 162},
	KnightSlip2:     {X: 226, Y: 364, W: 223, H: 162},
}

// Girl: 463x539, variable cols per row (measured separately)
// row0 y=4-184   row1 y=184-360   row2 y=360-531
var girlRects = map[FrameID]FrameRect{
	GirlFrontIdle:   {X: 6, Y: 4, W: 135, H: 180},
	GirlFrontWalk:   {X: 141, Y: 4, W: 123, H: 180},
	GirlFrontAttack: {X: 264, Y: 4, W: 192, H: 180},
	GirlSideWalk1:   {X: 5, Y: 184, W: 136, H: 176},
	GirlSideWalk2:   {X: 141, Y: 184, W: 127, H: 176},
	GirlSlip:        {X: 268, Y: 184, W: 181, H: 176},
	GirlExtra0:      {X: 16, Y: 360, W: 119, H: 171},
	GirlExtra1:      {X: 135, Y: 360, W: 120, H: 171},
	GirlExtra2:      {X: 255, Y: 360, W: 110, H: 171},
	GirlExtra3:      {X: 365, Y: 360, W: 92, H: 171},
}

// ── Direction → frame sequence ────────────────────────────────────────────────

var knightDirFrames = map[Direction][]FrameID{
	DirDown:  {KnightFrontIdle, KnightFrontWalk},
	DirUp:    {KnightFrontIdle, KnightFrontWalk},
	DirRight: {KnightSideWalk1, KnightSideWalk2},
	DirLeft:  {KnightSideWalk1, KnightSideWalk2},
	DirSlip:  {KnightSlip1, KnightSlip2},
}

var girlDirFrames = map[Direction][]FrameID{
	DirDown:  {GirlFrontIdle, GirlExtra2, GirlExtra1},
	DirUp:    {GirlFrontIdle, GirlExtra2, GirlExtra1},
	DirRight: {GirlExtra2, GirlFrontWalk, GirlSideWalk2},
	DirLeft:  {GirlExtra2, GirlFrontWalk, GirlSideWalk2},
	DirSlip:  {GirlFrontWalk, GirlSideWalk2, GirlSlip},
}

// ── Spritesheet ───────────────────────────────────────────────────────────────

type CharacterType int

const (
	KnightCharacter CharacterType = iota
	GirlCharacter
)

type CharacterOpts struct {
	sheetPath   string
	img         *ebiten.Image
	sheetCols   int
	sheetRows   int
	spritesheet *Spritesheet
}

var KnightOpts = &CharacterOpts{
	sheetPath: "./assets/character/knight_sprite.png",
	sheetCols: 2,
	sheetRows: 3,
}

var GirlOpts = &CharacterOpts{
	sheetPath: "./assets/character/girl_sprite.png",
	sheetCols: 3,
	sheetRows: 3,
}

func NewCharacterOpts(cType CharacterType) *CharacterOpts {
	switch cType {
	case KnightCharacter:
		if KnightOpts.img == nil {
			panic("knight img was not loaded")
		}
		KnightOpts.spritesheet = NewKnightSpritesheet(KnightOpts.img)
		return KnightOpts
	case GirlCharacter:
		if GirlOpts.img == nil {
			panic("girl img was not loaded")
		}
		GirlOpts.spritesheet = NewGirlSpritesheet(GirlOpts.img)
		return GirlOpts
	}
	return nil
}

type Spritesheet struct {
	frames    map[FrameID]*ebiten.Image
	dirFrames map[Direction][]FrameID
}

func NewSpritesheet(sheet *ebiten.Image, rects map[FrameID]FrameRect, dirFrames map[Direction][]FrameID) *Spritesheet {
	frames := make(map[FrameID]*ebiten.Image, len(rects))
	for id, r := range rects {
		rect := image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
		frames[id] = sheet.SubImage(rect).(*ebiten.Image)
	}
	return &Spritesheet{frames: frames, dirFrames: dirFrames}
}

func NewKnightSpritesheet(sheet *ebiten.Image) *Spritesheet {
	return NewSpritesheet(sheet, knightRects, knightDirFrames)
}

func NewGirlSpritesheet(sheet *ebiten.Image) *Spritesheet {
	return NewSpritesheet(sheet, girlRects, girlDirFrames)
}

func NewRamSpritesheet(sheet *ebiten.Image) *Spritesheet {
	return NewSpritesheet(sheet, ramRects, ramDirFrames)
}

func (s *Spritesheet) Frame(id FrameID) *ebiten.Image {
	return s.frames[id]
}

func (s *Spritesheet) FrameForState(dir Direction, state PlayerState, frameIdx, slipTick int) *ebiten.Image {
	if state == PlayerStateSlipping {
		dir = DirSlip
	}

	ids := s.dirFrames[dir]

	var id FrameID
	if state == PlayerStateSlipping {
		step := playerSlipDuration / (len(ids) * 2)
		i := slipTick / step
		if i >= len(ids) {
			i = len(ids) - 1
		}
		id = ids[i]
	} else {
		id = ids[frameIdx%len(ids)]
	}

	return s.frames[id]
}
