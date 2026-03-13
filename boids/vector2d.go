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
	return math.Sqrt(((currV.x - newV.x) * (currV.x - newV.x)) + ((currV.y - newV.y) * (currV.y - newV.y)))
}

func (currV Vector2D) Len() float64 {
	return math.Sqrt((currV.x * currV.x) + (currV.y * currV.y))
}

func (currV Vector2D) Normalize() Vector2D {
	d := currV.Len()
	if d == 0 {
		return Vector2D{0, 0}
	}

	return Vector2D{
		currV.x / d,
		currV.y / d,
	}
}

func (currV Vector2D) NormalizeByMag(mag float64) Vector2D {
	return Vector2D{
		currV.x / mag,
		currV.y / mag,
	}
}

func (currV Vector2D) LimitSpeed() Vector2D {
	mag := currV.Len()

	if mag == 0 {
		return Vector2D{minSpeed, 0}
	}

	if minSpeed > mag {
		return currV.NormalizeByMag(mag).Mul(minSpeed)
	}

	if maxSpeed < mag {
		return currV.NormalizeByMag(mag).Mul(maxSpeed)
	}

	return currV
}
