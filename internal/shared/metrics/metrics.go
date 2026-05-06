package metrics

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	httpRequests       atomic.Int64
	httpLatencyNanos   atomic.Int64
	orderSuccess       atomic.Int64
	orderFailure       atomic.Int64
	wsActive           atomic.Int64
	panicCount         atomic.Int64
	rateLimitRejection atomic.Int64

	mu           sync.Mutex
	statusCounts map[int]int64
}

var Default = New()

func New() *Metrics {
	return &Metrics{statusCounts: map[int]int64{}}
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		Default.RecordHTTP(c.Writer.Status(), time.Since(start))
	}
}

func Handler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, Default.Render(db))
	}
}

func (m *Metrics) RecordHTTP(status int, latency time.Duration) {
	m.httpRequests.Add(1)
	m.httpLatencyNanos.Add(latency.Nanoseconds())
	m.mu.Lock()
	m.statusCounts[status]++
	m.mu.Unlock()
}

func (m *Metrics) IncOrderSuccess() { m.orderSuccess.Add(1) }
func (m *Metrics) IncOrderFailure() { m.orderFailure.Add(1) }
func (m *Metrics) IncPanic()        { m.panicCount.Add(1) }
func (m *Metrics) IncRateLimited()  { m.rateLimitRejection.Add(1) }
func (m *Metrics) IncWSActive()     { m.wsActive.Add(1) }
func (m *Metrics) DecWSActive()     { m.wsActive.Add(-1) }

func (m *Metrics) Render(db *sql.DB) string {
	var b strings.Builder
	total := m.httpRequests.Load()
	latencySeconds := float64(m.httpLatencyNanos.Load()) / float64(time.Second)

	fmt.Fprintf(&b, "http_requests_total %d\n", total)
	fmt.Fprintf(&b, "http_request_latency_seconds_sum %.6f\n", latencySeconds)
	if total > 0 {
		fmt.Fprintf(&b, "http_request_latency_seconds_avg %.6f\n", latencySeconds/float64(total))
	}

	m.mu.Lock()
	statuses := make([]int, 0, len(m.statusCounts))
	for status := range m.statusCounts {
		statuses = append(statuses, status)
	}
	sort.Ints(statuses)
	for _, status := range statuses {
		fmt.Fprintf(&b, "http_responses_total{status_code=\"%d\"} %d\n", status, m.statusCounts[status])
	}
	m.mu.Unlock()

	fmt.Fprintf(&b, "order_creation_success_total %d\n", m.orderSuccess.Load())
	fmt.Fprintf(&b, "order_creation_failure_total %d\n", m.orderFailure.Load())
	fmt.Fprintf(&b, "websocket_active_connections %d\n", m.wsActive.Load())
	fmt.Fprintf(&b, "panic_total %d\n", m.panicCount.Load())
	fmt.Fprintf(&b, "rate_limit_rejections_total %d\n", m.rateLimitRejection.Load())

	if db != nil {
		stats := db.Stats()
		fmt.Fprintf(&b, "db_open_connections %d\n", stats.OpenConnections)
		fmt.Fprintf(&b, "db_in_use_connections %d\n", stats.InUse)
		fmt.Fprintf(&b, "db_idle_connections %d\n", stats.Idle)
		fmt.Fprintf(&b, "db_wait_count %d\n", stats.WaitCount)
	}
	return b.String()
}
