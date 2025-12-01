package interceptors

import (
"context"
"time"

"github.com/prometheus/client_golang/prometheus"
"google.golang.org/grpc"
"google.golang.org/grpc/status"
)

// GRPCMetrics 定义 gRPC 指标
type GRPCMetrics struct {
	serverStartedTotal  *prometheus.CounterVec
	serverHandledTotal  *prometheus.CounterVec
	serverHandlingSeconds *prometheus.HistogramVec
	eventsReceivedTotal *prometheus.CounterVec
	agentsOnline        *prometheus.GaugeVec
}

// NewGRPCMetrics 创建 gRPC 指标实例
func NewGRPCMetrics() *GRPCMetrics {
	m := &GRPCMetrics{
		serverStartedTotal: prometheus.NewCounterVec(
prometheus.CounterOpts{
Name: "grpc_server_started_total",
Help: "Total number of RPCs started on the server.",
},
[]string{"method"},
),
		serverHandledTotal: prometheus.NewCounterVec(
prometheus.CounterOpts{
Name: "grpc_server_handled_total",
Help: "Total number of RPCs completed on the server.",
},
[]string{"method", "code"},
),
		serverHandlingSeconds: prometheus.NewHistogramVec(
prometheus.HistogramOpts{
Name:    "grpc_server_handling_seconds",
Help:    "Histogram of response latency of gRPC server.",
Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
},
[]string{"method"},
),
		eventsReceivedTotal: prometheus.NewCounterVec(
prometheus.CounterOpts{
Name: "edr_events_received_total",
Help: "Total number of events received.",
},
[]string{"event_type", "tenant_id"},
),
		agentsOnline: prometheus.NewGaugeVec(
prometheus.GaugeOpts{
Name: "edr_agents_online",
Help: "Number of online agents.",
},
[]string{"tenant_id"},
),
	}
	return m
}

// Register 注册指标
func (m *GRPCMetrics) Register(reg prometheus.Registerer) {
	reg.MustRegister(m.serverStartedTotal)
	reg.MustRegister(m.serverHandledTotal)
	reg.MustRegister(m.serverHandlingSeconds)
	reg.MustRegister(m.eventsReceivedTotal)
	reg.MustRegister(m.agentsOnline)
}

// UnaryInterceptor 创建一元调用指标拦截器
func (m *GRPCMetrics) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
info *grpc.UnaryServerInfo,
handler grpc.UnaryHandler) (interface{}, error) {

		m.serverStartedTotal.WithLabelValues(info.FullMethod).Inc()
		start := time.Now()

		resp, err := handler(ctx, req)

		code := status.Code(err).String()
		m.serverHandledTotal.WithLabelValues(info.FullMethod, code).Inc()
		m.serverHandlingSeconds.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())

		return resp, err
	}
}

// StreamInterceptor 创建流式调用指标拦截器
func (m *GRPCMetrics) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream,
info *grpc.StreamServerInfo,
handler grpc.StreamHandler) error {

		m.serverStartedTotal.WithLabelValues(info.FullMethod).Inc()
		start := time.Now()

		err := handler(srv, ss)

		code := status.Code(err).String()
		m.serverHandledTotal.WithLabelValues(info.FullMethod, code).Inc()
		m.serverHandlingSeconds.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())

		return err
	}
}

// IncEventsReceived 增加事件接收计数
func (m *GRPCMetrics) IncEventsReceived(eventType, tenantID string, count int) {
	m.eventsReceivedTotal.WithLabelValues(eventType, tenantID).Add(float64(count))
}

// SetAgentsOnline 设置在线 Agent 数量
func (m *GRPCMetrics) SetAgentsOnline(tenantID string, count int64) {
	m.agentsOnline.WithLabelValues(tenantID).Set(float64(count))
}
