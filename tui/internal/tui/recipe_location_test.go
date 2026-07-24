package tui

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/lingtai-tui/internal/fs"
)

type recipeLocationRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn recipeLocationRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func recipeLocationResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(`{
			"city": "Austin",
			"region": "Texas",
			"country": "US",
			"timezone": "America/Chicago",
			"loc": "30.2672,-97.7431"
		}`)),
	}
}

func TestSubstituteGreetPlaceholdersLocationReusesResolvedLocation(t *testing.T) {
	humanDir := t.TempDir()
	manifest, err := json.MarshalIndent(map[string]interface{}{
		"agent_name": "human",
		"address":    "human",
		"admin":      nil,
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal human manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(humanDir, ".agent.json"), manifest, 0o644); err != nil {
		t.Fatalf("write human manifest: %v", err)
	}

	var requests atomic.Int32
	secondRequest := make(chan struct{})
	previousTransport := http.DefaultTransport
	http.DefaultTransport = recipeLocationRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		if requests.Add(1) == 2 {
			close(secondRequest)
		}
		return recipeLocationResponse(), nil
	})
	t.Cleanup(func() {
		http.DefaultTransport = previousTransport
	})

	got := SubstituteGreetPlaceholders("location={{location}}", "human", humanDir, "en", "60")
	if got != "location=Austin, Texas, US" {
		t.Fatalf("substituted greet = %q; want resolved location", got)
	}

	// Current main starts an untracked UpdateHumanLocation after the synchronous
	// lookup, producing a second request. The fixed path stores the Location it
	// already has before returning, so no observation sleep is needed for GREEN;
	// this bounded wait only makes the pre-fix second request deterministic.
	select {
	case <-secondRequest:
	case <-time.After(500 * time.Millisecond):
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		node, readErr := fs.ReadAgent(humanDir)
		if readErr == nil && node.Location != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("resolved location was not persisted: %v", readErr)
		}
		time.Sleep(5 * time.Millisecond)
	}

	if got := requests.Load(); got != 1 {
		t.Fatalf("recipe location fallback made %d resolver requests; want exactly 1", got)
	}
}
