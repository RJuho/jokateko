package model

// TagCount captures the usage metrics of a controlled vocabulary tag across entities.
type TagCount struct {
	Tag            string `json:"tag"`
	TaskCount      int    `json:"task_count"`
	MilestoneCount int    `json:"milestone_count"`
	StrategyCount  int    `json:"strategy_count"`
}

// TagList captures the list of tags returned to AI agents and Web UI clients.
type TagList struct {
	Enforced bool       `json:"enforced"`
	Tags     []TagCount `json:"tags"`
}
