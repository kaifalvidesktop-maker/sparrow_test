package main

import "net"

// Prevents a user from accidentally "sending to themselves" if their own
// device somehow appears reachable (e.g. two network interfaces, or a
// stale discovery entry before goodbye packets propagate).

func IsSelfIP(ip string) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.String() == ip {
			return true
		}
	}
	return ip == "127.0.0.1" || ip == "localhost"
}