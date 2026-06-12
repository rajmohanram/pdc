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

## Usage

```bash
pdc generate \
  -p ./proto \
  -o uos.pb            # entry files auto-discovered under -p

pdc inspect  -i uos.pb --missing-only
pdc validate -i uos.pb --fail-on-missing
pdc tree     -p ./proto --by-package    # services → methods → messages
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
