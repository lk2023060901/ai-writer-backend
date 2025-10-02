package data

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/agent/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"gorm.io/gorm"
)

// StringArrayJSON 自定义 JSONB 类型（用于存储字符串数组）
type StringArrayJSON []string

func (j *StringArrayJSON) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j StringArrayJSON) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal([]string{})
	}
	return json.Marshal(j)
}

// AgentPO 用户智能体数据库模型
type AgentPO struct {
	ID               string          `gorm:"type:uuid;primarykey"`
	OwnerID          string          `gorm:"type:uuid;not null;index:idx_agents_owner_id,where:deleted_at IS NULL"`
	Name             string          `gorm:"size:255;not null"`
	Emoji            string          `gorm:"size:10;default:'🤖'"`
	Prompt           string          `gorm:"type:text;not null"`
	KnowledgeBaseIDs StringArrayJSON `gorm:"type:jsonb;not null;default:'[]'"`
	Tags             StringArrayJSON `gorm:"type:jsonb;not null;default:'[]';index:idx_agents_tags,type:gin"`
	Type             string          `gorm:"size:50;not null;default:'agent'"`
	IsEnabled        bool            `gorm:"not null;default:true;index:idx_agents_is_enabled,where:deleted_at IS NULL"`
	CreatedAt        time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt        gorm.DeletedAt  `gorm:"index:idx_agents_deleted_at"`
}

func (AgentPO) TableName() string {
	return "agents"
}

// AgentRepo 用户智能体仓储实现
type AgentRepo struct {
	db *database.DB
}

// NewAgentRepo 创建用户智能体仓储
func NewAgentRepo(db *database.DB) biz.AgentRepo {
	return &AgentRepo{db: db}
}

// Create 创建智能体
func (r *AgentRepo) Create(ctx context.Context, agent *biz.Agent) error {
	po := &AgentPO{
		ID:               agent.ID,
		OwnerID:          agent.OwnerID,
		Name:             agent.Name,
		Emoji:            agent.Emoji,
		Prompt:           agent.Prompt,
		KnowledgeBaseIDs: agent.KnowledgeBaseIDs,
		Tags:             agent.Tags,
		Type:             agent.Type,
		IsEnabled:        agent.IsEnabled,
		CreatedAt:        agent.CreatedAt,
		UpdatedAt:        agent.UpdatedAt,
	}

	return r.db.WithContext(ctx).GetDB().Create(po).Error
}

// GetByID 根据ID获取智能体（需要验证所有者）
func (r *AgentRepo) GetByID(ctx context.Context, id string, ownerID string) (*biz.Agent, error) {
	var po AgentPO
	err := r.db.WithContext(ctx).GetDB().
		Where("id = ? AND owner_id = ? AND deleted_at IS NULL", id, ownerID).
		First(&po).Error

	if err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrAgentNotFound
		}
		return nil, err
	}

	return r.toAgent(&po), nil
}

// List 获取智能体列表（分页、过滤）
func (r *AgentRepo) List(ctx context.Context, req *biz.ListAgentsRequest) ([]*biz.Agent, int64, error) {
	var pos []AgentPO
	var total int64

	query := r.db.WithContext(ctx).GetDB().Model(&AgentPO{}).
		Where("owner_id = ? AND deleted_at IS NULL", req.UserID)

	// 启用状态过滤
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	// 关键词搜索（按名称）
	if req.Keyword != "" {
		query = query.Where("name ILIKE ?", "%"+req.Keyword+"%")
	}

	// 标签过滤（JSONB 包含查询）
	if len(req.Tags) > 0 {
		tagsJSON, _ := json.Marshal(req.Tags)
		query = query.Where("tags @> ?", string(tagsJSON))
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	err := query.
		Order("is_enabled DESC, created_at DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&pos).Error

	if err != nil {
		return nil, 0, err
	}

	agents := make([]*biz.Agent, len(pos))
	for i, po := range pos {
		agents[i] = r.toAgent(&po)
	}

	return agents, total, nil
}

// Update 更新智能体
func (r *AgentRepo) Update(ctx context.Context, agent *biz.Agent) error {
	updates := map[string]interface{}{
		"name":               agent.Name,
		"emoji":              agent.Emoji,
		"prompt":             agent.Prompt,
		"knowledge_base_ids": StringArrayJSON(agent.KnowledgeBaseIDs),
		"tags":               StringArrayJSON(agent.Tags),
		"is_enabled":         agent.IsEnabled,
		"updated_at":         agent.UpdatedAt,
	}

	result := r.db.WithContext(ctx).GetDB().
		Model(&AgentPO{}).
		Where("id = ? AND owner_id = ? AND deleted_at IS NULL", agent.ID, agent.OwnerID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrAgentNotFound
	}

	return nil
}

// Delete 删除智能体（软删除）
func (r *AgentRepo) Delete(ctx context.Context, id string, ownerID string) error {
	result := r.db.WithContext(ctx).GetDB().
		Where("id = ? AND owner_id = ?", id, ownerID).
		Delete(&AgentPO{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrAgentNotFound
	}

	return nil
}

// UpdateEnabled 更新启用状态
func (r *AgentRepo) UpdateEnabled(ctx context.Context, id string, ownerID string, enabled bool) error {
	result := r.db.WithContext(ctx).GetDB().
		Model(&AgentPO{}).
		Where("id = ? AND owner_id = ? AND deleted_at IS NULL", id, ownerID).
		Updates(map[string]interface{}{
			"is_enabled": enabled,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrAgentNotFound
	}

	return nil
}

// toAgent 转换 PO 到业务对象
func (r *AgentRepo) toAgent(po *AgentPO) *biz.Agent {
	return &biz.Agent{
		ID:               po.ID,
		OwnerID:          po.OwnerID,
		Name:             po.Name,
		Emoji:            po.Emoji,
		Prompt:           po.Prompt,
		KnowledgeBaseIDs: po.KnowledgeBaseIDs,
		Tags:             po.Tags,
		Type:             po.Type,
		IsEnabled:        po.IsEnabled,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}
