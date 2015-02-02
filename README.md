StatsProxy
======

StatsProxy receives incoming statsd traffic, hashes the metric name, and sends it to one of your statsd backends.

Doesn't do any fancy liveliness checks or consistent hashing.

Initial code & dist script borrowed from https://github.com/bitly/statsdaemon

###Running

```shell
$ go build
$ ./statsproxy -hosts myhost:8127 -hosts myhost:8129
```

```shell
$ ./statsproxy -help
Usage of ./statsproxy:
  -address=":8125": UDP service address
  -cpuprofile="": write cpu profile to file
  -debug=false: print verbose data about packets sent and servers chosen
  -hosts=[]: backend statsd hosts
  -tickerPeriod=5s: writes internal stats every tickerPeriod seconds
  -version=false: print version string
  -workers=2: number of goroutines working on incoming UDP data
```

### Future Work
- Packet coalescence, backend StatsD hosts could use batched UDP data. 
