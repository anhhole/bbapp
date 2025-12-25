package config

import (
	"fmt"
	"bbapp/internal/api"
)

type Manager struct {
	config          *api.Config
	bigoRoomIndex   map[string]*api.Streamer
	streamerIdIndex map[string]*api.Streamer
}

func NewManager(cfg *api.Config) *Manager {
	m := &Manager{
		config:          cfg,
		bigoRoomIndex:   make(map[string]*api.Streamer),
		streamerIdIndex: make(map[string]*api.Streamer),
	}

	// Build indexes
	for i := range cfg.Teams {
		for j := range cfg.Teams[i].Streamers {
			streamer := &cfg.Teams[i].Streamers[j]
			m.bigoRoomIndex[streamer.BigoRoomId] = streamer
			m.streamerIdIndex[streamer.StreamerId] = streamer
		}
	}

	return m
}

func (m *Manager) LookupStreamerByBigoRoom(bigoRoomId string) (*api.Streamer, error) {
	streamer, ok := m.bigoRoomIndex[bigoRoomId]
	if !ok {
		return nil, fmt.Errorf("no streamer found for Bigo room %s", bigoRoomId)
	}
	return streamer, nil
}

func (m *Manager) LookupStreamerById(streamerId string) (*api.Streamer, error) {
	streamer, ok := m.streamerIdIndex[streamerId]
	if !ok {
		return nil, fmt.Errorf("no streamer found with ID %s", streamerId)
	}
	return streamer, nil
}

func (m *Manager) GetAllBigoRoomIds() []string {
	rooms := make([]string, 0, len(m.bigoRoomIndex))
	for roomId := range m.bigoRoomIndex {
		rooms = append(rooms, roomId)
	}
	return rooms
}

func (m *Manager) GetConfig() *api.Config {
	return m.config
}
