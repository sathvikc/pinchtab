package schema

import _ "embed"

// ConfigJSON is the PinchTab config JSON Schema bundled with this build.
//
//go:embed config.json
var ConfigJSON []byte
