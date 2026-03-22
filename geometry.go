package main

import "math"

const (
	wgs84A  = 6378137.0
	wgs84F  = 1.0 / 298.257223563
	wgs84E2 = wgs84F * (2 - wgs84F)
)

func llhToECEF(latDeg, lonDeg, altMeters float64) Vec3 {
	lat := latDeg * math.Pi / 180.0
	lon := lonDeg * math.Pi / 180.0

	sinLat := math.Sin(lat)
	cosLat := math.Cos(lat)
	sinLon := math.Sin(lon)
	cosLon := math.Cos(lon)

	n := wgs84A / math.Sqrt(1-wgs84E2*sinLat*sinLat)

	x := (n + altMeters) * cosLat * cosLon
	y := (n + altMeters) * cosLat * sinLon
	z := (n*(1-wgs84E2) + altMeters) * sinLat

	return Vec3{X: x, Y: y, Z: z}
}

func ecefToLLH(pos Vec3) (float64, float64, float64) {
	lon := math.Atan2(pos.Y, pos.X)
	p := math.Sqrt(pos.X*pos.X + pos.Y*pos.Y)
	lat := math.Atan2(pos.Z, p*(1-wgs84E2))

	for i := 0; i < 6; i++ {
		sinLat := math.Sin(lat)
		n := wgs84A / math.Sqrt(1-wgs84E2*sinLat*sinLat)
		lat = math.Atan2(pos.Z+wgs84E2*n*sinLat, p)
	}

	sinLat := math.Sin(lat)
	n := wgs84A / math.Sqrt(1-wgs84E2*sinLat*sinLat)
	alt := p/math.Cos(lat) - n

	return lat * 180 / math.Pi, lon * 180 / math.Pi, alt
}

func distance(a, b Vec3) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
