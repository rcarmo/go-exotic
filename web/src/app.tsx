import { render } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";
import * as d3 from "d3";
import { BoundaryStatus, CapabilityResponse, getJSON, LoadState, PlacementPreview, probeShardExecution, RoutePreview } from "./api";
import "./style.css";

function useJSON<T>(path: string, deps: unknown[] = []): LoadState<T> & { refresh: () => void } {
  const [tick, setTick] = useState(0);
  const [state, setState] = useState<LoadState<T>>({ loading: true });
  useEffect(() => {
    let cancelled = false;
    setState({ loading: true, data: state.data });
    getJSON<T>(path)
      .then((data) => !cancelled && setState({ loading: false, data }))
      .catch((err: Error) => !cancelled && setState({ loading: false, error: err.message }));
    return () => { cancelled = true; };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path, tick, ...deps]);
  return { ...state, refresh: () => setTick((v) => v + 1) };
}

function App() {
  const [layers, setLayers] = useState(4);
  const [model, setModel] = useState("demo");
  const [modelPath, setModelPath] = useState("../go-pherence/models/demo");
  const caps = useJSON<CapabilityResponse>("/capabilities");
  const boundary = useBoundaryStatus();
  const placement = useJSON<PlacementPreview>(`/placement/preview?layers=${layers}&model=${encodeURIComponent(model)}`, [layers, model]);
  const routes = useJSON<RoutePreview>(`/routes/preview?layers=${layers}&model=${encodeURIComponent(model)}`, [layers, model]);

  return <main>
    <header class="hero">
      <div>
        <p class="eyebrow">go-exotic</p>
        <h1>Distributed inference planner</h1>
        <p>Peer capabilities, placement previews, route previews, and model setup helpers. Shard execution remains opt-in.</p>
      </div>
      <button onClick={() => { caps.refresh(); placement.refresh(); routes.refresh(); }}>Refresh</button>
    </header>

    <section class="controls card">
      <label>Model ID <input value={model} onInput={(e) => setModel((e.currentTarget as HTMLInputElement).value)} /></label>
      <label>Model path <input value={modelPath} onInput={(e) => setModelPath((e.currentTarget as HTMLInputElement).value)} /></label>
      <label>Layers <input type="number" min="1" value={layers} onInput={(e) => setLayers(Math.max(1, Number((e.currentTarget as HTMLInputElement).value || 1)))} /></label>
    </section>

    <section class="grid">
      <PeersCard state={caps} />
      <BoundaryCard state={boundary} />
      <ModelHelpers model={model} modelPath={modelPath} />
    </section>

    <section class="grid wide">
      <PreviewCard title="Placement" state={placement} kind="placement" />
      <PreviewCard title="Routes" state={routes} kind="routes" />
    </section>
  </main>;
}

function useBoundaryStatus(): LoadState<BoundaryStatus> & { refresh: () => void } {
  const [tick, setTick] = useState(0);
  const [state, setState] = useState<LoadState<BoundaryStatus>>({ loading: true });
  useEffect(() => {
    let cancelled = false;
    setState({ loading: true, data: state.data });
    probeShardExecution()
      .then((data) => !cancelled && setState({ loading: false, data }))
      .catch((err: Error) => !cancelled && setState({ loading: false, error: err.message }));
    return () => { cancelled = true; };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick]);
  return { ...state, refresh: () => setTick((v) => v + 1) };
}

function PeersCard({ state }: { state: LoadState<CapabilityResponse> }) {
  return <section class="card">
    <h2>Peers</h2>
    {state.loading && <p>Loading peers…</p>}
    {state.error && <p class="error">{state.error}</p>}
    {state.data?.capabilities.map((cap) => <div class="peer" key={cap.peer_id}>
      <strong>{cap.peer_id}</strong>
      <span>{cap.device.backend || "unknown backend"}</span>
      <span>{cap.device.memory_gb.toFixed(2)} GB</span>
      <small>{cap.metadata?.address || "no advertised address"}</small>
    </div>)}
  </section>;
}

