package provider

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	wsmmodule "github.com/go-go-golems/workspace-manager/pkg/wsmjs/module"
	"github.com/go-go-golems/workspace-manager/pkg/wsmjs/service"
)

const PackageID = "workspace-manager"

type Config struct {
	DefaultJobs int `json:"defaultJobs,omitempty"`
}

var configSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "defaultJobs": {
      "type": "integer",
      "minimum": 0,
      "description": "Default parallel job count used when JavaScript calls omit per-call jobs. Zero keeps the service default."
    }
  },
  "additionalProperties": false
}`)

func Register(registry *providerapi.Registry) error {
	return registry.Package(PackageID, providerapi.Module{
		Name:         wsmmodule.ModuleName,
		DefaultAs:    wsmmodule.ModuleName,
		Description:  "Workspace Manager automation module exposed as require(\"wsm\").",
		ConfigSchema: configSchema,
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			opts, err := optionsFromConfig(ctx.Config)
			if err != nil {
				return nil, fmt.Errorf("workspace-manager provider config: %w", err)
			}
			return wsmmodule.NewLoader(opts), nil
		},
	})
}

func optionsFromConfig(data json.RawMessage) (wsmmodule.Options, error) {
	cfg := Config{}
	if len(data) > 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return wsmmodule.Options{}, err
		}
	}
	if cfg.DefaultJobs < 0 {
		return wsmmodule.Options{}, fmt.Errorf("defaultJobs must be non-negative")
	}
	return wsmmodule.Options{
		ManagerOptions: service.ManagerOptions{DefaultJobs: cfg.DefaultJobs},
	}, nil
}
