export type SensorSnapshot = {
  id: string
  sensor_id: number
  lat: number
  lon: number
  alt: number
  last_seen: string
}

export type AircraftEstimate = {
  icao: string
  lat: number
  lon: number
  alt: number
  confidence: "high" | "medium" | "low"
  residual_m: number
  sensors: number
  updated_at: string
  raw_hex: string
}

export type StatsSnapshot = {
  active_sensors: number
  tracked_aircraft: number
  total_packets: number
  solved_clusters: number
  failed_clusters: number
  last_solution_at: string
  last_packet_ingest: string
  server_started_at: string
}

export type FeedEvent =
  | {
      type: "sensor"
      id: string
      lat: number
      lon: number
      alt: number
    }
  | {
      type: "aircraft"
      id: string
      lat: number
      lon: number
      alt: number
      confidence: "high" | "medium" | "low"
      residual_m: number
      sensors: number
      hexData: string
    }