function BoundaryCard({ state }: { state: LoadState<BoundaryStatus> }) {
  const status = state.data?.status || "error";
  return <section class={`card boundary ${status}`}>
    <h2>Execution boundary</h2>
    {state.loading && <p>Checking shard endpoint…</p>}
    {state.error && <p class="error">{state.error}</p>}
    {state.data && <>
      <p><strong>{status === "disabled" ? "Shard execution disabled" : status === "available" ? "Shard endpoint wired" : "Shard endpoint error"}</strong></p>
      <p>{state.data.detail}</p>
      <small>Route/placement previews are safe metadata calls. `/shards/execute` only runs when the server is explicitly started with shard-worker wiring.</small>
    </>}
  </section>;
}

function ModelHelpers({ model, modelPath }: { model: string; modelPath: string }) {
  const safeModel = model.trim() || "MODEL";
  const safePath = modelPath.trim() || `../go-pherence/models/${safeModel}`;
  const commands = [
    { label: "Create fixture directory", command: `mkdir -p ${safePath}` },
    { label: "List required files", command: `find ${safePath} -maxdepth 1 \\( -name 'config.json' -o -name 'tokenizer.json' -o -name '*.safetensors' \\) -print` },
    { label: "Local generation smoke", command: `go run ./cmd/go-exotic run -model ${safePath} -prompt "Hello" -tokens 1` },
    { label: "Planning preview", command: `go run ./cmd/go-exotic routes -layers 4 -model ${safeModel} -json` },
    { label: "Explicit shard-server opt-in", command: `go run ./cmd/go-exotic serve -addr 127.0.0.1:8089 -shard-model ${safePath}` },
  ];
  return <section class="card">
    <h2>Model helpers</h2>
    <p>Download orchestration is not automated yet. Stage a local go-pherence model fixture, then run the smoke checks below.</p>
    <ol>{["config.json", "tokenizer.json", "*.safetensors"].map((item) => <li>{item}</li>)}</ol>
    <div class="commands">{commands.map((item) => <CommandRow key={item.label} label={item.label} command={item.command} />)}</div>
  </section>;
}

function CommandRow({ label, command }: { label: string; command: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard?.writeText(command);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };
  return <div class="command-row">
    <div>
      <strong>{label}</strong>
      <code>{command}</code>
    </div>
    <button type="button" onClick={copy}>{copied ? "Copied" : "Copy"}</button>
  </div>;
}

function PreviewCard<T extends PlacementPreview | RoutePreview>({ title, state, kind }: { title: string; state: LoadState<T>; kind: "placement" | "routes" }) {
  const bars = useMemo(() => {
    if (!state.data) return [];
    if (kind === "placement") return (state.data as PlacementPreview).shards.map((s) => ({ id: s.device_id, start: s.start_layer, end: s.end_layer }));
    return (state.data as RoutePreview).routes.map((r) => ({ id: r.peer_id, start: r.shard.start_layer, end: r.shard.end_layer }));
  }, [state.data, kind]);
  return <section class="card">
    <h2>{title}</h2>
    {state.loading && <p>Loading {title.toLowerCase()}…</p>}
    {state.error && <p class="error">{state.error}</p>}
    {bars.length > 0 && <Timeline bars={bars} />}
    {state.data && <pre>{JSON.stringify(state.data, null, 2)}</pre>}
  </section>;
}

function Timeline({ bars }: { bars: { id: string; start: number; end: number }[] }) {
  const width = 520;
  const height = Math.max(90, bars.length * 34 + 30);
  const maxLayer = d3.max(bars, (d) => d.end + 1) || 1;
  const x = d3.scaleLinear().domain([0, maxLayer]).range([90, width - 20]);
  return <svg class="timeline" viewBox={`0 0 ${width} ${height}`} role="img">
    {bars.map((bar, i) => <g key={bar.id} transform={`translate(0 ${20 + i * 34})`}>
      <text x="8" y="18">{bar.id}</text>
      <rect x={x(bar.start)} y="4" width={Math.max(2, x(bar.end + 1) - x(bar.start))} height="20" rx="5" />
      <text class="range" x={x(bar.start) + 8} y="18">{bar.start}-{bar.end}</text>
    </g>)}
  </svg>;
}

render(<App />, document.getElementById("app")!);
