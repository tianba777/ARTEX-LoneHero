package selfupdate

import (
	"net/http"
	"testing"
)

func TestPrivateUpdateTokenStaysOnOwnAPI(t *testing.T) {
	t.Setenv("ARTEX_UPDATE_TOKEN", "test-token-not-real")
	for _, tc := range []struct {
		url     string
		allowed bool
	}{
		{"https://api.github.com/repos/" + Repo + "/releases/latest", true},
		{"https://api.github.com/repos/other/project/releases/latest", false},
		{"https://github.com/" + Repo + "/releases/download/v1/x.zip", false},
		{"https://release-assets.githubusercontent.com/object", false},
	} {
		req, _ := http.NewRequest("GET", tc.url, nil)
		authorizeUpdateRequest(req)
		if (req.Header.Get("Authorization") != "") != tc.allowed {
			t.Fatalf("credential boundary failed for %s", tc.url)
		}
	}
	c, err := NewClient("")
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("GET", "https://release-assets.githubusercontent.com/object", nil)
	req.Header.Set("Authorization", "Bearer test-token-not-real")
	if err := c.CheckRedirect(req, nil); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatal("redirect leaked update token")
	}
}
