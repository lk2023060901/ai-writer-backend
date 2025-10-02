package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/agent/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"gorm.io/gorm"
)

// OfficialAgentPO 官方智能体数据库模型（无 owner_id 字段）
type OfficialAgentPO struct {
	ID               string          `gorm:"type:uuid;primarykey"`
	Name             string          `gorm:"size:255;not null"`
	Emoji            string          `gorm:"size:10;default:'🤖'"`
	Prompt           string          `gorm:"type:text;not null"`
	KnowledgeBaseIDs StringArrayJSON `gorm:"type:jsonb;not null;default:'[]'"`
	Tags             StringArrayJSON `gorm:"type:jsonb;not null;default:'[]';index:idx_official_agents_tags,type:gin"`
	Type             string          `gorm:"size:50;not null;default:'agent'"`
	IsEnabled        bool            `gorm:"not null;default:true;index:idx_official_agents_is_enabled,where:deleted_at IS NULL"`
	CreatedAt        time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt        gorm.DeletedAt  `gorm:"index:idx_official_agents_deleted_at"`
}

func (OfficialAgentPO) TableName() string {
	return "official_agents"
}

// OfficialAgentRepo 官方智能体仓储实现
type OfficialAgentRepo struct {
	db *database.DB
}

// NewOfficialAgentRepo 创建官方智能体仓储
func NewOfficialAgentRepo(db *database.DB) biz.OfficialAgentRepo {
	return &OfficialAgentRepo{db: db}
}

// GetByID 根据ID获取官方智能体
func (r *OfficialAgentRepo) GetByID(ctx context.Context, id string) (*biz.Agent, error) {
	var po OfficialAgentPO
	err := r.db.WithContext(ctx).GetDB().
		Where("id = ? AND deleted_at IS NULL", id).
		First(&po).Error

	if err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrAgentNotFound
		}
		return nil, err
	}

	return r.toAgent(&po), nil
}

// List 获取官方智能体列表（分页、过滤）
func (r *OfficialAgentRepo) List(ctx context.Context, req *biz.ListAgentsRequest) ([]*biz.Agent, int64, error) {
	var pos []OfficialAgentPO
	var total int64

	query := r.db.WithContext(ctx).GetDB().Model(&OfficialAgentPO{}).
		Where("deleted_at IS NULL AND is_enabled = true") // 只返回启用的官方智能体

	// 启用状态过滤（可选，但官方智能体默认只显示启用的）
	if req.IsEnabled != nil && !*req.IsEnabled {
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
		Order("created_at DESC").
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

// toAgent 转换 PO 到业务对象（手动设置 OwnerID 为 SystemOwnerID）
func (r *OfficialAgentRepo) toAgent(po *OfficialAgentPO) *biz.Agent {
	return &biz.Agent{
		ID:               po.ID,
		OwnerID:          biz.SystemOwnerID, // 手动设置为系统 UUID
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
