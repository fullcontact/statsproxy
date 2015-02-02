package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	MAX_UDP_PACKET_SIZE = 512
	SOCKET_READ_BUFFER  = 4194304
)

var (
	serviceAddress = flag.String("address", ":8125", "UDP service address")
	debug          = flag.Bool("debug", false, "print verbose data about packets sent and servers chosen")
	showVersion    = flag.Bool("version", false, "print version string")
	workers        = flag.Int("workers", 2, "number of goroutines working on incoming UDP data")
	hosts          = Hosts{}
	cpuprofile     = flag.String("cpuprofile", "", "write cpu profile to file")
	tickerPeriod   = flag.Duration("tickerPeriod", 5*time.Second, "writes internal stats every tickerPeriod seconds")

	// State
	hostConnections = make(map[*Host]io.Writer)
	packetCount     uint32
)

var signalchan chan os.Signal

func init() {
	flag.Var(&hosts, "hosts",
		"backend statsd hosts")

	signalchan = make(chan os.Signal)
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

	ticker := time.NewTicker(*tickerPeriod)
signalLoop:
	for {
		select {
		case sig := <-signalchan:
			fmt.Printf("!! Caught signal %d... shutting down\n", sig)
			break signalLoop
		case <-ticker.C:
			sendStats()
		}
	}
}

func udpListener(listener *net.UDPConn) {
	message := make([]byte, MAX_UDP_PACKET_SIZE)
	for {
		n, remaddr, err := listener.ReadFromUDP(message)
		if err != nil {
			log.Printf("ERROR: reading UDP packet from %+v - %s", remaddr, err)
			continue
		}

		dataHandler(message[:n])
	}
}

func dataHandler(data []byte) {
	packets := parseMessage(data)
	for _, packet := range packets {
		packetHandler(&packet)
	}
}

func packetHandler(s *Packet) {
	atomic.AddUint32(&packetCount, 1)

	DPrintf("Received packet - %+v\n", s)
	host := hosts[hash(s.Bucket)%uint32(len(hosts))]
	DPrintf("Hashed packet to host - %+v\n", host)

	hostConnections[host].Write([]byte(s.Raw))
	DPrintf("Wrote packet to host - %+v\n", host)
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
			DPrintf("ERROR: failed to parse line: %s\n", string(line))
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

	DPrintf("Received %d packets in last 5 seconds (%f pps)\n", packetCount, float64(packetCount)/5)

	sendCounter("statsproxy.packets", int32(packetCount))
}

func sendCounter(path string, delta int32) {
	packetHandler(&Packet{path, path + ":" + strconv.Itoa(int(delta)) + "|c\n"})
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("statsproxy (built w/%s)\n", runtime.Version())
		return
	}

	signalchan = make(chan os.Signal, 1)
	signal.Notify(signalchan, syscall.SIGTERM)
	signal.Notify(signalchan, syscall.SIGINT)

	runtime.GOMAXPROCS(runtime.NumCPU())

	if len(hosts) == 0 {
		log.Fatalf("You must provide at least one downstream host to proxy to.\n")
	}

	log.Printf("Initialized with %v downstream hosts: %v\n", len(hosts), hosts)

	for _, host := range hosts {
		addr, err := net.ResolveUDPAddr("udp", host.String())
		if err != nil {
			log.Fatalf("Failed to resolve host %v - %s", host, err)
		}

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Fatalf("Failed to connect to host %v - %s", addr, err)
		}

		hostConnections[host] = conn
	}

	address, _ := net.ResolveUDPAddr("udp", *serviceAddress)
	log.Printf("listening on %s", address)
	listener, err := net.ListenUDP("udp4", address)
	listener.SetReadBuffer(SOCKET_READ_BUFFER)
	if err != nil {
		log.Fatalf("ERROR: ListenUDP - %s", err)
	}
	defer listener.Close()

	log.Printf("Using %v worker routines", *workers)
	for i := 0; i < *workers; i++ {
		go udpListener(listener)
	}

	monitor()
}
