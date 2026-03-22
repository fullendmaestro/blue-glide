package main

import (
	"context"
	"log"

	neuronsdk "github.com/NeuronInnovations/neuron-go-hedera-sdk"
	commonlib "github.com/NeuronInnovations/neuron-go-hedera-sdk/common-lib"

	"github.com/hashgraph/hedera-sdk-go/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func main() {
	const nrnProtocol = protocol.ID("neuron/ADSB/0.0.2")
	app, err := NewBlueGlideApp(8080, "location-override.json")
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	app.StartAPI()

	neuronsdk.LaunchSDK(
		"0.2",
		nrnProtocol,
		nil,
		func(_ context.Context, h host.Host, b *commonlib.NodeBuffers) {
			app.ConfigureBuyerHandler(h, string(nrnProtocol), b)
		},
		func(msg hedera.TopicMessage) {
			log.Printf("buyer topic message: %v", msg)
		},
		func(_ context.Context, _ host.Host, _ *commonlib.NodeBuffers) {
			// seller mode is intentionally not implemented for this challenge.
		},
		func(msg hedera.TopicMessage) {
			log.Printf("seller topic message: %v", msg)
		},
	)
}
