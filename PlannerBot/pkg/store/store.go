package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Bucket names for BoltDB.
var (
	bucketReminders = []byte("reminders")
	bucketFired     = []byte("fired")
	bucketMeta      = []byte("meta")
	bucketChats     = []byte("chats")
)

// Reminder represents a stored task with scheduling information.
type Reminder struct {
	// ID is a unique identifier (UUID or timestamp-based).
	ID string `json:"id"`

	// Task is the human-readable action to perform.
	Task string `json:"task"`

	// FireAt is the absolute UTC time when the reminder should trigger.
	FireAt time.Time `json:"fire_at"`

	// CronExpr is a cron expression for recurring tasks (empty = one-shot).
	CronExpr string `json:"cron_expr,omitempty"`

	// CreatedAt is when the reminder was created.
	CreatedAt time.Time `json:"created_at"`

	// Fired indicates whether the reminder has been triggered.
	Fired bool `json:"fired"`

	// FiredAt is when the reminder was actually triggered.
	FiredAt *time.Time `json:"fired_at,omitempty"`

	// Missed indicates the reminder was due while the device was asleep.
	Missed bool `json:"missed"`

	// OriginalTimeExpr stores the raw time string from the AI extraction.
	OriginalTimeExpr string `json:"original_time_expr,omitempty"`

	// UpdatedAt is the timestamp of the last modification (for CRDT LWW sync).
	UpdatedAt time.Time `json:"updated_at"`

	// Deleted marks this reminder as a tombstone for CRDT synchronization.
	Deleted bool `json:"deleted"`

	// DeletedAt records when the reminder was deleted.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// Version is an incremental mutation counter for conflict resolution.
	Version int64 `json:"version"`
}

// Store is the BoltDB-backed persistence layer for reminders.
type Store struct {
	db   *bolt.DB
	path string
}

// New opens or creates a BoltDB database at the specified path.
// If dbPath is empty, it defaults to ~/.planner_bot/reminders.db
func New(dbPath string) (*Store, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("store: could not determine home directory: %w", err)
		}
		dir := filepath.Join(home, ".planner_bot")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("store: could not create data directory: %w", err)
		}
		dbPath = filepath.Join(dir, "reminders.db")
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("store: could not create parent directory: %w", err)
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("store: could not open database at %s: %w", dbPath, err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{bucketReminders, bucketFired, bucketMeta, bucketChats, bucketMemoryNodes, bucketUserProfile} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("create bucket %s: %w", string(bucket), err)
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: could not initialize buckets: %w", err)
	}

	return &Store{db: db, path: dbPath}, nil
}

// Close closes the BoltDB database.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Path returns the file path of the database.
func (s *Store) Path() string {
	return s.path
}

// Save stores a reminder in the database. If the ID already exists, it overwrites it.
func (s *Store) Save(r *Reminder) error {
	if r.ID == "" {
		return fmt.Errorf("store: reminder ID cannot be empty")
	}

	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = time.Now().UTC()
	}
	if r.Version == 0 {
		r.Version = 1
	}

	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("store: marshal reminder: %w", err)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		return b.Put([]byte(r.ID), data)
	})
}

// Get retrieves a reminder by its ID.
func (s *Store) Get(id string) (*Reminder, error) {
	var r Reminder

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("store: reminder %q not found", id)
		}
		return json.Unmarshal(data, &r)
	})
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// Delete removes a reminder completely by its ID.
func (s *Store) Delete(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		return b.Delete([]byte(id))
	})
}

// SoftDelete marks a reminder as deleted (tombstone) with an updated timestamp so CRDT sync propagates the deletion.
func (s *Store) SoftDelete(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("store: reminder %q not found", id)
		}

		var r Reminder
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}

		now := time.Now().UTC()
		r.Deleted = true
		r.DeletedAt = &now
		r.UpdatedAt = now
		r.Version++

		updated, err := json.Marshal(&r)
		if err != nil {
			return err
		}

		return b.Put([]byte(id), updated)
	})
}

// ListPending returns all reminders that haven't been fired yet and aren't marked deleted.
func (s *Store) ListPending() ([]*Reminder, error) {
	var reminders []*Reminder

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		return b.ForEach(func(k, v []byte) error {
			var r Reminder
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if !r.Fired && !r.Deleted {
				reminders = append(reminders, &r)
			}
			return nil
		})
	})

	return reminders, err
}

