package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rcarmo/go-exotic/internal/exotic"
)

func main() {
	layers := flag.Int("layers", 0, "number of model layers to shard")
	flag.Parse()
	if *layers <= 0 {
		fmt.Fprintln(os.Stderr, "usage: go-exotic -layers N")
		os.Exit(2)
	}
	plan, err := exotic.PlanLayerShards([]exotic.Device{{ID: "local", MemoryGB: 1, Backend: "go-pherence"}}, *layers)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, shard := range plan {
		fmt.Printf("%s: layers %d-%d\n", shard.DeviceID, shard.StartLayer, shard.EndLayer)
	}
}
