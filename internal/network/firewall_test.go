package network

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockExec struct {
	mu     sync.Mutex
	calls  [][]string
	output string
	err    error

	outputs map[string]string
	errs    map[string]error
	fn      func(args []string) ([]byte, error)
}

func (m *mockExec) mockExecFunc(_ context.Context, args ...string) ([]byte, error) {
	m.mu.Lock()
	m.calls = append(m.calls, append([]string(nil), args...))
	m.mu.Unlock()

	if m.fn != nil {
		return m.fn(args)
	}
	key := strings.Join(args, " ")
	if err, ok := m.errs[key]; ok {
		return nil, err
	}
	if out, ok := m.outputs[key]; ok {
		return []byte(out), nil
	}
	if m.err != nil {
		return nil, m.err
	}
	return []byte(m.output), nil
}

func (m *mockExec) snapshot() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	dup := make([][]string, len(m.calls))
	for i, call := range m.calls {
		dup[i] = append([]string(nil), call...)
	}
	return dup
}

func TestFirewallWhitelist(t *testing.T) {
	tmp := t.TempDir()
	me := &mockExec{}
	fw := NewFirewall("abc123def456", "br-test", tmp, me.mockExecFunc)

	err := fw.Apply(context.Background(), RuleSet{
		Mode:       "whitelist",
		AllowedIPs: map[string]string{"1.2.3.4": "api.example.com"},
		DeniedIPs:  map[string]string{"9.9.9.9": "deny.example.com"},
	})
	require.NoError(t, err)

	calls := me.snapshot()
	require.Len(t, calls, 5)
	assert.Equal(t, []string{"-I", "FORWARD", "1", "-i", "br-test", "-d", "127.0.0.11", "-p", "udp", "--dport", "53", "-j", "ACCEPT", "-m", "comment", "--comment", "zone-abc123def456"}, calls[0])
	assert.Equal(t, []string{"-I", "FORWARD", "1", "-i", "br-test", "-d", "127.0.0.11", "-p", "tcp", "--dport", "53", "-j", "ACCEPT", "-m", "comment", "--comment", "zone-abc123def456"}, calls[1])
	assert.Contains(t, calls[2], "1.2.3.4")
	assert.Contains(t, calls[2], "ACCEPT")
	assert.Contains(t, calls[3], "9.9.9.9")
	assert.Contains(t, calls[3], "DROP")
	assert.Equal(t, []string{"-A", "FORWARD", "-i", "br-test", "-j", "DROP", "-m", "comment", "--comment", "zone-abc123def456"}, calls[4])
	assert.Equal(t, RuleSet{Mode: "whitelist", AllowedIPs: map[string]string{"1.2.3.4": "api.example.com"}, DeniedIPs: map[string]string{"9.9.9.9": "deny.example.com"}}, fw.rules)

	cache, err := os.ReadFile(filepath.Join(tmp, "firewall.rules"))
	require.NoError(t, err)
	assert.Contains(t, string(cache), "127.0.0.11")
	assert.Contains(t, string(cache), "1.2.3.4")
	assert.Contains(t, string(cache), "9.9.9.9")
	assert.Contains(t, string(cache), "-A FORWARD")
}

func TestFirewallBlocklist(t *testing.T) {
	me := &mockExec{}
	fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
	err := fw.Apply(context.Background(), RuleSet{
		Mode:      "blocklist",
		DeniedIPs: map[string]string{"5.6.7.8": "evil.com"},
	})
	require.NoError(t, err)
	calls := me.snapshot()
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"-I", "FORWARD", "1", "-i", "br-test", "-d", "5.6.7.8", "-j", "DROP", "-m", "comment", "--comment", "zone-abc123def456"}, calls[0])
}

func TestFirewallModeNone(t *testing.T) {
	me := &mockExec{}
	fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
	require.NoError(t, fw.Apply(context.Background(), RuleSet{Mode: "none"}))
	assert.Empty(t, me.snapshot())
}

func TestFirewallRemove(t *testing.T) {
	me := &mockExec{output: strings.Join([]string{
		`-I FORWARD 1 -i br-test -d 1.2.3.4 -j ACCEPT -m comment --comment "zone-abc123def456"`,
		`-A FORWARD -i br-test -j DROP -m comment --comment "zone-abc123def456"`,
	}, "\n")}
	fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
	require.NoError(t, fw.Remove(context.Background()))
	calls := me.snapshot()
	require.Len(t, calls, 3)
	assert.Equal(t, []string{"-S"}, calls[0])
	assert.Equal(t, []string{"-D", "FORWARD", "1", "-i", "br-test", "-d", "1.2.3.4", "-j", "ACCEPT", "-m", "comment", "--comment", "zone-abc123def456"}, calls[1])
	assert.Equal(t, []string{"-D", "FORWARD", "-i", "br-test", "-j", "DROP", "-m", "comment", "--comment", "zone-abc123def456"}, calls[2])
}

