package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
	exoticruntime "github.com/rcarmo/go-exotic/internal/runtime"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "plan":
			runPlan(os.Args[2:])
			return
		case "run":
			runLocal(os.Args[2:])
			return
		case "serve", "peers":
			fmt.Fprintf(os.Stderr, "%s is not implemented; distributed execution is disabled\n", os.Args[1])
			os.Exit(2)
		}
	}
	// Backward-compatible default: placement preview.
	runPlan(os.Args[1:])
}

func runPlan(args []string) {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	layers := fs.Int("layers", 0, "number of model layers to shard")
	jsonOut := fs.Bool("json", false, "print placement plan as JSON")
	_ = fs.Parse(args)
	if *layers <= 0 {
		fmt.Fprintln(os.Stderr, "usage: go-exotic plan -layers N [-json]")
		os.Exit(2)
	}
	plan, err := placement.NewPlan([]exotic.Device{{ID: "local", MemoryGB: 1, Backend: "go-pherence"}}, *layers)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonOut {
		data, err := plan.JSON()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}
	for _, shard := range plan.Shards {
		fmt.Printf("%s: layers %d-%d\n", shard.DeviceID, shard.StartLayer, shard.EndLayer)
	}
}

func runLocal(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	modelPath := fs.String("model", "../go-pherence/models/smollm2-135m", "go-pherence model directory")
	prompt := fs.String("prompt", "Hello", "prompt text")
	tokens := fs.Int("tokens", 2, "number of tokens to generate")
	timeout := fs.Duration("timeout", 2*time.Minute, "runtime timeout")
	_ = fs.Parse(args)
	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(os.Stderr, "empty model path")
		os.Exit(2)
	}
	if *tokens < 0 {
		fmt.Fprintln(os.Stderr, "tokens must be non-negative")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	adapter := exoticruntime.NewPherenceAdapter()
	meta, err := adapter.Metadata(ctx, *modelPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	generated, err := adapter.Generate(ctx, *modelPath, *prompt, *tokens)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("model=%s type=%s layers=%d hidden=%d vocab=%d tokens=%v\n", meta.ModelPath, meta.ModelType, meta.Layers, meta.HiddenSize, meta.VocabSize, generated)
}
