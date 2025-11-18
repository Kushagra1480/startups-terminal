package main

import (
	"encoding/json"
	"os"
	"time"
)

type Cache struct {
	Startups    []*Startup `json:"startups"`
	LastUpdated time.Time  `json:"last_updated"`
}

func SaveCache(startups []*Startup) error {
	cache := Cache{
		Startups:    startups,
		LastUpdated: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("startups_cache.json", data, 0644)
}

func LoadCache() (*Cache, error) {
	data, err := os.ReadFile("startups_cache.json")
	if err != nil {
		return nil, err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}
