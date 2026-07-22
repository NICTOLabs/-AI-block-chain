package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

const (
	BlocksTopic = "/chain/blocks/1.0.0"
	TxsTopic    = "/chain/txs/1.0.0"
	AgentTopic  = "/chain/agent/1.0.0"
)

// PeerScore tracks a simple reputation score for known peers.
type PeerScore struct {
	Score int
}

// Host is a lightweight libp2p host that supports discovery, gossip, and peer scoring.
type Host struct {
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *slog.Logger
	host           host.Host
	dht            *dht.IpfsDHT
	pubsub         *pubsub.PubSub
	blocksTopic    *pubsub.Topic
	txsTopic       *pubsub.Topic
	agentTopic    *pubsub.Topic
	peerScores     map[string]int
	peerScoresMu   sync.RWMutex
	bootstrapPeers []multiaddr.Multiaddr
}

// Config carries host initialization options.
type Config struct {
	ListenAddr string
	Logger     *slog.Logger
}

// NewHost creates a libp2p host with mdns discovery, DHT bootstrap, and gossip topics.
func NewHost(ctx context.Context, cfg Config) (*Host, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(&noopWriter{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	ctx, cancel := context.WithCancel(ctx)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(cfg.ListenAddr),
		libp2p.EnableNATService(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		cancel()
		_ = h.Close()
		return nil, fmt.Errorf("create dht: %w", err)
	}

	pubsubService, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		_ = h.Close()
		return nil, fmt.Errorf("create pubsub: %w", err)
	}

	blocksTopic, err := pubsubService.Join(BlocksTopic)
	if err != nil {
		cancel()
		_ = h.Close()
		return nil, fmt.Errorf("join blocks topic: %w", err)
	}
	transactionsTopic, err := pubsubService.Join(TxsTopic)
	if err != nil {
		cancel()
		_ = h.Close()
		return nil, fmt.Errorf("join txs topic: %w", err)
	}
	agentTopic, err := pubsubService.Join(AgentTopic)
	if err != nil {
		cancel()
		_ = h.Close()
		return nil, fmt.Errorf("join agent topic: %w", err)
	}

	host := &Host{
		ctx:         ctx,
		cancel:      cancel,
		logger:      cfg.Logger,
		host:        h,
		dht:         kademliaDHT,
		pubsub:      pubsubService,
		blocksTopic: blocksTopic,
		txsTopic:    transactionsTopic,
		agentTopic:  agentTopic,
		peerScores:  make(map[string]int),
	}

	if err := host.startDiscovery(); err != nil {
		cancel()
		_ = h.Close()
		return nil, fmt.Errorf("start discovery: %w", err)
	}

	return host, nil
}

// Start begins the DHT bootstrap and topic subscriptions.
func (h *Host) Start() error {
	if err := h.dht.Bootstrap(h.ctx); err != nil {
		return fmt.Errorf("bootstrap dht: %w", err)
	}
	if err := h.dht.Close(); err != nil {
		return fmt.Errorf("close dht: %w", err)
	}
	return nil
}

// PublishBlock publishes a serialized block payload to the blocks topic.
func (h *Host) PublishBlock(payload []byte) error {
	return h.blocksTopic.Publish(h.ctx, payload)
}

// PublishTransaction publishes a serialized transaction payload to the txs topic.
func (h *Host) PublishTransaction(payload []byte) error {
	return h.txsTopic.Publish(h.ctx, payload)
}

// PublishAgentMessage publishes a serialized agent message payload to the agent topic.
func (h *Host) PublishAgentMessage(payload []byte) error {
	return h.agentTopic.Publish(h.ctx, payload)
}

// PeerIDs returns the currently connected peer IDs.
func (h *Host) PeerIDs() []peer.ID {
	return h.host.Network().Peers()
}

// ScorePeer updates a peer score, penalizing suspicious behavior.
func (h *Host) ScorePeer(peerID string, delta int) {
	h.peerScoresMu.Lock()
	defer h.peerScoresMu.Unlock()
	h.peerScores[peerID] += delta
}

// PeerScoreFor returns the current score for a peer.
func (h *Host) PeerScoreFor(peerID string) int {
	h.peerScoresMu.RLock()
	defer h.peerScoresMu.RUnlock()
	return h.peerScores[peerID]
}

// Close shuts down the host and cancels background work.
func (h *Host) Close() error {
	h.cancel()
	return h.host.Close()
}

func (h *Host) startDiscovery() error {
	service := mdns.NewMdnsService(h.host, "tender-discovery", nil)
	return service.Start()
}

// Protocols returns the supported protocols.
func (h *Host) Protocols() []protocol.ID {
	return h.host.Mux().Protocols()
}

type noopWriter struct{}

func (w *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

var _ = time.Second
