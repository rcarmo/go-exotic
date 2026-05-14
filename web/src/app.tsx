import { render } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";
import * as d3 from "d3";
import { BoundaryStatus, CapabilityResponse, getJSON, LoadState, LocalModelsResponse, ModelHelperResponse, PlacementPreview, probeShardExecution, RoutePreview, UIStatusResponse } from "./api";
import { loadBoundedNumber, loadNumber, loadString, saveValue } from "./storage";
import "./style.css";

function useJSON<T>(path: string): LoadState<T> & { refresh: () => void } {
  const [tick, setTick] = useState(0);
  const [state, setState] = useState<LoadState<T>>({ loading: true });
  useEffect(() => {
    let cancelled = false;
    setState((current) => ({ loading: true, data: current.data }));
    getJSON<T>(path)
      .then((data) => !cancelled && setState({ loading: false, data }))
      .catch((err: Error) => !cancelled && setState({ loading: false, error: err.message }));
    return () => { cancelled = true; };
  }, [path, tick]);
  return { ...state, refresh: () => setTick((v) => v + 1) };
}

function App() {
  const [layers, setLayersState] = useState(() => loadNumber("go-exotic.layers", 4));
  const [model, setModelState] = useState(() => loadString("go-exotic.model", "demo"));
  const [modelPath, setModelPathState] = useState(() => loadString("go-exotic.modelPath", "../go-pherence/models/demo"));
  const [modelRoot, setModelRootState] = useState(() => loadString("go-exotic.modelRoot", "../go-pherence/models"));
  const [modelLimit, setModelLimitState] = useState(() => loadBoundedNumber("go-exotic.modelLimit", 50, 1, 200));
  const setLayers = (value: number) => { const next = Number.isFinite(value) ? Math.max(1, value) : 1; setLayersState(next); saveValue("go-exotic.layers", next); };
  const setModel = (value: string) => { setModelState(value); saveValue("go-exotic.model", value); };
  const setModelPath = (value: string) => { setModelPathState(value); saveValue("go-exotic.modelPath", value); };
  const setModelRoot = (value: string) => { setModelRootState(value); saveValue("go-exotic.modelRoot", value); };
  const setModelLimit = (value: number) => { const next = Number.isFinite(value) ? Math.max(1, Math.min(200, value)) : 1; setModelLimitState(next); saveValue("go-exotic.modelLimit", next); };
  const uiStatus = useJSON<UIStatusResponse>("/ui/status");
  const caps = useJSON<CapabilityResponse>("/capabilities");
  const boundary = useBoundaryStatus();
  const localModels = useJSON<LocalModelsResponse>(`/models/local?root=${encodeURIComponent(modelRoot)}&limit=${modelLimit}`);
  const helpers = useJSON<ModelHelperResponse>(`/models/helpers?model=${encodeURIComponent(model)}&path=${encodeURIComponent(modelPath)}`);
  const placement = useJSON<PlacementPreview>(`/placement/preview?layers=${layers}&model=${encodeURIComponent(model)}`);
  const routes = useJSON<RoutePreview>(`/routes/preview?layers=${layers}&model=${encodeURIComponent(model)}`);

  return <main>
    <header class="hero">
      <div>
        <p class="eyebrow">go-exotic</p>
        <h1>Distributed inference planner</h1>
        <p>Peer capabilities, placement previews, route previews, and model setup helpers. Shard execution remains opt-in.</p>
      </div>
      <button onClick={() => { uiStatus.refresh(); caps.refresh(); boundary.refresh(); localModels.refresh(); helpers.refresh(); placement.refresh(); routes.refresh(); }}>Refresh</button>
    </header>

    <section class="controls card">
      <label>Model ID <input value={model} onInput={(e) => setModel((e.currentTarget as HTMLInputElement).value)} /></label>
      <label>Model path <input value={modelPath} onInput={(e) => setModelPath((e.currentTarget as HTMLInputElement).value)} /></label>
      <label>Model root <input value={modelRoot} onInput={(e) => setModelRoot((e.currentTarget as HTMLInputElement).value)} /></label>
      <label>Inventory limit <input class={modelLimit < 1 || modelLimit > 200 ? "invalid" : ""} type="number" min="1" max="200" value={modelLimit} onInput={(e) => setModelLimit(Number((e.currentTarget as HTMLInputElement).value || 1))} /></label>
      <label>Layers <input class={layers < 1 ? "invalid" : ""} type="number" min="1" value={layers} onInput={(e) => setLayers(Number((e.currentTarget as HTMLInputElement).value || 1))} /></label>
    </section>

    <section class="grid">
      <StatusCard state={uiStatus} />
      <PeersCard state={caps} />
      <BoundaryCard state={boundary} />
      <ModelHelpers state={helpers} localModels={localModels} modelRoot={modelRoot} onSelectPreset={(id, path) => { setModel(id); setModelPath(path); }} />
    </section>

    <section class="grid wide">
      <PreviewCard title="Placement" state={placement} kind="placement" />
      <PreviewCard title="Routes" state={routes} kind="routes" />
    </section>
  </main>;
}

