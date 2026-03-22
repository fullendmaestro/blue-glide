package main

import (
	"math"
	"testing"
)

func TestLLHRoundTrip(t *testing.T) {
	lat := 50.10248
	lon := -5.66817
	alt := 184.5

	ecef := llhToECEF(lat, lon, alt)
	outLat, outLon, outAlt := ecefToLLH(ecef)

	if math.Abs(lat-outLat) > 1e-5 {
		t.Fatalf("lat roundtrip mismatch: want %.8f got %.8f", lat, outLat)
	}
	if math.Abs(lon-outLon) > 1e-5 {
		t.Fatalf("lon roundtrip mismatch: want %.8f got %.8f", lon, outLon)
	}
	if math.Abs(alt-outAlt) > 0.5 {
		t.Fatalf("alt roundtrip mismatch: want %.3f got %.3f", alt, outAlt)
	}
}
