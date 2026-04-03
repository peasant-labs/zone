// Package network provides host-side iptables rule generation and management.
package network

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/peasant-labs/zone/internal/config"
)

// ExecFunc executes an iptables command.
type ExecFunc func(ctx context.Context, args ...string) ([]byte, error)

var refreshInterval = 5 * time.Minute

// DefaultExecFunc runs sudo iptables with the provided arguments.
func DefaultExecFunc(ctx context.Context, args ...string) ([]byte, error) {
	cmdArgs := append([]string{"iptables"}, args...)
	cmd := exec.CommandContext(ctx, "sudo", cmdArgs...)
	return cmd.CombinedOutput()
}

// Firewall manages iptables rules for a single container bridge.
type Firewall struct {
	containerHash string
	bridgeIface   string
	cacheDir      string
	rules         RuleSet
	mu            sync.Mutex
	execFn        ExecFunc
	resolveFn     func(string) ([]string, error)
}

// NewFirewall creates a firewall manager for one zone container.
func NewFirewall(containerHash, bridgeIface, cacheDir string, execFn ExecFunc) *Firewall {
	if execFn == nil {
		execFn = DefaultExecFunc
	}
	return &Firewall{
		containerHash: containerHash,
		bridgeIface:   bridgeIface,
		cacheDir:      cacheDir,
		execFn:        execFn,
	}
}

func (f *Firewall) tag() string {
	return "zone-" + f.containerHash
}

// Apply installs iptables rules for the provided RuleSet.
func (f *Firewall) Apply(ctx context.Context, rs RuleSet) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.applyUnlocked(ctx, rs)
}

func (f *Firewall) applyUnlocked(ctx context.Context, rs RuleSet) error {
	if rs.Mode == "none" {
		f.rules = rs
		return nil
	}

	tag := f.tag()
	appliedRules := make([]string, 0, len(rs.AllowedIPs)+len(rs.DeniedIPs)+3)

	if rs.Mode == "whitelist" {
		for _, proto := range []string{"udp", "tcp"} {
			args := []string{"-I", "FORWARD", "1", "-i", f.bridgeIface, "-d", "127.0.0.11", "-p", proto, "--dport", "53", "-j", "ACCEPT", "-m", "comment", "--comment", tag}
			if _, err := f.execFn(ctx, args...); err != nil {
				return fmt.Errorf("allow Docker DNS %s: %w", proto, err)
			}
			appliedRules = append(appliedRules, "sudo iptables "+strings.Join(args, " "))
		}

		for ip, hostname := range rs.AllowedIPs {
			args := []string{"-I", "FORWARD", "1", "-i", f.bridgeIface, "-d", ip, "-j", "ACCEPT", "-m", "comment", "--comment", tag}
			if _, err := f.execFn(ctx, args...); err != nil {
				return fmt.Errorf("allow IP %s (%s): %w", ip, hostname, err)
			}
			appliedRules = append(appliedRules, fmt.Sprintf("sudo iptables %s  # %s", strings.Join(args, " "), hostname))
		}

		for ip, hostname := range rs.DeniedIPs {
			args := []string{"-I", "FORWARD", "1", "-i", f.bridgeIface, "-d", ip, "-j", "DROP", "-m", "comment", "--comment", tag}
			if _, err := f.execFn(ctx, args...); err != nil {
				return fmt.Errorf("deny IP %s (%s): %w", ip, hostname, err)
			}
			appliedRules = append(appliedRules, fmt.Sprintf("sudo iptables %s  # %s", strings.Join(args, " "), hostname))
		}

		args := []string{"-A", "FORWARD", "-i", f.bridgeIface, "-j", "DROP", "-m", "comment", "--comment", tag}
		if _, err := f.execFn(ctx, args...); err != nil {
			return fmt.Errorf("append default drop: %w", err)
		}
		appliedRules = append(appliedRules, "sudo iptables "+strings.Join(args, " "))
	}

	if rs.Mode == "blocklist" {
		for ip, hostname := range rs.DeniedIPs {
			args := []string{"-I", "FORWARD", "1", "-i", f.bridgeIface, "-d", ip, "-j", "DROP", "-m", "comment", "--comment", tag}
			if _, err := f.execFn(ctx, args...); err != nil {
				return fmt.Errorf("deny IP %s (%s): %w", ip, hostname, err)
			}
			appliedRules = append(appliedRules, fmt.Sprintf("sudo iptables %s  # %s", strings.Join(args, " "), hostname))
		}
	}

	f.rules = rs
	if err := f.writeRulesCache(appliedRules); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write firewall.rules cache: %v\n", err)
	}
	return nil
}

