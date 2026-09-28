package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"planner_bot/pkg/crypto"
)

// DefaultBeaconPort is the standard UDP port used for encrypted local peer discovery.
const DefaultBeaconPort = 42424

// Discovery handles encrypted UDP discovery beacons across the local Wi-Fi / network.
type Discovery struct {
	deviceID   string
	deviceName string
	syncPort   int
	beaconPort int
	key        []byte
	peers      map[string]*Peer
	mu         sync.RWMutex
	conn       *net.UDPConn
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	onPeerFound func(peer *Peer)
}

// DiscoveryConfig holds options for the discovery service.
type DiscoveryConfig struct {
	DeviceID    string
	DeviceName  string
	SyncPort    int
	BeaconPort  int
	UserCode    string
	OnPeerFound func(peer *Peer)
}

// NewDiscovery initializes the discovery service with encrypted presence broadcasting.
func NewDiscovery(cfg DiscoveryConfig) (*Discovery, error) {
	if cfg.BeaconPort <= 0 {
		cfg.BeaconPort = DefaultBeaconPort
	}
	if cfg.DeviceName == "" {
		cfg.DeviceName = "PlannerBot-" + cfg.DeviceID[:4]
	}

	key := crypto.DeriveKey(cfg.UserCode, nil)

	return &Discovery{
		deviceID:    cfg.DeviceID,
		deviceName:  cfg.DeviceName,
		syncPort:    cfg.SyncPort,
		beaconPort:  cfg.BeaconPort,
		key:         key,
		peers:       make(map[string]*Peer),
		onPeerFound: cfg.OnPeerFound,
	}, nil
}

// Start launches the UDP beacon broadcaster and listener.
func (d *Discovery) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", d.beaconPort))
	if err != nil {
		cancel()
		return fmt.Errorf("p2p: resolve UDP addr: %w", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		cancel()
		return fmt.Errorf("p2p: listen UDP on :%d: %w", d.beaconPort, err)
	}
	d.conn = conn

	// 1. Start listener goroutine
	d.wg.Add(1)
	go d.listenLoop(ctx)

	// 2. Start broadcaster goroutine (every 3 seconds)
	d.wg.Add(1)
	go d.broadcastLoop(ctx)

	// 3. Start peer cleanup goroutine (every 10 seconds)
	d.wg.Add(1)
	go d.cleanupLoop(ctx)

	return nil
}

// Stop terminates discovery listeners and broadcasters cleanly.
func (d *Discovery) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	if d.conn != nil {
		_ = d.conn.Close()
	}
	d.wg.Wait()
}

// GetPeers returns a snapshot of active discovered peers.
func (d *Discovery) GetPeers() []*Peer {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*Peer, 0, len(d.peers))
	for _, p := range d.peers {
		copied := *p
		result = append(result, &copied)
	}
	return result
}

// broadcastLoop sends encrypted UDP beacons to 255.255.255.255 every 3 seconds.
func (d *Discovery) broadcastLoop(ctx context.Context) {
	defer d.wg.Done()

	broadcastAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", d.beaconPort))
	if err != nil {
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Broadcast immediately on startup
	d.sendBeacon(broadcastAddr)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.sendBeacon(broadcastAddr)
		}
	}
}

func (d *Discovery) sendBeacon(target *net.UDPAddr) {
	payload := BeaconPayload{
		DeviceID:   d.deviceID,
		DeviceName: d.deviceName,
		Port:       d.syncPort,
		Timestamp:  time.Now().Unix(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	encrypted, err := crypto.Encrypt(data, d.key)
	if err != nil {
		return
	}

	// Send UDP packet
	_, _ = d.conn.WriteToUDP(encrypted, target)
}

// listenLoop reads incoming discovery packets and attempts AES decryption with the user's key.
func (d *Discovery) listenLoop(ctx context.Context) {
	defer d.wg.Done()

	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_ = d.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, srcAddr, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}

		// Decrypt payload. If decryption fails (different user code / unauthenticated node), silently ignore.
		plaintext, err := crypto.Decrypt(buf[:n], d.key)
		if err != nil {
			// Unauthorized packet or different user code on the network: DROP SILENTLY!
			continue
		}

		var beacon BeaconPayload
		if err := json.Unmarshal(plaintext, &beacon); err != nil {
			continue
		}

		// Don't discover ourselves
		if beacon.DeviceID == d.deviceID {
			continue
		}

		peer := &Peer{
			ID:       beacon.DeviceID,
			Name:     beacon.DeviceName,
			IP:       srcAddr.IP.String(),
			Port:     beacon.Port,
			LastSeen: time.Now(),
		}

		d.mu.Lock()
		_, isNew := d.peers[peer.ID]
		d.peers[peer.ID] = peer
		d.mu.Unlock()

		if !isNew && d.onPeerFound != nil {
			d.onPeerFound(peer)
		}
	}
}

// cleanupLoop removes stale peers not seen in the last 15 seconds.
func (d *Discovery) cleanupLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.mu.Lock()
			cutoff := time.Now().Add(-15 * time.Second)
			for id, p := range d.peers {
				if p.LastSeen.Before(cutoff) {
					delete(d.peers, id)
				}
			}
			d.mu.Unlock()
		}
	}
}
