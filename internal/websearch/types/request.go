package types

// SearchRequest represents a search request
type SearchRequest struct {
	Query          string                 `json:"query" validate:"required,min=1,max=1000"`
	MaxResults     int                    `json:"max_results,omitempty" validate:"omitempty,min=1,max=100"`
	SearchDepth    string                 `json:"search_depth,omitempty"` // "basic" or "advanced"
	IncludeDomains []string               `json:"include_domains,omitempty"`
	ExcludeDomains []string               `json:"exclude_domains,omitempty"`
	TimeRange      *TimeRange             `json:"time_range,omitempty"`
	Options        map[string]interface{} `json:"options,omitempty"` // Provider-specific options
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start string `json:"start,omitempty"` // ISO 8601 format
	End   string `json:"end,omitempty"`   // ISO 8601 format
}
