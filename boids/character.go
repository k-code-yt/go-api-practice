package main

import "github.com/hajimehoshi/ebiten/v2"

type CharacterType int

const (
	KnightCharacter CharacterType = iota
	GirlCharacter
)

type CharacterOpts struct {
	sheetPath  string
	img        *ebiten.Image
	sheetCols  int
	sheetRows  int
	frameCount int
}

var KnightOpts = &CharacterOpts{
	sheetPath: "./assets/character/knight_sprite.png",
	sheetCols: 2,
	sheetRows: 3,
}

// col, row
var dirPosKnight = map[Direction][]FrameCoords{
	DirDown:  []FrameCoords{[]int{0, 0}, []int{1, 0}},
	DirUp:    []FrameCoords{[]int{0, 0}, []int{1, 0}},
	DirRight: []FrameCoords{[]int{0, 1}, []int{1, 1}},
	DirLeft:  []FrameCoords{[]int{0, 1}, []int{1, 1}},
	DirSlip:  []FrameCoords{[]int{0, 2}, []int{1, 2}},
}

var GirlOpts = &CharacterOpts{
	sheetPath: "./assets/character/girl_sprite.png",
	sheetCols: 3,
	sheetRows: 3,
}

var dirPosGirl = map[Direction][]FrameCoords{
	DirDown:  []FrameCoords{[]int{0, 0}, []int{1, 2}, []int{0, 2}},
	DirUp:    []FrameCoords{[]int{0, 0}, []int{1, 2}, []int{0, 2}},
	DirRight: []FrameCoords{[]int{2, 2}, []int{2, 1}, []int{2, 0}, []int{1, 0}},
	DirLeft:  []FrameCoords{[]int{2, 2}, []int{2, 1}, []int{2, 0}, []int{1, 0}},
	DirSlip:  []FrameCoords{[]int{1, 0}, []int{1, 1}, []int{2, 1}},
}

func NewCharacterOpts(cType CharacterType) *CharacterOpts {
	switch cType {
	case KnightCharacter:
		if KnightOpts.frameCount == 0 {
			KnightOpts.frameCount = KnightOpts.sheetCols * KnightOpts.sheetRows
		}
		return KnightOpts
	case GirlCharacter:
		if GirlOpts.frameCount == 0 {
			GirlOpts.frameCount = GirlOpts.sheetCols * GirlOpts.sheetRows
		}
		return GirlOpts
	}
	return nil
}
