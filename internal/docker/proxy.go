// proxy.go resolves proxy configuration from zone config and host environment variables.
// Config values always take precedence over auto-detected host env values.
package docker

import (
	"os"

	"github.com/peasant-labs/zone/internal/config"
)

// resolveProxy returns the effective HTTP proxy, HTTPS proxy, and no-proxy values.
// Config values (cfg.HTTPProxy, cfg.HTTPSProxy, cfg.NoProxy) take precedence.
// If a config value is empty, the corresponding host environment variable is used
// (checking both uppercase and lowercase variants).
func resolveProxy(cfg *config.NetworkConfig) (httpProxy, httpsProxy, noProxy string) {
	httpProxy = cfg.HTTPProxy
	if httpProxy == "" {
		httpProxy = firstEnv("HTTP_PROXY", "http_proxy")
	}

	httpsProxy = cfg.HTTPSProxy
	if httpsProxy == "" {
		httpsProxy = firstEnv("HTTPS_PROXY", "https_proxy")
	}

	noProxy = cfg.NoProxy
	if noProxy == "" {
		noProxy = firstEnv("NO_PROXY", "no_proxy")
	}

	return httpProxy, httpsProxy, noProxy
}

// firstEnv returns the value of the first non-empty environment variable from keys.
// Returns "" if all keys are unset or empty.
func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// proxyBuildArgs returns proxy values formatted as Docker build-arg map[string]*string.
// Both uppercase and lowercase variants are included for each non-empty proxy value.
// Each pointer is a unique local variable to avoid pointer aliasing bugs.
func proxyBuildArgs(httpProxy, httpsProxy, noProxy string) map[string]*string {
	args := map[string]*string{}

	if httpProxy != "" {
		v1 := httpProxy
		v2 := httpProxy
		args["HTTP_PROXY"] = &v1
		args["http_proxy"] = &v2
	}
	if httpsProxy != "" {
		v1 := httpsProxy
		v2 := httpsProxy
		args["HTTPS_PROXY"] = &v1
		args["https_proxy"] = &v2
	}
	if noProxy != "" {
		v1 := noProxy
		v2 := noProxy
		args["NO_PROXY"] = &v1
		args["no_proxy"] = &v2
	}

	return args
}

// proxyEnvVars returns proxy values as a []string of "KEY=value" pairs suitable
// for container environment variables. Both uppercase and lowercase variants are
// included for each non-empty proxy value.
func proxyEnvVars(httpProxy, httpsProxy, noProxy string) []string {
	var envs []string

	if httpProxy != "" {
		envs = append(envs, "HTTP_PROXY="+httpProxy)
		envs = append(envs, "http_proxy="+httpProxy)
	}
	if httpsProxy != "" {
		envs = append(envs, "HTTPS_PROXY="+httpsProxy)
		envs = append(envs, "https_proxy="+httpsProxy)
	}
	if noProxy != "" {
		envs = append(envs, "NO_PROXY="+noProxy)
		envs = append(envs, "no_proxy="+noProxy)
	}

	return envs
}
