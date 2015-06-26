package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"github.com/frightenedmonkey/statsproxy/common"
	"github.com/frightenedmonkey/statsproxy/config"
	"github.com/frightenedmonkey/statsproxy/workers"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

var signalchan chan os.Signal

func monitor() {
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

signalLoop:
	for {
		select {
		case sig := <-signalchan:
			common.Logger.Info(fmt.Sprintf("!! Caught signal %d... shutting down\n", sig))
			break signalLoop
		case <-common.StatsTicker.C:
			workers.SendStats()
		}
	}
}

func initializeUDPListener() *net.UDPConn {
	address, er := net.ResolveUDPAddr("udp", config.Service.Port)
	if er != nil {
		log.Fatalf("ERROR: could not resolve local UDP address - %s", er)
	}

	common.Logger.Info(fmt.Sprintf("Listening on %s", address))
	listener, err := net.ListenUDP("udp4", address)

	if err != nil {
		log.Fatalf("ERROR: ListenUDP - %s", err)
	}

	listener.SetReadBuffer(config.Service.SocketReadBuffer)

	return listener

}

func initializeTCPListener() *net.TCPListener {
	a, err := net.ResolveTCPAddr("tcp4", ":8126")
	if err != nil {
		log.Fatal("Couldn't resolve on port 8126: %s", err)
	}

	l, er := net.ListenTCP("tcp4", a)
	if er != nil {
		log.Fatal("Couldn't listen on port %s", err)
	}
	return l
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
	common.InitializeTickers()

	signalchan = make(chan os.Signal, 1)
	signal.Notify(signalchan, syscall.SIGTERM)
	signal.Notify(signalchan, syscall.SIGINT)

	runtime.GOMAXPROCS(runtime.NumCPU())

	if len(config.Service.Statsd.Hosts) == 0 {
		log.Fatalf("You must provide at least one downstream host to proxy to.\n")
	}

	// Sets up the host connections for writing, and the channels for receiving
	// the packets that we'll coalesce in the writers.
	common.InitializeStatsdConns()

	common.Logger.Info(fmt.Sprintf("Initialized with %v downstream hosts",
		len(config.Service.Statsd.Hosts)))

	listenerUDP := initializeUDPListener()
	defer listenerUDP.Close()
	workers.StartUDPListeners(listenerUDP)
	workers.StartUDPWriters()

	listenerTCP := initializeTCPListener()
	defer listenerTCP.Close()
	workers.InitializeTCPHealthCheck(listenerTCP)

	http.HandleFunc("/healthcheck", healthCheck)
	go http.ListenAndServe(":8080", nil)

	monitor()
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("OK")); err != nil {
		common.Logger.Err(fmt.Sprintf("Error sending healthcheck: %v", err))
	}
}
