package common

import (
	"fmt"
	"github.com/fullcontact/statsproxy/config"
	"log"
	"net"
)

var (
	HostConns            = make(map[config.Host]*net.UDPConn, len(config.Service.Statsd.Hosts))
	HostChans            = make(map[config.Host]chan string, len(config.Service.Statsd.Hosts))
	PacketCountPerServer = make(map[config.Host]*uint32)
	PacketCount          uint32
)

func InitializeStatsdConns() {
	for _, host := range config.Service.Statsd.Hosts {
		addr, err := net.ResolveUDPAddr("udp", host.String())

		if err != nil {
			log.Fatalf("Failed to resolve host %v - %s", host, err)
		}

		Logger.Info(fmt.Sprintf("The host address is %v", addr.String()))

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Fatalf("Failed to connect to host %v - %s", addr, err)
		}

		HostConns[host] = conn
		HostChans[host] = make(chan string)
		zero := uint32(0)
		PacketCountPerServer[host] = &zero
	}
}
