package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"planner_bot/pkg/store"
)

// NotifyFunc is called when a reminder is due. Implementations should handle
// displaying the notification to the user (toast, terminal, etc.).
type NotifyFunc func(reminder *store.Reminder)

// Scheduler polls the store every tick interval and fires notifications for due reminders.
type Scheduler struct {
	store    *store.Store
	interval time.Duration
	notify   NotifyFunc
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// New creates a new Scheduler with the given store, poll interval, and notification function.
// If interval is 0, it defaults to 1 minute.
func New(s *store.Store, interval time.Duration, notify NotifyFunc) *Scheduler {
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	if notify == nil {
		notify = func(r *store.Reminder) {
			log.Printf("🔔 REMINDER: %s (due: %s)", r.Task, r.FireAt.Local().Format("3:04 PM"))
		}
	}
	return &Scheduler{
		store:    s,
		interval: interval,
		notify:   notify,
	}
}

// Start begins the polling loop in a background goroutine.
// It immediately checks for missed reminders from when the device was asleep.
func (sc *Scheduler) Start(ctx context.Context) error {
	sc.mu.Lock()
	if sc.running {
		sc.mu.Unlock()
		return fmt.Errorf("scheduler: already running")
	}
	sc.running = true

	childCtx, cancel := context.WithCancel(ctx)
	sc.cancel = cancel
	sc.mu.Unlock()

	// Check for missed reminders immediately on startup
	sc.checkMissedReminders()

	sc.wg.Add(1)
	go sc.pollLoop(childCtx)

	return nil
}

// Stop gracefully stops the polling loop and waits for it to finish.
func (sc *Scheduler) Stop() {
	sc.mu.Lock()
	if !sc.running {
		sc.mu.Unlock()
		return
	}
	sc.running = false
	sc.mu.Unlock()

	if sc.cancel != nil {
		sc.cancel()
	}
	sc.wg.Wait()
}

// IsRunning returns whether the scheduler is currently active.
func (sc *Scheduler) IsRunning() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.running
}

// pollLoop runs the ticker and checks for due reminders on each tick.
func (sc *Scheduler) pollLoop(ctx context.Context) {
	defer sc.wg.Done()

	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.checkAndFire()
		}
	}
}

// checkAndFire queries the store for due reminders and fires notifications.
func (sc *Scheduler) checkAndFire() {
	now := time.Now().UTC()

	due, err := sc.store.ListDue(now)
	if err != nil {
		log.Printf("scheduler: error listing due reminders: %v", err)
		return
	}

	for _, r := range due {
		// Fire the notification
		sc.notify(r)

		// Mark as fired
		if err := sc.store.MarkFired(r.ID, now); err != nil {
			log.Printf("scheduler: error marking reminder %s as fired: %v", r.ID, err)
		}
	}
}

// checkMissedReminders finds reminders that were due while the app was not running.
// These are marked as "missed" and immediately notified.
func (sc *Scheduler) checkMissedReminders() {
	now := time.Now().UTC()

	due, err := sc.store.ListDue(now)
	if err != nil {
		log.Printf("scheduler: error checking missed reminders: %v", err)
		return
	}

	for _, r := range due {
		// If the reminder was due more than 2 minutes ago, it was missed
		if now.Sub(r.FireAt) > 2*time.Minute {
			_ = sc.store.MarkMissed(r.ID)
			log.Printf("⚠️  MISSED REMINDER: %s (was due at %s)", r.Task, r.FireAt.Local().Format("3:04 PM"))
		}

		// Fire the notification regardless (alert on wake)
		sc.notify(r)

		if err := sc.store.MarkFired(r.ID, now); err != nil {
			log.Printf("scheduler: error marking missed reminder %s as fired: %v", r.ID, err)
		}
	}
}

// ForceCheck triggers an immediate check for due reminders outside the normal tick cycle.
func (sc *Scheduler) ForceCheck() {
	sc.checkAndFire()
}
