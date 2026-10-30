export interface LoneHeroOptions {
  lessons_mode: "off" | "shadow" | "inject";
  failure_escalation: boolean;
  goal_gate: boolean;
  quality_guidance: boolean;
}
export interface HeroLesson {
  id: number; kind: string; cue: string; action: string;
  company_id: number; category_id: number; status: "candidate" | "adopted" | "retired"; support: number;
}
export interface HeroGap { skill: string; agent_key: string; misses: number; hits: number; category_id: number }
export interface HeroOutcome {
  task_id: number; status: string; tool_calls: number; skill_misses: number; wall_seconds: number;
  findings: { total: number; confirmed: number; false_positive: number; duplicate: number };
  tokens: { input_tokens: number; output_tokens: number; cache_read_tokens: number; cache_write_tokens: number };
}
export interface HeroPrices { input: number; output: number; cache_read: number; cache_write: number }
