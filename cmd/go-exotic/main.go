package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
	exoticruntime "github.com/rcarmo/go-exotic/internal/runtime"
	"github.com/rcarmo/go-exotic/internal/server"
	"github.com/rcarmo/go-exotic/internal/sim"
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
		case "serve":
			runServe(os.Args[2:])
			return
		case "routes":
			runRoutes(os.Args[2:])
			return
		case "peers":
			runPeers(os.Args[2:])
			return
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

func runRoutes(args []string) {
	fs := flag.NewFlagSet("routes", flag.ExitOnError)
	layers := fs.Int("layers", 0, "number of model layers to route")
	modelID := fs.String("model", "", "model identifier for preview metadata")
	jsonOut := fs.Bool("json", false, "print route preview as JSON")
	_ = fs.Parse(args)
	if *layers <= 0 {
		fmt.Fprintln(os.Stderr, "usage: go-exotic routes -layers N [-model ID] [-json]")
		os.Exit(2)
	}
	preview, err := router.PreviewFromCapabilities(context.Background(), localCapabilities("http://127.0.0.1:0"), *modelID, *layers)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonOut {
		data, err := json.MarshalIndent(preview, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}
	for _, route := range preview.Routes {
		fmt.Printf("%s (%s): layers %d-%d via %s\n", route.PeerID, route.Address, route.Shard.StartLayer, route.Shard.EndLayer, route.Transport)
	}
}

func runPeers(args []string) {
	fs := flag.NewFlagSet("peers", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "print peers as JSON")
	_ = fs.Parse(args)
	caps := localCapabilities()
	if *jsonOut {
		data, err := json.MarshalIndent(protocol.CapabilitiesResponse{Capabilities: caps}, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}
	for _, cap := range caps {
		fmt.Printf("%s: backend=%s memory=%.2fGB\n", cap.PeerID, cap.Device.Backend, cap.Device.MemoryGB)
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8089", "HTTP listen address")
	verbose := fs.Bool("verbose", false, "enable structured diagnostic logging")
	shardModel := fs.String("shard-model", "", "enable local shard execution with this go-pherence model directory (explicit opt-in)")
	_ = fs.Parse(args)
	if strings.TrimSpace(*addr) == "" {
		fmt.Fprintln(os.Stderr, "empty listen address")
		os.Exit(2)
	}
	logger := newLogger(*verbose)
	serverOpts := []server.Option(nil)
	shardEnabled := strings.TrimSpace(*shardModel) != ""
	if shardEnabled {
		model, err := exoticruntime.LoadLocalModel(*shardModel)
		if err != nil {
			logger.Error("shard_model_load_error", "error", err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		exec, err := exoticruntime.NewPherenceLayerExecutor(model)
		if err != nil {
			logger.Error("shard_executor_error", "error", err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		worker := &sim.LayerExecutorWorker{Executor: exec, KVCacheK: make([][]float32, model.Config.NumLayers), KVCacheV: make([][]float32, model.Config.NumLayers), TotalLayers: model.Config.NumLayers}
		serverOpts = append(serverOpts, server.WithShardExecution(worker, model.Config.NumLayers))
		logger.Info("shard_execution_enabled", "model", *shardModel, "layers", model.Config.NumLayers)
	}
	srv := &http.Server{Addr: *addr, Handler: server.New(localCapabilities(advertiseHTTPAddress(*addr)), serverOpts...).Handler(), ReadHeaderTimeout: 5 * time.Second}
	logger.Info("serve_start", "addr", *addr, "distributed_generation", "disabled", "shard_execution", shardEnabled)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("serve_error", "error", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newLogger(verbose bool) *slog.Logger {
	if !verbose {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func localCapabilities(address ...string) []protocol.Capability {
	metadata := map[string]string{"mode": "local-only"}
	if len(address) > 0 && strings.TrimSpace(address[0]) != "" {
		metadata["address"] = advertiseHTTPAddress(address[0])
		metadata["transport"] = "http"
	}
	return []protocol.Capability{{
		PeerID:   "local",
		Device:   exotic.Device{ID: "local", MemoryGB: 1, Backend: "go-pherence"},
		Metadata: metadata,
	}}
}

func advertiseHTTPAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	} else if host, port, err := net.SplitHostPort(addr); err == nil && (host == "" || host == "0.0.0.0" || host == "::" || host == "[::]") {
		addr = net.JoinHostPort("127.0.0.1", port)
	}
	return "http://" + strings.TrimRight(addr, "/")
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
