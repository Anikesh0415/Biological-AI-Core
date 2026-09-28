package p2p

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"planner_bot/pkg/crdt"
	"planner_bot/pkg/crypto"
	"planner_bot/pkg/store"
)

// SyncServer serves the authenticated, encrypted CRDT sync API.
type SyncServer struct {
	deviceID   string
	key        []byte
	store      *store.Store
	server     *http.Server
	listener   net.Listener
	port       int
	mu         sync.Mutex
	challenges map[string]string // deviceID -> challenge
}

// NewSyncServer creates an encrypted CRDT sync server.
func NewSyncServer(deviceID string, userCode string, s *store.Store) (*SyncServer, error) {
	key := crypto.DeriveKey(userCode, nil)

	return &SyncServer{
		deviceID:   deviceID,
		key:        key,
		store:      s,
		challenges: make(map[string]string),
	}, nil
}

// Start binds to a free TCP port and begins serving requests.
func (s *SyncServer) Start() error {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("sync server: listen tcp: %w", err)
	}
	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/p2p/handshake", s.handleHandshake)
	mux.HandleFunc("/p2p/sync", s.handleSync)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		_ = s.server.Serve(ln)
	}()

	return nil
}

// Port returns the assigned listening port.
func (s *SyncServer) Port() int {
	return s.port
}

// Close gracefully stops the sync server.
func (s *SyncServer) Close() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleHandshake performs mutual authentication with a connecting peer.
func (s *SyncServer) handleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HandshakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Sign client's challenge with our derived key
	clientChallenge, err := hex.DecodeString(req.Challenge)
	if err != nil {
		http.Error(w, "Invalid challenge", http.StatusBadRequest)
		return
	}
	sig := crypto.HMACSign(clientChallenge, s.key)

	// Issue our own challenge to the client
	ourChallenge := crypto.GenerateNonceToken()
	s.mu.Lock()
	s.challenges[req.DeviceID] = ourChallenge
	s.mu.Unlock()

	resp := HandshakeResponse{
		DeviceID:  s.deviceID,
		Signature: hex.EncodeToString(sig),
		Challenge: ourChallenge,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleSync handles encrypted two-way CRDT reminder delta synchronization.
func (s *SyncServer) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	encryptedBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Read error", http.StatusBadRequest)
		return
	}

	// 1. Decrypt incoming delta with AES-256-GCM
	decrypted, err := crypto.Decrypt(encryptedBody, s.key)
	if err != nil {
		http.Error(w, "Authentication / Decryption failed", http.StatusUnauthorized)
		return
	}

	var incoming crdt.SyncDelta
	if err := json.Unmarshal(decrypted, &incoming); err != nil {
		http.Error(w, "Invalid delta JSON", http.StatusBadRequest)
		return
	}

	// 2. Apply incoming delta to local store using CRDT LWW rules
	_, err = crdt.ApplyDeltaToStore(s.store, incoming.Reminders)
	if err != nil {
		http.Error(w, "CRDT merge failed", http.StatusInternalServerError)
		return
	}

	// 3. Prepare response delta: items modified since the client's last sync timestamp
	localModifications, err := s.store.ListModifiedSince(incoming.SentAt)
	if err != nil {
		http.Error(w, "Store read error", http.StatusInternalServerError)
		return
	}

	responseDelta := crdt.SyncDelta{
		DeviceID:  s.deviceID,
		SentAt:    time.Now().UTC(),
		Reminders: localModifications,
	}

	responseJSON, err := json.Marshal(responseDelta)
	if err != nil {
		http.Error(w, "Marshal error", http.StatusInternalServerError)
		return
	}

	// 4. Encrypt response delta with AES-256-GCM
	encryptedResp, err := crypto.Encrypt(responseJSON, s.key)
	if err != nil {
		http.Error(w, "Encryption error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(encryptedResp)
}
