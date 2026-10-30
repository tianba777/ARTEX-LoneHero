"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
import type { HeroGap, HeroLesson, HeroOutcome, HeroPrices, LoneHeroOptions } from "@/lib/lonehero-types";
import type { LLMProfile, ProfileUsage, Task } from "@/lib/types";

const lessonStates = { candidate: "候选", adopted: "已采纳", retired: "已停用" };
const selectClass = "border-input bg-background rounded-md border p-2 text-sm";
const number = (n: number) => n.toLocaleString("zh-CN");

function PriceRow({ profile, usage, onSaved }: { profile: LLMProfile; usage?: ProfileUsage; onSaved: () => void }) {
  const [prices, setPrices] = useState<HeroPrices>({
    input: profile.price_input_per_m ?? 0,
    output: profile.price_output_per_m ?? 0,
    cache_read: profile.price_cache_read_per_m ?? 0,
    cache_write: profile.price_cache_write_per_m ?? 0,
  });
  const [saving, setSaving] = useState(false);
  const save = async () => {
    setSaving(true);
    try {
      await api.saveHeroPrices(Number(profile.id), prices);
      toast.success("单价已保存");
      onSaved();
    } catch (e) {
      toast.error(String(e));
    } finally {
      setSaving(false);
    }
  };
  return (
    <div className="grid gap-3 rounded-lg border p-4 md:grid-cols-[1.5fr_repeat(4,1fr)_auto]">
      <div>
        <div className="font-medium">{profile.name}</div>
        <div className="text-muted-foreground text-xs">
          {usage?.cost_eur == null ? "尚未估算费用" : `历史用量按当前单价估算 €${usage.cost_eur.toFixed(4)}`}
        </div>
      </div>
      {(
        [
          ["input", "输入"],
          ["output", "输出"],
          ["cache_read", "缓存读取"],
          ["cache_write", "缓存写入"],
        ] as const
      ).map(([key, label]) => (
        <label key={key} className="grid gap-1 text-xs">
          {label}
          <Input
            aria-label={`${profile.name} ${label}单价`}
            type="number"
            min="0"
            max="999999"
            step="any"
            value={prices[key]}
            onChange={(e) => setPrices((p) => ({ ...p, [key]: Number(e.target.value) }))}
          />
        </label>
      ))}
      <Button variant="outline" className="self-end" onClick={save} disabled={saving}>
        保存
      </Button>
    </div>
  );
}