// Remove deletes all rules tagged for this firewall.
func (f *Firewall) Remove(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return removeRulesForHash(ctx, f.execFn, f.containerHash)
}

func removeRulesForHash(ctx context.Context, execFn ExecFunc, hash string) error {
	if execFn == nil {
		execFn = DefaultExecFunc
	}
	out, err := execFn(ctx, "-S")
	if err != nil {
		return fmt.Errorf("list iptables rules: %w", err)
	}

	tag := "zone-" + hash
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, tag) {
			continue
		}

		switch {
		case strings.HasPrefix(line, "-A "):
			line = "-D " + strings.TrimPrefix(line, "-A ")
		case strings.HasPrefix(line, "-I "):
			line = "-D " + strings.TrimPrefix(line, "-I ")
		default:
			continue
		}

		args := strings.Fields(line)
		for i := range args {
			args[i] = strings.Trim(args[i], `"`)
		}
		if _, err := execFn(ctx, args...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove rule %q: %v\n", line, err)
		}
	}
	return nil
}

var zoneHashRe = regexp.MustCompile(`zone-([a-z0-9]+)(?:\s|$|")`)

// CleanStaleRules removes rules belonging to non-running zone containers.
func CleanStaleRules(ctx context.Context, execFn ExecFunc, runningHashes map[string]bool) error {
	if execFn == nil {
		execFn = DefaultExecFunc
	}
	out, err := execFn(ctx, "-S")
	if err != nil {
		return fmt.Errorf("list iptables rules for stale cleanup: %w", err)
	}

	seen := make(map[string]bool)
	for _, match := range zoneHashRe.FindAllSubmatch(out, -1) {
		seen[string(match[1])] = true
	}

	for hash := range seen {
		if runningHashes != nil && runningHashes[hash] {
			continue
		}
		if err := removeRulesForHash(ctx, execFn, hash); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean stale rules for %s: %v\n", hash, err)
		}
	}
	return nil
}

// StartRefresh periodically re-resolves hostnames and reapplies rules on change.
func (f *Firewall) StartRefresh(ctx context.Context, cfg *config.NetworkConfig) {
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = f.refreshOnce(ctx, cfg)
			}
		}
	}()
}

func (f *Firewall) refreshOnce(ctx context.Context, cfg *config.NetworkConfig) error {
	// Suppress warnings during periodic refresh (already shown at launch time)
	old := warnWriter
	warnWriter = io.Discard
	newRules, err := BuildRuleSet(cfg, f.resolveFn)
	warnWriter = old
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if RulesEqual(f.rules, newRules) {
		return nil
	}
	if f.rules.Mode != "none" {
		if err := removeRulesForHash(ctx, f.execFn, f.containerHash); err != nil {
			return err
		}
	}
	return f.applyUnlocked(ctx, newRules)
}

// writeRulesCache writes human-readable applied rules for inspection.
func (f *Firewall) writeRulesCache(rules []string) error {
	if f.cacheDir == "" {
		return nil
	}
	content := strings.Join([]string{
		"# zone firewall rules for zone-" + f.containerHash,
		"# Generated: " + time.Now().UTC().Format(time.RFC3339),
		"# Mode: " + f.rules.Mode,
		"",
		strings.Join(rules, "\n"),
		"",
	}, "\n")
	return os.WriteFile(filepath.Join(f.cacheDir, "firewall.rules"), []byte(content), 0o644)
}

// RemoveRulesCache deletes the cached firewall.rules file.
func (f *Firewall) RemoveRulesCache() error {
	if f.cacheDir == "" {
		return nil
	}
	err := os.Remove(filepath.Join(f.cacheDir, "firewall.rules"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
