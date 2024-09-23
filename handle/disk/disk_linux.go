//go:build linux
// +build linux

package handle

import (
	"encoding/json"
	"fmt"
	"go_collector/bin"
	"go_collector/utils"
	"strings"
	"sync"
)

func GetInfo() []DiskInfo {
	output, err := bin.RunCommand("smartctl", "--json=c", "--scan")
	if err != nil {
		utils.Log().Error("Error:%+v", err.Error())
		return nil
		// panic(err)
	}

	var s Smartctl
	if err := json.Unmarshal([]byte(output), &s); err != nil {
		utils.Log().Error("Error:%+v", err.Error())
		return nil
		// panic(err)
	}

	disks := []DiskInfo{}
	var wg sync.WaitGroup
	wg.Add(5)
	var jobs = make(chan *Device, len(s.Devices))

	for _, device := range s.Devices {
		jobs <- &device
	}

	for i := 0; i < 5; i++ {
		go func() {
			for {
				d, ok := <-jobs
				if !ok {
					fmt.Println("goroutine done")
					wg.Done()
					return
				}

				args := []string{"--json=c", "-a", d.InfoName}
				if d.Type != "scsi" {
					append_args := []string{"-d", d.Type}
					args = append(args, append_args...)
				}
				diskInfo := getDiskInfo(d.InfoName, args...)
				if diskInfo.ModelName == "" {
					return
				}
				fmt.Println("InfoName:", d.InfoName)
				fmt.Println("ModelName:", diskInfo.ModelName)
				fmt.Println("SerialNumber:", diskInfo.SerialNumber)
				fmt.Println("ModelType:", diskInfo.ModelType)
				fmt.Println("SmartStatus:", diskInfo.SmartStatus.Passed)
				fmt.Println("UserCapacity:", diskInfo.UserCapacity.Bytes)
				fmt.Println("Temperature:", diskInfo.Temperature.Current)
				fmt.Println("PowerOnTime:", diskInfo.PowerOnTime.Hours)
				println("=============")
				disks = append(disks, diskInfo)
			}
		}()
	}

	close(jobs)
	wg.Wait()

	return disks

}

func getDiskInfo(path string, args ...string) DiskInfo {
	output, err := bin.RunCommand("smartctl", args...)
	if err != nil {
		fmt.Println("getDiskInfo err:", err.Error())
	}

	var diskInfo DiskInfo
	if err := json.Unmarshal([]byte(output), &diskInfo); err != nil {
		fmt.Println("Failed to unmarshal JSON:", err)
		return DiskInfo{}
	}

	if diskInfo.ScsiVendor == "LSI" {
		return DiskInfo{}
	}

	protocol := ""
	modelType := ""
	if diskInfo.SetaVersion.String != "" {
		protocol = diskInfo.SetaVersion.String
	} else {
		if strings.HasPrefix(path, "/dev/sd") {
			protocol = "SATA"
		} else {
			protocol = diskInfo.Device.Protocol
			switch protocol {
			case "NVMe":
				modelType = "NVMe"
			default:
			}
		}
	}

	if diskInfo.RotationRate != nil {
		if diskInfo.RotationRate == 0 {
			modelType = "ssd"
		} else {
			modelType = "hdd"
		}
	} else {
		modelType = "ssd"
	}

	diskInfo.ModelType = protocol + " " + modelType

	return diskInfo
}
