package main

import (
	"bytes"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	sheepSheet       *ebiten.Image
	barnSheet        *ebiten.Image
	bananaEventSheet *ebiten.Image
	bananaPeelSheet  *ebiten.Image
	energySheet      *ebiten.Image
	ramSheet         *ebiten.Image
	sparkleSheet     *ebiten.Image
	dizzySheet       *ebiten.Image
	menuFontSheet    *ebiten.Image
	bgImage          *ebiten.Image
	rawBgImage       image.Image

	scoreFont *text.GoTextFace

	bgScaleX float64
	bgScaleY float64
)

func init() {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	scoreFont = &text.GoTextFace{
		Source: src,
		Size:   62,
	}

	// ----characters----
	knightImg, _, err := ebitenutil.NewImageFromFile(KnightOpts.sheetPath)
	if err != nil {
		log.Fatal("player sprite:", err)
	}
	KnightOpts.img = knightImg

	girlImg, _, err := ebitenutil.NewImageFromFile(GirlOpts.sheetPath)
	if err != nil {
		log.Fatal("player sprite:", err)
	}
	GirlOpts.img = girlImg

	madSciImg, _, err := ebitenutil.NewImageFromFile(MadSciOpts.sheetPath)
	if err != nil {
		log.Fatal("mad scientist sprite:", err)
	}
	MadSciOpts.img = madSciImg

	// ----boids----
	sheep, _, err := ebitenutil.NewImageFromFile(sheepImgPath)
	if err != nil {
		log.Fatal("sheep sprite:", err)
	}
	sheepSheet = sheep

	// ----events----
	bananaEvent, _, err := ebitenutil.NewImageFromFile(bananaEventPath)
	if err != nil {
		log.Fatal("bananaEvent sprite:", err)
	}
	bananaEventSheet = bananaEvent
	bananaPeel, _, err := ebitenutil.NewImageFromFile(bananaPeelPath)
	if err != nil {
		log.Fatal("bananaPeel sprite:", err)
	}
	bananaPeelSheet = bananaPeel
	energy, _, err := ebitenutil.NewImageFromFile(energyEventPath)
	if err != nil {
		log.Fatal("energy sprite:", err)
	}
	energySheet = energy

	ram, _, err := ebitenutil.NewImageFromFile(ramPath)
	if err != nil {
		log.Fatal("ram sprite:", err)
	}
	ramSheet = ram

	// ----effects----
	sparkle, _, err := ebitenutil.NewImageFromFile(sparkeSheetPath)
	if err != nil {
		log.Fatal("sparkle sprite:", err)
	}
	sparkleSheet = sparkle

	dizzy, _, err := ebitenutil.NewImageFromFile(dizzySheetPath)
	if err != nil {
		log.Fatal("dizzy sprite:", err)
	}
	dizzySheet = dizzy
	menu, _, err := ebitenutil.NewImageFromFile(menuFontPath)
	if err != nil {
		log.Fatal("menu sprite:", err)
	}
	menuFontSheet = menu

	menuFontBitMap = NewFontBitMap()

	loadBgImg()
	initScoreDisplay()
	loadDizzyFrames()
}

func loadBgImg() {
	img, rawImg, err := ebitenutil.NewImageFromFile(bgPath)
	if err != nil {
		log.Fatal(err)
	}
	bgImage = img
	rawBgImage = rawImg

	bgScaleX = screenWidth / float64(bgImage.Bounds().Dx())
	bgScaleY = screenHeight / float64(bgImage.Bounds().Dy())
}
