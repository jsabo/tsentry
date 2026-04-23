package locker

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	apiclient "github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/types"

	"github.com/jsabo/tsentry/internal/config"
)

// nonAlnum matches any character that is not alphanumeric or a hyphen.
var nonAlnum = regexp.MustCompile(`[^a-z0-9-]`)

// Locker creates Teleport locks via the API.
type Locker struct {
	client *apiclient.Client
	cfg    *config.Config
}

func New(c *apiclient.Client, cfg *config.Config) *Locker {
	return &Locker{client: c, cfg: cfg}
}

// Lock creates a user lock that expires after cfg.LockTTL.
func (l *Locker) Lock(ctx context.Context, username, sessionID string) error {
	expiry := time.Now().UTC().Add(l.cfg.LockTTL)
	lock, err := types.NewLock(lockName(username, sessionID), types.LockSpecV2{
		Target:  types.LockTarget{User: username},
		Message: lockMessage(sessionID),
		Expires: &expiry,
	})
	if err != nil {
		return fmt.Errorf("building lock: %w", err)
	}
	return l.client.UpsertLock(ctx, lock)
}

// Command returns the equivalent tctl command for display purposes.
func Command(username string, ttl time.Duration, sessionID string) string {
	return fmt.Sprintf(
		"tctl lock --user=%s --ttl=%s --message=%q",
		username, ttl, lockMessage(sessionID),
	)
}

func lockMessage(sessionID string) string {
	return fmt.Sprintf("tsentry: HIGH risk AI session summary (session: %s)", sessionID)
}

// lockName produces a deterministic, Teleport-safe lock name from the username
// and first 8 characters of the session ID. UpsertLock is idempotent on name,
// so repeated alerts for the same session overwrite rather than stack.
func lockName(username, sessionID string) string {
	safe := nonAlnum.ReplaceAllString(strings.ToLower(username), "-")
	short := sessionID
	if len(short) > 8 {
		short = short[:8]
	}
	return fmt.Sprintf("tsentry-%s-%s", safe, short)
}
