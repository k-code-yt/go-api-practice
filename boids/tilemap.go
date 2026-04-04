package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ── Sheet layout (trees.png, 736×224 px, single row, 5 trees) ────────────────
//
//	Tree 0  x   8–144  w=137  round green oak
//	Tree 1  x 157–284  w=128  red maple
//	Tree 2  x 306–430  w=125  wide willow
//	Tree 3  x 440–587  w=148  large twin oak
//	Tree 4  x 596–730  w=135  weeping willow
//
// Collision offsets are the trunk/root zone measured by pixel analysis
// (bottom ~35% of sprite content, in source-pixel space before scale).

type TreeDef struct {
	SrcX, SrcY int
	SrcW, SrcH int
	// Collision box relative to the sprite's top-left (source pixels).
	CollOffX, CollOffY float64
	CollW, CollH       float64
}

var treeDefs = [5]TreeDef{
	// 0 – round green oak
	{SrcX: 8, SrcY: 0, SrcW: 137, SrcH: 224,
		CollOffX: 3, CollOffY: 132, CollW: 130, CollH: 67},
	// 1 – red maple
	{SrcX: 157, SrcY: 0, SrcW: 128, SrcH: 224,
		CollOffX: 13, CollOffY: 138, CollW: 107, CollH: 75},
	// 2 – wide willow
	{SrcX: 306, SrcY: 0, SrcW: 125, SrcH: 224,
		CollOffX: 2, CollOffY: 138, CollW: 122, CollH: 65},
	// 3 – large twin oak
	{SrcX: 440, SrcY: 0, SrcW: 148, SrcH: 224,
		CollOffX: 3, CollOffY: 143, CollW: 138, CollH: 77},
	// 4 – weeping willow
	{SrcX: 596, SrcY: 0, SrcW: 135, SrcH: 224,
		CollOffX: 4, CollOffY: 134, CollW: 124, CollH: 73},
}

// ── Placed instance ───────────────────────────────────────────────────────────

type PlacedTree struct {
	DefIdx int
	X, Y   float64 // world-space top-left of the sprite
	Scale  float64
}

// ── Tilemap ───────────────────────────────────────────────────────────────────

type Tilemap struct {
	sheet *ebiten.Image
	trees []PlacedTree
	maskW int
	maskH int
	mask  []bool
}

func NewTilemap(sheetPath string) *Tilemap {
	img, _, err := ebitenutil.NewImageFromFile(sheetPath)
	if err != nil {
		panic("tilemap: cannot load sheet: " + err.Error())
	}
	mw, mh := int(screenWidth), int(screenHeight)
	return &Tilemap{
		sheet: img,
		maskW: mw,
		maskH: mh,
		mask:  make([]bool, mw*mh),
	}
}

// PlaceTree adds one tree and stamps its trunk footprint into the collision mask.
func (tm *Tilemap) PlaceTree(defIdx int, x, y, scale float64) {
	tm.trees = append(tm.trees, PlacedTree{DefIdx: defIdx, X: x, Y: y, Scale: scale})
	def := treeDefs[defIdx]
	tm.stampRect(
		x+def.CollOffX*scale,
		y+def.CollOffY*scale,
		def.CollW*scale,
		def.CollH*scale,
	)
}

func (tm *Tilemap) stampRect(wx, wy, ww, wh float64) {
	x0 := max(0, int(wx))
	y0 := max(0, int(wy))
	x1 := min(tm.maskW-1, int(wx+ww))
	y1 := min(tm.maskH-1, int(wy+wh))
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			tm.mask[py*tm.maskW+px] = true
		}
	}
}

// IsBush satisfies CollisionMask — identical API to BgCollisionMask.
func (tm *Tilemap) IsBush(x, y float64) bool {
	px, py := int(x), int(y)
	if px < 0 || py < 0 || px >= tm.maskW || py >= tm.maskH {
		return true
	}
	return tm.mask[py*tm.maskW+px]
}

