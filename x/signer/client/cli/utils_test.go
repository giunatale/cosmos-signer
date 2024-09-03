package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestNewFilterNullKeysJSON(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFilterNullKeysJSON(&buf)

	if writer == nil {
		t.Fatal("Expected NewFilterNullKeysJSON to return a non-nil value")
	}

	if writer.Output != &buf {
		t.Errorf("Expected Output to be %v, got %v", &buf, writer.Output)
	}

	if len(writer.NullKeys) != len(defaultNullKeys) {
		t.Errorf("Expected NullKeys to have %d elements, got %d", len(defaultNullKeys), len(writer.NullKeys))
	}
}

func TestFilterNullKeysJSON_Write(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFilterNullKeysJSON(&buf)

	// Test with valid JSON and null key present
	input := `{"tip": null, "other_key": "value"}`
	expectedOutput := `{"other_key":"value"}`

	n, err := writer.Write([]byte(input))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if n != len(expectedOutput) {
		t.Errorf("Expected %d bytes written, got %d", len(expectedOutput), n)
	}
	if buf.String() != expectedOutput {
		t.Errorf("Expected output %s, got %s", expectedOutput, buf.String())
	}

	// Test with valid JSON and no null key
	buf.Reset()
	input = `{"other_key": "value"}`
	expectedOutput = `{"other_key":"value"}`

	n, err = writer.Write([]byte(input))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if n != len(expectedOutput) {
		t.Errorf("Expected %d bytes written, got %d", len(expectedOutput), n)
	}
	if buf.String() != expectedOutput {
		t.Errorf("Expected output %s, got %s", expectedOutput, buf.String())
	}

	// Test with invalid JSON
	buf.Reset()
	input = `{invalid json}`
	expectedOutput = `{invalid json}`

	n, err = writer.Write([]byte(input))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if n != len(expectedOutput) {
		t.Errorf("Expected %d bytes written, got %d", len(expectedOutput), n)
	}
	if buf.String() != expectedOutput {
		t.Errorf("Expected output %s, got %s", expectedOutput, buf.String())
	}
}

func TestFilterNullKeysJSON_FilterNullJSONKeys(t *testing.T) {
	writer := NewFilterNullKeysJSON(nil)

	// Test with map containing null key
	input := map[string]interface{}{
		"tip":       nil,
		"other_key": "value",
	}
	expectedOutput := map[string]interface{}{
		"other_key": "value",
	}

	result := writer.FilterNullJSONKeys(input)
	if !jsonEqual(result, expectedOutput) {
		t.Errorf("Expected %v, got %v", expectedOutput, result)
	}

	// Test with nested map containing null key
	input = map[string]interface{}{
		"tip": map[string]interface{}{
			"inner_tip": nil,
		},
		"other_key": "value",
	}
	expectedOutput = map[string]interface{}{
		"other_key": "value",
		"tip":       map[string]interface{}{},
	}

	result = writer.FilterNullJSONKeys(input)
	if !jsonEqual(result, expectedOutput) {
		t.Errorf("Expected %v, got %v", expectedOutput, result)
	}

	// Test with slice containing map with null key
	inputSlice := []interface{}{
		map[string]interface{}{
			"tip":       nil,
			"other_key": "value",
		},
	}
	expectedOutputSlice := []interface{}{
		map[string]interface{}{
			"other_key": "value",
		},
	}

	resultSlice := writer.FilterNullJSONKeys(inputSlice)
	if !jsonEqual(resultSlice, expectedOutputSlice) {
		t.Errorf("Expected %v, got %v", expectedOutputSlice, resultSlice)
	}
}

func TestFilterNullJSONKeysFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test JSON to the file
	input := `{"tip": null, "other_key": "value"}`
	err = os.WriteFile(tempFile.Name(), []byte(input), 0644)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	// Call the function to filter null keys
	FilterNullJSONKeysFile(tempFile.Name())

	// Read the file content after filtering
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	// Verify the output is as expected
	expectedOutput := `{"other_key":"value"}`
	if !jsonEqual(content, []byte(expectedOutput)) {
		t.Errorf("Expected output %s, got %s", expectedOutput, content)
	}
}

func TestFilterNullJSONKeysFile_NoOutputDoc(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected no panic, but got one: %v", r)
		}
	}()

	FilterNullJSONKeysFile("")
}

func TestFilterNullKeysJSON_RemoveEmptyMaps(t *testing.T) {
	writer := NewFilterNullKeysJSON(nil)

	// Test case where nested map becomes empty after filtering
	input := map[string]interface{}{
		"outer": map[string]interface{}{
			"inner": map[string]interface{}{
				"key1": nil,
				"key2": nil,
			},
			"other_key": "value",
		},
	}
	expectedOutput := map[string]interface{}{
		"outer": map[string]interface{}{
			"inner":     map[string]interface{}{},
			"other_key": "value",
		},
	}

	result := writer.FilterNullJSONKeys(input)
	if !jsonEqual(result, expectedOutput) {
		t.Errorf("Expected %v, got %v", expectedOutput, result)
	}
}

func TestFilterNullKeysJSON_HandleNullArrayElems(t *testing.T) {
	writer := NewFilterNullKeysJSON(nil)

	// Test case where nested map becomes empty after filtering
	input := []interface{}{
		nil,
		"foo",
		nil,
		1,
	}
	expectedOutput := []interface{}{
		"foo",
		1,
	}

	result := writer.FilterNullJSONKeys(input)
	if !jsonEqual(result, expectedOutput) {
		t.Errorf("Expected %v, got %v", expectedOutput, result)
	}
}

func TestIsNil(t *testing.T) {
	// Test nil values
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"Test nil pointer", (*int)(nil), true},
		{"Test nil string pointer", (*string)(nil), true},
		{"Test nil interface{}", nil, true},
		{"Test nil map", (map[string]string)(nil), true},
		{"Test nil slice", ([]int)(nil), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isNil(tc.value)
			if result != tc.expected {
				t.Errorf("Expected %v for %v, got %v", tc.expected, tc.value, result)
			}
		})
	}

	// Test non-nil values
	tests = []struct {
		name     string
		value    any
		expected bool
	}{
		{"Test non-nil pointer", 1, false},
		{"Test non-nil string", "hello", false},
		{"Test non-nil map", map[string]string{"key": "value"}, false},
		{"Test non-nil slice", []int{1, 2, 3}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isNil(tc.value)
			if result != tc.expected {
				t.Errorf("Expected %v for %v, got %v", tc.expected, tc.value, result)
			}
		})
	}

	// Test invalid value (using interface{})
	t.Run("Test invalid value", func(t *testing.T) {
		var invalidValue interface{}
		result := isNil(invalidValue)
		if !result {
			t.Errorf("Expected true for invalid value, got %v", result)
		}
	})
}

// jsonEqual is a helper function to compare JSON objects
func jsonEqual(a, b interface{}) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(aJSON, bJSON)
}
