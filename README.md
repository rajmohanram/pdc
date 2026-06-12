# descforge

Annotate gRPC methods and forge a validated protobuf **`FileDescriptorSet`** for
the Envoy WASM authz filter — in one pure-Go binary, **no `protoc` required**.

`descforge`:
- compiles `.proto` files with **imports + source info always included**,
- **bundles** the well-known (`google/protobuf/*`) and `google/api/*` protos, so
  your protos don't need to vendor them,
- optionally **injects `google.api.http`** annotations on methods that lack one
  (which the wasm/authz filter would otherwise deny in fail-closed mode),
- emits a deterministic, self-validated descriptor.

> Status: **scaffold** — CLI wired, pipeline stubbed. See [DESIGN.md](DESIGN.md).

## Usage (target)

```bash
descforge generate \
  -p ./proto \
  -o uos.pb            # entry files auto-discovered under -p

descforge inspect  -i uos.pb --missing-only
descforge validate -i uos.pb --fail-on-missing
```

## Build

```bash
make build      # ./bin/descforge for the host
make cross      # ./dist/ for linux/{amd64,arm64} + windows/amd64
```

## Bundled protos

The well-known and `google/api` protos live under `protos/` and are embedded at
build time. Refresh/pin them with `make vendor-protos` (version recorded there).
