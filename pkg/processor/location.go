package processor

import (
	"math"
)

type Location struct {
	Latitude  float64
	Longitude float64
}

// https://community.fabric.microsoft.com/t5/Desktop/How-to-calculate-lat-long-distance/td-p/1488227#:~:text=You%20need%20Latitude%20and%20Longitude,is%20Earth%20radius%20in%20km.)
func (l Location) Distance(l2 Location) float64 {
	factor := 1 / (math.Pi / 180.0)
	return math.Acos(math.Sin(l.Latitude/factor)*math.Sin(l2.Latitude/factor)+math.Cos(l.Latitude/factor)*math.Cos(l2.Latitude/factor)*math.Cos(l2.Longitude/factor-l.Longitude/factor)) * 6371
}
