package main

import (
	"sort"
	"sync"
	"time"
)

type SensorManager struct {
	mu       sync.RWMutex
	sensors  map[string]SensorSnapshot
	aircraft map[string]AircraftEstimate
	stats    StatsSnapshot
	events   *EventBus
	started  time.Time
}

func NewSensorManager(events *EventBus) *SensorManager {
	now := time.Now().UTC()
	return &SensorManager{
		sensors:  make(map[string]SensorSnapshot),
		aircraft: make(map[string]AircraftEstimate),
		events:   events,
		started:  now,
		stats: StatsSnapshot{
			ServerStartedAt: now,
		},
	}
}

func (sm *SensorManager) RecordPacket(receiver *Receiver) {
	now := time.Now().UTC()
	sm.mu.Lock()
	sm.stats.TotalPackets++
	sm.stats.LastPacketIngest = now
	if receiver != nil {
		sm.sensors[receiver.ID] = SensorSnapshot{
			ID:        receiver.ID,
			SensorID:  receiver.SensorID,
			Latitude:  receiver.Latitude,
			Longitude: receiver.Longitude,
			Altitude:  receiver.Altitude,
			LastSeen:  now,
		}
		sm.stats.ActiveSensors = len(sm.sensors)
	}
	sm.mu.Unlock()

	if receiver != nil {
		sm.events.Publish(map[string]any{
			"type": "sensor",
			"id":   receiver.ID,
			"lat":  receiver.Latitude,
			"lon":  receiver.Longitude,
			"alt":  receiver.Altitude,
		})
	}
}

func (sm *SensorManager) RecordSolution(solution AircraftEstimate) {
	now := time.Now().UTC()
	solution.UpdatedAt = now

	sm.mu.Lock()
	sm.aircraft[solution.ICAO] = solution
	sm.stats.SolvedClusters++
	sm.stats.LastSolutionAt = now
	sm.stats.TrackedAircraft = len(sm.aircraft)
	sm.mu.Unlock()

	sm.events.Publish(map[string]any{
		"type":       "aircraft",
		"id":         solution.ICAO,
		"lat":        solution.Latitude,
		"lon":        solution.Longitude,
		"alt":        solution.Altitude,
		"confidence": solution.Confidence,
		"residual_m": solution.ResidualM,
		"sensors":    solution.Sensors,
		"hexData":    solution.RawHex,
	})
}

func (sm *SensorManager) RecordFailedSolve() {
	sm.mu.Lock()
	sm.stats.FailedClusters++
	sm.mu.Unlock()
}

func (sm *SensorManager) Sensors() []SensorSnapshot {
	sm.mu.RLock()
	out := make([]SensorSnapshot, 0, len(sm.sensors))
	for _, sensor := range sm.sensors {
		out = append(out, sensor)
	}
	sm.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (sm *SensorManager) Aircraft() []AircraftEstimate {
	sm.mu.RLock()
	out := make([]AircraftEstimate, 0, len(sm.aircraft))
	for _, ac := range sm.aircraft {
		out = append(out, ac)
	}
	sm.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (sm *SensorManager) Stats() StatsSnapshot {
	sm.mu.RLock()
	stats := sm.stats
	sm.mu.RUnlock()
	return stats
}
