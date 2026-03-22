package main

import (
	"math"
	"testing"
)

func TestSolveMLAT(t *testing.T) {
	sensors := []*Receiver{
		{ID: "s1", Position: llhToECEF(50.12993, -5.51370, 56.3)},
		{ID: "s2", Position: llhToECEF(50.10248, -5.66817, 184.5)},
		{ID: "s3", Position: llhToECEF(50.15499, -5.65337, 177.7)},
		{ID: "s4", Position: llhToECEF(50.07404, -5.62037, 172.7)},
	}

	truth := llhToECEF(50.11000, -5.59000, 9300)
	emissionT := 1000.0

	obs := make([]ReceiverObservation, 0, len(sensors))
	for _, s := range sensors {
		rangeM := distance(truth, s.Position)
		tRx := emissionT + rangeM/speedOfLightMPS
		obs = append(obs, ReceiverObservation{Receiver: s, ReceiveTime: tRx})
	}

	solved, rmseM, err := solveMLAT(obs)
	if err != nil {
		t.Fatalf("solveMLAT returned error: %v", err)
	}
	if rmseM > 1 {
		t.Fatalf("expected residual <= 1m, got %.3f", rmseM)
	}

	lat, lon, alt := ecefToLLH(solved)
	if math.Abs(lat-50.11) > 0.005 {
		t.Fatalf("lat mismatch: got %.6f", lat)
	}
	if math.Abs(lon-(-5.59)) > 0.005 {
		t.Fatalf("lon mismatch: got %.6f", lon)
	}
	if math.Abs(alt-9300) > 300 {
		t.Fatalf("alt mismatch: got %.2f", alt)
	}
}
