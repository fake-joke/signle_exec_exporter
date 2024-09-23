// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"go_collector/collector"
	"go_collector/handle"
	diskHandle "go_collector/handle/disk"
	"go_collector/utils"
	_ "net/http/pprof"
	"os"
	"os/user"
	"runtime"
	"sort"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
)

// handler wraps an unfiltered http.Handler but uses a filtered handler,
// created on the fly, if filtering is requested. Create instances with
// newHandler.

var filters = []string{
	"meminfo",
	"cpu",
	"diskstats",
	"filefd",
	"netclass",
	"netdev",
	"loadavg",
	"hwmon",
}

func main() {
	utils.BuildLogger("debug")
	var (
		disableDefaultCollectors = kingpin.Flag(
			"collector.disable-defaults",
			"Set all collectors to disabled by default.",
		).Default("false").Bool()
		maxProcs = kingpin.Flag(
			"runtime.gomaxprocs", "The target number of CPUs Go will run on (GOMAXPROCS)",
		).Envar("GOMAXPROCS").Default("1").Int()
	)

	r := prometheus.NewRegistry()
	// r.MustRegister(promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}), promcollectors.NewGoCollector())

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("node_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	if *disableDefaultCollectors {
		collector.DisableDefaultCollectors()
	}
	level.Info(logger).Log("msg", "Starting node_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		level.Warn(logger).Log("msg", "Node Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
	}
	runtime.GOMAXPROCS(*maxProcs)
	level.Debug(logger).Log("msg", "Go MAXPROCS", "procs", runtime.GOMAXPROCS(0))

	nc, err := collector.NewNodeCollector(logger, filters...)
	if err != nil {
		level.Error(logger).Log("couldn't create collector: %s", err)
	}

	level.Info(logger).Log("msg", "Enabled collectors")
	collectors := []string{}
	for n := range nc.Collectors {
		collectors = append(collectors, n)
	}
	sort.Strings(collectors)
	for _, c := range collectors {
		level.Info(logger).Log("collector", c)
	}
	if err := r.Register(nc); err != nil {
		level.Error(logger).Log("couldn't register node collector: %s", err)
	}

	if mfs, err := r.Gather(); err != nil {
		level.Error(logger).Log("err", err)
	} else {
		// 将指标转换为 JSON 格式
		var result []map[string]interface{}

		for _, mf := range mfs {
			for _, m := range mf.Metric {
				metric := make(map[string]interface{})
				metric["name"] = *mf.Name
				metric["help"] = *mf.Help
				metric["type"] = mf.Type.String()

				labels := make(map[string]string)
				for _, lp := range m.Label {
					labels[*lp.Name] = *lp.Value
				}
				metric["labels"] = labels

				switch {
				case m.Gauge != nil:
					metric["value"] = *m.Gauge.Value
				case m.Counter != nil:
					metric["value"] = *m.Counter.Value
				case m.Summary != nil:
					metric["count"] = *m.Summary.SampleCount
					metric["sum"] = *m.Summary.SampleSum
					quantiles := make(map[string]float64)
					for _, q := range m.Summary.Quantile {
						quantiles[fmt.Sprintf("%g", *q.Quantile)] = *q.Value
					}
					metric["quantiles"] = quantiles
				case m.Histogram != nil:
					metric["count"] = *m.Histogram.SampleCount
					metric["sum"] = *m.Histogram.SampleSum
					buckets := make(map[string]uint64)
					for _, b := range m.Histogram.Bucket {
						buckets[fmt.Sprintf("%g", *b.UpperBound)] = *b.CumulativeCount
					}
					metric["buckets"] = buckets
				}

				result = append(result, metric)
			}
		}

		handle.HandleCPU(r)
		handle.HandleMemory(r)
		diskHandle.GetInfo()

		// jsonStr, err := json.MarshalIndent(result, "", " ")
		// if err != nil {
		// 	fmt.Println("Error converting to JSON:", err)
		// 	os.Exit(1)
		// }
		// fmt.Println(string(jsonStr))

		// 打开文件（如果不存在则创建，如果存在则截断）
		file, err := os.OpenFile("system_metrics.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println("Error opening file:", err)
			os.Exit(1)
		}
		defer file.Close()

		// 将数据写入 JSON 文件
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}

		fmt.Println("Memory metrics have been written to system_metrics.json")
	}
}
