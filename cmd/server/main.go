package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/injector"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
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

	// Initialize logger
	log, err := initLogger(config)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer log.Sync()

	log.Info("config loaded successfully")

	// Initialize application with Wire dependency injection
	app, cleanup, err := injector.InitializeApp(config, log)
	if err != nil {
		log.Fatal("failed to initialize application", zap.Error(err))
	}
	defer cleanup()

	// Start servers
	startServers(app)

	// Graceful shutdown
	waitForShutdown(app)
}

// initLogger initializes the logger with configuration
func initLogger(config *conf.Config) (*logger.Logger, error) {
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
		return nil, err
	}

	// Initialize global logger
	if err := logger.InitGlobal(logConfig); err != nil {
		return nil, err
	}

	return log, nil
}

// startServers starts HTTP and gRPC servers in goroutines
func startServers(app *injector.App) {
	// Start HTTP server
	go func() {
		if err := app.HTTPServer.Start(); err != nil {
			app.Logger.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		if err := app.GRPCServer.Start(); err != nil {
			app.Logger.Fatal("failed to start gRPC server", zap.Error(err))
		}
	}()

	app.Logger.Info("servers started successfully")
}

// waitForShutdown waits for interrupt signal and performs graceful shutdown
func waitForShutdown(app *injector.App) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Logger.Info("shutting down servers...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop gRPC server
	app.GRPCServer.Stop()

	// Stop HTTP server
	if err := app.HTTPServer.Stop(ctx); err != nil {
		app.Logger.Error("HTTP server forced to shutdown", zap.Error(err))
	}

	app.Logger.Info("servers exited")
}
