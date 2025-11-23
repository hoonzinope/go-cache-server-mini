package router

import (
	"fmt"
	"net"
)

type ClusterManager struct {
	local_ip           string
	local_ip_err       error
	swarm_service_name string
}

func NewClusterManager(service_name string) *ClusterManager {
	return &ClusterManager{
		swarm_service_name: service_name,
	}
}

// return local node ip
func (cm *ClusterManager) GetLocalNodeIP() (string, error) {
	if cm.local_ip == "" || cm.local_ip_err != nil {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			cm.local_ip_err = err
			return "", err
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					cm.local_ip = ipnet.IP.String()
					cm.local_ip_err = nil
					return cm.local_ip, nil
				}
			}
		}
		cm.local_ip = ""
		cm.local_ip_err = fmt.Errorf("no non-loopback IPv4 address found")
	}
	return cm.local_ip, cm.local_ip_err
}

func (cm *ClusterManager) GetRemoteNodeIPs() ([]string, error) {
	ips, err := net.LookupHost(cm.swarm_service_name)
	if err != nil {
		return nil, err
	}
	var remote_ips []string
	local_ip, err := cm.GetLocalNodeIP()
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if ip != local_ip {
			remote_ips = append(remote_ips, ip)
		}
	}
	return remote_ips, nil
}
