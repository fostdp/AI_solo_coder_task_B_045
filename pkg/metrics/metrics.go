package metrics

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "battlefield_http_requests_total",
			Help: "HTTP请求总数",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "battlefield_http_request_duration_seconds",
			Help:    "HTTP请求耗时分布",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	battlefieldCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "battlefield_data_battlefields_total",
			Help: "战场遗址总数",
		},
	)

	regionCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "battlefield_data_regions_total",
			Help: "军事分区总数",
		},
	)

	analysisRuns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "battlefield_analysis_runs_total",
			Help: "分析任务执行次数",
		},
		[]string{"type"},
	)

	activeConns = atomic.Int64{}
)

func init() {
	registry.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		battlefieldCount,
		regionCount,
		analysisRuns,
	)
}

func StartPprof(port string) {
	go func() {
		log.Printf("pprof调试端口启动于 :%s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("pprof启动失败: %v", err)
		}
	}()
}

func StartMetricsServer(port string) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(registry, prometheus.DefaultGatherOpts))
		log.Printf("Prometheus指标端口启动于 :%s", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Printf("指标端口启动失败: %v", err)
		}
	}()
}

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		activeConns.Add(1)
		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()))
		c.Next()
		timer.ObserveDuration()
		activeConns.Add(-1)

		status := c.Writer.Status()
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), http.StatusText(status)).Inc()
	}
}

func SetBattlefieldCount(n int)   { battlefieldCount.Set(float64(n)) }
func SetRegionCount(n int)        { regionCount.Set(float64(n)) }
func IncAnalysisRuns(typ string)  { analysisRuns.WithLabelValues(typ).Inc() }
