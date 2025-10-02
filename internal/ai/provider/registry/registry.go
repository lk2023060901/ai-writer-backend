package registry

import (
	"fmt"
	"sync"

	"github.com/lk2023060901/ai-writer-backend/internal/ai/provider/types"
)

var (
	mu        sync.RWMutex
	providers = make(map[string]types.Provider)
	aliases   = make(map[string]string) // alias -> real name
)

// Register 注册 Provider（支持别名）
func Register(name string, provider types.Provider, aliasNames ...string) {
	mu.Lock()
	defer mu.Unlock()

	// 注册主 Provider
	providers[name] = provider

	// 注册别名
	for _, alias := range aliasNames {
		aliases[alias] = name
	}
}

// Get 获取 Provider（支持别名）
func Get(nameOrAlias string) (types.Provider, error) {
	mu.RLock()
	defer mu.RUnlock()

	// 解析别名
	realName := resolveAliasLocked(nameOrAlias)

	provider, ok := providers[realName]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", nameOrAlias)
	}
	return provider, nil
}

// ResolveAlias 解析别名为真实名称（公开方法）
func ResolveAlias(nameOrAlias string) string {
	mu.RLock()
	defer mu.RUnlock()
	return resolveAliasLocked(nameOrAlias)
}

// resolveAliasLocked 解析别名（内部方法，不加锁）
func resolveAliasLocked(nameOrAlias string) string {
	if realName, ok := aliases[nameOrAlias]; ok {
		return realName
	}
	return nameOrAlias
}

// IsAlias 检查是否为别名
func IsAlias(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := aliases[name]
	return ok
}

// GetAliases 获取指定 Provider 的所有别名
func GetAliases(name string) []string {
	mu.RLock()
	defer mu.RUnlock()

	result := []string{}
	for alias, realName := range aliases {
		if realName == name {
			result = append(result, alias)
		}
	}
	return result
}

// List 列出所有 Provider 名称（不包括别名）
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// ListAll 列出所有名称（包括别名）
func ListAll() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(providers)+len(aliases))
	for name := range providers {
		names = append(names, name)
	}
	for alias := range aliases {
		names = append(names, alias)
	}
	return names
}

// Unregister 注销 Provider（同时删除别名）
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()

	// 删除 Provider
	delete(providers, name)

	// 删除所有指向该 Provider 的别名
	for alias, realName := range aliases {
		if realName == name {
			delete(aliases, alias)
		}
	}
}

// UnregisterAlias 仅注销别名
func UnregisterAlias(alias string) {
	mu.Lock()
	defer mu.Unlock()
	delete(aliases, alias)
}

// Clear 清空所有 Provider 和别名
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	providers = make(map[string]types.Provider)
	aliases = make(map[string]string)
}
