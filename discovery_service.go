package main

import (
	"encoding/json"
	"log"
	"net"
	"time"
)

// ============================================================
// discovery_service.go
// UDP broadcaster + listener goroutines driving DeviceRegistry.
// ============================================================

type announcePacket struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Avatar  string `json:"avatar"`
	Version string `json:"version"`
}

type DiscoveryService struct {
	registry   *DeviceRegistry
	conn       *net.UDPConn
	selfID     string
	stopCh     chan struct{}
	onDeviceCB func(DeviceInfo)
}

func NewDiscoveryService(registry *DeviceRegistry, selfID string) *DiscoveryService {
	return &DiscoveryService{registry: registry, selfID: selfID, stopCh: make(chan struct{})}
}

func (d *DiscoveryService) Start() error {
	addr := &net.UDPAddr{Port: DiscoveryPort, IP: net.IPv4zero}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}
	d.conn = conn
	go d.listenLoop()
	go d.broadcastLoop()
	go d.pruneLoop()
	log.Println("[discovery] started on UDP", DiscoveryPort)
	return nil
}

func (d *DiscoveryService) Stop() {
	close(d.stopCh)
	d.sendGoodbye()
	if d.conn != nil {
		_ = d.conn.Close()
	}
}

func (d *DiscoveryService) listenLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		n, remoteAddr, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		var pkt announcePacket
		if jsonErr := json.Unmarshal(buf[:n], &pkt); jsonErr != nil {
			continue
		}
		if pkt.ID == d.selfID {
			continue
		}

		switch pkt.Type {
		case "sparrow-announce":
			info := DeviceInfo{
				ID: pkt.ID, Name: pkt.Name, IP: remoteAddr.IP.String(),
				Port: pkt.Port, Avatar: pkt.Avatar, Version: pkt.Version,
			}
			d.registry.Upsert(info)
			if d.onDeviceCB != nil {
				d.onDeviceCB(info)
			}
		case "sparrow-goodbye":
			d.registry.Remove(pkt.ID)
		}
	}
}

func (d *DiscoveryService) broadcastLoop() {
	ticker := time.NewTicker(discoveryBroadcastInterval)
	defer ticker.Stop()
	d.sendAnnounce()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.sendAnnounce()
		}
	}
}

func (d *DiscoveryService) pruneLoop() {
	ticker := time.NewTicker(discoveryExpireAfter / 2)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.registry.PruneExpired()
		}
	}
}

func (d *DiscoveryService) sendAnnounce() {
	pkt := announcePacket{
		Type: "sparrow-announce", ID: d.selfID, Name: state.deviceName,
		Port: TransferPort, Avatar: state.deviceAvatar, Version: state.appVersion,
	}
	d.broadcast(pkt)
}

func (d *DiscoveryService) sendGoodbye() {
	pkt := announcePacket{Type: "sparrow-goodbye", ID: d.selfID}
	d.broadcast(pkt)
}

func (d *DiscoveryService) broadcast(pkt announcePacket) {
	payload, err := json.Marshal(pkt)
	if err != nil {
		return
	}
	for _, bcast := range listBroadcastAddresses() {
		dst := &net.UDPAddr{IP: bcast, Port: DiscoveryPort}
		conn, err := net.DialUDP("udp4", nil, dst)
		if err != nil {
			continue
		}
		_, _ = conn.Write(payload)
		_ = conn.Close()
	}
}

func listBroadcastAddresses() []net.IP {
	var result []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		result = append(result, net.IPv4bcast)
		return result
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			bcast := make(net.IP, 4)
			mask := ipNet.Mask
			for i := 0; i < 4; i++ {
				bcast[i] = ip4[i] | ^mask[i]
			}
			result = append(result, bcast)
		}
	}
	if len(result) == 0 {
		result = append(result, net.IPv4bcast)
	}
	return result
}