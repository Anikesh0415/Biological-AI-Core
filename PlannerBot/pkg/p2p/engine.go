package p2p

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"planner_bot/pkg/store"
)

// EngineConfig configures the high-level P2P synchronization orchestrator.
type EngineConfig struct {
	DeviceID    string
	DeviceName  string
	UserCode    string // Shared secret pairing code (e.g. "PLAN-ABCD-1234")
	BeaconPort  int    // Defaults to DefaultBeaconPort (42424)
	Store       *store.Store
	SyncInterval time.Duration // Defaults to 30s
}

// Engine coordinates peer discovery, local server hosting, and periodic background sync.
type Engine struct {
	deviceID     string
	deviceName   string
	userCode     string
	store        *store.Store
	server       *SyncServer
	client       *SyncClient
	discovery    *Discovery
	syncInterval time.Duration
	lastSyncTime time.Time
	mu           sync.Mutex
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
}

// NewEngine creates a new P2P sync engine.
func NewEngine(cfg EngineConfig) (*Engine, error) {
	if cfg.DeviceID == "" {
		return nil, fmt.Errorf("p2p: DeviceID is required")
	}
	if cfg.UserCode == "" {
		return nil, fmt.Errorf("p2p: UserCode is required")
	}
	if cfg.Store == nil {
		return nil, fmt.Errorf("p2p: Store is required")
	}
	if cfg.SyncInterval <= 0 {
		cfg.SyncInterval = 30 * time.Second
	}

	server, err := NewSyncServer(cfg.DeviceID, cfg.UserCode, cfg.Store)
	if err != nil {
		return nil, fmt.Errorf("create sync server: %w", err)
	}

	client := NewSyncClient(cfg.DeviceID, cfg.UserCode, cfg.Store)

	engine := &Engine{
		deviceID:     cfg.DeviceID,
		deviceName:   cfg.DeviceName,
		userCode:     cfg.UserCode,
		store:        cfg.Store,
		server:       server,
		client:       client,
		syncInterval: cfg.SyncInterval,
		lastSyncTime: time.Time{}, // Start by syncing everything
	}

	return engine, nil
}

// Start boots the sync server, sets up discovery, and runs the sync loop.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("p2p: engine already running")
	}
	e.running = true
	childCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.mu.Unlock()

	// 1. Start HTTP sync server on auto-assigned port
	if err := e.server.Start(); err != nil {
		return fmt.Errorf("start sync server: %w", err)
	}

	// 2. Initialize encrypted UDP discovery
	disc, err := NewDiscovery(DiscoveryConfig{
		DeviceID:   e.deviceID,
		DeviceName: e.deviceName,
		SyncPort:   e.server.Port(),
		UserCode:   e.userCode,
		OnPeerFound: func(p *Peer) {
			// Trigger immediate sync when a new peer appears!
			go func() {
				syncCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				_, _ = e.SyncWithPeer(syncCtx, p)
			}()
		},
	})
	if err != nil {
		_ = e.server.Close()
		return fmt.Errorf("initialize discovery: %w", err)
	}
	e.discovery = disc

	// 3. Start discovery service
	if err := e.discovery.Start(childCtx); err != nil {
		// Even if UDP broadcast fails on restricted networks, the sync server is still functional for direct connection
		log.Printf("p2p: discovery start warning: %v", err)
	}

	// 4. Start periodic sync loop
	e.wg.Add(1)
	go e.syncLoop(childCtx)

	return nil
}

// SyncWithPeer performs an immediate sync session with a specific peer.
func (e *Engine) SyncWithPeer(ctx context.Context, p *Peer) (int, error) {
	e.mu.Lock()
	lastSync := e.lastSyncTime
	e.mu.Unlock()

	changes, err := e.client.SyncWithPeer(ctx, p, lastSync)
	if err != nil {
		return 0, err
	}

	e.mu.Lock()
	e.lastSyncTime = time.Now().UTC()
	e.mu.Unlock()

	return changes, nil
}

// SyncNow triggers a synchronization pass across all currently discovered peers.
func (e *Engine) SyncNow(ctx context.Context) (int, error) {
	peers := e.Peers()
	if len(peers) == 0 {
		return 0, nil
	}

	totalChanges := 0
	for _, p := range peers {
		changes, err := e.SyncWithPeer(ctx, p)
		if err != nil {
			log.Printf("p2p: sync with peer %s (%s:%d) error: %v", p.Name, p.IP, p.Port, err)
			continue
		}
		totalChanges += changes
	}

	return totalChanges, nil
}

// Peers returns a list of actively discovered peers on the local network.
func (e *Engine) Peers() []*Peer {
	if e.discovery == nil {
		return nil
	}
	return e.discovery.GetPeers()
}

// Port returns the local sync server's listening port.
func (e *Engine) Port() int {
	if e.server == nil {
		return 0
	}
	return e.server.Port()
}

// PairingCode returns the user code used by this engine.
func (e *Engine) PairingCode() string {
	return e.userCode
}

// Close gracefully stops the sync engine, discovery, and HTTP server.
func (e *Engine) Close() error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = false
	e.mu.Unlock()

	if e.cancel != nil {
		e.cancel()
	}
	if e.discovery != nil {
		e.discovery.Stop()
	}
	if e.server != nil {
		_ = e.server.Close()
	}

	e.wg.Wait()
	return nil
}

// syncLoop periodically triggers a sync across all active peers.
func (e *Engine) syncLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			_, _ = e.SyncNow(syncCtx)
			cancel()
		}
	}
}
