export type UIStatusResponse = { name: string; api_version: string; web_ui: boolean; started_at: string; uptime_seconds: number; web_bundle?: string; endpoints: string[]; boundary: string };

export type Capability = {
  peer_id: string;
  device: { id: string; memory_gb: number; backend?: string };
  metadata?: Record<string, string>;
};

export type CapabilityResponse = { capabilities: Capability[] };
export type Shard = { device_id: string; start_layer: number; end_layer: number };
export type PlacementPreview = { model_id: string; layers: number; shards: Shard[] };
export type RouteEntry = { peer_id: string; address: string; transport: string; shard: Shard };
export type RoutePreview = { model_id: string; layers: number; routes: RouteEntry[] };
export type LoadState<T> = { loading: boolean; data?: T; error?: string };

export type ModelPreset = { id: string; name: string; path: string; description: string };
export type ModelFileStatus = { pattern: string; present: boolean; matches?: string[] };
export type LocalModel = { id: string; path: string; files: ModelFileStatus[]; complete: boolean };
export type LocalModelsResponse = { root: string; models: LocalModel[]; truncated: boolean; limit: number };
export type ModelCommand = { label: string; command: string };
export type ModelHelperResponse = { status: string; model_path: string; presets: ModelPreset[]; required_files: string[]; files: ModelFileStatus[]; commands: ModelCommand[] };

export type BoundaryStatus = {
  status: "disabled" | "available" | "error";
  detail: string;
};

export class APIError extends Error {
  constructor(public status: number, public statusText: string, message: string) {
    super(`${status} ${statusText}: ${message}`);
    this.name = "APIError";
  }
}

function parseJSON(text: string): { ok: true; value: unknown } | { ok: false } {
  if (!text) return { ok: true, value: undefined };
  try {
    return { ok: true, value: JSON.parse(text) };
  } catch {
    return { ok: false };
  }
}

export async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { accept: "application/json" } });
  const text = await res.text();
  const parsed = parseJSON(text);
  const value = parsed.ok ? parsed.value as { error?: string } | undefined : undefined;
  if (!res.ok) throw new APIError(res.status, res.statusText, value?.error || text || "request failed");
  if (!parsed.ok) throw new APIError(res.status, res.statusText, "invalid JSON response");
  return parsed.value as T;
}

export async function probeShardExecution(): Promise<BoundaryStatus> {
  const res = await fetch("/shards/execute", { method: "POST", headers: { "content-type": "application/json", accept: "application/json" }, body: "{}" });
  const text = await res.text();
  const parsed = parseJSON(text);
  const value = parsed.ok ? parsed.value as { error?: string } | undefined : undefined;
  const detail = value?.error || text || `${res.status} ${res.statusText}`;
  if (res.status === 503) return { status: "disabled", detail };
  if (res.status === 400) return { status: "available", detail: "Shard endpoint is wired and validating requests" };
  if (res.ok) return { status: "available", detail };
  return { status: "error", detail };
}
