// Package anvilogicas is the module root. It exists so that the Wave-1
// api-surface knowledge base can be embedded into the binary: go:embed
// directives cannot reference parent directories, so the embed must live
// here, at the package that physically contains api-surface/.
package anvilogicas

import _ "embed"

// EndpointsYAML is the embedded Wave-1 endpoint registry
// (api-surface/endpoints.yaml). internal/registry parses it; a --registry
// flag or ANVILOGIC_REGISTRY_PATH overrides it at runtime.
//
//go:embed api-surface/endpoints.yaml
var EndpointsYAML []byte
