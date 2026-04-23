package watcher

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

// jsonEvent matches the JSON field names that Teleport writes to its audit log
// for session.summarized events. All fields are optional so partial JSON (e.g.
// hand-crafted test payloads) still parses without error.
type jsonEvent struct {
	EventType        string `json:"event"`
	Time             string `json:"time"`
	ID               string `json:"uid"`
	ClusterName      string `json:"cluster_name"`
	SessionID        string `json:"sid"`
	RiskLevel        string `json:"risk_level"`
	Username         string `json:"username"`
	SessionType      string `json:"session_type"`
	ShortDescription string `json:"short_description"`
}

// RunStdin reads newline-delimited JSON from r, printing an alert for every
// session.summarized event it finds. Non-matching lines are silently skipped.
// No Teleport connection is required; auto-lock is disabled in this mode.
func (w *Watcher) RunStdin(ctx context.Context, r io.Reader) error {
	slog.Info("tsentry stdin mode",
		"threshold", w.cfg.Threshold,
		"note", "auto-lock disabled in stdin mode",
	)
	fmt.Println("Reading events from stdin — press Ctrl-C to stop.")

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] != '{' {
			continue
		}

		var raw jsonEvent
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			slog.Debug("skipping non-JSON line", "error", err)
			continue
		}
		if raw.EventType != "session.summarized" {
			continue
		}

		evt := &sessionEvent{
			id:          raw.ID,
			clusterName: raw.ClusterName,
			eventTime:   parseTime(raw.Time),
			sessionID:   raw.SessionID,
			sessionType: raw.SessionType,
			username:    raw.Username,
			riskLevel:   raw.RiskLevel,
			summary:     raw.ShortDescription,
		}
		w.handle(ctx, evt)
	}

	return scanner.Err()
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
