# Official CycloneDX 1.6 validation schemas

Unmodified files from [CycloneDX/specification tag 1.6](https://github.com/CycloneDX/specification/tree/55343ba19dee1785acf1ce9191540d5fd7b590db/schema),
commit `55343ba19dee1785acf1ce9191540d5fd7b590db`:
`bom-1.6.schema.json`, `spdx.schema.json`, `jsf-0.82.schema.json` and the
repository's Apache-2.0 `LICENSE`. Exact SHA-256 hashes are checked by
`build/validate_sbom.cjs` before compilation. No schema downloads occur during
validation; the two external references resolve to these local files.

The validator uses build-time Ajv 6.15.0 and ajv-formats-draft2019 1.6.1 for the
schema's international URI/email formats; none enter the served UI. It checks the
official schema plus Flow's complete, unique, closed dependency graph. This
establishes format and graph correctness, not the completeness of supplied
license texts or proof that every source component is linked in a binary.
