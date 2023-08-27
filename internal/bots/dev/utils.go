package dev

import (
	"crypto/rand"
	_ "embed"
	"gonum.org/v1/plot/plotter"
	"math/big"
)

// randomPoints returns some random x, y points.
func randomPoints(n int) plotter.XYs {
	pts := make(plotter.XYs, n)
	for i := range pts {
		num, _ := rand.Int(rand.Reader, big.NewInt(100))
		if i == 0 {
			pts[i].X = float64(num.Int64())
		} else {
			pts[i].X = pts[i-1].X + float64(num.Int64())
		}
		pts[i].Y = pts[i].X + 10*float64(num.Int64())
	}
	return pts
}
