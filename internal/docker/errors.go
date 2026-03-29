// errors.go defines sentinel errors for the docker package.
package docker

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
