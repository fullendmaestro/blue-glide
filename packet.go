package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

const (
	sensorIDSize             = 8
	sensorLongitudeSize      = 8
	sensorLatitudeSize       = 8
	sensorAltitudeSize       = 8
	secondsSinceMidnightSize = 8
	nanosecondsSize          = 8
	minFixedSize             = sensorIDSize + sensorLatitudeSize + sensorLongitudeSize + sensorAltitudeSize + secondsSinceMidnightSize + nanosecondsSize
)

func readExact(stream network.Stream, buf []byte) error {
	var total int
	for total < len(buf) {
		n, err := stream.Read(buf[total:])
		if err != nil {
			return err
		}
		if n == 0 {
			return io.EOF
		}
		total += n
	}
	return nil
}

func float64FromByte(bytes []byte) float64 {
	bits := binary.BigEndian.Uint64(bytes)
	return math.Float64frombits(bits)
}

func int64FromByte(bytes []byte) int64 {
	bits := binary.BigEndian.Uint64(bytes)
	return int64(bits)
}

func uint64FromByte(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}

func ReadAndParseModeSPacket(stream network.Stream) (*ModeSPacket, error) {
	if err := stream.SetReadDeadline(time.Now().Add(8 * time.Second)); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}

	lengthBuf := make([]byte, 1)
	if err := readExact(stream, lengthBuf); err != nil {
		return nil, err
	}

	totalPacketSize := int(lengthBuf[0])
	if totalPacketSize == 0 {
		return nil, fmt.Errorf("empty packet")
	}

	packetBytes := make([]byte, totalPacketSize)
	if err := readExact(stream, packetBytes); err != nil {
		return nil, err
	}

	if len(packetBytes) < minFixedSize {
		return nil, fmt.Errorf("packet too short: got %d, need >= %d", len(packetBytes), minFixedSize)
	}

	offset := 0
	sensorID := int64FromByte(packetBytes[offset : offset+sensorIDSize])
	offset += sensorIDSize

	latitude := float64FromByte(packetBytes[offset : offset+sensorLatitudeSize])
	offset += sensorLatitudeSize

	longitude := float64FromByte(packetBytes[offset : offset+sensorLongitudeSize])
	offset += sensorLongitudeSize

	altitude := float64FromByte(packetBytes[offset : offset+sensorAltitudeSize])
	offset += sensorAltitudeSize

	secondsSinceMidnight := uint64FromByte(packetBytes[offset : offset+secondsSinceMidnightSize])
	offset += secondsSinceMidnightSize

	nanoseconds := uint64FromByte(packetBytes[offset : offset+nanosecondsSize])
	offset += nanosecondsSize

	raw := append([]byte(nil), packetBytes[offset:]...)
	if len(raw) == 0 {
		return nil, fmt.Errorf("packet without modes payload")
	}

	return &ModeSPacket{
		SensorID:  sensorID,
		Latitude:  latitude,
		Longitude: longitude,
		Altitude:  altitude,
		Timestamp: Timestamp{
			SecondsSinceMidnight: secondsSinceMidnight,
			Nanoseconds:          nanoseconds,
		},
		RawBytes: raw,
		Received: time.Now().UTC(),
	}, nil
}
