// Package handle use to handle the data which collected by node exporter
package handle

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

type Core struct {
	Mode  string
	Value float64
}

type CollectCPUInfoStruct map[string][]Core

var PrevCollectCPUInfo *CollectCPUInfoStruct = &CollectCPUInfoStruct{}
var LastCollectCPUInfo *CollectCPUInfoStruct = &CollectCPUInfoStruct{}

type CPUAttr struct {
	ID     string `json:"cpu"`
	Value  string `json:"value"`
	Sensor string `json:"sensor"`
}

type CPUInfoStruct struct {
	Usage       []CPUAttr `json:"usage"`
	Temperature []CPUAttr `json:"temperature"`
}

var CPUInfo CPUInfoStruct

func setCPUCollect(mfs []*io_prometheus_client.MetricFamily, CollectCPUInfo *CollectCPUInfoStruct) {
	for _, mf := range mfs {
		for _, m := range mf.Metric {
			if *mf.Name == "node_cpu_seconds_total" {
				var key string
				var mode string
				for _, lp := range m.Label {
					if *lp.Name == "cpu" {
						key = *lp.Value
					}
					if *lp.Name == "mode" {
						mode = *lp.Value
					}
				}

				if _, exists := (*CollectCPUInfo)[key]; !exists {
					(*CollectCPUInfo)[key] = []Core{}
				}

				(*CollectCPUInfo)[key] = append((*CollectCPUInfo)[key], Core{
					Mode:  mode,
					Value: *m.Counter.Value,
				})
			}
		}
	}
}

func setCPUTemperature(mfs []*io_prometheus_client.MetricFamily) {
	_tempTemperature := []CPUAttr{}
	for _, mf := range mfs {
	outloop:
		for _, m := range mf.Metric {
			if *mf.Name == "node_hwmon_sensor_label" {
				var id string
				var sensor string
				var chip string
				for _, lp := range m.Label {
					if *lp.Name == "label" {
						temp := strings.Split(*lp.Value, " ")
						if temp[0] != "Core" {
							continue outloop
						}
						id = temp[len(temp)-1]
					}
					if *lp.Name == "chip" {
						temp := strings.Split(*lp.Value, "_")
						chip = temp[len(temp)-1]
					}
					if *lp.Name == "sensor" {
						sensor = chip + "_" + *lp.Value
					}
				}
				CPUInfo.Temperature = append(CPUInfo.Temperature, CPUAttr{
					ID:     chip + "_" + id,
					Sensor: sensor,
				})
			}
			if *mf.Name == "node_hwmon_temp_celsius" {
				var sensor string
				var chip string
				for _, lp := range m.Label {
					if *lp.Name == "chip" {
						temp := strings.Split(*lp.Value, "_")
						chip = temp[len(temp)-1]
					}
					if *lp.Name == "sensor" {
						sensor = chip + "_" + *lp.Value
					}
				}
				_tempTemperature = append(_tempTemperature, CPUAttr{
					Sensor: sensor,
					Value:  strconv.FormatFloat(*m.Gauge.Value, 'f', 2, 64),
				})
			}
		}
	}
	CPUInfo.Temperature = mergeCPUTemperature(CPUInfo.Temperature, _tempTemperature)
}

func HandleCPU(r *prometheus.Registry) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		if prevCollect, err := r.Gather(); err == nil {
			setCPUCollect(prevCollect, PrevCollectCPUInfo)
		}
		wg.Done()
	}()
	go func() {
		time.Sleep(time.Second * 1)
		if lastCollect, err := r.Gather(); err == nil {
			setCPUCollect(lastCollect, LastCollectCPUInfo)
			//采集最新数据时一并处理温度数据
			setCPUTemperature(lastCollect)
		}
		wg.Done()
	}()
	wg.Wait()

	for CoreID, CoreInfo := range *LastCollectCPUInfo {
		prevCoreInfo := (*PrevCollectCPUInfo)[CoreID]
		var totalSecond float64 = 0
		var idleSecond float64 = 0
		var prevTotalSecond float64 = 0
		var prevIdleSecond float64 = 0
		for _, core := range CoreInfo {
			if core.Mode == "idle" {
				idleSecond = core.Value
			}
			totalSecond += core.Value
		}
		for _, core := range prevCoreInfo {
			if core.Mode == "idle" {
				prevIdleSecond = core.Value
			}
			prevTotalSecond += core.Value
		}
		CPUInfo.Usage = append(CPUInfo.Usage, CPUAttr{
			ID:    CoreID,
			Value: strconv.FormatFloat(1-(idleSecond-prevIdleSecond)/(totalSecond-prevTotalSecond), 'f', 2, 64),
		})
	}

	file, err := os.OpenFile("cpu_info.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	// 将数据写入 JSON 文件
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(CPUInfo); err != nil {
		fmt.Println("Error encoding JSON:", err)
		os.Exit(1)
	}

	fmt.Println("Memory metrics have been written to cpu_info.json")
}

func mergeCPUTemperature(label []CPUAttr, temperature []CPUAttr) []CPUAttr {
	// 创建一个 map 来存储合并后的结果
	mergedMap := make(map[string]CPUAttr)

	// 处理 Usage 数据
	for _, u := range label {
		mergedMap[u.Sensor] = CPUAttr{
			ID:     u.ID,
			Value:  u.Value,
			Sensor: u.Sensor,
		}
	}

	// 处理 Temperature 数据
	for _, t := range temperature {
		if existing, ok := mergedMap[t.Sensor]; ok {
			// 如果 Sensor 已存在，更新 Value
			existing.Value = t.Value
			mergedMap[t.Sensor] = existing
		}
	}

	// 将 map 转换回切片
	result := make([]CPUAttr, 0, len(mergedMap))
	for _, v := range mergedMap {
		result = append(result, v)
	}

	return result
}
