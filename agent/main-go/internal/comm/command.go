// Package comm 提供 Agent 与云端的通信功能
package comm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/log"
	pb "github.com/houzhh15/EDR-POC/agent/main-go/pkg/proto/edr/v1"
)

// CommandExecutor 命令执行器接口
type CommandExecutor interface {
	// Execute 执行命令，返回输出和错误
	Execute(ctx context.Context, cmd *pb.Command) (output string, err error)
	// SupportedCommands 返回支持的命令类型列表
	SupportedCommands() []string
}

// CommandConfig 命令客户端配置
type CommandConfig struct {
	AgentID         string        // Agent ID
	DefaultTimeout  time.Duration // 默认超时时间
	MaxConcurrent   int           // 最大并发执行数
	ReconnectDelay  time.Duration // 重连延迟
	HeartbeatPeriod time.Duration // 心跳周期（保持流活跃）
}

// CommandClient 命令执行客户端
type CommandClient struct {
	config   CommandConfig
	conn     *Connection
	logger   *log.Logger
	executor CommandExecutor

	mu         sync.RWMutex
	isRunning  bool
	stopCh     chan struct{}
	activeCmds sync.WaitGroup // 跟踪活跃命令
	semaphore  chan struct{}  // 并发控制信号量
}

// NewCommandClient 创建命令客户端
func NewCommandClient(conn *Connection, config CommandConfig, logger *log.Logger) *CommandClient {
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 60 * time.Second
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 5
	}
	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = 5 * time.Second
	}
	if config.HeartbeatPeriod <= 0 {
		config.HeartbeatPeriod = 30 * time.Second
	}

	return &CommandClient{
		config:    config,
		conn:      conn,
		logger:    logger,
		stopCh:    make(chan struct{}),
		semaphore: make(chan struct{}, config.MaxConcurrent),
	}
}

// SetExecutor 设置命令执行器
func (c *CommandClient) SetExecutor(executor CommandExecutor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.executor = executor
}

// Start 启动命令通道（双向流）
func (c *CommandClient) Start(ctx context.Context) {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = true
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	c.logger.Info("Command client starting",
		zap.String("agent_id", c.config.AgentID),
		zap.Int("max_concurrent", c.config.MaxConcurrent),
	)

	// 持续运行，断连后重连
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Command client stopped by context")
			c.shutdown()
			return
		case <-c.stopCh:
			c.logger.Info("Command client stopped by signal")
			c.shutdown()
			return
		default:
			err := c.runCommandStream(ctx)
			if err != nil {
				c.logger.Warn("Command stream disconnected, will reconnect",
					zap.Error(err),
					zap.Duration("delay", c.config.ReconnectDelay),
				)
			}

			// 等待重连延迟
			select {
			case <-ctx.Done():
				c.shutdown()
				return
			case <-c.stopCh:
				c.shutdown()
				return
			case <-time.After(c.config.ReconnectDelay):
				// 继续重连
			}
		}
	}
}

// Stop 停止命令客户端
func (c *CommandClient) Stop() {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	close(c.stopCh)
}

// shutdown 优雅关闭
func (c *CommandClient) shutdown() {
	c.logger.Info("Waiting for active commands to complete")
	c.activeCmds.Wait()

	c.mu.Lock()
	c.isRunning = false
	c.mu.Unlock()

	c.logger.Info("Command client shutdown complete")
}

// runCommandStream 运行命令流
func (c *CommandClient) runCommandStream(ctx context.Context) error {
	conn := c.conn.GetConn()
	if conn == nil {
		return fmt.Errorf("not connected")
	}

	client := pb.NewAgentServiceClient(conn)

	stream, err := client.ExecuteCommand(ctx)
	if err != nil {
		return fmt.Errorf("failed to open command stream: %w", err)
	}

	c.logger.Info("Command stream established")

	// 启动心跳 goroutine（保持流活跃）
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()

	go c.streamHeartbeat(heartbeatCtx, stream)

	// 接收并处理命令
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		default:
			cmd, err := stream.Recv()
			if err != nil {
				return fmt.Errorf("receive command failed: %w", err)
			}

			c.logger.Info("Received command",
				zap.String("command_id", cmd.CommandId),
				zap.String("command_type", cmd.CommandType),
				zap.Int32("timeout_seconds", cmd.TimeoutSeconds),
			)

			// 异步执行命令
			c.activeCmds.Add(1)
			go func(cmd *pb.Command) {
				defer c.activeCmds.Done()
				c.executeAndReport(ctx, stream, cmd)
			}(cmd)
		}
	}
}

// streamHeartbeat 发送流心跳以保持连接
func (c *CommandClient) streamHeartbeat(ctx context.Context, stream pb.AgentService_ExecuteCommandClient) {
	ticker := time.NewTicker(c.config.HeartbeatPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 发送空的心跳结果
			result := &pb.CommandResult{
				CommandId: "_heartbeat",
				Status:    pb.CommandStatus_COMMAND_STATUS_SUCCESS,
			}
			if err := stream.Send(result); err != nil {
				c.logger.Debug("Stream heartbeat failed", zap.Error(err))
				return
			}
		}
	}
}