export default function LoneHeroPage() {
  const [options, setOptions] = useState<LoneHeroOptions | null>(null);
  const [lessons, setLessons] = useState<HeroLesson[]>([]);
  const [gaps, setGaps] = useState<HeroGap[]>([]);
  const [profiles, setProfiles] = useState<LLMProfile[]>([]);
  const [usage, setUsage] = useState<ProfileUsage[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [taskId, setTaskId] = useState("");
  const [outcome, setOutcome] = useState<HeroOutcome | null>(null);
  const [busy, setBusy] = useState(false);
  const [loadError, setLoadError] = useState("");
  const load = useCallback(async () => {
    try {
      const [o, l, g, p, u, t] = await Promise.all([
        api.heroOptions(),
        api.heroLessons(),
        api.heroGaps(),
        api.llmProfiles(),
        api.usageStats(),
        api.tasks(),
      ]);
      setOptions(o);
      setLessons(l.lessons);
      setGaps(g.gaps);
      setProfiles(p);
      setUsage(u.by_profile);
      setTasks(t.tasks);
      setLoadError("");
    } catch (e) {
      setLoadError(String(e));
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    let live = true;
    setOutcome(null);
    if (taskId)
      void api
        .heroOutcome(taskId)
        .then((o) => {
          if (live) setOutcome(o);
        })
        .catch((e) => {
          if (live) toast.error(String(e));
        });
    return () => {
      live = false;
    };
  }, [taskId]);
  const perform = async (action: () => Promise<unknown>, message: string) => {
    setBusy(true);
    try {
      await action();
      toast.success(message);
      await load();
    } catch (e) {
      toast.error(String(e));
    } finally {
      setBusy(false);
    }
  };
  const currentTask = tasks.find((t) => t.id === taskId);
  return (
    <div className="mx-auto grid w-full max-w-6xl gap-6 p-4 md:p-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="font-semibold text-2xl">孤胆英豪</h1>
          <p className="text-muted-foreground mt-1 text-sm">运行增强、经验复用与用量管理</p>
        </div>
        <Button variant="outline" onClick={() => void load()}>
          刷新
        </Button>
      </div>
      {loadError && (
        <div role="alert" className="rounded-lg border border-destructive p-4 text-sm">
          加载失败：{loadError}
          <Button variant="link" onClick={() => void load()}>
            重试
          </Button>
        </div>
      )}
      <Card>
        <CardHeader>
          <CardTitle>运行设置</CardTitle>
          <CardDescription>原有任务调度继续生效。以下增强可分别调整。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          {!options ? (
            <p className="text-muted-foreground text-sm">正在读取设置…</p>
          ) : (
            <>
              <label className="grid gap-2 text-sm">
                历史经验的使用方式
                <select
                  className={selectClass}
                  value={options.lessons_mode}
                  onChange={(e) =>
                    setOptions({ ...options, lessons_mode: e.target.value as LoneHeroOptions["lessons_mode"] })
                  }
                >
                  <option value="off">关闭</option>
                  <option value="shadow">记录候选经验，暂不用于决策</option>
                  <option value="inject">使用已人工采纳的经验</option>
                </select>
              </label>
              {(
                [
                  ["quality_guidance", "加强证据质量与不可信内容提示"],
                  ["failure_escalation", "连续调用失败后尝试任务链中的下一个模型"],
                  ["goal_gate", "创建任务时检查目标描述是否足够明确"],
                ] as const
              ).map(([key, label]) => (
                <label key={key} className="flex items-center gap-3 text-sm">
                  <input
                    type="checkbox"
                    checked={options[key]}
                    onChange={(e) => setOptions({ ...options, [key]: e.target.checked })}
                  />
                  {label}
                </label>
              ))}
              <p className="text-muted-foreground text-xs">
                设置对后续执行生效；已运行中的会话可能继续使用已有上下文。经验模式也可由部署环境指定。
              </p>
              <Button
                className="justify-self-start"
                disabled={busy}
                onClick={() => void perform(() => api.saveHeroOptions(options), "设置已保存")}
              >
                保存设置
              </Button>
            </>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>任务结果</CardTitle>
          <CardDescription>查看已有任务的产出与消耗，并从已完成任务提取候选经验。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <select
            aria-label="选择任务"
            className={selectClass}
            value={taskId}
            onChange={(e) => setTaskId(e.target.value)}
          >
            <option value="">选择任务</option>
            {tasks.map((t) => (
              <option key={t.id} value={t.id}>
                #{t.id} {t.name || t.description}
              </option>
            ))}
          </select>
          {outcome && (
            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              {[
                ["确认发现", outcome.findings.confirmed],
                ["误报 / 重复", `${outcome.findings.false_positive} / ${outcome.findings.duplicate}`],
                [
                  "输入 / 输出 Token",
                  `${number(outcome.tokens.input_tokens)} / ${number(outcome.tokens.output_tokens)}`,
                ],
                ["工具调用", outcome.tool_calls],
              ].map(([label, value]) => (
                <div key={label} className="rounded-lg bg-muted p-3">
                  <div className="text-muted-foreground text-xs">{label}</div>
                  <div className="mt-1 font-semibold">{value}</div>
                </div>
              ))}
            </div>
          )}
          {currentTask && (
            <div className="flex flex-wrap items-center gap-4">
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={currentTask.plan_quiet_mode ?? false}
                  disabled={busy}
                  onChange={(e) => void perform(() => api.setHeroTaskQuiet(taskId, e.target.checked), "规划设置已保存")}
                />
                工作进行中减少重复规划；人工线索与新发现仍会触发规划
              </label>
              <Button
                variant="outline"
                disabled={busy || !["done", "failed", "timeout"].includes(currentTask.status)}
                onClick={() => void perform(() => api.extractHeroLessons(taskId), "候选经验已更新")}
              >
                提取候选经验
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>经验审核</CardTitle>
          <CardDescription>
            列出最近 200 条。候选经验不会自动进入模型上下文；采纳后仍需启用“使用已人工采纳的经验”。
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3">
          {lessons.length === 0 && <p className="text-muted-foreground text-sm">尚无经验记录。</p>}
          {lessons.map((l) => (
            <div key={l.id} className="grid gap-2 rounded-lg border p-4">
              <div className="flex flex-wrap justify-between gap-2">
                <span className="font-medium">{l.cue}</span>
                <span className="text-muted-foreground text-xs">
                  {lessonStates[l.status]} · {l.support} 条来源 ·{" "}
                  {l.company_id ? `公司 #${l.company_id}` : "原任务范围"}
                </span>
              </div>
              <p className="whitespace-pre-wrap break-words text-sm">{l.action}</p>
              <div className="flex gap-2">
                {(
                  [
                    ["adopted", "采纳"],
                    ["candidate", "退回候选"],
                    ["retired", "停用"],
                  ] as const
                )
                  .filter(([status]) => status !== l.status)
                  .map(([status, label]) => (
                    <Button
                      key={status}
                      size="sm"
                      variant="outline"
                      disabled={busy}
                      onClick={() => void perform(() => api.setHeroLessonStatus(l.id, status), "经验状态已更新")}
                    >
                      {label}
                    </Button>
                  ))}
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>模型费用估算</CardTitle>
          <CardDescription>
            单价单位为 EUR / 百万 Token。历史用量按当前单价估算，供比较配置使用；未填写价格时不显示费用。
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3">
          {profiles.length === 0 ? (
            <p className="text-muted-foreground text-sm">先在 LLM 页面添加模型配置。</p>
          ) : (
            profiles.map((p) => (
              <PriceRow
                key={`${p.id}-${p.price_input_per_m}-${p.price_output_per_m}-${p.price_cache_read_per_m}-${p.price_cache_write_per_m}`}
                profile={p}
                usage={usage.find((u) => u.profile_name === p.name)}
                onSaved={() => void load()}
              />
            ))
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>需要补充的 Skills</CardTitle>
          <CardDescription>根据任务实际调用失败记录整理，名称需经人工核对后再决定是否新增。</CardDescription>
        </CardHeader>
        <CardContent>
          {gaps.length === 0 ? (
            <p className="text-muted-foreground text-sm">尚无缺口记录。</p>
          ) : (
            <div className="overflow-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr>
                    <th className="p-2">Skill</th>
                    <th>Agent</th>
                    <th>未找到 / 调用成功</th>
                  </tr>
                </thead>
                <tbody>
                  {gaps.map((g, i) => (
                    <tr key={`${g.skill}-${g.agent_key}-${g.category_id}-${i}`} className="border-t">
                      <td className="p-2">{g.skill}</td>
                      <td>{g.agent_key}</td>
                      <td>
                        {g.misses} / {g.hits}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
