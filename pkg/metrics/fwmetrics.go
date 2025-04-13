package fwmetrics

import (
	"runtime/metrics"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const totalCpuSecondsMetricName = "/cpu/classes/scavenge/total:cpu-seconds"

var log = logging.GetLog("fwmetrics")

// https://pkg.go.dev/runtime/metrics
// https://pkg.go.dev/runtime/metrics#example-Read-ReadingOneMetric
// "/cpu/classes/scavenge/total:cpu-seconds"
//   Estimated total CPU time spent performing tasks that return
//   unused memory to the underlying platform. This metric is an
//   overestimate, and not directly comparable to system CPU time
//   measurements. Compare only with other /cpu/classes metrics.
//   Sum of all metrics in /cpu/classes/scavenge.
var metricsTicker *time.Ticker

var totalCpuSeconds *prometheus.CounterVec

func Setup(prefix string) {
	totalCpuSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: prefix + "_total_cpu_seconds",
		Help: "The total number of seconds the CPU has been used, an overestimate",
	},[]string{})
	
    go func() {
		last := setTotalCpuSeconds(0.0)
        metricsTicker = time.NewTicker(10 * time.Second) // frequency matches scrape rate
        for range metricsTicker.C {
			last = setTotalCpuSeconds(last)
        }
    }()
}

func AddAll(router *gin.Engine) {
    // standard prometheus metrics - different endpoint, since we want to scrape these more frequently.
    // note that we also configure metrics in middelware.go, where we set up tracking http calls, 
    // as well as in database.go where we set up gorm metrics
    promHandler := promhttp.Handler()
    router.Handle("GET", "/metrics-prom", func(c *gin.Context) {
        promHandler.ServeHTTP(c.Writer, c.Request)
    })
}

func setTotalCpuSeconds(last float64) float64 {
	sample := make([]metrics.Sample, 1)
	sample[0].Name = totalCpuSecondsMetricName

	metrics.Read(sample)

	// Check if the metric is actually supported.
	// If it's not, the resulting value will always have
	// kind KindBad.
	if sample[0].Value.Kind() == metrics.KindBad {
		log.Fatal().Msgf("metric %q no longer supported", totalCpuSecondsMetricName)
	}

	// Handle the result.
	//
	// It's OK to assume a particular Kind for a metric;
	// they're guaranteed not to change.
	tcs := sample[0].Value.Float64()
	diff := tcs - last

	log.Debug().Msgf("totalCpuSeconds: %f, diff: %f", tcs, diff)

	totalCpuSeconds.WithLabelValues().Add(diff)

	return tcs
}
