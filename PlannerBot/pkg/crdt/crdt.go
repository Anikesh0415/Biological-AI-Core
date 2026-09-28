package crdt

import (
	"fmt"
	"time"

	"planner_bot/pkg/store"
)

// SyncDelta represents a batch of CRDT mutations transmitted between devices.
type SyncDelta struct {
	DeviceID  string            `json:"device_id"`
	SentAt    time.Time         `json:"sent_at"`
	Reminders []*store.Reminder `json:"reminders"`
}

// MergeResult summarizes the outcomes of a CRDT merge operation.
type MergeResult struct {
	Added     int
	Updated   int
	Deletions int
	Ignored   int
}

// ResolveConflict determines whether the incoming reminder should overwrite the existing local reminder.
// It implements Last-Write-Wins (LWW) with tombstones and deterministic tie-breaking.
func ResolveConflict(local, incoming *store.Reminder) bool {
	// Rule 1: Newer UpdatedAt timestamp strictly wins.
	if incoming.UpdatedAt.After(local.UpdatedAt) {
		return true
	}
	if incoming.UpdatedAt.Before(local.UpdatedAt) {
		return false
	}

	// Rule 2: If timestamps are equal, higher Version counter wins.
	if incoming.Version > local.Version {
		return true
	}
	if incoming.Version < local.Version {
		return false
	}

	// Rule 3: If version is also equal, Tombstones (Deleted=true) take precedence to prevent resurrection.
	if incoming.Deleted && !local.Deleted {
		return true
	}
	if !incoming.Deleted && local.Deleted {
		return false
	}

	// Rule 4: If Fired state differs, Fired=true takes precedence (reminders shouldn't un-fire).
	if incoming.Fired && !local.Fired {
		return true
	}
	if !incoming.Fired && local.Fired {
		return false
	}

	// Rule 5: Deterministic content tie-breaker (lexicographical comparison of Task).
	return incoming.Task > local.Task
}

// MergeFieldLevel merges local and incoming reminder records at the individual field level:
// - Tombstones (Deleted=true) take absolute precedence to prevent zombie resurrects.
// - Fired=true takes precedence (completions are monotonic; a completed task is never reverted).
// - Task description and FireAt schedule are merged from the newer modification.
// Returns the merged record and a boolean indicating whether any field was updated.
func MergeFieldLevel(local, incoming *store.Reminder) (*store.Reminder, bool) {
	if local == nil {
		copied := *incoming
		return &copied, true
	}
	if incoming == nil {
		copied := *local
		return &copied, false
	}

	merged := *local
	changed := false

	// Rule 1: Tombstones take absolute precedence
	if incoming.Deleted && !local.Deleted {
		merged.Deleted = true
		merged.DeletedAt = incoming.DeletedAt
		changed = true
	}

	// Rule 2: Fired state monotonicity (completed never reverts to uncompleted)
	if incoming.Fired && !local.Fired {
		merged.Fired = true
		merged.FiredAt = incoming.FiredAt
		changed = true
	}

	// Rule 3: Task & FireAt: newer timestamp or higher version wins
	if incoming.UpdatedAt.After(local.UpdatedAt) || (incoming.UpdatedAt.Equal(local.UpdatedAt) && incoming.Version > local.Version) {
		if incoming.Task != local.Task {
			merged.Task = incoming.Task
			changed = true
		}
		if !incoming.FireAt.Equal(local.FireAt) {
			merged.FireAt = incoming.FireAt
			merged.OriginalTimeExpr = incoming.OriginalTimeExpr
			changed = true
		}
		if incoming.UpdatedAt.After(merged.UpdatedAt) {
			merged.UpdatedAt = incoming.UpdatedAt
		}
		if incoming.Version > merged.Version {
			merged.Version = incoming.Version
		}
	} else if incoming.UpdatedAt.Equal(local.UpdatedAt) && incoming.Version == local.Version {
		// Tie-breaker
		if incoming.Task > local.Task {
			merged.Task = incoming.Task
			changed = true
		}
	}

	if changed {
		if merged.UpdatedAt.Before(incoming.UpdatedAt) {
			merged.UpdatedAt = incoming.UpdatedAt
		}
		if merged.Version < incoming.Version {
			merged.Version = incoming.Version
		}
	}

	return &merged, changed
}

// MergeReminders merges a slice of incoming reminders into a map of local reminders using field-level CRDT semantics.
func MergeReminders(localMap map[string]*store.Reminder, incoming []*store.Reminder) (map[string]*store.Reminder, MergeResult) {
	var res MergeResult
	result := make(map[string]*store.Reminder, len(localMap)+len(incoming))

	// Copy existing local reminders
	for k, v := range localMap {
		copied := *v
		result[k] = &copied
	}

	for _, inc := range incoming {
		if inc == nil || inc.ID == "" {
			continue
		}

		existing, exists := result[inc.ID]
		if !exists {
			// New record
			copied := *inc
			result[inc.ID] = &copied
			if inc.Deleted {
				res.Deletions++
			} else {
				res.Added++
			}
			continue
		}

		merged, changed := MergeFieldLevel(existing, inc)
		if changed {
			result[inc.ID] = merged
			if inc.Deleted && !existing.Deleted {
				res.Deletions++
			} else {
				res.Updated++
			}
		} else {
			res.Ignored++
		}
	}

	return result, res
}

// ApplyDeltaToStore applies an incoming CRDT delta directly to a BoltDB store using field-level merging.
func ApplyDeltaToStore(s *store.Store, incoming []*store.Reminder) (MergeResult, error) {
	var res MergeResult

	for _, inc := range incoming {
		if inc == nil || inc.ID == "" {
			continue
		}

		existing, err := s.Get(inc.ID)
		if err != nil {
			// Item doesn't exist locally: insert it
			if err := s.Save(inc); err != nil {
				return res, fmt.Errorf("crdt: save incoming reminder %q: %w", inc.ID, err)
			}
			if inc.Deleted {
				res.Deletions++
			} else {
				res.Added++
			}
			continue
		}

		merged, changed := MergeFieldLevel(existing, inc)
		if changed {
			if err := s.Save(merged); err != nil {
				return res, fmt.Errorf("crdt: update incoming reminder %q: %w", inc.ID, err)
			}
			if inc.Deleted && !existing.Deleted {
				res.Deletions++
			} else {
				res.Updated++
			}
		} else {
			res.Ignored++
		}
	}

	return res, nil
}