function StatusCard({ state }: { state: LoadState<UIStatusResponse> }) {
  return <section class="card">
    <h2>Server</h2>
    {state.loading && <p>Loading server status…</p>}
    {state.error && <p class="error">{state.error}</p>}
    {state.data && <>
      <p><strong>{state.data.name}</strong> <span>{state.data.api_version}</span></p>
      <p>{state.data.boundary}</p>
      <small>started {state.data.started_at} · uptime {formatDuration(state.data.uptime_seconds)} · bundle {state.data.web_bundle || "unknown"} · {state.data.endpoints.length} API endpoints advertised</small>
    </>}
  </section>;
}

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  if (mins < 1) return `${secs}s`;
  const hours = Math.floor(mins / 60);
  if (hours < 1) return `${mins}m ${secs}s`;
  return `${hours}h ${mins % 60}m`;
}

function useBoundaryStatus(): LoadState<BoundaryStatus> & { refresh: () => void } {
  const [tick, setTick] = useState(0);
  const [state, setState] = useState<LoadState<BoundaryStatus>>({ loading: true });
  useEffect(() => {
    let cancelled = false;
    setState((current) => ({ loading: true, data: current.data }));
    probeShardExecution()
      .then((data) => !cancelled && setState({ loading: false, data }))
      .catch((err: Error) => !cancelled && setState({ loading: false, error: err.message }));
    return () => { cancelled = true; };
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

function ModelHelpers({ state, localModels, modelRoot, onSelectPreset }: { state: LoadState<ModelHelperResponse>; localModels: LoadState<LocalModelsResponse>; modelRoot: string; onSelectPreset: (id: string, path: string) => void }) {
  const commands = state.data?.commands || [];
  const modelCount = localModels.data?.models.length || 0;
  const completeCount = localModels.data?.models.filter((item) => item.complete).length || 0;
  return <section class="card">
    <h2>Model helpers</h2>
    <p>Download orchestration is not automated yet. Stage a local go-pherence model fixture, then run the smoke checks below.</p>
    {state.loading && <p>Loading model helpers…</p>}
    {state.error && <p class="error">{state.error}</p>}
    {state.data?.presets && <div class="presets">{state.data.presets.map((preset) => <button type="button" key={preset.id} onClick={() => onSelectPreset(preset.id, preset.path)} title={preset.description}>{preset.name}</button>)}</div>}
    <div class="local-models"><strong>Local fixtures</strong><small>root: {modelRoot}</small>{localModels.data && <span class="inventory-summary">{completeCount}/{modelCount} complete</span>}{localModels.loading && <span>Scanning…</span>}{localModels.error && <span class="error">{localModels.error}</span>}{localModels.data?.truncated && <span class="warn">showing first {localModels.data.limit}</span>}{localModels.data?.models.map((item) => <button type="button" class={item.complete ? "complete" : "incomplete"} key={item.path} onClick={() => onSelectPreset(item.id, item.path)}>{item.complete ? "✓" : "…"} {item.id}</button>)}</div>
    {state.data?.files ? <ul class="file-status">{state.data.files.map((file) => <li class={file.present ? "present" : "missing"} key={file.pattern}>
      <span>{file.present ? "✓" : "×"}</span> <strong>{file.pattern}</strong> <small>{file.matches?.join(", ") || "not found"}{file.truncated ? ` … first ${file.limit}` : ""}</small>
    </li>)}</ul> : <ol>{["config.json", "tokenizer.json", "*.safetensors"].map((item) => <li>{item}</li>)}</ol>}
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
  const totalLayers = state.data?.layers || 0;
  const summary = bars.length > 0 ? `${bars.length} ${kind === "placement" ? "shards" : "routes"} covering ${totalLayers} layers` : "";
  return <section class="card">
    <h2>{title}</h2>
    {state.loading && <p>Loading {title.toLowerCase()}…</p>}
    {state.error && <p class="error">{state.error}</p>}
    {summary && <p class="preview-summary">{summary}</p>}
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
