package main

import (
	"time"
)

const speedOfLightMPS = 299792458.0

type Timestamp struct {
	SecondsSinceMidnight uint64
	Nanoseconds          uint64
}

func (t Timestamp) Seconds() float64 {
	return float64(t.SecondsSinceMidnight) + float64(t.Nanoseconds)/1e9
}

type ModeSPacket struct {
	SensorID  int64
	Latitude  float64
	Longitude float64
	Altitude  float64
	Timestamp Timestamp
	RawBytes  []byte
	Received  time.Time
}

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

type Receiver struct {
	ID        string
	SensorID  int64
	Latitude  float64
	Longitude float64
	Altitude  float64
	Position  Vec3
	LastSeen  time.Time
}

type ReceiverObservation struct {
	Receiver    *Receiver
	ReceiveTime float64
	RawBytes    []byte
	ObservedAt  time.Time
}

type AircraftEstimate struct {
	ICAO       string    `json:"icao"`
	Latitude   float64   `json:"lat"`
	Longitude  float64   `json:"lon"`
	Altitude   float64   `json:"alt"`
	Confidence string    `json:"confidence"`
	ResidualM  float64   `json:"residual_m"`
	Sensors    int       `json:"sensors"`
	UpdatedAt  time.Time `json:"updated_at"`
	RawHex     string    `json:"raw_hex"`
}

type SensorSnapshot struct {
	ID        string    `json:"id"`
	SensorID  int64     `json:"sensor_id"`
	Latitude  float64   `json:"lat"`
	Longitude float64   `json:"lon"`
	Altitude  float64   `json:"alt"`
	LastSeen  time.Time `json:"last_seen"`
}

type StatsSnapshot struct {
	ActiveSensors    int       `json:"active_sensors"`
	TrackedAircraft  int       `json:"tracked_aircraft"`
	TotalPackets     uint64    `json:"total_packets"`
	SolvedClusters   uint64    `json:"solved_clusters"`
	FailedClusters   uint64    `json:"failed_clusters"`
	LastSolutionAt   time.Time `json:"last_solution_at"`
	LastPacketIngest time.Time `json:"last_packet_ingest"`
	ServerStartedAt  time.Time `json:"server_started_at"`
}
