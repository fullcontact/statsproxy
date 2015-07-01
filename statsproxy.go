package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"github.com/fullcontact/statsproxy/common"
	"github.com/fullcontact/statsproxy/config"
	"github.com/fullcontact/statsproxy/workers"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var signalchan chan os.Signal

func initializeUDPListener() *net.UDPConn {
	address, err := net.ResolveUDPAddr("udp", config.Service.Port)
	if err != nil {
		log.Fatalf("ERROR: could not resolve local UDP address - %s", err)
	}

	common.Logger.Info(fmt.Sprintf("Listening on %s", address))
	listener, err := net.ListenUDP("udp4", address)

	if err != nil {
		log.Fatalf("ERROR: ListenUDP - %s", err)
	}

	listener.SetReadBuffer(config.Service.SocketReadBuffer)

	return listener

}

// Our loadbalancer relies on the statsd tcp management port for doing health
// checks, so we need to setup and emulate the behavior.
func initializeTCPListener() *net.TCPListener {
	a, err := net.ResolveTCPAddr("tcp4", config.Service.MgmtPort)
	if err != nil {
		log.Fatal("Couldn't resolve the tcp management port: %s", err)
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
	go http.ListenAndServe(config.Service.HTTPPort, nil)

signalLoop:
	for {
		select {
		case sig := <-signalchan:
			common.Logger.Info(fmt.Sprintf("!! Caught signal %d... shutting down\n", sig))
			break signalLoop
		case <-common.StatsTicker.C:
			common.Logger.Info("Sending statsproxy service stats to statsd")
			workers.SendStats()
		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("OK")); err != nil {
		common.Logger.Err(fmt.Sprintf("Error sending healthcheck: %v", err))
	}
}
