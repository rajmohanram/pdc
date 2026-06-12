# protos (bundled)

Vendored `.proto` **source** embedded into the binary at build time so user
projects don't need to supply them:

- `google/protobuf/*.proto` — well-known types (incl. `descriptor.proto`)
- `google/api/http.proto`, `google/api/annotations.proto`

Embedding the source (rather than the linked-in registry) means these files also
carry source info in the generated descriptor.

Populate/refresh with `make vendor-protos` (pin and record the upstream version
here). Resolver precedence: these win over user-supplied `google/*` unless
`--prefer-local-wkt` is passed.
