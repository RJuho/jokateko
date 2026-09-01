import * as v from "valibot";

export const PrioritySchema = v.picklist(["low", "medium", "high", "critical"]);

export const MilestoneStatusSchema = v.picklist(["open", "closed"]);

export const TierSchema = v.picklist([1, 2, 3]);

export const ColumnSchema = v.object({
  id: v.string(),
  name: v.string(),
  color: v.string(),
});

export const TaskSchema = v.object({
  id: v.string(),
  title: v.string(),
  status: v.string(),
  priority: v.optional(PrioritySchema, "medium"),
  milestone: v.optional(v.nullish(v.string())),
  tags: v.optional(v.array(v.string()), []),
  summary: v.optional(v.string(), ""),
  dependencies: v.optional(v.array(v.string()), []),
  body: v.optional(v.string(), ""),
  total_criteria: v.optional(v.number(), 0),
  completed_criteria: v.optional(v.number(), 0),
  file_path: v.optional(v.string()),
  mod_time: v.optional(v.string()),
});

export const MilestoneSchema = v.object({
  id: v.string(),
  title: v.string(),
  status: v.optional(MilestoneStatusSchema, "open"),
  is_archived: v.optional(v.boolean(), false),
  target_date: v.optional(v.nullish(v.string())),
  tags: v.optional(v.array(v.string()), []),
  summary: v.optional(v.string(), ""),
  body: v.optional(v.string(), ""),
  total_tasks: v.optional(v.number(), 0),
  completed_tasks: v.optional(v.number(), 0),
  progress_percentage: v.optional(v.number(), 0),
  file_path: v.optional(v.string()),
  mod_time: v.optional(v.string()),
});

export const StrategySchema = v.object({
  id: v.string(),
  title: v.string(),
  tier: TierSchema,
  tags: v.optional(v.array(v.string()), []),
  summary: v.optional(v.string(), ""),
  body: v.optional(v.string(), ""),
  file_path: v.optional(v.string()),
  mod_time: v.optional(v.string()),
});

export const GlossaryTermSchema = v.object({
  id: v.string(),
  title: v.string(),
  tags: v.optional(v.array(v.string()), []),
  summary: v.optional(v.string(), ""),
  body: v.optional(v.string(), ""),
  file_path: v.optional(v.string()),
  mod_time: v.optional(v.string()),
});

export const ProjectConfigSchema = v.object({
  name: v.string(),
  description: v.optional(v.string(), ""),
});

export const BoardConfigSchema = v.object({
  columns: v.array(ColumnSchema),
});

export const TagsConfigSchema = v.object({
  allowed: v.optional(v.array(v.string()), []),
  enforce_allowed: v.optional(v.boolean(), false),
});

export const SnapshotConfigSchema = v.object({
  project: ProjectConfigSchema,
  board: BoardConfigSchema,
  tags: v.optional(TagsConfigSchema),
});

export const SnapshotSchema = v.object({
  config: SnapshotConfigSchema,
  tasks: v.array(TaskSchema),
  milestones: v.array(MilestoneSchema),
  strategies: v.array(StrategySchema),
  glossary: v.array(GlossaryTermSchema),
});

export const ColumnStateSchema = v.object({
  id: v.string(),
  name: v.string(),
  color: v.string(),
  tasks: v.array(TaskSchema),
  count: v.number(),
});

export const BoardStateSchema = v.object({
  project_name: v.string(),
  columns: v.array(ColumnStateSchema),
});

// Automatically inferred TypeScript types (Single source of truth)
export type Priority = v.InferOutput<typeof PrioritySchema>;
export type MilestoneStatus = v.InferOutput<typeof MilestoneStatusSchema>;
export type Tier = v.InferOutput<typeof TierSchema>;
export type Column = v.InferOutput<typeof ColumnSchema>;
export type Task = v.InferOutput<typeof TaskSchema>;
export type Milestone = v.InferOutput<typeof MilestoneSchema>;
export type Strategy = v.InferOutput<typeof StrategySchema>;
export type GlossaryTerm = v.InferOutput<typeof GlossaryTermSchema>;
export type ColumnState = v.InferOutput<typeof ColumnStateSchema>;
export type BoardState = v.InferOutput<typeof BoardStateSchema>;
export type SnapshotConfig = v.InferOutput<typeof SnapshotConfigSchema>;
export type Snapshot = v.InferOutput<typeof SnapshotSchema>;
export type TaskInput = v.InferInput<typeof TaskSchema>;
export type SnapshotInput = v.InferInput<typeof SnapshotSchema>;

