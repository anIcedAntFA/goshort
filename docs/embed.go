// Package docs embeds static assets compiled into the binary.
package docs

import _ "embed"

// OpenAPISpec is the OpenAPI 3.x specification for the GoShort HTTP API.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
