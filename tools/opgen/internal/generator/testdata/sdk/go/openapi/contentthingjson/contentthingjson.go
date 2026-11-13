// Package contentthingjson is a stand-in for a decoder of the JSON content of a resource.
package contentthingjson

import "encoding/json"

// Unmarshal decodes the JSON text into the target.
func Unmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
