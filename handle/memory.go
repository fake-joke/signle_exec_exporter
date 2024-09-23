package handle

import (
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

type MemoryStruct struct {
	Total float64 `json:"total"`
	Free  float64 `json:"free"`
}

var Memory *MemoryStruct = &MemoryStruct{}

func setMemory(mfs []*io_prometheus_client.MetricFamily) {
	for _, mf := range mfs {
		for _, m := range mf.Metric {
			if *mf.Name == "node_memory_MemFree_bytes" {
				(*Memory).Free = *m.Gauge.Value
			}
			if *mf.Name == "node_memory_MemTotal_bytes" {
				(*Memory).Free = *m.Gauge.Value
			}
		}
	}
}

func HandleMemory(r *prometheus.Registry) {
	if Collect, err := r.Gather(); err == nil {
		setMemory(Collect)
	}
}
