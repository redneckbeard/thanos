package facades

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

//go:embed *.json
var builtinFacades embed.FS

// LoadBuiltins reads all embedded JSON facade files and returns a merged
// FacadeConfig. These are the facades that ship with thanos out of the box.
func LoadBuiltins() (types.FacadeConfig, error) {
	merged := types.FacadeConfig{}

	entries, err := builtinFacades.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded facades: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := builtinFacades.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read facade %s: %w", entry.Name(), err)
		}

		var config types.FacadeConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("invalid facade %s: %w", filepath.Base(entry.Name()), err)
		}

		for k, v := range config {
			merged[k] = v
		}
	}

	return merged, nil
}
