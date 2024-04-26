package cli

import (
	"encoding/json"
	"io"
	"os"
)

var (
	defaultNullKeys = map[string]struct{}{
		"tip": {},
	}
)

// FilterNullKeysJSON implements the io.Writer interface
type FilterNullKeysJSON struct {
	Output   io.Writer
	NullKeys map[string]struct{}
}

func NewFilterNullKeysJSON(output io.Writer) *FilterNullKeysJSON {
	return &FilterNullKeysJSON{
		Output:   output,
		NullKeys: defaultNullKeys,
	}
}

func (w *FilterNullKeysJSON) Write(p []byte) (n int, err error) {
	var data interface{}
	if err := json.Unmarshal(p, &data); err != nil {
		return 0, err
	}
	filteredData := w.FilterNullJSONKeys(data)
	filteredBytes, err := json.Marshal(filteredData)
	if err != nil {
		return 0, err
	}
	return w.Output.Write(filteredBytes)
}

// FilterNullJSONKeys recursively filters out null values from JSON for specified keys
func (w *FilterNullKeysJSON) FilterNullJSONKeys(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if _, ok := w.NullKeys[key]; ok && val == nil {
				if val == nil {
					delete(v, key)
				} else {
					v[key] = w.FilterNullJSONKeys(val)
				}
			} else {
				v[key] = w.FilterNullJSONKeys(val)
			}
		}
		return v
	case []interface{}:
		for i := range v {
			v[i] = w.FilterNullJSONKeys(v[i])
		}
		return v
	default:
		return v
	}
}

func (w *FilterNullKeysJSON) FilterNullJSONKeysFromFile(outputDoc string) {
	if outputDoc != "" {
		content, err := os.ReadFile(outputDoc)
		if err != nil {
			panic(err)
		}
		var data interface{}
		if err := json.Unmarshal(content, &data); err != nil {
			panic(err)
		}
		filteredData := w.FilterNullJSONKeys(data)
		filteredBytes, err := json.Marshal(filteredData)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(outputDoc, filteredBytes, 0644); err != nil {
			panic(err)
		}
	}
}
