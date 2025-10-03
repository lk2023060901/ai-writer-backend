package injector

import (
	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	kbqueue "github.com/lk2023060901/ai-writer-backend/internal/knowledge/queue"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/server"
)

// App encapsulates all application dependencies
type App struct {
	Config         *conf.Config
	Logger         *logger.Logger
	HTTPServer     *server.HTTPServer
	GRPCServer     *server.GRPCServer
	DocumentWorker *kbqueue.Worker
	cleanup        func()
}

// Cleanup releases all resources (kept for backward compatibility)
func (a *App) Cleanup() {
	if a.cleanup != nil {
		a.cleanup()
	}
}
