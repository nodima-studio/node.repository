# Contributing nodes

1. Add a new semantic-version directory under `nodes/<name>/` without changing
   an already published version.
2. Include `package.template.json`, `config.schema.json`, `ui.json`,
   `README.md`, `repository.json`, and a `source/` directory.
3. Build with `./scripts/build` and commit only source and metadata. The script
   disables parent Go workspaces so it always uses the SDK version pinned in
   `go.mod`; `dist/` is ignored.
4. Open a pull request. CI builds twice, compares the archives byte-for-byte,
   validates every package, and checks that `catalog.json` is reproducible.
5. After merge, create the tag `<package-id>@<version>`. The publish workflow
   attaches the exact archive to a GitHub Release and deploys the generated
   catalog to GitHub Pages.

Package IDs use reverse-domain lowercase names. JavaScript packages define
`function process(row, config)` and cannot request capabilities. WASM packages
target WASI Preview 1 and communicate through the versioned Nodima runner ABI.

## Updating the SDK

Update the `github.com/nodima-studio/nodima-sdk` version in `go.mod`, run
`GOWORK=off go mod tidy`, and rebuild all packages twice before merging. Commit
the regenerated `catalog.json`. ABI changes require a new catalog ABI and must
not rewrite existing releases.
