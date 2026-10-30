package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Autumn-27/artex/db"
)

func TestLoneHeroSettingsRequireAuthAndPreserveCompatibility(t *testing.T) {
	s, _, _ := trafficEvidenceServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/lonehero/settings", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated settings: %d", rec.Code)
	}
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		return w
	}
	if err := s.m.pg.SetSetting("lonehero_options", ""); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = s.m.pg.Exec(`DELETE FROM settings WHERE key='lonehero_options'`) })
	var initial loneHeroOptions
	if err := json.Unmarshal(call("GET", "/api/lonehero/settings", "").Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.LessonsMode != "shadow" || initial.FailureEscalation || initial.GoalGate {
		t.Fatalf("unsafe compatibility defaults: %+v", initial)
	}
	if w := call("PUT", "/api/lonehero/settings", `{"lessons_mode":"inject","quality_guidance":true,"failure_escalation":true}`); w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	if got := s.loneHeroOptions(); got.LessonsMode != "inject" || !got.FailureEscalation {
		t.Fatalf("settings lost: %+v", got)
	}
	if w := call("PUT", "/api/lonehero/settings", `{"lessons_mode":"unknown"}`); w.Code != 400 {
		t.Fatalf("invalid mode accepted: %d", w.Code)
	}
}

func TestLoneHeroLessonsNeedAdoptionAndStayInScope(t *testing.T) {
	s, _, _ := trafficEvidenceServer(t)
	pg := s.m.pg
	a, err := s.m.CreateTask("lesson scope A", "goal", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.m.CreateTask("lesson scope B", "goal", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	aid, _ := strconv.ParseInt(a.ID, 10, 64)
	bid, _ := strconv.ParseInt(b.ID, 10, 64)
	t.Cleanup(func() {
		_, _ = pg.Exec(`DELETE FROM rsi_lessons WHERE origin_task_id=ANY($1::bigint[])`, []int64{aid, bid})
		_ = pg.DeleteTask(aid)
		_ = pg.DeleteTask(bid)
	})
	for _, id := range []int64{aid, bid} {
		if _, err := pg.Exec(`INSERT INTO findings(task_id,name,vulnclass,summary,severity,status) VALUES($1,'scope probe','xss','same observation','high','confirmed')`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := pg.ExtractLessonsForTask(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
	list, err := pg.TaskLessons(context.Background(), aid, 50)
	if err != nil || len(list) != 1 {
		t.Fatalf("cross-task leak or missing candidate: %+v %v", list, err)
	}
	approved, err := pg.TaskLessons(context.Background(), aid, 50, true)
	if err != nil || len(approved) != 0 {
		t.Fatalf("unapproved lessons injected: %+v %v", approved, err)
	}
	if _, err := pg.Exec(`UPDATE rsi_lessons SET status='adopted' WHERE id=$1`, list[0].ID); err != nil {
		t.Fatal(err)
	}
	approved, err = pg.TaskLessons(context.Background(), aid, 50, true)
	if err != nil || len(approved) != 1 {
		t.Fatalf("adopted lesson missing: %+v %v", approved, err)
	}
	other, err := pg.TaskLessons(context.Background(), bid, 50, true)
	if err != nil || len(other) != 0 {
		t.Fatalf("adoption crossed tasks: %+v %v", other, err)
	}
}

func TestLoneHeroProfileEditKeepsPricesAndKey(t *testing.T) {
	s, _, _ := trafficEvidenceServer(t)
	id, err := s.m.pg.SaveProfile(&db.LLMProfile{Name: "hero-price-preserve", Format: "openai", Model: "m", APIKey: "test-only", PriceInputPerM: 3})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = s.m.pg.Exec(`DELETE FROM llm_profiles WHERE id=$1`, id) })
	body, _ := json.Marshal(map[string]any{"id": id, "name": "hero-price-preserve", "format": "openai", "model": "changed"})
	rec := httptest.NewRecorder()
	s.pgSaveProfile(rec, httptest.NewRequest("POST", "/api/llm/profiles", bytes.NewReader(body)))
	if rec.Code != 200 {
		t.Fatalf("profile edit failed: %d %s", rec.Code, rec.Body.String())
	}
	p, err := s.m.pg.ProfileByID(id)
	if err != nil || p.PriceInputPerM != 3 || p.APIKey != "test-only" {
		t.Fatalf("edit lost price/key: %+v %v", p, err)
	}
}

func inactiveLoneHeroTestContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
