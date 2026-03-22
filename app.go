package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	commonlib "github.com/NeuronInnovations/neuron-go-hedera-sdk/common-lib"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type BlueGlideApp struct {
	sensorManager   *SensorManager
	apiServer       *APIServer
	locationManager *LocationOverrideManager
	mlatTracker     *MlatTracker

	mu        sync.Mutex
	receivers map[string]*Receiver
}

func NewBlueGlideApp(apiPort int, overridePath string) (*BlueGlideApp, error) {
	eventBus := NewEventBus()
	sensorManager := NewSensorManager(eventBus)
	locationManager, err := NewLocationOverrideManager(overridePath)
	if err != nil {
		return nil, err
	}

	app := &BlueGlideApp{
		sensorManager:   sensorManager,
		apiServer:       NewAPIServer(apiPort, sensorManager, eventBus),
		locationManager: locationManager,
		receivers:       make(map[string]*Receiver),
	}

	app.mlatTracker = NewMlatTracker(450*time.Millisecond, app.sensorManager.RecordSolution, app.sensorManager.RecordFailedSolve)
	return app, nil
}

func (app *BlueGlideApp) StartAPI() {
	app.apiServer.Start()
}

func (app *BlueGlideApp) ConfigureBuyerHandler(h host.Host, protocolID string, buffers *commonlib.NodeBuffers) {
	h.SetStreamHandler(protocol.ID(protocolID), func(stream network.Stream) {
		peerID := stream.Conn().RemotePeer().String()
		buffers.SetStreamHandler(stream.Conn().RemotePeer(), &stream)
		app.handleStreamFromPeer(stream, peerID)
	})
}

func (app *BlueGlideApp) handleStreamFromPeer(stream network.Stream, peerID string) {
	defer stream.Close()
	log.Printf("[%s] stream established", peerID)

	for {
		packet, err := ReadAndParseModeSPacket(stream)
		if err != nil {
			if isTimeoutError(err) {
				continue
			}
			if errors.Is(err, io.EOF) {
				log.Printf("[%s] stream closed by peer", peerID)
				return
			}
			if err != context.Canceled {
				log.Printf("[%s] stream read stopped: %v", peerID, err)
			}
			return
		}

		app.locationManager.ApplyOverride(peerID, packet)
		receiver := app.getOrCreateReceiver(peerID, packet)
		receiver.LastSeen = time.Now().UTC()
		app.sensorManager.RecordPacket(receiver)

		obs := ReceiverObservation{
			Receiver:    receiver,
			ReceiveTime: packet.Timestamp.Seconds(),
			RawBytes:    packet.RawBytes,
			ObservedAt:  packet.Received,
		}
		app.mlatTracker.AddObservation(obs)
	}
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func (app *BlueGlideApp) getOrCreateReceiver(peerID string, packet *ModeSPacket) *Receiver {
	app.mu.Lock()
	defer app.mu.Unlock()

	receiver, ok := app.receivers[peerID]
	if ok {
		receiver.SensorID = packet.SensorID
		receiver.Latitude = packet.Latitude
		receiver.Longitude = packet.Longitude
		receiver.Altitude = packet.Altitude
		receiver.Position = llhToECEF(packet.Latitude, packet.Longitude, packet.Altitude)
		return receiver
	}

	receiver = &Receiver{
		ID:        peerID,
		SensorID:  packet.SensorID,
		Latitude:  packet.Latitude,
		Longitude: packet.Longitude,
		Altitude:  packet.Altitude,
		Position:  llhToECEF(packet.Latitude, packet.Longitude, packet.Altitude),
		LastSeen:  time.Now().UTC(),
	}
	app.receivers[peerID] = receiver
	return receiver
}
