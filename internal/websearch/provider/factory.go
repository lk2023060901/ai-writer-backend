package provider

import (
	"fmt"
	"sync"

	"github.com/lk2023060901/ai-writer-backend/internal/websearch/types"
)

// Factory creates provider instances
type Factory struct {
	mu           sync.RWMutex
	constructors map[types.ProviderID]func(*types.ProviderConfig) (Provider, error)
}

// NewFactory creates a new provider factory
func NewFactory() *Factory {
	f := &Factory{
		constructors: make(map[types.ProviderID]func(*types.ProviderConfig) (Provider, error)),
	}

	// Register built-in providers
	f.Register(types.ProviderTavily, NewTavilyProvider)
	f.Register(types.ProviderSearXNG, NewSearXNGProvider)
	f.Register(types.ProviderExa, NewExaProvider)
	f.Register(types.ProviderZhipu, NewZhipuProvider)
	f.Register(types.ProviderBocha, NewBochaProvider)

	return f
}

// Register registers a provider constructor
func (f *Factory) Register(id types.ProviderID, constructor func(*types.ProviderConfig) (Provider, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.constructors[id] = constructor
}

// Create creates a provider instance from configuration
func (f *Factory) Create(config *types.ProviderConfig) (Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	f.mu.RLock()
	constructor, exists := f.constructors[config.ID]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, config.ID)
	}

	return constructor(config)
}

// ListProviders returns a list of all registered provider IDs
func (f *Factory) ListProviders() []types.ProviderID {
	f.mu.RLock()
	defer f.mu.RUnlock()

	ids := make([]types.ProviderID, 0, len(f.constructors))
	for id := range f.constructors {
		ids = append(ids, id)
	}
	return ids
}
