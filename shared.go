package main

import (
	"hash/fnv"
	"log"
)

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func DPrintf(format string, v ...interface{}) {
	if *debug {
		log.Printf(format, v...)
	}
}
