package p2p

import (
	"time"
)

// Peer represents an authenticated discovered node on the local network.
type Peer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Port      int       `json:"port"`
	LastSeen  time.Time `json:"last_seen"`
}

// BeaconPayload is the unencrypted content of the periodic LAN presence beacon.
type BeaconPayload struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Port       int    `json:"port"`
	Timestamp  int64  `json:"timestamp"`
}

// HandshakeRequest initiates mutual authentication.
type HandshakeRequest struct {
	DeviceID  string `json:"device_id"`
	Challenge string `json:"challenge"`
}

// HandshakeResponse responds to client's challenge and issues server's challenge.
type HandshakeResponse struct {
	DeviceID  string `json:"device_id"`
	Signature string `json:"signature"`
	Challenge string `json:"challenge"`
}

// HandshakeAck finishes mutual authentication.
type HandshakeAck struct {
	Signature string `json:"signature"`
}
