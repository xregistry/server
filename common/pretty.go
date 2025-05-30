package common

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// PrettyPrintJSON takes a JSON []byte, prefix, and indent, and returns
// pretty-printed JSON []byte  with the original order of attributes and
// arrays preserved.
// Note, it will NOT do error/syntax checking - maybe we should at some point
func PrettyPrintJSON(data []byte, prefix, indent string) ([]byte, error) {
	// Handle empty object and empty array cases
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 2 {
		if trimmed[0] == '{' && trimmed[1] == '}' {
			return []byte("{}"), nil
		}
		if trimmed[0] == '[' && trimmed[1] == ']' {
			return []byte("[]"), nil
		}
	}

	obj, err := ParseJSONToObject(data)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(obj, prefix, indent)
}

func ParseJSONToObject(data []byte) (interface{}, error) {
	// Parse JSON while preserving key order

	var parseJSON func(decoder *json.Decoder) (interface{}, error)
	parseJSON = func(decoder *json.Decoder) (interface{}, error) {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case json.Delim:
			if t == '{' {
				ordered := &OrderedMap{Values: make(map[string]interface{})}
				for decoder.More() {
					key, err := decoder.Token()
					if err != nil {
						return nil, err
					}
					keyStr, ok := key.(string)
					if !ok {
						return nil, fmt.Errorf("expected string key")
					}
					ordered.Keys = append(ordered.Keys, keyStr)
					value, err := parseJSON(decoder)
					if err != nil {
						return nil, err
					}
					ordered.Values[keyStr] = value
				}
				// Consume closing '}'
				if _, err := decoder.Token(); err != nil {
					return nil, err
				}
				return ordered, nil
			} else if t == '[' {
				// In the case of `[]` make sure we return zero-size not null
				arr := []interface{}{}
				for decoder.More() {
					value, err := parseJSON(decoder)
					if err != nil {
						return nil, err
					}
					arr = append(arr, value)
				}
				// Consume closing ']'
				if _, err := decoder.Token(); err != nil {
					return nil, err
				}
				return arr, nil
			}
			return nil, fmt.Errorf("unexpected delimiter: %v", t)
		default:
			return t, nil // Scalars (string, number, bool, null)
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return parseJSON(decoder)
}

// OrderedMap holds key-value pairs with order preservation
type OrderedMap struct {
	Keys   []string
	Values map[string]interface{}
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve key order
func (o *OrderedMap) UnmarshalJSON(data []byte) error {
	o.Values = make(map[string]interface{})
	o.Keys = nil // Reset Keys to ensure no leftover keys

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if t, err := decoder.Token(); err != nil {
		return err
	} else if t != json.Delim('{') {
		return fmt.Errorf("expected object")
	}

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return err
		}
		keyStr, ok := key.(string)
		if !ok {
			return fmt.Errorf("expected string key")
		}
		o.Keys = append(o.Keys, keyStr)
		var value interface{}
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		o.Values[keyStr] = value
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling to preserve key order
func (o *OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, key := range o.Keys {
		if i > 0 {
			buf.WriteString(",")
		}
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteString(":")
		valueBytes, err := json.Marshal(o.Values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valueBytes)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}
