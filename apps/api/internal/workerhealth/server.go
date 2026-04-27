// Package workerhealth exposes a small HTTP server attached to the jobworker
// and connectorworker binaries so operators can ask "is this worker alive and
// keeping up?" without parsing process logs or polling Redis directly.
//
// Endpoints (per worker, on a separate port from the API):
//   - GET /health      — liveness only ({"status":"ok"}); always public.
//   - GET /ops/health  — full snapshot (DB ping, Redis ping, queue depths,
//                        last-processed timestamps); bearer-gated outside
//                        local dev (same OPS_AUTH_TOKEN convention as the
//                        API's /ops/health, see PRODUCTION_HARDENING.md).
//
// Tracker is the optional last-processed-per-task timestamp store. Workers
// that want this signal wrap their asynq.HandlerFunc with Tracker.Wrap so a
// successful handler call records the timestamp; the snapshot then reports
// "task X last completed at Y", which is a much better stuck-worker signal
// than "queue depth is high" alone.
package workerhealth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Tracker records last successful completion timestamps per task type.
// Methods are goroutine-safe.
type Tracker struct {
	mu         sync.RWMutex
	lastByTask map[string]time.Time
}

// NewTracker returns an empty Tracker.
func NewTracker() *Tracker {
	return &Tracker{lastByTask: make(map[string]time.Time)}
}

// Mark records that taskType was processed successfully right now.
func (t *Tracker) Mark(taskType string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.lastByTask[taskType] = time.Now().UTC()
	t.mu.Unlock()
}

// Snapshot returns a copy of the per-task timestamps.
func (t *Tracker) Snapshot() map[string]time.Time {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make(map[string]time.Time, len(t.lastByTask))
	for k, v := range t.lastByTask {
		out[k] = v
	}
	return out
}

// Wrap instruments an asynq.HandlerFunc to call Mark(taskType) after the
// handler returns nil. Failures are not recorded so a stuck-but-failing task
// does not look healthy.
func (t *Tracker) Wrap(taskType string, h asynq.HandlerFunc) asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		err := h(ctx, task)
		if err == nil {
			t.Mark(taskType)
		}
		return err
	}
}

// Config wires the health server. Inspector, Queues and Tracker are optional
// (the snapshot omits whichever section is unconfigured).
type Config struct {
	WorkerName string
	Addr       string // ":9001" / ":9002"; if empty server is not started
	DB         *pgxpool.Pool
	Redis      *redis.Client
	Inspector  *asynq.Inspector
	Queues     []string
	Tracker    *Tracker
	OpsToken   string
	LocalDev   bool
	StartedAt  time.Time
}

// QueueStat is a small subset of asynq.QueueInfo for JSON output.
type QueueStat struct {
	Pending   int `json:"pending"`
	Active    int `json:"active"`
	Scheduled int `json:"scheduled"`
	Retry     int `json:"retry"`
	Archived  int `json:"archived"`
	Failed    int `json:"failed"`
}

// Snapshot is the JSON returned by /ops/health.
type Snapshot struct {
	Worker     string               `json:"worker"`
	Status     string               `json:"status"`
	UptimeSec  int64                `json:"uptime_seconds"`
	Database   string               `json:"database"`
	Redis      string               `json:"redis,omitempty"`
	Queues     map[string]QueueStat `json:"queues,omitempty"`
	LastByTask map[string]string    `json:"last_processed_by_task,omitempty"`
}

// Start launches the health HTTP server in a goroutine and returns a shutdown
// function. If cfg.Addr is empty the function is a no-op (operator opted out).
// Start NEVER kills the worker on listen failure — workers must keep
// processing tasks even if the health port is unavailable.
func Start(cfg Config) func(context.Context) error {
	if strings.TrimSpace(cfg.Addr) == "" {
		return func(context.Context) error { return nil }
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/ops/health", func(w http.ResponseWriter, r *http.Request) {
		if !cfg.LocalDev {
			if cfg.OpsToken == "" || !bearerAuthorized(r, cfg.OpsToken) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		writeJSON(w, http.StatusOK, BuildSnapshot(r.Context(), cfg))
	})
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("%s: health server on %s", cfg.WorkerName, cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("%s: health server: %v", cfg.WorkerName, err)
		}
	}()
	return srv.Shutdown
}

// BuildSnapshot collects the current health state.
func BuildSnapshot(ctx context.Context, cfg Config) Snapshot {
	snap := Snapshot{
		Worker:    cfg.WorkerName,
		Status:    "ok",
		UptimeSec: int64(time.Since(cfg.StartedAt).Seconds()),
	}
	if cfg.DB != nil {
		if err := cfg.DB.Ping(ctx); err != nil {
			snap.Database = err.Error()
			snap.Status = "degraded"
		} else {
			snap.Database = "ok"
		}
	} else {
		snap.Database = "unconfigured"
	}
	if cfg.Redis != nil {
		if err := cfg.Redis.Ping(ctx).Err(); err != nil {
			snap.Redis = err.Error()
			snap.Status = "degraded"
		} else {
			snap.Redis = "ok"
		}
	}
	if cfg.Inspector != nil && len(cfg.Queues) > 0 {
		snap.Queues = make(map[string]QueueStat, len(cfg.Queues))
		for _, q := range cfg.Queues {
			info, err := cfg.Inspector.GetQueueInfo(q)
			if err != nil {
				// Missing queue or inspector error: omit; do not flip status,
				// because empty queues are often not yet created.
				continue
			}
			snap.Queues[q] = QueueStat{
				Pending:   info.Pending,
				Active:    info.Active,
				Scheduled: info.Scheduled,
				Retry:     info.Retry,
				Archived:  info.Archived,
				Failed:    info.Failed,
			}
		}
	}
	if cfg.Tracker != nil {
		raw := cfg.Tracker.Snapshot()
		if len(raw) > 0 {
			snap.LastByTask = make(map[string]string, len(raw))
			for k, v := range raw {
				snap.LastByTask[k] = v.Format(time.RFC3339)
			}
		}
	}
	return snap
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Best-effort error log; the response header is already written.
		fmt.Fprintf(w, `{"error":%q}`, err.Error())
	}
}

func bearerAuthorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if len(got) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
