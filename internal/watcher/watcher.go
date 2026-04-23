package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apiclient "github.com/gravitational/teleport/api/client"
	apievents "github.com/gravitational/teleport/api/types/events"
	"github.com/gravitational/teleport/api/types"

	"github.com/jsabo/tsentry/internal/config"
	"github.com/jsabo/tsentry/internal/locker"
)

const (
	summaryEventType = "session.summarized"
	namespace        = "default"
	pageSize         = 100
	divider          = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

// sessionEvent holds the fields needed for display and locking. It is
// populated from either a typed API event (polling mode) or parsed JSON
// (stdin mode), so both paths share the same handle() logic.
type sessionEvent struct {
	id          string
	clusterName string
	eventTime   time.Time
	sessionID   string
	sessionType string
	username    string
	riskLevel   string
	summary     string
}

func fromTyped(s *apievents.SessionSummarized) *sessionEvent {
	return &sessionEvent{
		id:          s.GetID(),
		clusterName: s.GetClusterName(),
		eventTime:   s.GetTime(),
		sessionID:   s.SessionID,
		sessionType: s.SessionType,
		username:    s.Username,
		riskLevel:   s.RiskLevel,
		summary:     s.ShortDescription,
	}
}

// Watcher polls the Teleport audit log for session.summarized events and
// prints alerts (and optionally issues locks) for sessions above the
// configured risk threshold.
type Watcher struct {
	client *apiclient.Client // nil in stdin mode
	cfg    *config.Config
	locker *locker.Locker // nil when auto-lock is disabled or in stdin mode
}

func New(c *apiclient.Client, cfg *config.Config) *Watcher {
	w := &Watcher{client: c, cfg: cfg}
	if cfg.AutoLock {
		w.locker = locker.New(c, cfg)
	}
	return w
}

// NewStdin creates a Watcher that reads from stdin. No Teleport client or
// locker is created; auto-lock is silently disabled.
func NewStdin(cfg *config.Config) *Watcher {
	return &Watcher{cfg: cfg}
}

// Run starts the polling loop and blocks until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	cursor := time.Now().UTC().Add(-w.cfg.LookBack)
	seen := make(map[string]struct{})

	mode := "alert-only"
	if w.cfg.AutoLock {
		mode = fmt.Sprintf("auto-lock (ttl=%s)", w.cfg.LockTTL)
	}
	slog.Info("tsentry started",
		"mode", mode,
		"threshold", w.cfg.Threshold,
		"poll_interval", w.cfg.PollInterval,
		"look_back", w.cfg.LookBack,
	)

	// Run immediately, then on each tick.
	w.poll(ctx, &cursor, seen)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.poll(ctx, &cursor, seen)
		}
	}
}

func (w *Watcher) poll(ctx context.Context, cursor *time.Time, seen map[string]struct{}) {
	now := time.Now().UTC()

	evts, err := w.fetchAll(ctx, *cursor, now)
	if err != nil {
		slog.Warn("poll failed", "error", err)
		return
	}

	for _, evt := range evts {
		id := evt.GetID()
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		summ, ok := evt.(*apievents.SessionSummarized)
		if !ok {
			continue
		}
		w.handle(ctx, fromTyped(summ))
	}

	*cursor = now
}

// fetchAll pages through SearchEvents until all results for the window are collected.
func (w *Watcher) fetchAll(ctx context.Context, from, to time.Time) ([]apievents.AuditEvent, error) {
	var all []apievents.AuditEvent
	startKey := ""
	for {
		page, nextKey, err := w.client.SearchEvents(
			ctx, from, to, namespace,
			[]string{summaryEventType},
			pageSize, types.EventOrderAscending, startKey, "",
		)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if nextKey == "" {
			break
		}
		startKey = nextKey
	}
	return all, nil
}

func (w *Watcher) handle(ctx context.Context, s *sessionEvent) {
	risk := config.ParseRiskLevel(s.riskLevel)
	meetsThreshold := risk != config.RiskUnknown && risk >= w.cfg.Threshold

	t := s.eventTime
	if t.IsZero() {
		t = time.Now().UTC()
	}

	fmt.Printf("\n%s\n", divider)
	fmt.Printf("[ALERT] %s  risk=%-8s  type=%s\n",
		t.UTC().Format(time.RFC3339),
		strings.ToUpper(s.riskLevel),
		s.sessionType,
	)
	fmt.Printf("  user:    %s\n", s.username)
	fmt.Printf("  cluster: %s\n", s.clusterName)
	fmt.Printf("  session: %s\n", s.sessionID)
	if s.summary != "" {
		fmt.Printf("  summary: %s\n", indent(s.summary, "           "))
	}

	if meetsThreshold {
		fmt.Printf("\n  ▶ %s\n", locker.Command(s.username, w.cfg.LockTTL, s.sessionID))
	}
	fmt.Printf("%s\n", divider)

	if !meetsThreshold || w.locker == nil {
		return
	}

	if err := w.locker.Lock(ctx, s.username, s.sessionID); err != nil {
		slog.Error("lock failed", "user", s.username, "session", s.sessionID, "error", err)
		fmt.Printf("[ERROR] lock failed for user=%s: %v\n", s.username, err)
		return
	}
	fmt.Printf("[LOCKED] user=%-20s ttl=%s\n\n", s.username, w.cfg.LockTTL)
}

// indent makes continuation lines of multi-line text align with the first line.
func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
