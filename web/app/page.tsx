"use client"

import { useEffect, useMemo, useRef, useState } from "react"
import type * as Leaflet from "leaflet"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

type SensorEvent = {
  type: "sensor"
  id: string
  lat: number
  lon: number
}

type AircraftEvent = {
  type: "aircraft"
  id: string
  lat: number
  lon: number
  hexData: string
}

type FeedEvent = SensorEvent | AircraftEvent

type FlightCard = {
  id: string
  hexData: string
  seenAt: number
}

const DEFAULT_CENTER: [number, number] = [50.15, -5.65]
const DEFAULT_ZOOM = 8

export default function Page() {
  const mapNodeRef = useRef<HTMLDivElement | null>(null)
  const mapRef = useRef<Leaflet.Map | null>(null)
  const markersRef = useRef<Record<string, Leaflet.CircleMarker | Leaflet.Marker>>({})
  const [isConnected, setIsConnected] = useState(false)
  const [flights, setFlights] = useState<FlightCard[]>([])

  const flightCount = useMemo(() => flights.length, [flights])

  useEffect(() => {
    if (!mapNodeRef.current || mapRef.current) {
      return
    }

    let active = true
    let eventSource: EventSource | null = null
    let map: Leaflet.Map | null = null

    void import("leaflet").then((L) => {
      if (!active || !mapNodeRef.current) {
        return
      }

      map = L.map(mapNodeRef.current, {
        zoomControl: true,
        attributionControl: true,
      }).setView(DEFAULT_CENTER, DEFAULT_ZOOM)
      const currentMap = map

      mapRef.current = currentMap

      L.tileLayer("https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png", {
        attribution: "&copy; OpenStreetMap contributors &copy; CARTO",
      }).addTo(currentMap)

      const planeIcon = L.icon({
        iconUrl: "https://cdn-icons-png.flaticon.com/512/3135/3135715.png",
        iconSize: [20, 20],
        className: "plane-icon",
      })

      eventSource = new EventSource("/events")

      eventSource.onopen = () => {
        setIsConnected(true)
      }

      eventSource.onerror = () => {
        setIsConnected(false)
      }

      eventSource.onmessage = (event) => {
        let data: FeedEvent
        try {
          data = JSON.parse(event.data) as FeedEvent
        } catch {
          return
        }

        if (!Number.isFinite(data.lat) || !Number.isFinite(data.lon)) {
          return
        }

        if (data.type === "sensor") {
          if (!markersRef.current[data.id]) {
            markersRef.current[data.id] = L.circleMarker([data.lat, data.lon], {
              color: "#10b981",
              radius: 6,
              fillOpacity: 1,
            })
              .addTo(currentMap)
              .bindPopup(`<b>Receiver Tower</b><br>ID: ${data.id}`)
          }
          return
        }

        if (markersRef.current[data.id]) {
          markersRef.current[data.id].setLatLng([data.lat, data.lon])
        } else {
          markersRef.current[data.id] = L.marker([data.lat, data.lon], {
            icon: planeIcon,
          })
            .addTo(currentMap)
            .bindPopup(`<b>Aircraft ICAO:</b> ${data.id}`)

          setFlights((prev) => {
            if (prev.some((flight) => flight.id === data.id)) {
              return prev
            }

            const next = [{ id: data.id, hexData: data.hexData, seenAt: Date.now() }, ...prev]
            return next.slice(0, 100)
          })
        }
      }
    })

    return () => {
      active = false
      eventSource?.close()
      setIsConnected(false)
      map?.remove()
      mapRef.current = null
      markersRef.current = {}
    }
  }, [])

  return (
    <main className="h-svh overflow-hidden bg-slate-950 text-slate-100">
      <div className="relative flex h-full flex-col md:flex-row">
        <aside className="z-20 flex h-[45%] w-full flex-col border-b border-slate-700/80 bg-slate-900/95 backdrop-blur md:h-full md:w-[350px] md:border-r md:border-b-0">
          <Card className="rounded-none border-0 border-b border-slate-700/90 bg-slate-950/80 py-0 ring-0">
            <CardHeader className="space-y-2 p-5">
              <CardTitle className="text-lg tracking-[0.16em] uppercase text-sky-400">
                Blue-Glide Oracle
              </CardTitle>
              <div className="flex items-center gap-2 text-xs text-emerald-400">
                <span className="relative inline-flex size-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-70" />
                  <span className="relative inline-flex size-2 rounded-full bg-emerald-400" />
                </span>
                {isConnected ? "Network Connected" : "Reconnecting Feed"}
              </div>
            </CardHeader>
          </Card>

          <div className="flex items-center justify-between px-5 py-3">
            <p className="text-xs tracking-[0.12em] text-slate-400 uppercase">Active Flights</p>
            <Badge
              variant="outline"
              className="border-sky-400/40 bg-sky-500/10 font-mono text-sky-300"
            >
              {flightCount}
            </Badge>
          </div>

          <div className="min-h-0 flex-1 space-y-2 overflow-y-auto px-3 pb-3">
            {flights.length === 0 ? (
              <Card className="border border-slate-700/80 bg-slate-800/80 py-0 ring-0">
                <CardContent className="p-3 text-xs text-slate-400">
                  Waiting for aircraft events from /events.
                </CardContent>
              </Card>
            ) : (
              flights.map((flight) => (
                <Card
                  key={flight.id}
                  className="border border-slate-700/80 bg-slate-700/70 py-0 ring-0"
                >
                  <CardContent className="space-y-1 border-l-4 border-l-sky-400 p-3 font-mono text-sm">
                    <p>
                      ICAO: <strong>{flight.id}</strong>
                    </p>
                    <p className="text-[11px] tracking-[0.08em] text-slate-300 uppercase">
                      VERIFIED BY: MLAT ENGINE
                    </p>
                    <p className="text-[11px] text-slate-400">
                      LATEST HEX: {flight.hexData.substring(0, 14)}...
                    </p>
                  </CardContent>
                </Card>
              ))
            )}
          </div>
        </aside>

        <section className="relative min-h-0 flex-1">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_15%,rgba(56,189,248,0.14),transparent_28%),radial-gradient(circle_at_82%_84%,rgba(16,185,129,0.12),transparent_34%)]" />
          <div ref={mapNodeRef} className="absolute inset-0" />
        </section>
      </div>
    </main>
  )
}