// ListAll returns all reminders in the database (excluding deleted tombstones).
func (s *Store) ListAll() ([]*Reminder, error) {
	var reminders []*Reminder

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		return b.ForEach(func(k, v []byte) error {
			var r Reminder
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if !r.Deleted {
				reminders = append(reminders, &r)
			}
			return nil
		})
	})

	return reminders, err
}

// ListDue returns all reminders whose FireAt time is at or before the given time, aren't deleted, and haven't fired.
func (s *Store) ListDue(now time.Time) ([]*Reminder, error) {
	var due []*Reminder

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		return b.ForEach(func(k, v []byte) error {
			var r Reminder
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if !r.Fired && !r.Deleted && !r.FireAt.IsZero() && !r.FireAt.After(now) {
				due = append(due, &r)
			}
			return nil
		})
	})

	return due, err
}

// ListModifiedSince returns all reminders modified since the given timestamp (including tombstones) for CRDT delta sync.
func (s *Store) ListModifiedSince(since time.Time) ([]*Reminder, error) {
	var modified []*Reminder

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		return b.ForEach(func(k, v []byte) error {
			var r Reminder
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if r.UpdatedAt.After(since) || r.UpdatedAt.Equal(since) {
				modified = append(modified, &r)
			}
			return nil
		})
	})

	return modified, err
}

// MarkFired marks a reminder as fired and moves it to the fired bucket for history.
func (s *Store) MarkFired(id string, firedAt time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("store: reminder %q not found", id)
		}

		var r Reminder
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}

		r.Fired = true
		r.FiredAt = &firedAt

		updatedData, err := json.Marshal(&r)
		if err != nil {
			return err
		}

		// Update in reminders bucket
		if err := b.Put([]byte(id), updatedData); err != nil {
			return err
		}

		// Also store in fired bucket for history
		fired := tx.Bucket(bucketFired)
		return fired.Put([]byte(id), updatedData)
	})
}

// MarkMissed marks a reminder as missed (device was asleep when it was due).
func (s *Store) MarkMissed(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("store: reminder %q not found", id)
		}

		var r Reminder
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}

		r.Missed = true

		updatedData, err := json.Marshal(&r)
		if err != nil {
			return err
		}

		return b.Put([]byte(id), updatedData)
	})
}

// Count returns the total number of reminders in the database.
func (s *Store) Count() (int, error) {
	var count int

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		count = b.Stats().KeyN
		return nil
	})

	return count, err
}

// PurgeCompleted removes all fired reminders from the main bucket.
func (s *Store) PurgeCompleted() (int, error) {
	purged := 0

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketReminders)
		var toDelete [][]byte

		err := b.ForEach(func(k, v []byte) error {
			var r Reminder
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if r.Fired {
				toDelete = append(toDelete, append([]byte{}, k...))
			}
			return nil
		})
		if err != nil {
			return err
		}

		for _, key := range toDelete {
			if err := b.Delete(key); err != nil {
				return err
			}
			purged++
		}

		return nil
	})

	return purged, err
}

// ChatMessage represents a lightweight conversation record between the user and PlannerBot.
type ChatMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action,omitempty"` // "reminder_created", "analysis", "list", "general"
	Meta      string    `json:"meta,omitempty"`   // optional metadata summary (e.g. task ID or tag)
}

// SaveChat persists a single chat message to BoltDB.
// The key is formatted with timestamp prefix for natural chronological ordering in BoltDB B-tree.
func (s *Store) SaveChat(msg *ChatMessage) error {
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("store: marshal chat message: %w", err)
	}

	key := []byte(fmt.Sprintf("%s_%s", msg.Timestamp.Format("20060102150405.000000000"), msg.ID))

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketChats)
		if b == nil {
			return fmt.Errorf("store: chats bucket not found")
		}
		return b.Put(key, data)
	})
}

// ListRecentChats retrieves the most recent N chat messages in chronological order.
// If limit <= 0, returns up to 50 recent messages.
func (s *Store) ListRecentChats(limit int) ([]*ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	var messages []*ChatMessage
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketChats)
		if b == nil {
			return nil
		}

		c := b.Cursor()
		// Traverse backward from the last key to collect the latest messages
		for k, v := c.Last(); k != nil && len(messages) < limit; k, v = c.Prev() {
			var msg ChatMessage
			if err := json.Unmarshal(v, &msg); err != nil {
				continue
			}
			messages = append(messages, &msg)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reverse to restore chronological order (oldest to newest)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// ClearChats removes all conversation history.
func (s *Store) ClearChats() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(bucketChats); err != nil && err != bolt.ErrBucketNotFound {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(bucketChats)
		return err
	})
}

