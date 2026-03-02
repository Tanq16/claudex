package parser

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tanq16/claudex/internal/model"
)

func ParseStatsCache(configDir string) (model.StatsCache, error) {
	var stats model.StatsCache

	data, err := os.ReadFile(filepath.Join(configDir, "stats-cache.json"))
	if err != nil {
		return stats, err
	}

	err = json.Unmarshal(data, &stats)
	return stats, err
}
