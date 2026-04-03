// errors.go defines sentinel errors for the docker package.
package docker

import "errors"

// Sentinel errors returned by Manager operations.
var (
	// ErrDockerNotRunning is returned when the Docker daemon is not reachable.
	ErrDockerNotRunning = errors.New("docker daemon is not running. Start Docker Desktop or the Docker daemon and try again")

	// ErrNoContainer is returned when an operation requires a running container but none exists.
	ErrNoContainer = errors.New("no container is running for this repo. Run `zone launch` first")

	// ErrNetworkUnsupported is returned when network sandboxing is requested on an unsupported platform.
	ErrNetworkUnsupported = errors.New("network sandboxing is not supported on this platform")

	// ErrFirewallSetup is returned when firewall rule application fails.
	ErrFirewallSetup = errors.New("failed to apply firewall rules. Check sudo permissions and iptables availability")

	// ErrSudoUnavailable is returned when sudo is not available for iptables commands.
	ErrSudoUnavailable = errors.New("sudo is not available. Network filtering requires sudo and iptables. Set [network] mode = \"none\" to suppress this warning")

	// ErrIPTablesUnavailable is returned when iptables cannot be executed.
	ErrIPTablesUnavailable = errors.New("iptables is not available. Network filtering requires sudo and iptables. Falling back to unrestricted network access. Set [network] mode = \"none\" to suppress this warning")
)

// SecurityConfig holds hardened container security settings per spec section 12.
type SecurityConfig struct {
	SecurityOpt      []string
	CapDrop          []string
	CapAdd           []string
	DefaultPidsLimit int64
}

// ContainerSecurityFlags returns the security settings applied to every zone container.
// Containers run with no-new-privileges, all capabilities dropped, and only
// CHOWN, DAC_OVERRIDE, SETGID, SETUID, FOWNER added back.
func ContainerSecurityFlags() SecurityConfig {
	return SecurityConfig{
		SecurityOpt:      []string{"no-new-privileges"},
		CapDrop:          []string{"ALL"},
		CapAdd:           []string{"CHOWN", "DAC_OVERRIDE", "SETGID", "SETUID", "FOWNER"},
		DefaultPidsLimit: 512,
	}
}
