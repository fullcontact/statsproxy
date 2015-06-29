package workers

import (
	"bytes"
	"fmt"
	"github.com/frightenedmonkey/statsproxy/common"
	"github.com/frightenedmonkey/statsproxy/config"
	"net"
	"strconv"
	"sync/atomic"
)

type Packet struct {
	Bucket string
	Raw    string
}

func StartUDPListeners(l *net.UDPConn) {
	for i := 0; i < config.Service.Workers; i++ {
		go udpListener(l)
	}
}

func StartUDPWriters() {
	common.Logger.Info("Starting UDP writers")
	for i := 0; i < config.Service.WriterMultiplier; i++ {
		for _, host := range config.Service.Statsd.Hosts {
			common.Logger.Info("Spawning writer for host " + host.FQDN)
			go hostWriter(host)
		}
	}
}

func hostWriter(host config.Host) {
	common.Logger.Info(fmt.Sprintf("Initializing writer for host "+host.FQDN+" on port %v", host.Port))
	coalescedPacket := make([]byte, 0)
	for {
		newMessage := <-common.HostChans[host]
		common.Logger.Dev("Writer received message " + newMessage)
		if len(coalescedPacket)+len([]byte(newMessage)) > config.Service.MaxCoalescedPacketSize {
			common.HostConns[host].Write(coalescedPacket)
			coalescedPacket = []byte(newMessage)
		} else {
			coalescedPacket = append(coalescedPacket, []byte(newMessage)...)
		}
	}
}

func udpListener(l *net.UDPConn) {
	message := make([]byte, config.Service.MaxUDPPacketSize)
	for {
		n, remaddr, err := l.ReadFromUDP(message)
		if err != nil {
			common.Logger.Err(fmt.Sprintf("ERROR: reading UDP packet from %+v - %s",
				remaddr, err))
			continue
		}

		dataHandler(message[:n])
	}
}

func dataHandler(data []byte) {
	packets := parseMessage(data)
	for _, packet := range packets {
		packetHandler(packet)
	}
}

func parseMessage(data []byte) []Packet {
	var input []byte
	var packets = make([]Packet, 0)

	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}

		input = line

		index := bytes.IndexByte(input, ':')
		if index < 0 || index == len(input)-1 {
			common.Logger.Err(fmt.Sprintf("ERROR: failed to parse line: %s\n",
				string(line)))
			continue
		}
		common.Logger.Dev("Received metric " + string(line))

		name := input[:index]

		packet := Packet{
			Bucket: string(name),
			Raw:    string(line) + "\n",
		}

		packets = append(packets, packet)
	}

	return packets
}

func packetHandler(s Packet) {
	host := config.Service.Statsd.Hosts[common.Hash(s.Bucket)%uint32(len(config.Service.Statsd.Hosts))]

	// increment packet counts and whatnot for the metrics that statsproxy sends
	// about itself.
	atomic.AddUint32(&common.PacketCount, 1)
	atomic.AddUint32(common.PacketCountPerServer[host], 1)

	common.HostChans[host] <- s.Raw
}

func SendStats() {
	packetCount := atomic.SwapUint32(&common.PacketCount, 0)
	common.Logger.Info(fmt.Sprintf("Received %d packets in last 5 seconds (%f pps)\n",
		packetCount, float64(packetCount)/5))
	sendCounter("statsproxy.packets", int32(packetCount))

	for host, stat := range common.PacketCountPerServer {
		packets := atomic.SwapUint32(stat, 0)
		common.Logger.Info(fmt.Sprintf(
			"Received %d packets in last 5 seconds (%f pps) for host %s\n",
			packets, float64(packets)/5, host))
		sendCounter("statsproxy.host."+config.Service.MetricsNamespace+".packets", int32(packets))
	}
}

func sendCounter(path string, delta int32) {
	packetHandler(Packet{path, path + ":" + strconv.Itoa(int(delta)) + "|c\n"})
}
