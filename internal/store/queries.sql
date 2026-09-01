-- Tasks
-- name: UpsertTask :exec
INSERT INTO tasks (
    id, title, status, priority, milestone_id, summary, body, total_criteria, completed_criteria, filepath, mtime
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    status = excluded.status,
    priority = excluded.priority,
    milestone_id = excluded.milestone_id,
    summary = excluded.summary,
    body = excluded.body,
    total_criteria = excluded.total_criteria,
    completed_criteria = excluded.completed_criteria,
    filepath = excluded.filepath,
    mtime = excluded.mtime;

-- name: GetTask :one
SELECT id, title, status, priority, milestone_id, summary, body, total_criteria, completed_criteria, filepath, mtime
FROM tasks
WHERE id = ?;

-- name: ListTasks :many
SELECT id, title, status, priority, milestone_id, summary, body, total_criteria, completed_criteria, filepath, mtime
FROM tasks
ORDER BY id ASC;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: UpdateTaskStatus :exec
UPDATE tasks
SET status = ?, mtime = ?
WHERE id = ?;

-- name: UpdateTaskCriteria :exec
UPDATE tasks
SET total_criteria = ?, completed_criteria = ?, body = ?, mtime = ?
WHERE id = ?;

-- Dependencies
-- name: ClearTaskDependencies :exec
DELETE FROM task_dependencies WHERE task_id = ?;

-- name: AddTaskDependency :exec
INSERT OR IGNORE INTO task_dependencies (task_id, depends_on_task_id)
VALUES (?, ?);

-- name: GetTaskDependencies :many
SELECT depends_on_task_id
FROM task_dependencies
WHERE task_id = ?
ORDER BY depends_on_task_id ASC;

-- name: GetDownstreamTasks :many
SELECT t.id, t.title, t.status
FROM task_dependencies td
JOIN tasks t ON td.task_id = t.id
WHERE td.depends_on_task_id = ?;

-- name: CountUnfinishedDependencies :one
SELECT COUNT(*)
FROM task_dependencies td
JOIN tasks t ON td.depends_on_task_id = t.id
WHERE td.task_id = ? AND t.status != 'done';

-- Milestones
-- name: UpsertMilestone :exec
INSERT INTO milestones (
    id, title, status, target_date, summary, body, filepath, mtime
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
) ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    status = excluded.status,
    target_date = excluded.target_date,
    summary = excluded.summary,
    body = excluded.body,
    filepath = excluded.filepath,
    mtime = excluded.mtime;

-- name: GetMilestone :one
SELECT id, title, status, target_date, summary, body, filepath, mtime
FROM milestones
WHERE id = ?;

-- name: ListMilestones :many
SELECT id, title, status, target_date, summary, body, filepath, mtime
FROM milestones
ORDER BY id ASC;

-- name: DeleteMilestone :exec
DELETE FROM milestones WHERE id = ?;

-- name: GetMilestoneTaskMetrics :one
SELECT
    COUNT(*) AS total_tasks,
    COUNT(CASE WHEN status = 'done' THEN 1 END) AS completed_tasks
FROM tasks
WHERE milestone_id = ?;

-- Strategies
-- name: UpsertStrategy :exec
INSERT INTO strategies (
    id, title, tier, summary, body, filepath, mtime
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
) ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    tier = excluded.tier,
    summary = excluded.summary,
    body = excluded.body,
    filepath = excluded.filepath,
    mtime = excluded.mtime;

-- name: GetStrategy :one
SELECT id, title, tier, summary, body, filepath, mtime
FROM strategies
WHERE id = ?;

-- name: ListStrategies :many
SELECT id, title, tier, summary, body, filepath, mtime
FROM strategies
ORDER BY tier ASC, id ASC;

-- name: ListStrategiesByTier :many
SELECT id, title, tier, summary, body, filepath, mtime
FROM strategies
WHERE tier = ?
ORDER BY id ASC;

-- name: DeleteStrategy :exec
DELETE FROM strategies WHERE id = ?;

-- Glossary
-- name: UpsertGlossaryTerm :exec
INSERT INTO glossary (
    id, title, summary, body, filepath, mtime
) VALUES (
    ?, ?, ?, ?, ?, ?
) ON CONFLICT(id) DO UPDATE SET
    title = excluded.title,
    summary = excluded.summary,
    body = excluded.body,
    filepath = excluded.filepath,
    mtime = excluded.mtime;

-- name: GetGlossaryTerm :one
SELECT id, title, summary, body, filepath, mtime
FROM glossary
WHERE id = ?;

-- name: ListGlossaryTerms :many
SELECT id, title, summary, body, filepath, mtime
FROM glossary
ORDER BY id ASC;

-- name: DeleteGlossaryTerm :exec
DELETE FROM glossary WHERE id = ?;

-- Tags
-- name: ClearEntityTags :exec
DELETE FROM entity_tags WHERE entity_type = ? AND entity_id = ?;

-- name: AddEntityTag :exec
INSERT OR IGNORE INTO entity_tags (entity_type, entity_id, tag)
VALUES (?, ?, ?);

-- name: GetEntityTags :many
SELECT tag
FROM entity_tags
WHERE entity_type = ? AND entity_id = ?
ORDER BY tag ASC;

-- name: GetTagCounts :many
SELECT
    tag,
    COUNT(CASE WHEN entity_type = 'task' THEN 1 END) AS task_count,
    COUNT(CASE WHEN entity_type = 'milestone' THEN 1 END) AS milestone_count,
    COUNT(CASE WHEN entity_type = 'strategy' THEN 1 END) AS strategy_count
FROM entity_tags
GROUP BY tag
ORDER BY tag ASC;

-- Full-Text Search (FTS5)
-- name: InsertFTSEntity :exec
INSERT INTO fts_entities (entity_type, entity_id, title, summary, body)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteFTSEntity :exec
DELETE FROM fts_entities WHERE entity_type = ? AND entity_id = ?;
