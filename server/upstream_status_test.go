package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Autumn-27/artex/integration"
)

type upstreamTransport func(*http.Request) (*http.Response, error)

func (f upstreamTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func upstreamResponse(code int, body string) (*http.Response, error) {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}
func upstreamFixture(t *testing.T, releaseCode int, comparison string) *http.Client {
	t.Helper()
	return &http.Client{Transport: upstreamTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("public upstream request crossed host/credential boundary")
		}
		switch {
		case strings.Contains(r.URL.Path, "/commits/"):
			return upstreamResponse(200, `{"sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","commit":{"message":"New feature\nLong body","committer":{"date":"2026-10-30T00:00:00Z"}}}`)
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			if releaseCode != 200 {
				return upstreamResponse(releaseCode, `{"message":"secret diagnostic must not appear"}`)
			}
			return upstreamResponse(200, `{"tag_name":"v0.3.26","published_at":"2026-10-30T00:00:00Z","html_url":"https://malicious.invalid/"}`)
		case strings.Contains(r.URL.Path, "/compare/"):
			return upstreamResponse(200, `{"status":"`+comparison+`","ahead_by":3,"behind_by":1}`)
		}
		t.Errorf("unexpected request %s", r.URL.Path)
		return upstreamResponse(404, `{}`)
	})}
}
func upstreamSource() integration.Source {
	return integration.Source{Repository: "RuoJi6/ARTEX", Branch: "main", Commit: strings.Repeat("a", 40)}
}
func TestUpstreamSnapshotUsesPublicReadsAndFixedLinks(t *testing.T) {
	t.Setenv("ARTEX_UPDATE_TOKEN", "test-private-release-token")
	row := fetchUpstream(t.Context(), upstreamFixture(t, 200, "ahead"), upstreamSource(), "mainline")
	if row.Comparison != "ahead" || row.Ahead != 3 || row.LatestCommit == nil || row.LatestCommit.Title != "New feature" {
		t.Fatalf("unexpected snapshot: %+v", row)
	}
	if row.ReleaseStatus != "available" || row.LatestRelease == nil || row.LatestRelease.HTMLURL != "https://github.com/RuoJi6/ARTEX/releases/tag/v0.3.26" {
		t.Fatalf("unexpected release/link: %+v", row)
	}
	if row.CheckedAt.IsZero() || row.CompareURL == "" {
		t.Fatal("snapshot lacks verification time or comparison link")
	}
}
func TestUpstreamNoReleaseAndDivergenceAreSeparateStates(t *testing.T) {
	row := fetchUpstream(t.Context(), upstreamFixture(t, 404, "diverged"), upstreamSource(), "donor")
	if row.ReleaseStatus != "none" || row.ReleaseError != "" || row.Comparison != "diverged" || row.Ahead != 3 || row.Behind != 1 {
		t.Fatalf("lost no-release/divergence distinction: %+v", row)
	}
}
func TestUpstreamFailuresNeverMeanNoChangesOrNoRelease(t *testing.T) {
	client := &http.Client{Transport: upstreamTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			return upstreamResponse(404, `{}`)
		}
		return upstreamResponse(403, `{"message":"credential-bearing upstream diagnostic"}`)
	})}
	row := fetchUpstream(t.Context(), client, upstreamSource(), "mainline")
	if row.Comparison != "unknown" || row.CommitError == "" || row.ReleaseStatus != "error" || row.LatestCommit != nil {
		t.Fatalf("inaccessible project must remain unknown: %+v", row)
	}
	b, _ := json.Marshal(row)
	if strings.Contains(string(b), "credential-bearing") {
		t.Fatal("remote error body leaked into UI")
	}
}
func TestUpstreamReleaseErrorKeepsKnownCommitRelationship(t *testing.T) {
	row := fetchUpstream(t.Context(), upstreamFixture(t, 429, "ahead"), upstreamSource(), "mainline")
	if row.ReleaseStatus != "error" || row.Comparison != "ahead" || row.LatestCommit == nil {
		t.Fatalf("partial result lost: %+v", row)
	}
	client := &http.Client{Transport: upstreamTransport(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("proxy://user:private-password@host connection failed")
	})}
	row = fetchUpstream(t.Context(), client, upstreamSource(), "mainline")
	if strings.Contains(row.CommitError, "private-password") {
		t.Fatal("connection credentials leaked")
	}
}
func TestUpstreamIdentityAvoidsUnnecessaryCompareRequest(t *testing.T) {
	src := upstreamSource()
	src.Commit = strings.Repeat("b", 40)
	row := fetchUpstream(t.Context(), upstreamFixture(t, 404, "unexpected"), src, "mainline")
	if row.Comparison != "identical" || row.ComparisonError != "" {
		t.Fatalf("identity should use known SHA: %+v", row)
	}
}
func fakeUpstreamRows() []upstreamProject {
	return []upstreamProject{{Repository: "RuoJi6/ARTEX", Comparison: "identical", ReleaseStatus: "none"}, {Repository: "BaaRai/ARTEX", Comparison: "identical", ReleaseStatus: "none"}}
}
func TestUpstreamCacheCachesRefreshesAndSeparatesProxyRoutes(t *testing.T) {
	var calls atomic.Int32
	c := &upstreamStatusCache{fetch: func(context.Context, *http.Client) []upstreamProject { calls.Add(1); return fakeUpstreamRows() }}
	for i := 0; i < 3; i++ {
		if _, err := c.get(t.Context(), nil, false, ""); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("page loads did not reuse cache")
	}
	_, _ = c.get(t.Context(), nil, true, "")
	_, _ = c.get(t.Context(), nil, false, "proxy-route")
	if calls.Load() != 3 {
		t.Fatalf("refresh or changed proxy failed to refetch: %d", calls.Load())
	}
	c.at = time.Now().Add(-releaseTTL - time.Second)
	_, _ = c.get(t.Context(), nil, false, "proxy-route")
	if calls.Load() != 4 {
		t.Fatal("expired cache was reused")
	}
}
func TestUpstreamCacheCoalescesConcurrentPageLoads(t *testing.T) {
	var calls atomic.Int32
	c := &upstreamStatusCache{fetch: func(context.Context, *http.Client) []upstreamProject { calls.Add(1); return fakeUpstreamRows() }}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.get(t.Context(), nil, false, "")
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("duplicate network refreshes: %d", calls.Load())
	}
}
func TestUpstreamCacheCancellationIsNotStored(t *testing.T) {
	c := &upstreamStatusCache{fetch: func(ctx context.Context, _ *http.Client) []upstreamProject { <-ctx.Done(); return fakeUpstreamRows() }}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.get(ctx, nil, false, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled query: %v", err)
	}
	if !c.at.IsZero() {
		t.Fatal("cancelled request poisoned cache")
	}
}
func TestUpstreamCacheFailureIsCachedForTenMinutes(t *testing.T) {
	var calls int
	c := &upstreamStatusCache{fetch: func(context.Context, *http.Client) []upstreamProject {
		calls++
		rows := fakeUpstreamRows()
		rows[0].CommitError = "offline"
		return rows
	}}
	_, _ = c.get(t.Context(), nil, false, "")
	c.at = time.Now().Add(-11 * time.Minute)
	_, _ = c.get(t.Context(), nil, false, "")
	if calls != 2 {
		t.Fatal("stale failure was kept for normal 30-minute TTL")
	}
}
func TestUpstreamEndpointRequiresAuthentication(t *testing.T) {
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := New(inactiveLoneHeroTestContext(), m, t.TempDir(), t.TempDir(), t.TempDir())
	old := upstreamCache
	defer func() { upstreamCache = old }()
	upstreamCache = &upstreamStatusCache{fetch: func(context.Context, *http.Client) []upstreamProject { return fakeUpstreamRows() }}
	h := s.Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/update/upstreams", nil))
	if w.Code != 401 {
		t.Fatalf("unauthenticated access: %d", w.Code)
	}
	token, _ := signJWT(s.jwtKey)
	r := httptest.NewRequest("GET", "/api/update/upstreams", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "BaaRai/ARTEX") {
		t.Fatalf("authenticated result: %d %s", w.Code, w.Body.String())
	}
}
