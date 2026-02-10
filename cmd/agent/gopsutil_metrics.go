package main

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func (a *Agent) collectGopsutilMetrics() {
	fmt.Printf("%s \n", "collect gopsutil metrics "+time.Now().String())

	vm, err := mem.VirtualMemory()
	if err == nil {
		a.Gauge["TotalMemory"] = float64(vm.Total)
		a.Gauge["FreeMemory"] = float64(vm.Free)
	}

	cpuPercents, err := cpu.Percent(0, true)
	if err == nil {
		for i, p := range cpuPercents {
			a.Gauge[fmt.Sprintf("CPUutilization%d", i+1)] = p
		}
	}
}
