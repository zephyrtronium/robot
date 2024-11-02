package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func NewPromCounter(m prometheus.Counter) Observer {
	return &PrometheusMetric{
		observe: func(val float64, labels ...string) {
			m.Add(val)
		},
		Collector: m,
	}
}

func NewPromGauge(m prometheus.Counter) Observer {
	return &PrometheusMetric{
		observe: func(val float64, labels ...string) {
			m.Add(val)
		},
		Collector: m,
	}
}

// for histogram or summary vecs
func NewPromObserverVec(m prometheus.ObserverVec) Observer {
	return &PrometheusMetric{
		observe: func(val float64, labels ...string) {
			m.WithLabelValues(labels...).Observe(val)
		},
		Collector: m,
	}
}

func NewPromHistogram(m prometheus.Histogram) Observer {
	return &PrometheusMetric{
		observe: func(val float64, labels ...string) {
			m.Observe(val)
		},
		Collector: m,
	}
}

type PrometheusMetric struct {
	observe func(val float64, labels ...string)
	prometheus.Collector
}

func (m *PrometheusMetric) Observe(val float64, labels ...string) {
	m.observe(val, labels...)
}
