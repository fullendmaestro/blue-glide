package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type messageGroup struct {
	key          string
	rawHex       string
	firstSeen    time.Time
	observations map[string]ReceiverObservation
	timer        *time.Timer
}

type MlatTracker struct {
	mu      sync.Mutex
	pending map[string]*messageGroup
	delay   time.Duration
	window  time.Duration
	onSolve func(AircraftEstimate)
	onFail  func()
}

func NewMlatTracker(delay time.Duration, onSolve func(AircraftEstimate), onFail func()) *MlatTracker {
	return &MlatTracker{
		pending: make(map[string]*messageGroup),
		delay:   delay,
		window:  2 * time.Millisecond,
		onSolve: onSolve,
		onFail:  onFail,
	}
}

func (mt *MlatTracker) AddObservation(obs ReceiverObservation) {
	icao, ok := decodeICAO(obs.RawBytes)
	if !ok {
		return
	}

	bucket := int64(obs.ReceiveTime / mt.window.Seconds())
	key := fmt.Sprintf("%s:%d:%s", icao, bucket, hex.EncodeToString(obs.RawBytes))

	mt.mu.Lock()
	group, exists := mt.pending[key]
	if !exists {
		rawHex := hex.EncodeToString(obs.RawBytes)
		group = &messageGroup{
			key:          key,
			rawHex:       rawHex,
			firstSeen:    obs.ObservedAt,
			observations: make(map[string]ReceiverObservation),
		}
		group.timer = time.AfterFunc(mt.delay, func() {
			mt.resolve(key)
		})
		mt.pending[key] = group
	}
	group.observations[obs.Receiver.ID] = obs
	mt.mu.Unlock()
}

func (mt *MlatTracker) resolve(key string) {
	mt.mu.Lock()
	group, exists := mt.pending[key]
	if !exists {
		mt.mu.Unlock()
		return
	}
	delete(mt.pending, key)
	mt.mu.Unlock()

	obs := make([]ReceiverObservation, 0, len(group.observations))
	for _, o := range group.observations {
		obs = append(obs, o)
	}

	if len(obs) < 3 {
		if mt.onFail != nil {
			mt.onFail()
		}
		return
	}

	sort.Slice(obs, func(i, j int) bool {
		return obs[i].ReceiveTime < obs[j].ReceiveTime
	})

	if obs[len(obs)-1].ReceiveTime-obs[0].ReceiveTime > 0.010 {
		if mt.onFail != nil {
			mt.onFail()
		}
		return
	}

	position, rmseMeters, err := solveMLAT(obs)
	if err != nil {
		if mt.onFail != nil {
			mt.onFail()
		}
		log.Printf("mlat solve failed: %v", err)
		return
	}

	lat, lon, alt := ecefToLLH(position)
	if alt < -500 || alt > 20000 || rmseMeters > 8000 {
		if mt.onFail != nil {
			mt.onFail()
		}
		return
	}

	if !withinNetworkEnvelope(position, obs) {
		if mt.onFail != nil {
			mt.onFail()
		}
		return
	}

	icao, ok := decodeICAO(obs[0].RawBytes)
	if !ok {
		if mt.onFail != nil {
			mt.onFail()
		}
		return
	}
	confidence := scoreConfidence(rmseMeters, len(obs))

	estimate := AircraftEstimate{
		ICAO:       icao,
		Latitude:   lat,
		Longitude:  lon,
		Altitude:   alt,
		Confidence: confidence,
		ResidualM:  rmseMeters,
		Sensors:    len(obs),
		RawHex:     strings.ToUpper(group.rawHex),
	}

	if mt.onSolve != nil {
		mt.onSolve(estimate)
	}
}

func scoreConfidence(rmseMeters float64, sensors int) string {
	if sensors >= 4 && rmseMeters < 350 {
		return "high"
	}
	if sensors >= 3 && rmseMeters < 1200 {
		return "medium"
	}
	return "low"
}

func decodeICAO(msg []byte) (string, bool) {
	if len(msg) != 14 {
		return "", false
	}

	df := msg[0] >> 3
	if df != 17 {
		return "", false
	}

	typeCode := msg[4] >> 3
	if typeCode < 9 || typeCode > 18 {
		return "", false
	}

	return strings.ToUpper(hex.EncodeToString(msg[1:4])), true
}

func withinNetworkEnvelope(solution Vec3, obs []ReceiverObservation) bool {
	var center Vec3
	for _, o := range obs {
		center.X += o.Receiver.Position.X
		center.Y += o.Receiver.Position.Y
		center.Z += o.Receiver.Position.Z
	}
	n := float64(len(obs))
	center.X /= n
	center.Y /= n
	center.Z /= n

	maxReceiverRadius := 0.0
	for _, o := range obs {
		r := distance(o.Receiver.Position, center)
		if r > maxReceiverRadius {
			maxReceiverRadius = r
		}
	}

	d := distance(solution, center)
	maxAllowed := math.Max(350000, maxReceiverRadius+250000)
	return d <= maxAllowed
}
