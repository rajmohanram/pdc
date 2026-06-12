# pdc

Annotate gRPC methods and forge a validated protobuf **`FileDescriptorSet`** for
the Envoy WASM authz filter — in one pure-Go binary, **no `protoc` required**.

`pdc`:

- compiles `.proto` files with **imports + source info always included**,
- **bundles** the well-known (`google/protobuf/*`) and `google/api/*` protos, so
  your protos don't need to vendor them,
- sets a **standard `google.api.http`** annotation on every method (overwriting
  any existing one) so the wasm/authz filter inspects every gRPC method,
- emits a deterministic, self-validated descriptor.

> Status: **working** — `generate`/`inspect`/`validate` implemented. See [DESIGN.md](DESIGN.md).

## Install

```bash
go install github.com/rajmohanram/pdc@latest   # -> $(go env GOPATH)/bin/pdc
# or build from a checkout:
make build                                      # -> ./bin/pdc
```

Prebuilt linux/windows binaries are attached to each GitHub release.

## Usage

Every command takes either `-p <proto dir>` (compile from source) or
`-i <descriptor.pb>` (read a prebuilt set); `-h` on any subcommand lists its flags.

```bash
# generate a descriptor (auto-discovers all roots, annotates every method)
pdc generate -p ./proto -o uos.pb

# the service graph
pdc tree -p ./proto --by-package        # group services under their package
pdc tree -p ./proto --methods-only      # compact: services + methods only
pdc tree -i uos.pb  --fields --depth 0  # expand message fields (0 = unlimited)

# annotation status / validation
pdc inspect  -p ./proto --missing-only  # methods the filter would deny
pdc validate -i uos.pb  --fail-on-missing
```

`pdc tree` renders the service graph. Add `--fields` to expand message fields
(`--depth N` for nested messages, `0` = unlimited), or `--methods-only` for a
compact services+methods view:

```
demo.Greeter
├─ Hello
│  ├─ request: demo.Req
│  └─ response: demo.Resp
└─ Health
   ├─ request: demo.Req
   └─ response: demo.Resp
```

## Build

```bash
make build      # ./bin/pdc for the host
make cross      # ./dist/ for linux/{amd64,arm64} + windows/amd64
```

## Bundled protos

The `google/api` protos live under `protos/` and are embedded at build time;
the well-known `google/protobuf` types come from the protobuf runtime. Refresh
the bundled `google/api` protos with `make vendor-protos` — they are pinned to a
googleapis commit recorded in `protos/PROTO_VERSION`.

## CI / Release

- **CI** (`.github/workflows/ci.yml`) on every PR to `main`: gofmt, `go vet`,
  golangci-lint, `go test -race`, build.
- **Release** (`.github/workflows/release.yml`) on merge to `main`:
  release-please maintains a release PR from Conventional Commits; merging it
  tags the version and goreleaser publishes the linux/windows binaries.
