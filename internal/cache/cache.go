// Package cache manages the .zone/ directory, config hashing, and build log storage.
package cache

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Cache manages the .zone/ directory for a single repository.
// All reads and writes go through this struct to ensure atomic operations.
type Cache struct {
	dir string // absolute path to .zone/ directory
}

// New returns a Cache whose directory is {repoDir}/.zone/.
func New(repoDir string) *Cache {
	return &Cache{dir: filepath.Join(repoDir, ".zone")}
}

// Dir returns the absolute path of the .zone/ directory.
func (c *Cache) Dir() string { return c.dir }

// EnsureDir creates the .zone/ and .zone/logs/ directories if they do not exist.
func (c *Cache) EnsureDir() error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(c.dir, "logs"), 0755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}
	return nil
}

// writeAtomic writes content to .zone/{name} via a .tmp- intermediate file
// and an os.Rename for atomic replacement.
func (c *Cache) writeAtomic(name, content string) error {
	tmpPath := filepath.Join(c.dir, ".tmp-"+name)
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	target := filepath.Join(c.dir, name)
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath) // best-effort cleanup
		return fmt.Errorf("rename to %s: %w", name, err)
	}
	return nil
}

// readTrimmed reads .zone/{name} and trims surrounding whitespace.
// If the file does not exist, it returns ("", nil) — not-found is not an error.
func (c *Cache) readTrimmed(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(c.dir, name))
	if os.IsNotExist(err) {
		return "", nil // not set yet is not an error
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// SetImageID atomically persists the Docker image ID.
func (c *Cache) SetImageID(id string) error { return c.writeAtomic("image_id", id) }

// SetContainerID atomically persists the Docker container ID.
func (c *Cache) SetContainerID(id string) error { return c.writeAtomic("container_id", id) }

// SetNetworkID atomically persists the Docker network ID.
func (c *Cache) SetNetworkID(id string) error { return c.writeAtomic("network_id", id) }

// SetConfigHash atomically persists the computed config hash.
func (c *Cache) SetConfigHash(h string) error { return c.writeAtomic("config.hash", h) }

// ImageID returns the cached Docker image ID, or ("", nil) if not set.
func (c *Cache) ImageID() (string, error) { return c.readTrimmed("image_id") }

// ContainerID returns the cached Docker container ID, or ("", nil) if not set.
func (c *Cache) ContainerID() (string, error) { return c.readTrimmed("container_id") }

// NetworkID returns the cached Docker network ID, or ("", nil) if not set.
func (c *Cache) NetworkID() (string, error) { return c.readTrimmed("network_id") }

// ConfigHash returns the cached config hash, or ("", nil) if not set.
func (c *Cache) ConfigHash() (string, error) { return c.readTrimmed("config.hash") }

// Clean removes the entire .zone/ directory and all its contents.
func (c *Cache) Clean() error {
	return os.RemoveAll(c.dir)
}

// EnsureGitignore adds .zone/ to the git root's .gitignore.
// cwd is the directory zone is invoked from (where zone.toml lives).
// Idempotent: no-op if entry already present. Silently skips non-git repos.
func EnsureGitignore(cwd string) error {
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil // not in a git repo — silently skip
	}
	gitRoot := strings.TrimSpace(string(out))

	rel, err := filepath.Rel(gitRoot, cwd)
	if err != nil {
		return fmt.Errorf("relative path: %w", err)
	}
	var entry string
	if rel == "." {
		entry = ".zone/"
	} else {
		entry = rel + "/.zone/"
	}

	gitignorePath := filepath.Join(gitRoot, ".gitignore")

	// Check if entry already present (exact string match per line)
	if data, err := os.ReadFile(gitignorePath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == entry {
				return nil // already present
			}
		}
	}

	// Append entry (or create minimal .gitignore)
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open .gitignore: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n", entry)
	return err
}

// CreateBuildLog opens .zone/logs/last_build.log for writing (truncate if exists).
// Returns a writer that tees to both w (terminal) and the log file, plus a closer func.
// Callers MUST defer closer() unconditionally — partial logs are kept on build failure.
func (c *Cache) CreateBuildLog(w io.Writer, configHash, version string) (io.Writer, func(), error) {
	logPath := filepath.Join(c.dir, "logs", "last_build.log")
	f, err := os.Create(logPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create build log: %w", err)
	}

	header := fmt.Sprintf("# zone build | %s | config hash: %s | zone %s\n",
		time.Now().Format(time.RFC3339), configHash, version)
	_, _ = f.WriteString(header)

	tee := io.MultiWriter(w, f)
	closer := func() { _ = f.Close() }
	return tee, closer, nil
}
