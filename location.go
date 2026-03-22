package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type LocationOverride struct {
	PublicKey string  `json:"public_key"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Altitude  float64 `json:"alt"`
	Name      string  `json:"name"`
}

type LocationOverrideManager struct {
	mu        sync.RWMutex
	overrides map[string]LocationOverride
}

func NewLocationOverrideManager(path string) (*LocationOverrideManager, error) {
	mgr := &LocationOverrideManager{overrides: make(map[string]LocationOverride)}
	if strings.TrimSpace(path) == "" {
		return mgr, nil
	}

	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return mgr, nil
		}
		return nil, fmt.Errorf("read overrides: %w", err)
	}

	var entries []LocationOverride
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parse overrides: %w", err)
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, entry := range entries {
		if entry.PublicKey != "" {
			mgr.overrides[strings.ToLower(entry.PublicKey)] = entry
		}
		if entry.Name != "" {
			mgr.overrides[strings.ToLower(entry.Name)] = entry
		}
	}

	return mgr, nil
}

func (l *LocationOverrideManager) ApplyOverride(identity string, packet *ModeSPacket) {
	if packet == nil {
		return
	}

	l.mu.RLock()
	override, ok := l.overrides[strings.ToLower(identity)]
	l.mu.RUnlock()
	if !ok {
		return
	}

	packet.Latitude = override.Latitude
	packet.Longitude = override.Longitude
	packet.Altitude = override.Altitude
}
