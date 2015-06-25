package main

import (
	"bytes"
	_ "expvar"
	"flag"
	"fmt"
	"github.com/frightenedmonkey/statsproxy/common"
	"github.com/frightenedmonkey/statsproxy/config"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	// State
	hostConnections      = make(map[config.Host]io.Writer)
	packetCount          uint32
	packetCountPerServer = make(map[config.Host]*uint32)
)

var signalchan chan os.Signal

type Packet struct {
	Bucket string
	Raw    string
}

func monitor() {
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ticker := time.NewTicker(time.Duration(config.Service.TickerPeriod) * time.Second)
signalLoop:
	for {
		select {
		case sig := <-signalchan:
			common.Logger.Info(fmt.Sprintf("!! Caught signal %d... shutting down\n", sig))
			break signalLoop
		case <-ticker.C:
			sendStats()
		}
	}
}

func udpListener(listener *net.UDPConn) {
	message := make([]byte, config.Service.MaxUDPPacketSize)
	for {
		n, remaddr, err := listener.ReadFromUDP(message)
		if err != nil {
			common.Logger.Err(fmt.Sprintf("ERROR: reading UDP packet from %+v - %s",
				remaddr, err))
			continue
		}

		dataHandler(message[:n])
	}
}

func tcpHealthCheckListener(listener *net.TCPListener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		m := make([]byte, 6)
		conn.Read(m)
		conn.Write([]byte("health: up"))
	}
}

func createTCPListener() *net.TCPListener {
	a, err := net.ResolveTCPAddr("tcp4", ":8126")
	if err != nil {
		log.Fatal(err)
	}

	l, er := net.ListenTCP("tcp4", a)
	if er != nil {
		log.Fatal(err)
	}
	return l
}

func dataHandler(data []byte) {
	packets := parseMessage(data)
	for _, packet := range packets {
		packetHandler(packet)
	}
}

func packetHandler(s Packet) {
	host := config.Service.Statsd.Hosts[common.Hash(s.Bucket)%uint32(len(config.Service.Statsd.Hosts))]

	// increment packet counts and whatnot for the metrics that statsproxy sends
	// about itself.
	atomic.AddUint32(&packetCount, 1)
	atomic.AddUint32(packetCountPerServer[host], 1)

	hostConnections[host].Write([]byte(s.Raw))
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

		name := input[:index]

		packet := Packet{
			Bucket: string(name),
			Raw:    string(line) + "\n",
		}

		packets = append(packets, packet)
	}

	return packets
}

func sendStats() {
	packetCount := atomic.SwapUint32(&packetCount, 0)
	common.Logger.Info(fmt.Sprintf("Received %d packets in last 5 seconds (%f pps)\n",
		packetCount, float64(packetCount)/5))
	sendCounter("statsproxy.packets", int32(packetCount))

	for host, stat := range packetCountPerServer {
		packets := atomic.SwapUint32(stat, 0)
		common.Logger.Info(fmt.Sprintf(
			"Received %d packets in last 5 seconds (%f pps) for host %s\n",
			packets, float64(packets)/5, host))
		sendCounter("statsproxy.host."+sanitize(host)+".packets", int32(packets))
	}
}

func sanitize(h config.Host) string {
	return strings.Replace(strings.Replace(h.String(), ".", "_", -1), ":", "_", -1)
}

func sendCounter(path string, delta int32) {
	packetHandler(Packet{path, path + ":" + strconv.Itoa(int(delta)) + "|c\n"})
}

func main() {

	configFile := flag.String("config",
		"config/config.json",
		"path to config file (defaults to config/config.json)")
	flag.Parse()

	if err := config.InitializeConfig(*configFile); err != nil {
		log.Fatalf("Error loading service config - %v", err)
	}

	common.InitializeLogger()

	signalchan = make(chan os.Signal, 1)
	signal.Notify(signalchan, syscall.SIGTERM)
	signal.Notify(signalchan, syscall.SIGINT)

	runtime.GOMAXPROCS(runtime.NumCPU())

	if len(config.Service.Statsd.Hosts) == 0 {
		log.Fatalf("You must provide at least one downstream host to proxy to.\n")
	}

	common.Logger.Info(fmt.Sprintf("Initialized with %v downstream hosts",
		len(config.Service.Statsd.Hosts)))

	for _, host := range config.Service.Statsd.Hosts {
		addr, err := net.ResolveUDPAddr("udp", host.String())
		if err != nil {
			log.Fatalf("Failed to resolve host %v - %s", host, err)
		}

		common.Logger.Info(fmt.Sprintf("The host address is %v", addr.String()))

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Fatalf("Failed to connect to host %v - %s", addr, err)
		}

		hostConnections[host] = conn
		zero := uint32(0)
		packetCountPerServer[host] = &zero
	}

	address, _ := net.ResolveUDPAddr("udp", config.Service.Port)
	common.Logger.Info(fmt.Sprintf("listening on %s", address))
	listener, err := net.ListenUDP("udp4", address)

	// There were some issues setting this locally during development. It may need
	// to be configured automatically during development vs. production.
	listener.SetReadBuffer(config.Service.SocketReadBuffer)
	if err != nil {
		log.Fatalf("ERROR: ListenUDP - %s", err)
	}
	defer listener.Close()

	common.Logger.Info(fmt.Sprintf("Using %v worker routines", config.Service.Workers))
	for i := 0; i < config.Service.Workers; i++ {
		go udpListener(listener)
	}

	l := createTCPListener()
	go tcpHealthCheckListener(l)

	monitor()
}