func TestRuleTagging(t *testing.T) {
	me := &mockExec{}
	fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
	require.NoError(t, fw.Apply(context.Background(), RuleSet{
		Mode:       "whitelist",
		AllowedIPs: map[string]string{"1.2.3.4": "api.example.com"},
		DeniedIPs:  map[string]string{"5.6.7.8": "evil.com"},
	}))
	for _, call := range me.snapshot() {
		assert.Contains(t, strings.Join(call, " "), `-m comment --comment zone-abc123def456`)
	}
}

func TestFirewallRulesCache(t *testing.T) {
	tmp := t.TempDir()
	me := &mockExec{}
	fw := NewFirewall("abc123def456", "br-test", tmp, me.mockExecFunc)
	require.NoError(t, fw.Apply(context.Background(), RuleSet{
		Mode:       "whitelist",
		AllowedIPs: map[string]string{"1.2.3.4": "api.example.com"},
	}))
	content, err := os.ReadFile(filepath.Join(tmp, "firewall.rules"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "zone firewall rules for zone-abc123def456")
	assert.Contains(t, string(content), "api.example.com")
	assert.Contains(t, string(content), "sudo iptables")
}

func TestRuleEvalOrder(t *testing.T) {
	me := &mockExec{}
	fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
	require.NoError(t, fw.Apply(context.Background(), RuleSet{
		Mode:       "whitelist",
		AllowedIPs: map[string]string{"1.2.3.4": "api.example.com"},
		DeniedIPs:  map[string]string{"5.6.7.8": "evil.com"},
	}))
	calls := me.snapshot()
	assert.Equal(t, "-I", calls[2][0])
	assert.Contains(t, calls[2], "ACCEPT")
	assert.Equal(t, "-I", calls[3][0])
	assert.Contains(t, calls[3], "DROP")
	assert.Equal(t, []string{"-A", "FORWARD", "-i", "br-test", "-j", "DROP", "-m", "comment", "--comment", "zone-abc123def456"}, calls[4])
}

func TestCleanStaleRules(t *testing.T) {
	iptablesDump := strings.Join([]string{
		`-I FORWARD 1 -i br-a -d 1.1.1.1 -j ACCEPT -m comment --comment "zone-livehash"`,
		`-I FORWARD 1 -i br-b -d 2.2.2.2 -j DROP -m comment --comment "zone-stalehash"`,
	}, "\n")
	me := &mockExec{
		outputs: map[string]string{
			"-S": iptablesDump,
		},
	}
	require.NoError(t, CleanStaleRules(context.Background(), me.mockExecFunc, map[string]bool{"livehash": true}))
	calls := me.snapshot()
	require.Len(t, calls, 3)
	assert.Equal(t, []string{"-S"}, calls[0])
	assert.Equal(t, []string{"-S"}, calls[1])
	assert.Equal(t, []string{"-D", "FORWARD", "1", "-i", "br-b", "-d", "2.2.2.2", "-j", "DROP", "-m", "comment", "--comment", "zone-stalehash"}, calls[2])
}

func TestFirewallRefresh(t *testing.T) {
	t.Run("stops on context cancellation", func(t *testing.T) {
		prev := refreshInterval
		refreshInterval = 10 * time.Millisecond
		defer func() { refreshInterval = prev }()

		me := &mockExec{}
		fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
		fw.resolveFn = mockResolver(map[string][]string{"api.example.com": {"1.1.1.1"}})
		ctx, cancel := context.WithCancel(context.Background())
		fw.StartRefresh(ctx, &config.NetworkConfig{Mode: "whitelist", Allow: []string{"api.example.com"}})
		time.Sleep(25 * time.Millisecond)
		cancel()
		count := len(me.snapshot())
		time.Sleep(30 * time.Millisecond)
		assert.Len(t, me.snapshot(), count)
	})

	t.Run("reapply on rule diff", func(t *testing.T) {
		me := &mockExec{output: ""}
		fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
		fw.rules = RuleSet{Mode: "whitelist", AllowedIPs: map[string]string{"1.1.1.1": "api.example.com"}, DeniedIPs: map[string]string{}}
		fw.resolveFn = mockResolver(map[string][]string{"api.example.com": {"2.2.2.2"}})

		err := fw.refreshOnce(context.Background(), &config.NetworkConfig{Mode: "whitelist", Allow: []string{"api.example.com"}})
		require.NoError(t, err)
		calls := me.snapshot()
		assert.NotEmpty(t, calls)
		assert.Equal(t, []string{"-S"}, calls[0])
		assert.Equal(t, "2.2.2.2", firstCallContaining(t, calls, "2.2.2.2")[6])
	})

	t.Run("apply failure surfaces", func(t *testing.T) {
		me := &mockExec{err: errors.New("boom")}
		fw := NewFirewall("abc123def456", "br-test", "", me.mockExecFunc)
		err := fw.Apply(context.Background(), RuleSet{Mode: "blocklist", DeniedIPs: map[string]string{"5.6.7.8": "evil.com"}})
		require.Error(t, err)
	})
}

func firstCallContaining(t *testing.T, calls [][]string, needle string) []string {
	t.Helper()
	for _, call := range calls {
		for _, part := range call {
			if part == needle {
				return call
			}
		}
	}
	t.Fatalf("no call contained %q", needle)
	return nil
}
