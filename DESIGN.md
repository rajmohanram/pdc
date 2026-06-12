# pdc — design

A pure-Go CLI that compiles `.proto` files into a self-contained, validated
`FileDescriptorSet` for the Envoy WASM authz filter, optionally injecting
`google.api.http` annotations on methods that lack them.

## Goals
- One static binary per platform (Linux, Windows) — **no `protoc` dependency**.
- Output: a **fully resolved, source-info-bearing** descriptor the wasm/authz
  pipeline can inspect without external files.
- Make unannotated methods inspectable (the filter denies methods with no
  `google.api.http` annotation in fail-closed mode).

## Architecture
- **Compiler engine: `github.com/bufbuild/protocompile`** (pure Go; what `buf`
  uses). No shelling to `protoc`.
- **Always on** (not flags): `--include_imports` semantics + source info.
- **Bundled protos** (`go:embed` of `.proto` *source*): `google/protobuf/*`
  (WKTs) and `google/api/{http,annotations}.proto`. Embedding the *source*
  (not the linked-in registry) means even those files carry source info in the
  output. Resolver precedence: **bundled `google/*` wins**; user-supplied
  `google/*` is ignored unless `--prefer-local-wkt`.

## Commands
| Command | Purpose |
|---------|---------|
| `generate` | compile [+ annotate] → descriptor `.pb` |
| `inspect` | list services/methods + annotation status (`--missing-only`) |
| `validate` | load a `.pb` and assert it's complete/authz-ready |
| `version` | build info |

## Flags (`generate`)
| Flag | Short | Default | Notes |
|------|-------|---------|-------|
| `--proto-path` | `-p` | — | import root(s), repeatable |
| `--proto-files` | `-f` | (auto) | entry files; **if omitted, discover all `*.proto` under `-p`** |
| `--output` | `-o` | — | descriptor output path |
| `--annotate` | | `true` | inject missing http annotations |
| `--http-method` | | `post` | method for synthetic annotations (GET ⇒ no body) |
| `--path-template` | | `/{pkg}.{service}/{method}` | synthetic path; collision-free by construction |
| `--mapping` | | — | YAML/JSON per-method overrides (real paths) |
| `--exclude` | | — | method glob(s) to leave unannotated (internal-only) |
| `--fail-on-missing` | | `false` | CI gate: non-zero if any non-excluded method unannotated |
| `--dry-run` | | `false` | report, write nothing |
| `--prefer-local-wkt` | | `false` | use user `google/*` instead of bundled |

## Annotation strategy (decided)
The WAF does not need a real REST path — only that a consistent path **exists**.
So for each method missing `google.api.http`, inject a **synthetic** one:

- HTTP method: `--http-method` (default `post`; `body: "*"` for non-GET).
- Path: the fully-qualified `<pkg>.<Service>.<Method>` with **`.` replaced by
  `/`**, i.e. `post: "/<pkg>/<Service>/<Method>"`. The `<pkg>` segment is
  **omitted when the service has no package** → `"/<Service>/<Method>"`
  (`<Service>/<Method>` is always available). Unique by construction.
- Override per method via `--mapping`; skip methods matching `--exclude`.

**Apply mode: descriptor injection only** — the option is set on `MethodOptions`
in memory after compile; the `.proto` source is never modified. (Files that gain
an annotation also get `google/api/annotations.proto` added to their
`dependency` list so the descriptor stays self-consistent.)

## generate pipeline
1. Resolve inputs (discover roots if `-f` omitted; drop bundled `google/*` from user roots).
2. Compile (protocompile, `SourceInfoMode=Standard`, composite resolver).
3. Inject `google.api.http` on methods missing it.
4. Re-link / validate the mutated descriptors.
5. Assemble `FileDescriptorSet` = all transitive files (with source info).
6. Deterministic marshal → reproducible bytes.
7. Self-check: re-read, assert file/service/method counts + zero unannotated (minus excludes).
8. Summary + CI exit code.

## Edge cases
- Mixed proto2/proto3 (`descriptor.proto` is proto2).
- Streaming RPCs — annotate (path only); `GET` ⇒ omit body.
- Pre-annotated methods never overwritten.
- Path collisions — synthetic `/<pkg>/<Service>/<Method>` is unique by construction; a `--mapping` collision ⇒ error.
- Package with multiple segments (`a.b`) ⇒ `/a/b/<Service>/<Method>` (all dots become slashes).
- Duplicate filenames across `-p` roots ⇒ ambiguity error.
- Orphan root files not imported by anything (e.g. `telemetry.proto`) — covered by auto-discovery.
- Idempotent: re-runs produce byte-identical output.

## Build / release
Pure Go ⇒ `goreleaser` matrix `linux/{amd64,arm64}`, `windows/amd64`,
`CGO_ENABLED=0`, version via `-ldflags`.

## Decisions (locked)
1. Paths — **synthetic**, `/<pkg>/<Service>/<Method>` with `.`→`/`, package optional. WAF needs no real path.
2. Apply mode — **descriptor injection only**; source untouched.
3. Tool name — **`pdc`**.
