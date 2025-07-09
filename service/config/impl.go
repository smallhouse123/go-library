package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	Service = fx.Provide(New)
)

type Impl struct {
	env           string
	configMapPath string
	vaultPath     string
	data          map[string]interface{}
}

type Params struct {
	fx.In

	Env           string `name:"environment"`
	ConfigMapPath string `name:"configMapPath"`
	VaultPath     string `name:"vaultPath"`
	Logger        *zap.Logger
}

func New(p Params) Config {
	config := make(map[string]interface{})

	config, err := LoadAndMergeFiles(p.ConfigMapPath, p.VaultPath)
	if err != nil {
		p.Logger.Sugar().Errorf("error reading and merging files: %v", err)
	}

	return &Impl{
		env:           p.Env,
		configMapPath: p.ConfigMapPath,
		vaultPath:     p.VaultPath,
		data:          config,
	}
}

// LoadAndMergeFiles loads all JSON or YAML files from the given paths and merges them into a single map
func LoadAndMergeFiles(paths ...string) (map[string]interface{}, error) {
	mergedMap := make(map[string]interface{})

	for _, path := range paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && (strings.HasSuffix(filePath, ".json") || strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml")) {
				fileData, err := os.ReadFile(filePath)
				if err != nil {
					return err
				}

				var fileMap map[string]interface{}
				if strings.HasSuffix(filePath, ".json") {
					err = json.Unmarshal(fileData, &fileMap)
					if err != nil {
						return fmt.Errorf("error parsing JSON file %s: %v", filePath, err)
					}
				} else {
					err = yaml.Unmarshal(fileData, &fileMap)
					if err != nil {
						return fmt.Errorf("error parsing YAML file %s: %v", filePath, err)
					}
				}

				mergedMap = mergeMaps(mergedMap, fileMap)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return mergedMap, nil
}

// mergeMaps recursively merges two maps
func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	for key, value := range src {
		if existingValue, ok := dst[key]; ok {
			switch existingValueTyped := existingValue.(type) {
			case map[string]interface{}:
				if valueTyped, ok := value.(map[string]interface{}); ok {
					dst[key] = mergeMaps(existingValueTyped, valueTyped)
				} else {
					dst[key] = value
				}
			default:
				dst[key] = value
			}
		} else {
			dst[key] = value
		}
	}

	return dst
}

func (im *Impl) Get(key string) (interface{}, error) {
	if envValue, exist := im.data[im.env]; exist {
		if value, exist := envValue.(map[string]interface{})[key]; exist {
			return value, nil
		}
		// production vault doesn't have env, look the key directly
		if value, exist := im.data[key]; exist {
			return value, nil
		}
		return nil, fmt.Errorf("key '%s' not found in environment '%s'", key, im.env)
	}
	return nil, fmt.Errorf("environment '%s' not found", im.env)
}
