package cli

import (
	"encoding/json"
	"io"
	"os"
)

var (
	nullKeys = []string{"tip"}
)

// FilterNullValJSON implements the io.Writer interface
type FilterNullValJSON struct {
	Output   io.Writer
	NullKeys []string
}

func NewFilterNullValJSON(output io.Writer) *FilterNullValJSON {
	return &FilterNullValJSON{
		Output:   output,
		NullKeys: nullKeys,
	}
}

func (w *FilterNullValJSON) Write(p []byte) (n int, err error) {
	var data interface{}
	if err := json.Unmarshal(p, &data); err != nil {
		return 0, err
	}
	filteredData := FilterNullJSONKeys(data)
	filteredBytes, err := json.Marshal(filteredData)
	if err != nil {
		return 0, err
	}
	return w.Output.Write(filteredBytes)
}

// FilterNullJSONKeys recursively filters out null values from JSON for specified keys
func FilterNullJSONKeys(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if shouldCheckKey(key) {
				if val == nil {
					delete(v, key)
				} else {
					v[key] = FilterNullJSONKeys(val)
				}
			} else {
				v[key] = FilterNullJSONKeys(val) // Recursively process all keys
			}
		}
		return v
	case []interface{}:
		for i := range v {
			v[i] = FilterNullJSONKeys(v[i]) // Recursively process array elements
		}
		return v
	default:
		return v
	}
}

// shouldCheckKey checks if the key is one of the specified keys to check
func shouldCheckKey(key string) bool {
	for _, k := range nullKeys {
		if key == k {
			return true
		}
	}
	return false
}

func FilterNullJSONKeysFromFile(outputDoc string) {
	if outputDoc != "" {
		content, err := os.ReadFile(outputDoc)
		if err != nil {
			panic(err)
		}
		var data interface{}
		if err := json.Unmarshal(content, &data); err != nil {
			panic(err)
		}
		filteredData := FilterNullJSONKeys(data)
		filteredBytes, err := json.Marshal(filteredData)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(outputDoc, filteredBytes, 0644); err != nil {
			panic(err)
		}
	}
}
