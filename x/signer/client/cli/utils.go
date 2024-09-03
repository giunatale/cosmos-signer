package cli

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
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
		return w.Output.Write(p)
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
			// Recursively filter the value
			v[key] = w.FilterNullJSONKeys(val)

			// Remove the key if it's in NullKeys and the value is nil
			if _, ok := w.NullKeys[key]; ok && v[key] == nil {
				delete(v, key)
			}

			// Remove the key if contains only nil values after filtering
			if nestedMap, isMap := v[key].(map[string]interface{}); isMap {
				for nestedKey, nestedVal := range nestedMap {
					if nestedVal == nil {
						delete(nestedMap, nestedKey)
					}
				}
			}
		}

		return v

	case []interface{}:
		var tmpRet interface{}
		j := 0
		for i := range v {
			tmpRet = w.FilterNullJSONKeys(v[i])
			if !isNil(tmpRet) {
				v[j] = tmpRet
				j++
			}
		}

		// avoid memory leak
		for k := j; k < len(v); k++ {
			v[k] = nil
		}

		v = v[:j]
		return v

	default:
		return v
	}
}

func FilterNullJSONKeysFile(outputDoc string) {
	w := NewFilterNullKeysJSON(nil)
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

func isNil(val any) bool {
	if val == nil {
		return true
	}

	v := reflect.ValueOf(val)
	k := v.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()

	case reflect.Invalid:
		return true
	}

	return false
}
