StatsProxy
======

StatsProxy receives incoming statsd traffic, hashes the metric name, and sends it to your statsd backends.

Doesn't do any fancy liveliness checks.

StatsD proxy.js is garbage.

###Running

```shell
$ go build
$ ./statsproxy
```