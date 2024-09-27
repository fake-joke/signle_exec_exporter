package handle

import (
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

type InterfaceStruct struct {
	Receive  float64 `json:"receive"`
	Transmit float64 `json:"transmit"`
}

var Network map[string]*InterfaceStruct = map[string]*InterfaceStruct{}

func setNetwork(mfs []*io_prometheus_client.MetricFamily) {
	for _, mf := range mfs {
		for _, m := range mf.Metric {
			if *mf.Name == "node_network_receive_bytes_total" {
				for _, lp := range m.Label {
					if *lp.Name == "device" {
						if _, ok := Network[*lp.Value]; !ok {
							Network[*lp.Value] = &InterfaceStruct{}
						}
						Network[*lp.Value].Receive = *m.Counter.Value
					}
				}
			}
			if *mf.Name == "node_network_transmit_bytes_total" {
				for _, lp := range m.Label {
					if *lp.Name == "device" {
						if _, ok := Network[*lp.Value]; !ok {
							Network[*lp.Value] = &InterfaceStruct{}
						}
						Network[*lp.Value].Transmit = *m.Counter.Value
					}
				}
			}
		}
	}
}

func HandleNetwork(r *prometheus.Registry) {
	if Collect, err := r.Gather(); err == nil {
		setNetwork(Collect)
	}
}
