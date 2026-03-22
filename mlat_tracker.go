package main

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
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
	onSolve func(AircraftEstimate)
	onFail  func()
}

func NewMlatTracker(delay time.Duration, onSolve func(AircraftEstimate), onFail func()) *MlatTracker {
	return &MlatTracker{
		pending: make(map[string]*messageGroup),
		delay:   delay,
		onSolve: onSolve,
		onFail:  onFail,
	}
}

func (mt *MlatTracker) AddObservation(obs ReceiverObservation) {
	key := hex.EncodeToString(obs.RawBytes)

	mt.mu.Lock()
	group, exists := mt.pending[key]
	if !exists {
		group = &messageGroup{
			key:          key,
			rawHex:       key,
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

	position, rmseMeters, err := solveMLAT(obs)
	if err != nil {
		if mt.onFail != nil {
			mt.onFail()
		}
		log.Printf("mlat solve failed: %v", err)
		return
	}

	lat, lon, alt := ecefToLLH(position)
	if alt < -500 || alt > 20000 {
		if mt.onFail != nil {
			mt.onFail()
		}
		return
	}

	icao := decodeICAO(obs[0].RawBytes)
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

func decodeICAO(msg []byte) string {
	if len(msg) >= 4 {
		df := msg[0] >> 3
		if df == 17 || df == 18 {
			return strings.ToUpper(hex.EncodeToString(msg[1:4]))
		}
	}

	sum := sha1.Sum(msg)
	return strings.ToUpper(hex.EncodeToString(sum[:3]))
}
