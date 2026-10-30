"use client";

import * as React from "react";
import { ExternalLinkIcon, RefreshCwIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import type { UpstreamProject } from "@/lib/upstream-types";

function dateText(value?: string) {
  if (!value || value.startsWith("0001-")) return "时间未知";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "时间未知" : date.toLocaleString("zh-CN");
}

function relation(project: UpstreamProject) {
  switch (project.comparison) {
    case "identical":
      return "与记录快照一致";
    case "ahead":
      return `快照之后新增 ${project.ahead} 个提交`;
    case "behind":
      return "远端分支早于记录快照";
    case "diverged":
      return "提交历史已分叉，需人工核对";
    default:
      return "提交关系暂时未知";
  }
}

export function UpstreamStatus({ revision, disabled }: { revision: number; disabled: boolean }) {
  const [projects, setProjects] = React.useState<UpstreamProject[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState("");
  const sequence = React.useRef(0);
  const load = React.useCallback(async (force: boolean) => {
    const seq = ++sequence.current;
    setLoading(true);
    setError("");
    try {
      const result = await api.checkUpstreams(force);
      if (sequence.current !== seq) return;
      if (!result.projects?.length) throw new Error("未能读取上游来源记录");
      setProjects(result.projects);
    } catch (e) {
      if (sequence.current === seq) setError((e as Error).message);
    } finally {
      if (sequence.current === seq) setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load(revision > 0);
    return () => {
      sequence.current++;
    };
  }, [revision, load]);

  return (
    <section aria-label="上游项目动态" className="space-y-3 border-t pt-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="font-medium text-sm">上游项目动态</h3>
        <Button variant="outline" size="sm" disabled={loading || disabled} onClick={() => void load(true)}>
          <RefreshCwIcon className={loading ? "size-3.5 animate-spin" : "size-3.5"} />
          {loading ? "正在查询…" : "刷新上游动态"}
        </Button>
      </div>
      <p className="text-muted-foreground text-xs">
        RuoJi6 为主干，BaaRai 为能力供体。以下动态用于评估后续整合，程序升级仍使用本项目的私有正式版。
      </p>
      {error && (
        <p role="alert" className="text-destructive text-xs">
          查询失败：{error}。已显示的信息可能来自上次查询。
        </p>
      )}
      {projects.map((project) => (
        <div key={project.repository} className="space-y-2 rounded-md border p-3 text-xs">
          <div className="flex flex-wrap items-center gap-2">
            <a
              href={project.repository_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 font-medium underline-offset-4 hover:underline"
            >
              {project.repository} <ExternalLinkIcon className="size-3" />
            </a>
            <Badge variant="outline">{project.role === "mainline" ? "主干" : "能力供体"}</Badge>
            <Badge variant={project.comparison === "ahead" ? "secondary" : "outline"}>{relation(project)}</Badge>
          </div>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-muted-foreground">
            <span>跟踪分支 {project.branch}</span>
            <a href={project.baseline_url} target="_blank" rel="noreferrer" className="underline underline-offset-4">
              {project.role === "mainline" ? "集成基线" : "参考快照"} {project.baseline.slice(0, 7)}
            </a>
          </div>
          <p>
            正式版：
            {project.latest_release ? (
              <a
                href={project.latest_release.html_url}
                target="_blank"
                rel="noreferrer"
                className="underline underline-offset-4"
              >
                {project.latest_release.tag} · {dateText(project.latest_release.published_at)}
              </a>
            ) : project.release_status === "none" ? (
              "尚无正式发布"
            ) : (
              "暂时无法查询"
            )}
          </p>
          {project.latest_commit ? (
            <div className="space-y-1">
              <p className="break-words">
                最新提交：
                <a
                  href={project.latest_commit.html_url}
                  target="_blank"
                  rel="noreferrer"
                  className="underline underline-offset-4"
                >
                  {project.latest_commit.sha.slice(0, 7)} · {project.latest_commit.title}
                </a>
              </p>
              <p className="text-muted-foreground">提交时间：{dateText(project.latest_commit.date)}</p>
            </div>
          ) : (
            <p className="text-muted-foreground">最新提交暂时无法查询。</p>
          )}
          {[project.commit_error, project.release_error, project.comparison_error]
            .filter(Boolean)
            .map((message, index) => (
              <p key={`${index}-${message}`} className="text-destructive">
                {message}
              </p>
            ))}
          {project.compare_url && project.comparison !== "identical" && (
            <a
              href={project.compare_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 underline underline-offset-4"
            >
              查看与快照的差异 <ExternalLinkIcon className="size-3" />
            </a>
          )}
          <p className="text-muted-foreground">最近查询：{dateText(project.checked_at)}</p>
        </div>
      ))}
      <p className="text-muted-foreground text-xs">
        提交差异以记录的快照为准，不等于本项目待安装的更新；供体能力仍需选择性评估。
      </p>
    </section>
  );
}
