package fwmetrics

import (
	"runtime/metrics"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
var totalCpuSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "tickets_total_cpu_seconds",
	Help: "The total number of seconds the CPU has been used, an overestimate",
},[]string{})

var metricsTicker *time.Ticker

func Setup() {
    go func() {
		last := setTotalCpuSeconds(0.0)
        metricsTicker = time.NewTicker(10 * time.Second) // frequency matches scrape rate
        for range metricsTicker.C {
			last = setTotalCpuSeconds(last)
        }
    }()
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
