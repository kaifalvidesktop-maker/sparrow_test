package main

import (
	"fmt"
	"net"
	"time"
)

type PingResult struct {
	Reachable  bool    `json:"reachable"`
	LatencyMs  float64 `json:"latencyMs"`
	Error      string  `json:"error,omitempty"`
}

// PingDevice measures TCP connect latency to a device's transfer port —
// a lightweight reachability + round-trip check for the "Troubleshoot"
// section shown in the Devices/Home UI when a send fails.
func PingDevice(ip string, port int) PingResult {
	addr := fmt.Sprintf("%s:%d", ip, port)
	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	elapsed := time.Since(start).Seconds() * 1000

	if err != nil {
		return PingResult{Reachable: false, Error: err.Error()}
	}
	defer conn.Close()

	return PingResult{Reachable: true, LatencyMs: elapsed}
}

// CheckLANHealth reports whether discovery and the transfer server both
// look healthy, and how many devices are currently visible — used for a
// simple "is my network OK" status readout.
type LANHealth struct {
	DiscoveryRunning bool `json:"discoveryRunning"`
	TransferRunning  bool `json:"transferRunning"`
	DeviceCount      int  `json:"deviceCount"`
}

func CheckLANHealth() LANHealth {
	return LANHealth{
		DiscoveryRunning: state.discovery != nil,
		TransferRunning:  state.transferReceiver != nil,
		DeviceCount:      globalDeviceRegistry.Count(),
	}
}