export interface UpstreamProject {
  repository: string;
  role: "mainline" | "donor";
  branch: string;
  baseline: string;
  baseline_url: string;
  repository_url: string;
  compare_url?: string;
  latest_commit?: { sha: string; title: string; date: string; html_url: string };
  latest_release?: { tag: string; published_at: string; html_url: string };
  release_status: "available" | "none" | "error";
  comparison: "identical" | "ahead" | "behind" | "diverged" | "unknown";
  ahead: number;
  behind: number;
  commit_error?: string;
  release_error?: string;
  comparison_error?: string;
  checked_at: string;
}
