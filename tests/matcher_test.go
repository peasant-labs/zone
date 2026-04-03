// Tests for hostname glob matching.
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/peasant-labs/zone/internal/network"
)

func TestCompile(t *testing.T) {
	t.Run("literal", func(t *testing.T) {
		pattern, err := network.Compile("api.anthropic.com")
		require.NoError(t, err)
		assert.False(t, pattern.IsGlob())
		assert.Equal(t, "api.anthropic.com", pattern.String())
	})

	t.Run("glob", func(t *testing.T) {
		pattern, err := network.Compile("*.anthropic.com")
		require.NoError(t, err)
		assert.True(t, pattern.IsGlob())
		assert.Equal(t, "*.anthropic.com", pattern.String())
	})

	for _, pattern := range []string{"**.anthropic.com", "foo[bar].com", "foo/bar.com", "foo{bar}.com"} {
		t.Run(pattern, func(t *testing.T) {
			_, err := network.Compile(pattern)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported pattern")
		})
	}
}

func TestMatch(t *testing.T) {
	literal, err := network.Compile("api.anthropic.com")
	require.NoError(t, err)
	assert.True(t, literal.Match("api.anthropic.com"))
	assert.False(t, literal.Match("other.anthropic.com"))

	glob, err := network.Compile("*.anthropic.com")
	require.NoError(t, err)
	assert.True(t, glob.Match("api.anthropic.com"))
	assert.True(t, glob.Match("docs.anthropic.com"))
	assert.False(t, glob.Match("anthropic.com"))
	assert.False(t, glob.Match("sub.api.anthropic.com"))

	example, err := network.Compile("*.example.com")
	require.NoError(t, err)
	assert.True(t, example.Match("www.example.com"))
	assert.False(t, example.Match("example.com"))
}

func TestMatcher(t *testing.T) {
	patterns, err := network.CompileAll([]string{"api.github.com", "*.anthropic.com"})
	require.NoError(t, err)

	assert.True(t, network.MatchAny("docs.anthropic.com", patterns))
	assert.True(t, network.MatchAny("api.github.com", patterns))
	assert.False(t, network.MatchAny("api.gitlab.com", patterns))
	assert.False(t, network.MatchAny("anything.example.com", nil))

	_, err = network.CompileAll([]string{"api.github.com", "foo[bar].com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported pattern")
}
