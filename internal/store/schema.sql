-- SQLite schema definition for Jokateko in-memory store and index
-- Compatible with pure Go modernc.org/sqlite and sqlc

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    milestone_id TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    total_criteria INTEGER NOT NULL DEFAULT 0,
    completed_criteria INTEGER NOT NULL DEFAULT 0,
    filepath TEXT NOT NULL DEFAULT '',
    mtime INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT '',
    changed_at TEXT NOT NULL DEFAULT '',
    target_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_milestone ON tasks(milestone_id);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

CREATE TABLE IF NOT EXISTS milestones (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    target_date TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    filepath TEXT NOT NULL DEFAULT '',
    mtime INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_milestones_status ON milestones(status);

CREATE TABLE IF NOT EXISTS strategies (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    tier INTEGER NOT NULL,
    summary TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    filepath TEXT NOT NULL DEFAULT '',
    mtime INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_strategies_tier ON strategies(tier);

CREATE TABLE IF NOT EXISTS glossary (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    filepath TEXT NOT NULL DEFAULT '',
    mtime INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id TEXT NOT NULL,
    depends_on_task_id TEXT NOT NULL,
    PRIMARY KEY (task_id, depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_deps_dep ON task_dependencies(depends_on_task_id);

CREATE TABLE IF NOT EXISTS entity_tags (
    entity_type TEXT NOT NULL, -- 'task', 'milestone', 'strategy', 'glossary'
    entity_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (entity_type, entity_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_entity_tags_tag ON entity_tags(tag);
CREATE INDEX IF NOT EXISTS idx_entity_tags_lookup ON entity_tags(entity_type, entity_id);

-- Full-Text Search (FTS5) table across all entities
CREATE VIRTUAL TABLE IF NOT EXISTS fts_entities USING fts5(
    entity_type UNINDEXED,
    entity_id UNINDEXED,
    title,
    summary,
    body
);