// executeAndReport 执行命令并上报结果
func (c *CommandClient) executeAndReport(ctx context.Context, stream pb.AgentService_ExecuteCommandClient, cmd *pb.Command) {
	// 获取并发信号量
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		c.reportResult(stream, cmd.CommandId, pb.CommandStatus_COMMAND_STATUS_FAILED, "", "context cancelled")
		return
	}

	// 设置超时
	timeout := c.config.DefaultTimeout
	if cmd.TimeoutSeconds > 0 {
		timeout = time.Duration(cmd.TimeoutSeconds) * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 报告开始执行
	c.reportResult(stream, cmd.CommandId, pb.CommandStatus_COMMAND_STATUS_RUNNING, "", "")

	// 执行命令
	c.mu.RLock()
	executor := c.executor
	c.mu.RUnlock()

	if executor == nil {
		c.reportResult(stream, cmd.CommandId, pb.CommandStatus_COMMAND_STATUS_FAILED, "", "no executor configured")
		return
	}

	output, err := executor.Execute(execCtx, cmd)

	// 判断结果状态
	var finalStatus pb.CommandStatus
	var errorMsg string

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			finalStatus = pb.CommandStatus_COMMAND_STATUS_TIMEOUT
			errorMsg = "command execution timeout"
		} else {
			finalStatus = pb.CommandStatus_COMMAND_STATUS_FAILED
			errorMsg = err.Error()
		}
	} else {
		finalStatus = pb.CommandStatus_COMMAND_STATUS_SUCCESS
	}

	c.reportResult(stream, cmd.CommandId, finalStatus, output, errorMsg)

	c.logger.Info("Command execution completed",
		zap.String("command_id", cmd.CommandId),
		zap.String("status", finalStatus.String()),
	)
}

// reportResult 上报命令执行结果
func (c *CommandClient) reportResult(stream pb.AgentService_ExecuteCommandClient, cmdID string, status pb.CommandStatus, output, errorMsg string) {
	result := &pb.CommandResult{
		CommandId:    cmdID,
		Status:       status,
		Output:       output,
		ErrorMessage: errorMsg,
		CompletedAt:  timestamppb.Now(),
	}

	if err := stream.Send(result); err != nil {
		c.logger.Error("Failed to send command result",
			zap.String("command_id", cmdID),
			zap.Error(err),
		)
	}
}

// IsRunning 返回是否正在运行
func (c *CommandClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// ExecuteOnce 执行单次命令（不依赖流，用于测试或特殊场景）
func (c *CommandClient) ExecuteOnce(ctx context.Context, cmd *pb.Command) (*pb.CommandResult, error) {
	c.mu.RLock()
	executor := c.executor
	c.mu.RUnlock()

	if executor == nil {
		return nil, fmt.Errorf("no executor configured")
	}

	// 设置超时
	timeout := c.config.DefaultTimeout
	if cmd.TimeoutSeconds > 0 {
		timeout = time.Duration(cmd.TimeoutSeconds) * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := executor.Execute(execCtx, cmd)

	result := &pb.CommandResult{
		CommandId:   cmd.CommandId,
		CompletedAt: timestamppb.Now(),
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Status = pb.CommandStatus_COMMAND_STATUS_TIMEOUT
			result.ErrorMessage = "command execution timeout"
		} else {
			result.Status = pb.CommandStatus_COMMAND_STATUS_FAILED
			result.ErrorMessage = err.Error()
		}
	} else {
		result.Status = pb.CommandStatus_COMMAND_STATUS_SUCCESS
		result.Output = output
	}

	return result, nil
}

// DefaultCommandExecutor 默认命令执行器（示例实现）
type DefaultCommandExecutor struct {
	handlers map[string]CommandHandler
	mu       sync.RWMutex
}

// CommandHandler 单个命令处理函数
type CommandHandler func(ctx context.Context, params map[string]string) (string, error)

// NewDefaultCommandExecutor 创建默认命令执行器
func NewDefaultCommandExecutor() *DefaultCommandExecutor {
	return &DefaultCommandExecutor{
		handlers: make(map[string]CommandHandler),
	}
}

// RegisterHandler 注册命令处理器
func (e *DefaultCommandExecutor) RegisterHandler(cmdType string, handler CommandHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[cmdType] = handler
}

// Execute 执行命令
func (e *DefaultCommandExecutor) Execute(ctx context.Context, cmd *pb.Command) (string, error) {
	e.mu.RLock()
	handler, ok := e.handlers[cmd.CommandType]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unsupported command type: %s", cmd.CommandType)
	}

	return handler(ctx, cmd.Parameters)
}

// SupportedCommands 返回支持的命令类型
func (e *DefaultCommandExecutor) SupportedCommands() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	commands := make([]string, 0, len(e.handlers))
	for cmd := range e.handlers {
		commands = append(commands, cmd)
	}
	return commands
}

// isCommandRetryable 判断命令错误是否可重试
func isCommandRetryable(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false // 非 gRPC 错误不重试
	}

	switch st.Code() {
	case codes.Unavailable, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
