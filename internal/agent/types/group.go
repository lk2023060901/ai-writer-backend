package types

// Common agent groups/categories
const (
	GroupCareer    = "职业"
	GroupBusiness  = "商业"
	GroupTool      = "工具"
	GroupWriting   = "写作"
	GroupMarketing = "营销"
	GroupTech      = "技术"
	GroupEducation = "教育"
	GroupLife      = "生活"
)

// AgentGroup represents an agent category
type AgentGroup struct {
	Name  string `json:"name"`
	Count int    `json:"count"` // Number of agents in this group
}
