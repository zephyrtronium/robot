package metrics

import "github.com/prometheus/client_golang/prometheus"

type Observer interface {
	Observe(val float64, labels ...string)

	// for now we will tightly couple to the prometheus collector type
	// the go otel metrics sdk also has a prometheus adapter that implements this interface.
	prometheus.Collector
}

type Metrics struct {
	TMIMsgsCount              Observer
	TMICommandCount           Observer
	LearnedCount              Observer
	ForgotCount               Observer
	SpeakLatency              Observer
	LearnLatency              Observer
	UsedMessagesForGeneration Observer
}

func (m Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.ForgotCount,
		m.LearnedCount,
		m.SpeakLatency,
		m.TMICommandCount,
		m.TMIMsgsCount,
		m.LearnLatency,
	}
}
