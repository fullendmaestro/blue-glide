"use client"

import { useEffect, useMemo, useRef, useState } from "react"
import type * as Leaflet from "leaflet"
import { Radar, TowerControl } from "lucide-react"

import { api } from "@/lib/api"
import type { AircraftEstimate, FeedEvent, SensorSnapshot, StatsSnapshot } from "@/lib/types"

const DEFAULT_CENTER: [number, number] = [50.11, -5.59]
const DEFAULT_ZOOM = 9

const FALLBACK_STATS: StatsSnapshot = {
  active_sensors: 0,
  tracked_aircraft: 0,
  total_packets: 0,
  solved_clusters: 0,
  failed_clusters: 0,
  last_solution_at: "",
  last_packet_ingest: "",
  server_started_at: "",
}

type MarkerStore = {
  sensors: Record<string, Leaflet.CircleMarker>
  aircraft: Record<string, Leaflet.Marker>
}

function confidenceClass(confidence: AircraftEstimate["confidence"]): string {
  if (confidence === "high") return "chip chip-high"
  if (confidence === "medium") return "chip chip-medium"
  return "chip chip-low"
}

export default function Page() {
  const mapNodeRef = useRef<HTMLDivElement | null>(null)
  const mapRef = useRef<Leaflet.Map | null>(null)
  const markersRef = useRef<MarkerStore>({ sensors: {}, aircraft: {} })

  const [status, setStatus] = useState<"connecting" | "live" | "down">("connecting")
  const [sensors, setSensors] = useState<SensorSnapshot[]>([])
  const [aircraft, setAircraft] = useState<AircraftEstimate[]>([])
  const [stats, setStats] = useState<StatsSnapshot>(FALLBACK_STATS)
  const [error, setError] = useState<string>("")

  const activeAircraft = useMemo(() => aircraft.slice(0, 40), [aircraft])

  useEffect(() => {
    let mounted = true
    ;(async () => {
      try {
        const [health, initialSensors, initialAircraft, initialStats] = await Promise.all([
          api.health(),
          api.sensors(),
          api.aircraft(),
          api.stats(),
        ])
        if (!mounted) return
        if (!health.ok) {
          throw new Error("API health check failed")
        }
        setSensors(initialSensors)
        setAircraft(initialAircraft)
        setStats(initialStats)
      } catch (e) {
        setError(e instanceof Error ? e.message : "Unable to reach MLAT API")
      }
    })()

    return () => {
      mounted = false
    }
  }, [])

  useEffect(() => {
    if (!mapNodeRef.current || mapRef.current) return

    let active = true
    let source: EventSource | null = null
    let map: Leaflet.Map | null = null

    void import("leaflet").then((L) => {
      if (!active || !mapNodeRef.current) return

      map = L.map(mapNodeRef.current, {
        zoomControl: true,
        attributionControl: true,
      }).setView(DEFAULT_CENTER, DEFAULT_ZOOM)
      mapRef.current = map

      L.tileLayer("https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png", {
        attribution: "&copy; OpenStreetMap contributors &copy; CARTO",
      }).addTo(map)

      const sensorStyle = {
        color: "#1d4ed8",
        fillColor: "#60a5fa",
        radius: 7,
        fillOpacity: 0.9,
        weight: 2,
      }

      const planeIcon = L.divIcon({
        html: '<span class="plane-dot"></span>',
        className: "plane-pin",
        iconSize: [20, 20],
      })

      source = new EventSource("/events")
      source.onopen = () => {
        setStatus("live")
      }

      source.onerror = () => {
        setStatus("down")
      }

      source.onmessage = (event) => {
        let payload: FeedEvent
        try {
          payload = JSON.parse(event.data) as FeedEvent
        } catch {
          return
        }

        if (!Number.isFinite(payload.lat) || !Number.isFinite(payload.lon)) {
          return
        }

        if (payload.type === "sensor") {
          setSensors((prev) => {
            const existing = prev.find((sensor) => sensor.id === payload.id)
            if (existing) {
              return prev.map((sensor) =>
                sensor.id === payload.id
                  ? {
                      ...sensor,
                      lat: payload.lat,
                      lon: payload.lon,
                      alt: payload.alt,
                      last_seen: new Date().toISOString(),
                    }
                  : sensor,
              )
            }
            return [
              {
                id: payload.id,
                sensor_id: 0,
                lat: payload.lat,
                lon: payload.lon,
                alt: payload.alt,
                last_seen: new Date().toISOString(),
              },
              ...prev,
            ]
          })

          const existingMarker = markersRef.current.sensors[payload.id]
          if (existingMarker) {
            existingMarker.setLatLng([payload.lat, payload.lon])
          } else if (map) {
            markersRef.current.sensors[payload.id] = L.circleMarker([payload.lat, payload.lon], sensorStyle)
              .addTo(map)
              .bindTooltip(`Sensor ${payload.id.slice(0, 10)}...`)
          }
          return
        }

        setAircraft((prev) => {
          const next: AircraftEstimate = {
            icao: payload.id,
            lat: payload.lat,
            lon: payload.lon,
            alt: payload.alt,
            confidence: payload.confidence,
            residual_m: payload.residual_m,
            sensors: payload.sensors,
            updated_at: new Date().toISOString(),
            raw_hex: payload.hexData,
          }

          const idx = prev.findIndex((item) => item.icao === payload.id)
          if (idx >= 0) {
            const updated = [...prev]
            updated[idx] = next
            updated.sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at))
            return updated
          }

          return [next, ...prev].slice(0, 120)
        })

        const existingFlightMarker = markersRef.current.aircraft[payload.id]
        if (existingFlightMarker) {
          existingFlightMarker.setLatLng([payload.lat, payload.lon])
        } else if (map) {
          markersRef.current.aircraft[payload.id] = L.marker([payload.lat, payload.lon], {
            icon: planeIcon,
          })
            .addTo(map)
            .bindTooltip(`ICAO ${payload.id}`)
        }
      }
    })

    return () => {
      active = false
      source?.close()
      map?.remove()
      mapRef.current = null
      markersRef.current = { sensors: {}, aircraft: {} }
    }
  }, [])

  useEffect(() => {
    const map = mapRef.current
    if (!map) return

    void import("leaflet").then((L) => {
      const sensorStyle = {
        color: "#1d4ed8",
        fillColor: "#60a5fa",
        radius: 7,
        fillOpacity: 0.9,
        weight: 2,
      }

      for (const sensor of sensors) {
        const existing = markersRef.current.sensors[sensor.id]
        if (existing) {
          existing.setLatLng([sensor.lat, sensor.lon])
          continue
        }

        markersRef.current.sensors[sensor.id] = L.circleMarker([sensor.lat, sensor.lon], sensorStyle)
          .addTo(map)
          .bindTooltip(`Sensor ${sensor.id.slice(0, 10)}...`)
      }
    })
  }, [sensors])

  useEffect(() => {
    const map = mapRef.current
    if (!map) return

    void import("leaflet").then((L) => {
      const planeIcon = L.divIcon({
        html: '<span class="plane-dot"></span>',
        className: "plane-pin",
        iconSize: [20, 20],
      })

      for (const flight of aircraft) {
        const existing = markersRef.current.aircraft[flight.icao]
        if (existing) {
          existing.setLatLng([flight.lat, flight.lon])
          continue
        }

        markersRef.current.aircraft[flight.icao] = L.marker([flight.lat, flight.lon], {
          icon: planeIcon,
        })
          .addTo(map)
          .bindTooltip(`ICAO ${flight.icao}`)
      }
    })
  }, [aircraft])

  useEffect(() => {
    setStats((prev) => ({
      ...prev,
      active_sensors: sensors.length,
      tracked_aircraft: aircraft.length,
    }))
  }, [aircraft.length, sensors.length])

  return (
    <main className="hud-root">
      <div className="hud-shell">
        <header className="hud-header">
          <div>
            <p className="hud-kicker">4DSKY CHALLENGE</p>
            <h1>Blue Glide Multilateration Console</h1>
          </div>
          <div className="status-wrap">
            <span className={`status-dot ${status}`} />
            <span className="status-text">
              {status === "live" ? "LIVE FEED" : status === "connecting" ? "CONNECTING" : "RECONNECTING"}
            </span>
          </div>
        </header>

        <section className="metrics">
          <article className="metric-card">
            <TowerControl size={16} />
            <div>
              <p>Sensors</p>
              <strong>{stats.active_sensors}</strong>
            </div>
          </article>
          <article className="metric-card">
            <Radar size={16} />
            <div>
              <p>Tracked Aircraft</p>
              <strong>{stats.tracked_aircraft}</strong>
            </div>
          </article>
          <article className="metric-card">
            <span className="metric-label">Packets</span>
            <strong>{stats.total_packets.toLocaleString()}</strong>
          </article>
          <article className="metric-card">
            <span className="metric-label">Solved</span>
            <strong>{stats.solved_clusters.toLocaleString()}</strong>
          </article>
        </section>

        <section className="workspace">
          <aside className="aircraft-panel">
            <div className="panel-title-row">
              <h2>Latest Solutions</h2>
              <span className="chip">{activeAircraft.length} on scope</span>
            </div>

            {error ? <p className="error-callout">{error}</p> : null}

            <ul className="aircraft-list">
              {activeAircraft.length === 0 ? (
                <li className="empty-state">Waiting for clusters with 3+ sensors...</li>
              ) : (
                activeAircraft.map((flight) => (
                  <li key={flight.icao} className="aircraft-item">
                    <div className="flight-top">
                      <h3>{flight.icao}</h3>
                      <span className={confidenceClass(flight.confidence)}>{flight.confidence}</span>
                    </div>
                    <p>
                      {flight.lat.toFixed(5)}, {flight.lon.toFixed(5)} | alt {Math.round(flight.alt)} m
                    </p>
                    <p>
                      residual {Math.round(flight.residual_m)} m with {flight.sensors} sensors
                    </p>
                    <p className="mono">{flight.raw_hex.slice(0, 22)}...</p>
                  </li>
                ))
              )}
            </ul>
          </aside>

          <div className="map-wrap">
            <div className="map-overlay" />
            <div ref={mapNodeRef} className="map-canvas" />
          </div>
        </section>
      </div>
    </main>
  )
}
