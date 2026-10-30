package server

// Lone Hero additions live behind one settings surface; upstream task, scheduler,
// evidence and update implementations remain authoritative.
import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type loneHeroOptions struct {
	LessonsMode       string `json:"lessons_mode"`
	FailureEscalation bool   `json:"failure_escalation"`
	GoalGate          bool   `json:"goal_gate"`
	QualityGuidance   bool   `json:"quality_guidance"`
}

func (s *Server) saveLoneHeroPrices(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("pid"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "配置 ID 无效")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var p struct {
		Input  float64 `json:"input"`
		Output float64 `json:"output"`
		Read   float64 `json:"cache_read"`
		Write  float64 `json:"cache_write"`
	}
	if decode(r, &p) != nil {
		writeErr(w, 400, "单价格式无效")
		return
	}
	for _, v := range []float64{p.Input, p.Output, p.Read, p.Write} {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) || v > 999999 {
			writeErr(w, 400, "单价超出范围")
			return
		}
	}
	result, err := s.m.PG().ExecContext(r.Context(), `UPDATE llm_profiles SET price_input_per_m=$2,price_output_per_m=$3,price_cache_read_per_m=$4,price_cache_write_per_m=$5 WHERE id=$1`, id, p.Input, p.Output, p.Read, p.Write)
	if err != nil {
		writeErr(w, 500, "保存单价失败")
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		writeErr(w, 404, "配置不存在")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) loneHeroOptions() loneHeroOptions {
	o := loneHeroOptions{LessonsMode: "shadow", QualityGuidance: true}
	if s.m.PG() != nil {
		if raw, ok, err := s.m.PG().GetSetting("lonehero_options"); err == nil && ok {
			_ = json.Unmarshal([]byte(raw), &o)
		}
	}
	if mode := strings.ToLower(strings.TrimSpace(os.Getenv("ARTEX_RSI_LESSONS"))); mode == "off" || mode == "shadow" || mode == "inject" {
		o.LessonsMode = mode
	}
	return o
}

func (s *Server) getLoneHeroOptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.loneHeroOptions())
}

func (s *Server) saveLoneHeroOptions(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var o loneHeroOptions
	if err := decode(r, &o); err != nil {
		writeErr(w, 400, "设置格式无效")
		return
	}
	if o.LessonsMode != "off" && o.LessonsMode != "shadow" && o.LessonsMode != "inject" {
		writeErr(w, 400, "经验模式必须为 off、shadow 或 inject")
		return
	}
	data, _ := json.Marshal(o)
	if err := s.m.PG().SetSetting("lonehero_options", string(data)); err != nil {
		writeErr(w, 500, "保存失败")
		return
	}
	s.invalidateTaskAgents()
	writeJSON(w, 200, s.loneHeroOptions())
}

func (s *Server) checkLoneHeroGoal(goal string) error {
	if !s.loneHeroOptions().GoalGate {
		return nil
	}
	return goalGateCheck(goal)
}

func (s *Server) updateLessonStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("lid"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "经验 ID 无效")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		Status string `json:"status"`
	}
	if decode(r, &body) != nil || (body.Status != "candidate" && body.Status != "adopted" && body.Status != "retired") {
		writeErr(w, 400, "状态必须为 candidate、adopted 或 retired")
		return
	}
	result, err := s.m.PG().ExecContext(r.Context(), `UPDATE rsi_lessons SET status=$2,updated_at=now() WHERE id=$1`, id, body.Status)
	if err != nil {
		writeErr(w, 500, "更新经验失败")
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		writeErr(w, 404, "经验不存在")
		return
	}
	s.invalidateTaskAgents()
	writeJSON(w, 200, map[string]any{"id": id, "status": body.Status})
}

func (s *Server) listLoneHeroLessons(w http.ResponseWriter, r *http.Request) {
	rows, err := s.m.PG().QueryContext(r.Context(), `SELECT id,kind,cue,action,COALESCE(company_id,0),COALESCE(category_id,0),status,(SELECT COUNT(*) FROM rsi_lesson_evidence e WHERE e.lesson_id=l.id) FROM rsi_lessons l ORDER BY updated_at DESC,id DESC LIMIT 200`)
	if err != nil {
		writeErr(w, 500, "读取经验失败")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, company, category, support int64
		var kind, cue, action, status string
		if err := rows.Scan(&id, &kind, &cue, &action, &company, &category, &status, &support); err != nil {
			writeErr(w, 500, "读取经验失败")
			return
		}
		out = append(out, map[string]any{"id": id, "kind": kind, "cue": cue, "action": action, "company_id": company, "category_id": category, "status": status, "support": support})
	}
	if rows.Err() != nil {
		writeErr(w, 500, "读取经验失败")
		return
	}
	writeJSON(w, 200, map[string]any{"lessons": out, "limit": 200})
}
