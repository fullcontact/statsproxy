package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Packet struct {
	Bucket string
	Raw    string
}

type Hosts []*Host
type Host struct {
	Host string
	Port int
}

func (a *Hosts) Set(s string) error {
	portIdx := strings.Index(s, ":")
	if portIdx < 0 {
		log.Fatalf("Host is missing port: %s\n", s)
	}

	port, err := strconv.Atoi(s[portIdx+1:])
	if err != nil {
		log.Fatalf("Error parsing host: %s - %s\n", s, err)
	}

	*a = append(*a, &Host{s[0:portIdx], port})
	return nil
}

func (p *Host) String() string {
	return fmt.Sprintf("%v:%v", p.Host, p.Port)
}

func (a *Hosts) String() string {
	return fmt.Sprintf("%v", *a)
}
