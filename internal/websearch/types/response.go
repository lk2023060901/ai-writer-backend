package types

// SearchResponse represents a search response
type SearchResponse struct {
	Query      string          `json:"query"`
	Results    []*SearchResult `json:"results"`
	TotalCount int             `json:"total_count,omitempty"`
	Took       int64           `json:"took"` // milliseconds
	Provider   ProviderID      `json:"provider"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title       string                 `json:"title"`
	URL         string                 `json:"url"`
	Content     string                 `json:"content"` // Snippet or full content
	Score       float32                `json:"score,omitempty"`
	PublishedAt string                 `json:"published_at,omitempty"`
	Author      string                 `json:"author,omitempty"`
	Images      []string               `json:"images,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
