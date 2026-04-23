package config

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// RiskLevel is an ordered numeric representation of session risk.
type RiskLevel int

const (
	RiskUnknown  RiskLevel = -1
	RiskLow      RiskLevel = 0
	RiskMedium   RiskLevel = 1
	RiskHigh     RiskLevel = 2
	RiskCritical RiskLevel = 3
)

func ParseRiskLevel(s string) RiskLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return RiskLow
	case "medium":
		return RiskMedium
	case "high":
		return RiskHigh
	case "critical":
		return RiskCritical
	default:
		return RiskUnknown
	}
}

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "LOW"
	case RiskMedium:
		return "MEDIUM"
	case RiskHigh:
		return "HIGH"
	case RiskCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Config holds all runtime configuration.
type Config struct {
	ProxyAddr     string
	Cluster       string
	PollInterval  time.Duration
	LookBack      time.Duration
	Threshold     RiskLevel
	AutoLock      bool
	LockTTL       time.Duration
	IdentityFile  string // tbot identity file; overrides tsh profile when set
	TSHProfileDir string // defaults to ~/.tsh
	Stdin         bool   // read JSON events from stdin instead of polling Teleport
}

func Parse() (*Config, error) {
	cfg := &Config{}
	var threshold string

	flag.StringVar(&cfg.ProxyAddr, "proxy", "", "Teleport proxy address (default: from active tsh profile)")
	flag.StringVar(&cfg.Cluster, "cluster", "", "Teleport cluster name (default: from active tsh profile)")
	flag.DurationVar(&cfg.PollInterval, "poll-interval", 30*time.Second, "How often to poll for new events")
	flag.DurationVar(&cfg.LookBack, "look-back", 5*time.Minute, "How far back to scan on startup")
	flag.StringVar(&threshold, "risk-threshold", "high", "Minimum risk level to alert/lock: low|medium|high|critical")
	flag.BoolVar(&cfg.AutoLock, "auto-lock", false, "Execute tctl lock for sessions above threshold (default: alert only)")
	flag.DurationVar(&cfg.LockTTL, "lock-ttl", 15*time.Minute, "Duration of lock when --auto-lock is set")
	flag.StringVar(&cfg.IdentityFile, "identity", "", "Path to tbot identity file (overrides tsh profile)")
	flag.StringVar(&cfg.TSHProfileDir, "tsh-profile-dir", "", "tsh profile directory (default: ~/.tsh)")
	flag.BoolVar(&cfg.Stdin, "stdin", false, "Read newline-delimited JSON events from stdin (no Teleport connection required)")
	flag.Parse()

	cfg.Threshold = ParseRiskLevel(threshold)
	if cfg.Threshold == RiskUnknown {
		return nil, fmt.Errorf("invalid --risk-threshold %q: must be low, medium, high, or critical", threshold)
	}

	return cfg, nil
}
