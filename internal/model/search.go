package model

// SearchResult represents a matched entity from full-text search queries.
type SearchResult struct {
	ID         string   `json:"id"`
	Type       string   `json:"type,omitempty"` // "task", "milestone", "strategy", "glossary"
	Title      string   `json:"title"`
	Tags       []string `json:"tags"`
	Snippet    string   `json:"snippet"`
	Score      float64  `json:"score"`
	Status     string   `json:"status,omitempty"`
	Priority   Priority `json:"priority,omitempty"`
	TargetDate string   `json:"target_date,omitempty"`
	Tier       Tier     `json:"tier,omitzero"`
}
