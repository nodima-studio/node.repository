# Nodima Node Repository

This repository is the curated public catalog for Nodima Studio node packages. Nodes
may be written in JavaScript or in any language that compiles to WASI Preview 1.
Published archives are immutable, deterministic, and pinned in `catalog.json`
by size and SHA-256.

Source packages live at `nodes/<name>/<version>/`. Run `./scripts/build` to
build every package and regenerate the catalog locally. See
[`CONTRIBUTING.md`](CONTRIBUTING.md) for the package and release workflow. The
runner contracts, Go guest SDK, and package CLI come from
[`nodima-sdk`](https://github.com/nodima-studio/nodima-sdk); this repository
contains only nodes and catalog publishing logic.
