package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net"
	"net/http"
	"server/diagnostics"
	"server/flow"
	"server/searchstatus"
	config "server/settings"
	"server/torr"
	"server/torr/storage/torrstor"
	"sync"
	"time"
)

var metricsOnce sync.Once
var metricsHandler http.Handler

type flowCollector struct{}

func (flowCollector) Describe(ch chan<- *prometheus.Desc) {
	// The complete fixed descriptor set is emitted by Collect, with no dynamic
	// resource or identity labels. Unchecked registration is local to this registry.
}
func (flowCollector) Collect(ch chan<- prometheus.Metric) {
	gauge := func(name, help string, value float64) {
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("flow_"+name, help, nil, nil), prometheus.GaugeValue, value)
	}
	counter := func(name, help string, value uint64) {
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("flow_"+name, help, nil, nil), prometheus.CounterValue, float64(value))
	}
	flag := func(v bool) float64 {
		if v {
			return 1
		}
		return 0
	}
	enabled := config.CurrentFlow().MetricsEnabled
	gauge("observations_enabled", "Whether ordinary Flow streaming observations are enabled.", flag(enabled))
	gauge("active_streams", "Current in-flight server media requests; not decoded player state.", float64(torr.GetActiveStreams()))
	mem := diagnostics.CachedMemory()
	resources := torrstor.Resources()
	if !mem.SampledAt.IsZero() {
		gauge("memory_sample_timestamp_seconds", "Timestamp of the last ordinary memory sample.", float64(mem.SampledAt.UnixMilli())/1000)
		if mem.RSSAvailable {
			gauge("process_resident_memory_bytes", "Cached process RSS including native allocations.", float64(mem.RSSBytes))
		}
		gauge("go_heap_bytes", "Cached allocated Go heap bytes.", float64(mem.GoHeapBytes))
		gauge("go_goroutines", "Cached Go goroutine count.", float64(mem.Goroutines))
		gauge("memory_pressure", "Whether the ordinary allocator reports memory pressure.", flag(mem.Pressure))
	}
	if !resources.SampledAt.IsZero() {
		gauge("cache_budget_bytes", "Current aggregate RAM cache budget.", float64(resources.BudgetBytes))
		gauge("cache_warm_budget_bytes", "Current warm/background cache budget.", float64(resources.WarmBudgetBytes))
		gauge("cache_resident_bytes", "Cached aggregate RAM cache occupancy.", float64(resources.RAMResidentBytes))
		gauge("cache_disk_bytes", "Cached aggregate evicting disk cache occupancy.", float64(resources.DiskCacheBytes))
		gauge("cache_protected_overcommit", "Whether protected working sets exceed the cache budget.", flag(resources.ProtectedOvercommit))
		gauge("background_limited", "Whether the allocator is limiting background work.", flag(resources.BackgroundLimited))
		gauge("cache_budget_changes", "Current allocator budget-change count (sampled state).", float64(resources.BudgetChanges))
	}
	// These remain process-monotonic while observation collection is disabled;
	// the flag above explains why no new streaming observations are arriving.
	totals := flow.Totals()
	counter("cache_hit_bytes_total", "Observed server cache hit bytes since process start.", totals.HitBytes)
	counter("cache_miss_bytes_total", "Observed server cache miss bytes since process start.", totals.MissBytes)
	counter("read_waits_total", "Observed server data-read waits since process start.", totals.Waits)
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("flow_read_wait_seconds_total", "Total observed server data-read wait time.", nil, nil), prometheus.CounterValue, float64(totals.WaitNS)/float64(time.Second))
	counter("server_read_stalls_total", "Observed server read waits of at least 0.5 seconds; not player frame stalls.", totals.Stalls)
	counter("seek_recoveries_total", "Observed successful server seek delivery recoveries.", totals.Recoveries)
	for i, outcome := range []string{"ready", "failed", "timeout", "cancelled"} {
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("flow_startup_outcomes_total", "Observed startup terminal outcomes.", []string{"outcome"}, nil), prometheus.CounterValue, float64(totals.Startup[i]), outcome)
	}
	for _, row := range searchstatus.OutcomeMetrics() {
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("flow_search_outcomes_total", "Configured endpoint search/test outcomes since process start.", []string{"source", "outcome"}, nil), prometheus.CounterValue, float64(row.Count), row.Source, row.State)
	}
}
func metrics(c *gin.Context) {
	if !config.CurrentFlow().PrometheusEnabled {
		c.Status(http.StatusNotFound)
		return
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	ip := net.ParseIP(host)
	if err != nil || ip == nil {
		abortWithErrorCode(c, 403, "METRICS_PEER_REQUIRED", errors.New("connecting peer is unavailable"))
		return
	}
	_, authenticated := c.Get(gin.AuthUserKey)
	if !ip.IsLoopback() && (!config.HttpAuth || !authenticated) {
		abortWithErrorCode(c, 403, "METRICS_AUTH_REQUIRED", errors.New("remote monitoring requires enabled HTTP authentication and valid credentials"))
		return
	}
	metricsOnce.Do(func() {
		registry := prometheus.NewRegistry()
		registry.MustRegister(flowCollector{})
		metricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{MaxRequestsInFlight: 2, Timeout: 5 * time.Second})
	})
	c.Header("Cache-Control", "private, no-store")
	metricsHandler.ServeHTTP(c.Writer, c.Request)
}
