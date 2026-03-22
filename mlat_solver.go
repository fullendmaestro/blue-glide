package main

import (
	"errors"
	"math"
)

func solveMLAT(observations []ReceiverObservation) (Vec3, float64, error) {
	if len(observations) < 3 {
		return Vec3{}, 0, errors.New("need at least 3 observations")
	}

	for _, obs := range observations {
		if obs.Receiver == nil {
			return Vec3{}, 0, errors.New("missing receiver")
		}
	}

	x := averagePosition(observations)
	t0 := initialTransmitTime(observations, x)
	state := [4]float64{x.X, x.Y, x.Z, t0}

	var rmse float64
	for i := 0; i < 25; i++ {
		jtj, jtr, nextRMSE := normalEquations(observations, state)
		rmse = nextRMSE
		delta, ok := solve4x4(jtj, jtr)
		if !ok {
			return Vec3{}, 0, errors.New("ill-conditioned system")
		}

		state[0] -= delta[0]
		state[1] -= delta[1]
		state[2] -= delta[2]
		state[3] -= delta[3]

		step := math.Abs(delta[0]) + math.Abs(delta[1]) + math.Abs(delta[2]) + math.Abs(delta[3])*speedOfLightMPS
		if step < 0.05 {
			break
		}
	}

	return Vec3{X: state[0], Y: state[1], Z: state[2]}, rmse, nil
}

func averagePosition(observations []ReceiverObservation) Vec3 {
	var sumX, sumY, sumZ float64
	for _, obs := range observations {
		sumX += obs.Receiver.Position.X
		sumY += obs.Receiver.Position.Y
		sumZ += obs.Receiver.Position.Z
	}
	n := float64(len(observations))
	return Vec3{X: sumX / n, Y: sumY / n, Z: sumZ / n}
}

func initialTransmitTime(observations []ReceiverObservation, start Vec3) float64 {
	minT := observations[0].ReceiveTime
	var avgRange float64
	for _, obs := range observations {
		if obs.ReceiveTime < minT {
			minT = obs.ReceiveTime
		}
		avgRange += distance(start, obs.Receiver.Position)
	}
	avgRange /= float64(len(observations))
	return minT - avgRange/speedOfLightMPS
}

func normalEquations(observations []ReceiverObservation, state [4]float64) ([4][4]float64, [4]float64, float64) {
	var jtj [4][4]float64
	var jtr [4]float64

	x := Vec3{X: state[0], Y: state[1], Z: state[2]}
	t0 := state[3]
	var sumSq float64

	for _, obs := range observations {
		receiverPos := obs.Receiver.Position
		rangeMeters := distance(x, receiverPos)
		if rangeMeters < 1 {
			rangeMeters = 1
		}

		pred := rangeMeters/speedOfLightMPS + t0
		residual := pred - obs.ReceiveTime
		sumSq += residual * residual

		jx := (x.X - receiverPos.X) / (speedOfLightMPS * rangeMeters)
		jy := (x.Y - receiverPos.Y) / (speedOfLightMPS * rangeMeters)
		jz := (x.Z - receiverPos.Z) / (speedOfLightMPS * rangeMeters)
		jt := 1.0

		jac := [4]float64{jx, jy, jz, jt}

		for r := 0; r < 4; r++ {
			jtr[r] += jac[r] * residual
			for c := 0; c < 4; c++ {
				jtj[r][c] += jac[r] * jac[c]
			}
		}
	}

	for i := 0; i < 4; i++ {
		jtj[i][i] += 1e-9
	}

	rmseSeconds := math.Sqrt(sumSq / float64(len(observations)))
	return jtj, jtr, rmseSeconds * speedOfLightMPS
}

func solve4x4(a [4][4]float64, b [4]float64) ([4]float64, bool) {
	aug := [4][5]float64{}
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			aug[r][c] = a[r][c]
		}
		aug[r][4] = b[r]
	}

	for col := 0; col < 4; col++ {
		pivot := col
		for r := col + 1; r < 4; r++ {
			if math.Abs(aug[r][col]) > math.Abs(aug[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(aug[pivot][col]) < 1e-14 {
			return [4]float64{}, false
		}
		if pivot != col {
			aug[col], aug[pivot] = aug[pivot], aug[col]
		}

		div := aug[col][col]
		for c := col; c < 5; c++ {
			aug[col][c] /= div
		}

		for r := 0; r < 4; r++ {
			if r == col {
				continue
			}
			factor := aug[r][col]
			for c := col; c < 5; c++ {
				aug[r][c] -= factor * aug[col][c]
			}
		}
	}

	return [4]float64{aug[0][4], aug[1][4], aug[2][4], aug[3][4]}, true
}
