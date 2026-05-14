export function loadString(key: string, fallback: string): string {
  try {
    return window.localStorage.getItem(key) || fallback;
  } catch {
    return fallback;
  }
}

export function loadNumber(key: string, fallback: number): number {
  const raw = loadString(key, String(fallback));
  const parsed = Number(raw);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

export function loadBoundedNumber(key: string, fallback: number, min: number, max: number): number {
  const value = loadNumber(key, fallback);
  return Math.max(min, Math.min(max, value));
}

export function saveValue(key: string, value: string | number): void {
  try {
    window.localStorage.setItem(key, String(value));
  } catch {
    // Ignore storage failures; controls still work for the current session.
  }
}
