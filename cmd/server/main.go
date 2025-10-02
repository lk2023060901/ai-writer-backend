package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/data"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/server"
	"github.com/lk2023060901/ai-writer-backend/internal/user/biz"
	userdata "github.com/lk2023060901/ai-writer-backend/internal/user/data"
	"github.com/lk2023060901/ai-writer-backend/internal/user/service"
	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "config.yaml", "config file path")
)

func main() {
	flag.Parse()

	// Load configuration
	config, err := conf.LoadConfig(*configFile)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger with config
	logConfig := &logger.Config{
		Level:            config.Log.Level,
		Format:           config.Log.Format,
		Output:           config.Log.Output,
		EnableCaller:     config.Log.EnableCaller,
		EnableStacktrace: config.Log.EnableStacktrace,
		File: logger.FileConfig{
			Filename:   config.Log.File.Filename,
			MaxSize:    config.Log.File.MaxSize,
			MaxAge:     config.Log.File.MaxAge,
			MaxBackups: config.Log.File.MaxBackups,
			Compress:   config.Log.File.Compress,
		},
	}

	log, err := logger.New(logConfig)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer log.Sync()

	// Initialize global logger
	if err := logger.InitGlobal(logConfig); err != nil {
		log.Fatal("failed to initialize global logger", zap.Error(err))
	}

	log.Info("config loaded successfully")

	// Initialize data layer
	d, cleanup, err := data.NewData(config, log.Logger)
	if err != nil {
		log.Fatal("failed to initialize data layer", zap.Error(err))
	}
	defer cleanup()

	// Initialize repositories
	userRepo := userdata.NewUserRepo(d.DB)

	// Initialize use cases
	userUseCase := biz.NewUserUseCase(userRepo)

	// Initialize services
	userService := service.NewUserService(userUseCase, log.Logger)

	// Initialize HTTP server
	httpServer := server.NewHTTPServer(config, log.Logger, userService)

	// Start server in goroutine
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()

	log.Info("server started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Stop(ctx); err != nil {
		log.Error("server forced to shutdown", zap.Error(err))
	}

	log.Info("server exited")
}
