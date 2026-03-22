import type { AircraftEstimate, SensorSnapshot, StatsSnapshot } from "@/lib/types"

async function fetchJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, { cache: "no-store" })
  if (!response.ok) {
    throw new Error(`Failed to fetch ${path}: ${response.status}`)
  }
  return (await response.json()) as T
}

export const api = {
  health: () => fetchJSON<{ ok: boolean; service: string }>("/api/health"),
  sensors: () => fetchJSON<SensorSnapshot[]>("/api/sensors"),
  aircraft: () => fetchJSON<AircraftEstimate[]>("/api/aircraft"),
  stats: () => fetchJSON<StatsSnapshot>("/api/stats"),
}
