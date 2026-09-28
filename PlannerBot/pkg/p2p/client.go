package p2p

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"planner_bot/pkg/crdt"
	"planner_bot/pkg/crypto"
	"planner_bot/pkg/store"
)

// SyncClient executes two-way encrypted CRDT sync sessions with remote peers.
type SyncClient struct {
	deviceID   string
	key        []byte
	store      *store.Store
	httpClient *http.Client
}

// NewSyncClient creates a sync client configured with the user's pairing key.
func NewSyncClient(deviceID string, userCode string, s *store.Store) *SyncClient {
	key := crypto.DeriveKey(userCode, nil)

	return &SyncClient{
		deviceID: deviceID,
		key:      key,
		store:    s,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SyncWithPeer performs mutual authentication and a full two-way CRDT sync exchange with the target peer.
func (c *SyncClient) SyncWithPeer(ctx context.Context, peer *Peer, lastSyncTime time.Time) (int, error) {
	baseURL := fmt.Sprintf("http://%s:%d", peer.IP, peer.Port)

	// Step 1: Mutual authentication handshake
	if err := c.performHandshake(ctx, baseURL); err != nil {
		return 0, fmt.Errorf("handshake failed with %s: %w", peer.ID, err)
	}

	// Step 2: Query local modifications since last sync
	localDeltas, err := c.store.ListModifiedSince(lastSyncTime)
	if err != nil {
		return 0, fmt.Errorf("list local modifications: %w", err)
	}

	deltaReq := crdt.SyncDelta{
		DeviceID:  c.deviceID,
		SentAt:    lastSyncTime,
		Reminders: localDeltas,
	}

	deltaJSON, err := json.Marshal(deltaReq)
	if err != nil {
		return 0, fmt.Errorf("marshal delta: %w", err)
	}

	// Step 3: Encrypt outgoing payload with AES-256-GCM
	encryptedPayload, err := crypto.Encrypt(deltaJSON, c.key)
	if err != nil {
		return 0, fmt.Errorf("encrypt delta payload: %w", err)
	}

	// Step 4: Transmit delta to peer
	syncURL := baseURL + "/p2p/sync"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, syncURL, bytes.NewReader(encryptedPayload))
	if err != nil {
		return 0, fmt.Errorf("create sync request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("execute sync request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("sync rejected (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Step 5: Read and decrypt peer's response delta
	encryptedResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read sync response: %w", err)
	}

	decryptedResp, err := crypto.Decrypt(encryptedResp, c.key)
	if err != nil {
		return 0, fmt.Errorf("decrypt sync response: %w", err)
	}

	var peerDelta crdt.SyncDelta
	if err := json.Unmarshal(decryptedResp, &peerDelta); err != nil {
		return 0, fmt.Errorf("unmarshal peer delta: %w", err)
	}

	// Step 6: Apply peer's mutations to local BoltDB store via CRDT rules
	mergeResult, err := crdt.ApplyDeltaToStore(c.store, peerDelta.Reminders)
	if err != nil {
		return 0, fmt.Errorf("apply peer delta: %w", err)
	}

	totalChanges := mergeResult.Added + mergeResult.Updated + mergeResult.Deletions
	return totalChanges, nil
}

// performHandshake executes challenge-response authentication with the peer.
func (c *SyncClient) performHandshake(ctx context.Context, baseURL string) error {
	ourChallenge := crypto.GenerateNonceToken()

	reqBody := HandshakeRequest{
		DeviceID:  c.deviceID,
		Challenge: ourChallenge,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/p2p/handshake", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer rejected handshake (status %d)", resp.StatusCode)
	}

	var handshakeResp HandshakeResponse
	if err := json.NewDecoder(resp.Body).Decode(&handshakeResp); err != nil {
		return err
	}

	// Verify server's signature of our challenge
	serverSig, err := hex.DecodeString(handshakeResp.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature hex: %w", err)
	}

	ourChallengeBytes, _ := hex.DecodeString(ourChallenge)
	if !crypto.HMACVerify(ourChallengeBytes, serverSig, c.key) {
		return fmt.Errorf("peer signature verification failed (invalid pairing key)")
	}

	return nil
}
