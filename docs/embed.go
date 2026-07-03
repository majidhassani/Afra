// Package docs embeds the OpenAPI specification served at /swagger.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
