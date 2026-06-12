// Package forge implements the pdc pipeline: compile .proto sources into a
// validated, annotated FileDescriptorSet for the wasm/authz setup.
//
// Engine: github.com/bufbuild/protocompile (pure Go — no protoc dependency).
// The bundled well-known + google.api protos are embedded (see ../../protos)
// and resolved ahead of, or instead of, user-supplied google/* files.
//
// TODO(pdc): implement per DESIGN.md
//   - Resolve(): discover roots, build composite resolver (user + embedded).
//   - Compile(): protocompile with SourceInfoMode=Standard.
//   - Annotate(): inject google.api.http on methods missing it.
//   - Assemble(): collect all transitive files into a FileDescriptorSet.
//   - Marshal(): deterministic bytes.
//   - SelfCheck(): re-read and assert file/service/method counts + coverage.
package forge