// Draw renders all placed trees in insertion order.
func (tm *Tilemap) Draw(screen *ebiten.Image) {
	for _, pt := range tm.trees {
		def := treeDefs[pt.DefIdx]
		src := image.Rect(def.SrcX, def.SrcY,
			def.SrcX+def.SrcW, def.SrcY+def.SrcH)
		sub := tm.sheet.SubImage(src).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(pt.Scale, pt.Scale)
		op.GeoM.Translate(pt.X, pt.Y)
		screen.DrawImage(sub, op)

		if isDebugMode {
			rx := float32(pt.X + def.CollOffX*pt.Scale)
			ry := float32(pt.Y + def.CollOffY*pt.Scale)
			rw := float32(def.CollW * pt.Scale)
			rh := float32(def.CollH * pt.Scale)
			vector.StrokeRect(screen, rx, ry, rw, rh, 1,
				color.RGBA{G: 220, A: 220}, false)
		}
	}
}

// ── Border ring + triangle clusters ──────────────────────────────────────────
//
// PopulateBorderTiles does two things:
//
//  1. BORDER RING — a single continuous row of trees around all four edges.
//     Trees are sized so their trunks sit on the screen boundary and the
//     canopy faces inward, forming an impassable wall.
//
//  2. TRIANGLE CLUSTERS — groups of 3 trees in a triangle formation scattered
//     inside the play area, away from the barns, giving the same feel as the
//     old random bush patches.
func PopulateBorderTiles(tm *Tilemap, rng *rand.Rand) {
	const (
		borderH       = 100.0 // visual height of one border tree (world px)
		overlapFactor = 0.78  // horizontal step multiplier (<1 = overlap)

		// How far the trunk-bottom of the INNER border row is from the edge.
		// Positive = fully on screen. Must be > 0 so row 2 is visible.
		// The sprite top = trunkBottom - collOffY*scale, so even at trunkBottom=borderH
		// the canopy starts at ~0 and the trunk sits at borderH.
		innerRowOffset = borderH * 1.05 // row-2 trunk bottom this far from edge

		// Side columns: x offset of each column from the screen edge.
		// col0AnchorFrac = fraction of treeWidth that is ON screen for column 0.
		// col1AnchorFrac = same for column 1 (should be >= col0 so it's further in).
		col0VisibleFrac = 0.55 // 55% of tree width visible for outer side column
		col1VisibleFrac = 1.10 // column 2 fully on screen + slight extra inset

		// Side column vertical step as fraction of borderH.
		// 0.55 = 45% overlap between adjacent trees vertically.
		sideVStep = borderH * 0.55

		// Secondary ring: measured as "distance from nearest screen edge".
		// outerEdge = inner edge of the 2-row border band (~innerRowOffset + treeH).
		// The zone visible in the purple rectangles starts at ~120px from edge.
		secondaryOuter   = borderH * 2.2 // where secondary zone starts (from edge)
		secondaryInner   = borderH * 5.5 // where secondary zone ends (from edge)
		secondaryH       = 85.0          // tree height in secondary ring
		secondaryGridSz  = 95.0          // candidate grid spacing
		secondaryMaxProb = 0.95          // spawn prob at outer edge of secondary zone
		secondaryMinProb = 0.08          // spawn prob at inner edge (near centre)
	)

	scaleFor := func(idx int, h float64) float64 {
		return h / float64(treeDefs[idx].SrcH)
	}
	worldW := func(idx int, scale float64) float64 {
		return float64(treeDefs[idx].SrcW) * scale
	}
	pick := func() int { return rng.Intn(5) }

	// spriteTopY returns the sprite's top-left Y so its trunk bottom lands at trunkBottomY.
	spriteTopY := func(idx int, s, trunkBottomY float64) float64 {
		def := treeDefs[idx]
		return trunkBottomY - (def.CollOffY+def.CollH)*s
	}

	// placeHRow places one full horizontal row; trunkBottomY anchors the trunk.
	placeHRow := func(trunkBottomY float64) {
		x := -30.0
		for x < screenWidth+30 {
			idx := pick()
			s := scaleFor(idx, borderH) * (0.9 + rng.Float64()*0.2)
			tm.PlaceTree(idx, x, spriteTopY(idx, s, trunkBottomY), s)
			x += worldW(idx, s) * overlapFactor
		}
	}

	// ── 1. TOP EDGE — 2 rows ──────────────────────────────────────────────────
	// Row 0 (outermost): trunk bottom = 0  → roots at top edge, canopy above screen.
	// Row 1 (inner):     trunk bottom = innerRowOffset → fully visible on screen.
	placeHRow(0)
	placeHRow(innerRowOffset)

	// ── 2. BOTTOM EDGE — 2 rows ───────────────────────────────────────────────
	// Row 0 (outermost): trunk bottom = screenHeight → roots at bottom edge.
	// Row 1 (inner):     trunk bottom = screenHeight - innerRowOffset.
	placeHRow(screenHeight)
	placeHRow(screenHeight - innerRowOffset)

	// ── 3. LEFT EDGE — 2 columns ─────────────────────────────────────────────
	// Column 0: push tree left so col0VisibleFrac of its width is on screen.
	// Column 1: push further right so it's fully visible with a little inset.
	// Vertical step is sideVStep so trees overlap ~45% vertically.
	startY := -borderH * 0.2
	endY := screenHeight + borderH*0.2
	for col := 0; col < 2; col++ {
		visFrac := col0VisibleFrac
		if col == 1 {
			visFrac = col1VisibleFrac
		}
		y := startY
		for y < endY {
			idx := pick()
			s := scaleFor(idx, borderH) * (0.9 + rng.Float64()*0.2)
			tw := worldW(idx, s)
			x := -tw * (1.0 - visFrac) // negative x pushes tree left off screen
			tm.PlaceTree(idx, x, y, s)
			y += sideVStep
		}
	}

	// ── 4. RIGHT EDGE — 2 columns ────────────────────────────────────────────
	for col := 0; col < 2; col++ {
		visFrac := col0VisibleFrac
		if col == 1 {
			visFrac = col1VisibleFrac
		}
		y := startY
		for y < endY {
			idx := pick()
			s := scaleFor(idx, borderH) * (0.9 + rng.Float64()*0.2)
			tw := worldW(idx, s)
			x := screenWidth - tw*visFrac
			tm.PlaceTree(idx, x, y, s)
			y += sideVStep
		}
	}

	// ── 5. SECONDARY RING ────────────────────────────────────────────────────
	//
	// Walk a jittered grid over the whole screen. For each candidate point,
	// compute its distance from the nearest screen edge. Only place trees
	// inside [secondaryOuter, secondaryInner]. Spawn probability falls
	// linearly from secondaryMaxProb (at outer edge) to secondaryMinProb
	// (at inner edge), so density thins out toward the centre.
	//
	//  screen edge
	//  |--- border rows (0..secondaryOuter) ---|--- secondary zone ---|--- empty centre
	//  prob:                                    1.0 ───────────────── 0.08

	cols2 := int(math.Floor(screenWidth/secondaryGridSz)) + 2
	rows2 := int(math.Floor(screenHeight/secondaryGridSz)) + 2
	for row := 0; row < rows2; row++ {
		for col := 0; col < cols2; col++ {
			cx := float64(col)*secondaryGridSz + (rng.Float64()-0.5)*secondaryGridSz*0.7
			cy := float64(row)*secondaryGridSz + (rng.Float64()-0.5)*secondaryGridSz*0.7

			// Distance from nearest screen edge (= inward depth).
			dist := min(min(cx, screenWidth-cx), min(cy, screenHeight-cy))

			if dist < secondaryOuter || dist > secondaryInner {
				continue
			}

			// t: 0.0 at outer edge → 1.0 at inner edge.
			t := (dist - secondaryOuter) / (secondaryInner - secondaryOuter)
			prob := secondaryMaxProb - t*(secondaryMaxProb-secondaryMinProb)
			if rng.Float64() > prob {
				continue
			}

			idx := pick()
			s := scaleFor(idx, secondaryH) * (0.8 + rng.Float64()*0.4)
			tw := worldW(idx, s)
			tm.PlaceTree(idx, cx-tw/2, cy-secondaryH/2, s)
		}
	}
}
