package main

import "math"

type Vector2D struct {
	x float64
	y float64
}

func (currV Vector2D) Add(newV Vector2D) Vector2D {
	return Vector2D{currV.x + newV.x, currV.y + newV.y}
}

func (currV Vector2D) Sub(newV Vector2D) Vector2D {
	return Vector2D{currV.x - newV.x, currV.y - newV.y}
}

func (currV Vector2D) Mul(d float64) Vector2D {
	return Vector2D{currV.x * d, currV.y * d}
}

func (currV Vector2D) Div(d float64) Vector2D {
	return Vector2D{currV.x / d, currV.y / d}
}

func (currV Vector2D) AddVal(d float64) Vector2D {
	return Vector2D{currV.x + d, currV.y + d}
}

func (currV Vector2D) SubVal(d float64) Vector2D {
	return Vector2D{currV.x - d, currV.y - d}
}

func (currV Vector2D) Limit(lower, upper float64) Vector2D {
	return Vector2D{math.Min(math.Max(currV.x, lower), upper), math.Min(math.Max(currV.y, lower), upper)}
}

func (currV Vector2D) Distance(newV Vector2D) float64 {
	return math.Sqrt(math.Pow(currV.x-newV.x, 2) + math.Pow(currV.y-newV.y, 2))
}
