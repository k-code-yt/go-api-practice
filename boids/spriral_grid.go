package main

import "math"

type SpiralGrid struct {
	cells    [][]int
	rows     int
	cols     int
	cellSize float64
}

func NewSpiralGrid(cellSize float64) *SpiralGrid {
	cols := int(math.Ceil(screenWidth / cellSize))
	rows := int(math.Ceil(screenHeight / cellSize))

	cells := make([][]int, cols*rows)
	return &SpiralGrid{
		cells:    cells,
		cols:     cols,
		rows:     rows,
		cellSize: cellSize,
	}
}

func (sg *SpiralGrid) Insert(b *Boid) {
	idx := sg.GetCellIndex(b.position.x, b.position.y)
	// TODO -> why I got outof range here?
	sg.cells[idx] = append(sg.cells[idx], b.id)
}

func (sg *SpiralGrid) Clean() {
	for i := range sg.cells {
		sg.cells[i] = sg.cells[i][:0] // -> keep cap, remove len
	}
}

func (sg *SpiralGrid) GetCellIndex(x, y float64) int {
	col := int(x / sg.cellSize)
	row := int(y / sg.cellSize)
	return row*sg.cols + col
}

func (sg *SpiralGrid) GetNeighbours(b *Boid, results *[]int) {
	bCol := int(b.position.x / sg.cellSize)
	bRow := int(b.position.y / sg.cellSize)

	for nc := -1; nc <= 1; nc++ {
		for nr := -1; nr <= 1; nr++ {
			col := bCol + nc
			row := bRow + nr

			if col < 0 || row < 0 || col >= sg.cols || row >= sg.rows {
				continue
			}

			cellIdx := row*sg.cols + col
			for _, boidIdx := range sg.cells[cellIdx] {
				*results = append(*results, boidIdx)
			}
		}
	}
}
