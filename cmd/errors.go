package cmd

import (
	"errors"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
)

// mapError maps a sentinel error to a user-facing remediation message and exit code.
// Exit codes per spec section 3.3:
//   - 0 = success
//   - 1 = generic/unknown
//   - 2 = config error
//   - 3 = Docker error
//   - 4 = network error
//   - 5 = cache/lock error
//   - 6 = no container
func mapError(err error) (string, int) {
	if err == nil {
		return "", 0
	}

	switch {
	case errors.Is(err, docker.ErrNoContainer):
		return "Error: No running zone container for this repo.\n\n" +
			"  Run `zone launch` to start one, or `zone ls` to see all containers.", 6

	case errors.Is(err, cache.ErrLockContention):
		return "Error: Another zone process is operating on this repo.\n\n" +
			"  Wait for the other process to finish, or run `zone clean` if it crashed.", 5

	case errors.Is(err, docker.ErrFirewallSetup):
		return "Error: Failed to apply network firewall rules.\n\n" +
			"  Check that `sudo iptables -L -n` works on this machine.\n" +
			"  If sudo requires a password, configure NOPASSWD for iptables.\n" +
			"  Or set [network] mode = \"none\" to disable filtering.", 4

	case errors.Is(err, docker.ErrSudoUnavailable):
		return "Error: sudo is not available for iptables commands.\n\n" +
			"  Network filtering requires `sudo iptables`. Install sudo or\n" +
			"  set [network] mode = \"none\" to disable network filtering.", 4

	case errors.Is(err, docker.ErrIPTablesUnavailable):
		return "Error: iptables is not available on this system.\n\n" +
			"  Install iptables: `sudo apt-get install iptables`\n" +
			"  Or set [network] mode = \"none\" to disable filtering.", 4

	case errors.Is(err, docker.ErrNetworkUnsupported):
		return "Error: Network sandboxing is not supported on this platform.\n\n" +
			"  Network filtering requires Linux with iptables.\n" +
			"  Set [network] mode = \"none\" to disable filtering.", 4

	case errors.Is(err, docker.ErrDockerNotRunning):
		return "Error: Docker daemon is not running.\n\n" +
			"  macOS:  Open Docker Desktop, or run `open -a Docker`\n" +
			"  Linux:  Run `sudo systemctl start docker`\n\n" +
			"Zone requires Docker to create sandboxed workspaces.", 3

	case errors.Is(err, config.ErrNoConfig):
		return "Error: No zone.toml found.\n\n" +
			"  Run `zone init --harness <name>` to create one,\n" +
			"  or `zone launch --harness <name>` for zero-config quickstart.", 2

	case errors.Is(err, config.ErrVersionMismatch):
		return "Error: Unsupported config version in zone.toml.\n\n" +
			"  Zone currently supports `version = 1`.\n" +
			"  Check zone.toml and update the version field.", 2

	default:
		var uke *config.UnknownKeysError
		if errors.As(err, &uke) {
			return "Error: " + err.Error() + "\n\n" +
				"  Run `zone validate` to review config issues and suggestions,\n" +
				"  then fix unknown keys in zone.toml before retrying.", 2
		}

		return "Error: " + err.Error() + "\n\n" +
			"  Re-run with `--debug` for more details and inspect command usage with `zone --help`.\n" +
			"  If this persists, run `zone validate` to confirm configuration health.", 1
	}
}

// MapError is exported so main.go can call it for remediation output.
var MapError = mapError
