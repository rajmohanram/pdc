// Package protos holds the bundled google.api protos embedded into the binary so
// user projects don't need to vendor them. The well-known google/protobuf types
// come from the protobuf runtime (protocompile's standard imports); only the
// google/api files are embedded as source here.
package protos

import "embed"

// FS contains the bundled .proto source (google/api/*).
//
//go:embed google
var FS embed.FS
