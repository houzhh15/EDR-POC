// Package main 是 API Gateway 服务的入口点
//
// API Gateway 负责：
//   - RESTful API 接口（Console 调用）
//   - Agent gRPC 接口（Agent 连接）
//   - 请求路由和负载均衡
//   - 认证鉴权
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 版本信息
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("API Gateway starting",
		zap.String("version", Version),
		zap.String("commit", GitCommit),
	)

	// 创建 Gin 路由
	router := gin.New()
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": Version,
		})
	})

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// TODO: 添加 API 路由
		v1.GET("/agents", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"agents": []string{}})
		})
		v1.GET("/alerts", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"alerts": []string{}})
		})
		v1.GET("/events", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"events": []string{}})
		})
	}

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 启动服务器
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("API Gateway stopped")
}
