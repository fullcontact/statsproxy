package main

import (
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"
)

func init() {
	*debug = true
}

func TestParseMessage(t *testing.T) {
	sourceMsg := []byte("databus.consumer.test:1|g\n")

	pktSlice := parseMessage(sourceMsg)

	assert.Equal(t, "databus.consumer.test", pktSlice[0].Bucket)
	assert.Equal(t, "databus.consumer.test:1|g\n", pktSlice[0].Raw)
}

func TestParseMultipleMessages(t *testing.T) {
	sourceMsg := []byte("databus.consumer.test:1|g\ndatabus.consumer.test2:1|g\n")

	pktSlice := parseMessage(sourceMsg)

	assert.Equal(t, "databus.consumer.test", pktSlice[0].Bucket)
	assert.Equal(t, "databus.consumer.test:1|g\n", pktSlice[0].Raw)

	assert.Equal(t, "databus.consumer.test2", pktSlice[1].Bucket)
	assert.Equal(t, "databus.consumer.test2:1|g\n", pktSlice[1].Raw)
}

type w []byte

func (w *w) Write(b []byte) (int, error) {
	*w = append(*w, b...)
	return len(b), nil
}

func TestPacketRouting(t *testing.T) {
	host_a := &Host{"a", 1234}
	host_b := &Host{"b", 1235}
	hosts = Hosts{host_a, host_b}

	var aio w
	var bio w
	hostConnections = make(map[*Host]io.Writer)
	hostConnections[host_a] = iotest.NewWriteLogger("hosta", &aio)
	hostConnections[host_b] = iotest.NewWriteLogger("hostb", &bio)

	packetHandler(&Packet{"test", "test:1|c\n"})   // hostb
	packetHandler(&Packet{"test3", "test3:1|c\n"}) // hosta

	assert.Equal(t, "test:1|c\n", string(bio))
	assert.Equal(t, "test3:1|c\n", string(aio))
}

func TestPeriodicStats(t *testing.T) {
	host_a := &Host{"a", 1234}
	hosts = Hosts{host_a}
	var aio w
	hostConnections = make(map[*Host]io.Writer)
	hostConnections[host_a] = iotest.NewWriteLogger("hosta", &aio)

	atomic.StoreUint32(&packetCount, 1000)

	*tickerPeriod = 500 * time.Millisecond

	go func() {
		time.Sleep(600 * time.Millisecond)
		signalchan <- os.Interrupt
	}()

	monitor()

	assert.Equal(t, "statsproxy.packets:1000|c\n", string(aio))
}

func TestDataHandler(t *testing.T) {
	host_a := &Host{"a", 1234}
	hosts = Hosts{host_a}
	var aio w
	hostConnections = make(map[*Host]io.Writer)
	hostConnections[host_a] = iotest.NewWriteLogger("hosta", &aio)

	data := "stat1:1|c\nstat2:1|g\n"

	dataHandler([]byte(data))

	assert.Equal(t, data, string(aio))
}

func BenchmarkPacketRouting(b *testing.B) {
	b.ReportAllocs()
	*debug = false

	host_a := &Host{"a", 1234}
	host_b := &Host{"b", 1235}
	hosts = Hosts{host_a, host_b}

	var aio w
	var bio w

	hostConnections = make(map[*Host]io.Writer)
	hostConnections[host_a] = &aio
	hostConnections[host_b] = &bio

	sources := []Packet{Packet{"test", "test:1|c\n"}, Packet{"test3", "test3:1|c\n"}}

	for i := 0; i < b.N; i++ {
		packetHandler(&sources[i%2]) // hosta
	}

	*debug = true
}

var pkt []Packet

func BenchmarkParseMessage(b *testing.B) {
	b.ReportAllocs()
	*debug = false

	sourceMsg := []byte("databus.consumer.test:1|g\n")

	for i := 0; i < b.N; i++ {
		pkt = parseMessage(sourceMsg)
	}

	*debug = true
}
