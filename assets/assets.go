// Package assets provides embedded asset files for the application.
package assets

import _ "embed"

// DefaultMappingJSON holds the embedded default mapping configuration.
//
//go:embed mapping.json
var DefaultMappingJSON []byte
