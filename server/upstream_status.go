package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Autumn-27/artex/integration"
)

type upstreamCommit struct {
	SHA     string    `json:"sha"`
	Title   string    `json:"title"`
	Date    time.Time `json:"date"`
	HTMLURL string    `json:"html_url"`
}

type upstreamRelease struct {
	Tag         string    `json:"tag"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
}

type upstreamProject struct {
	Repository      string           `json:"repository"`
	Role            string           `json:"role"`
	Branch          string           `json:"branch"`
	Baseline        string           `json:"baseline"`
	BaselineURL     string           `json:"baseline_url"`
	RepositoryURL   string           `json:"repository_url"`
	CompareURL      string           `json:"compare_url,omitempty"`
	LatestCommit    *upstreamCommit  `json:"latest_commit,omitempty"`
	LatestRelease   *upstreamRelease `json:"latest_release,omitempty"`
	ReleaseStatus   string           `json:"release_status"` // available | none | error
	Comparison      string           `json:"comparison"`     // identical | ahead | behind | diverged | unknown
	Ahead           int              `json:"ahead"`
	Behind          int              `json:"behind"`
	CommitError     string           `json:"commit_error,omitempty"`
	ReleaseError    string           `json:"release_error,omitempty"`
	ComparisonError string           `json:"comparison_error,omitempty"`
	CheckedAt       time.Time        `json:"checked_at"`
}

var upstreamSHA = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

// These are public reads. Never call the private updater's authorization helper,
// forward a browser JWT, or send ARTEX_UPDATE_TOKEN to either upstream.
func upstreamJSON(ctx context.Context, client *http.Client, repo, path string, into any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+path, nil)
	if err != nil {
		return 0, fmt.Errorf("无法创建上游查询")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "artex-lonehero-upstream-status")
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("无法连接 GitHub，请检查更新代理或稍后重试")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 403 || resp.StatusCode == 429 {
			return resp.StatusCode, fmt.Errorf("GitHub 暂时拒绝查询或已限流，请稍后重试")
		}
		return resp.StatusCode, fmt.Errorf("GitHub 查询返回 HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(into); err != nil {
		return resp.StatusCode, fmt.Errorf("无法解析 GitHub 返回的数据")
	}
	return resp.StatusCode, nil
}

func fetchUpstream(ctx context.Context, client *http.Client, source integration.Source, role string) upstreamProject {
	baseURL := "https://github.com/" + source.Repository
	p := upstreamProject{
		Repository: source.Repository, Role: role, Branch: source.Branch,
		Baseline: source.Commit, RepositoryURL: baseURL,
		BaselineURL: baseURL + "/commit/" + source.Commit,
		Comparison:  "unknown", ReleaseStatus: "error",
	}
	var head struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message   string `json:"message"`
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	var release struct {
		Tag         string    `json:"tag_name"`
		PublishedAt time.Time `json:"published_at"`
	}
	var headErr, releaseErr error
	var releaseStatus int
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, headErr = upstreamJSON(ctx, client, source.Repository, "/commits/"+url.PathEscape(source.Branch), &head)
	}()
	go func() {
		defer wg.Done()
		releaseStatus, releaseErr = upstreamJSON(ctx, client, source.Repository, "/releases/latest", &release)
	}()
	wg.Wait()
	if releaseStatus == 404 {
		if headErr == nil && upstreamSHA.MatchString(head.SHA) {
			p.ReleaseStatus = "none"
		} else {
			p.ReleaseError = "暂时无法确认上游的正式发布信息"
		}
	} else if releaseErr != nil {
		p.ReleaseError = releaseErr.Error()
	} else if strings.TrimSpace(release.Tag) == "" {
		p.ReleaseError = "GitHub 未返回有效的正式版本信息"
	} else {
		p.ReleaseStatus = "available"
		p.LatestRelease = &upstreamRelease{Tag: release.Tag, PublishedAt: release.PublishedAt,
			HTMLURL: baseURL + "/releases/tag/" + url.PathEscape(release.Tag)}
	}
	if headErr != nil {
		p.CommitError = headErr.Error()
	} else if !upstreamSHA.MatchString(head.SHA) {
		p.CommitError = "GitHub 未返回有效的提交信息"
	} else {
		p.LatestCommit = &upstreamCommit{SHA: head.SHA, Title: strings.SplitN(head.Commit.Message, "\n", 2)[0],
			Date: head.Commit.Committer.Date, HTMLURL: baseURL + "/commit/" + head.SHA}
		p.CompareURL = baseURL + "/compare/" + source.Commit + "..." + head.SHA
		if strings.EqualFold(source.Commit, head.SHA) {
			p.Comparison = "identical"
		} else {
			var compare struct {
				Status string `json:"status"`
				Ahead  int    `json:"ahead_by"`
				Behind int    `json:"behind_by"`
			}
			_, err := upstreamJSON(ctx, client, source.Repository, "/compare/"+source.Commit+"..."+head.SHA+"?per_page=1", &compare)
			if err != nil {
				p.ComparisonError = err.Error()
			} else {
				switch compare.Status {
				case "identical", "ahead", "behind", "diverged":
					if compare.Ahead < 0 || compare.Behind < 0 {
						p.ComparisonError = "GitHub 提交计数无效"
					} else {
						p.Comparison, p.Ahead, p.Behind = compare.Status, compare.Ahead, compare.Behind
					}
				default:
					p.ComparisonError = "GitHub 提交关系未知"
				}
			}
		}
	}
	p.CheckedAt = time.Now().UTC()
	return p
}

type upstreamStatusCache struct {
	mu      sync.Mutex
	rows    []upstreamProject
	at      time.Time
	network string
	fetch   func(context.Context, *http.Client) []upstreamProject
}

var upstreamCache = &upstreamStatusCache{}

func loadUpstreams(ctx context.Context, client *http.Client) []upstreamProject {
	main, donor, err := integration.Sources()
	if err != nil || main.Repository != "RuoJi6/ARTEX" || donor.Repository != "BaaRai/ARTEX" ||
		!upstreamSHA.MatchString(main.Commit) || !upstreamSHA.MatchString(donor.Commit) || main.Branch == "" || donor.Branch == "" {
		return nil
	}
	rows := make([]upstreamProject, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); rows[0] = fetchUpstream(ctx, client, main, "mainline") }()
	go func() { defer wg.Done(); rows[1] = fetchUpstream(ctx, client, donor, "donor") }()
	wg.Wait()
	return rows
}

func (c *upstreamStatusCache) get(ctx context.Context, client *http.Client, force bool, network string) ([]upstreamProject, error) {
	started := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	ttl := releaseTTL
	for _, row := range c.rows {
		if row.CommitError != "" || row.ReleaseError != "" || row.ComparisonError != "" {
			ttl = 10 * time.Minute
		}
	}
	if c.network == network && !c.at.IsZero() && ((!force && time.Since(c.at) < ttl) || c.at.After(started)) {
		return append([]upstreamProject(nil), c.rows...), nil
	}
	fetch := c.fetch
	if fetch == nil {
		fetch = loadUpstreams
	}
	queryCtx, cancel := context.WithTimeout(ctx, releaseTimeout)
	defer cancel()
	rows := fetch(queryCtx, client)
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil, ctx.Err()
	}
	if len(rows) != 2 {
		return nil, fmt.Errorf("无法读取本项目记录的上游来源")
	}
	c.rows, c.at, c.network = rows, time.Now(), network
	return append([]upstreamProject(nil), rows...), nil
}

func (s *Server) updateUpstreams(w http.ResponseWriter, r *http.Request) {
	client, network, err := s.updateClient()
	if err != nil {
		writeErr(w, 400, "读取更新网络设置失败")
		return
	}
	defer client.CloseIdleConnections()
	rows, err := upstreamCache.get(r.Context(), client, r.URL.Query().Get("force") == "1", network)
	if err != nil {
		writeErr(w, 503, "暂时无法读取上游动态，请稍后重试")
		return
	}
	writeJSON(w, 200, map[string]any{"projects": rows})
}
