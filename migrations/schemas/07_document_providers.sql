-- 文档处理服务商配置模块
-- 系统预设的文档解析服务提供商（如 MinerU、Unstructured 等）

CREATE TABLE document_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_type VARCHAR(50) NOT NULL UNIQUE,
    provider_name VARCHAR(100) NOT NULL,
    api_base_url VARCHAR(255),  -- API 地址（远程服务）或本地路径
    api_key TEXT,  -- API 密钥（开发阶段明文存储）
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_document_providers_type ON document_providers(provider_type);

-- 注释
COMMENT ON TABLE document_providers IS '文档处理服务商配置';
COMMENT ON COLUMN document_providers.id IS '主键ID';
COMMENT ON COLUMN document_providers.provider_type IS '服务商类型（唯一标识）';
COMMENT ON COLUMN document_providers.provider_name IS '服务商显示名称';
COMMENT ON COLUMN document_providers.api_base_url IS 'API 地址或本地路径';
COMMENT ON COLUMN document_providers.api_key IS 'API 密钥（开发环境明文，生产环境加密）';
COMMENT ON COLUMN document_providers.is_enabled IS '是否启用';

-- 插入预设数据
INSERT INTO document_providers (provider_type, provider_name, api_base_url, api_key, is_enabled) VALUES
('mineru', 'MinerU', 'https://mineru.net', NULL, true),
('unstructured', 'Unstructured', NULL, NULL, true),
('pymupdf', 'PyMuPDF', NULL, NULL, true);
