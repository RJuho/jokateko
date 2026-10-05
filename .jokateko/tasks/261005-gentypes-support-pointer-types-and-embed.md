+++
title = 'gentypes: support pointer types and embedded struct fields'
status = 'done'
priority = 'low'
tags = ['backend', 'build', 'frontend']
summary = 'cmd/gentypes now mirrors encoding/json. Pointers become `T | null`, or `?: T` with omitempty/omitzero. Untagged embedded structs are flattened using the json dominance rules. Unsupported constructs produce warnings on stderr. Golden and json.Marshal tests check the output.'
created_at = '2026-10-05T06:18:28Z'
changed_at = '2026-10-05T06:41:21Z'
+++

## Context
Found during the Go coverage work (task 261005-go-test-coverage-t4-cmd-build-tools-and). `make generate` runs `go run ./cmd/gentypes` to produce `web/src/types/generated.ts` from `internal/model`.

- `goTypeToTS` (`cmd/gentypes/main.go`) has no `*ast.StarExpr` case, so `*string`, `*int` and `*time.Time` fall to the default `unknown`.
- The field loop skips fields that have no name and no json tag (`fieldName == ""` → `continue`). An embedded struct like `type A struct { B }` loses all of B's fields, although encoding/json promotes them into A.
- Neither pattern is in `internal/model` today, so `generated.ts` is correct now. This is about keeping it correct.

## Acceptance Criteria
- [x] `*ast.StarExpr` maps to the element type, and the field is marked optional (`?`), or `| null` if the JSON shows `null`. Pick one convention, document it, and check it against encoding/json output (pointer without omitempty → `null`; with omitempty/omitzero → omitted).
- [x] `*time.Time` → `string` (optional)
- [x] Embedded struct fields: an untagged embedded struct from the same package is flattened into the parent interface (or emitted as `interface A extends B`), matching encoding/json promotion. A tagged embedded field is treated as a normal named field. Embedded pointer structs are handled too.
- [x] Unsupported cases (embedded types from other packages, generics) are reported with a warning on stderr instead of being silently dropped or written as `unknown`
- [x] Table tests in `cmd/gentypes` for each case, plus a golden comparison against `json.Marshal` of a sample struct
- [x] `make generate` produces an unchanged `web/src/types/generated.ts` for the current models; `go test ./...` and the Bun unit tests pass

## Notes

### [2026-10-05 06:37 UTC]

**Implementation (cmd/gentypes):**
- **Convention (mirrors encoding/json, documented in the package doc of `cmd/gentypes/main.go`):**
  - `*T` without omitempty/omitzero → `name: T | null`. With either option → `name?: T`.
  - Pointers inside slices and maps keep the null: `(T | null)[]` and `Record<string, T | null>`.
  - `*time.Time` follows the same rule with `string`. `[]byte` → `string`.
- **Embedded fields:**
  - Untagged embedded same-package structs are flattened in place. This includes unexported and `*Struct` embeds.
  - Fields promoted through an embedded pointer are optional, because a nil pointer omits them.
  - The encoding/json dominance rules apply. The shallowest field wins. At the same depth a tagged field wins. Otherwise the name is dropped and a warning is printed.
  - A tagged embedded field is a normal named field. An embedded exported non-struct type becomes a field named after the type. Recursive embedding stops.
- **Fix found by the golden test:** untagged fields now use the Go name verbatim (`NoTag`), as encoding/json does. Before, the generator lowercased the first letter (`noTag`). Unexported named fields are skipped. Every `internal/model` field has a json tag, so `generated.ts` is byte-identical.
- **Warnings** go to stderr (`gentypes: warning: ...`) for:
  - embedded types from other packages
  - generic struct declarations and instantiations
  - other-package types (except `time.Time`), funcs and chans, which are written as `unknown`
- `_test.go` files in the model dir are now ignored.
- `generate(modelDir, warn)` is split out of `main` so it can be tested.
- **Tests:**
  - table tests `TestGoTypeToTS` and `TestGenerateFields`
  - a golden file `testdata/samplemodel.ts.golden` for the sample package `cmd/gentypes/internal/samplemodel`; refresh it with `go test ./cmd/gentypes -update`
  - `TestGoldenMatchesJSON`, which checks the interfaces against `json.Marshal` of zero and populated values
  - Coverage for gentypes is 94.8%.
- **Known limitation, out of scope:** nil slices and maps also encode as `null`, but they stay non-nullable in TS to keep `generated.ts` unchanged. Named non-struct types such as `Status` are still referenced without being declared in TS.
- **Side finding:** `make generate` also reruns sqlc v1.31.1. That rewrites `internal/store/queries.sql.go` (`GetMilestoneTaskMetricsRow.TargetStartAt`/`TargetEndAt` go from `string` to `interface{}`). I reverted it because it is unrelated to this task. It needs its own task: the query needs a cast, or the committed file needs regenerating.

## Completion Summary
- **Completed At:** 2026-10-05T06:41:21Z

### What Was Done
- Split `generate(modelDir, warn)` out of `main`. `_test.go` files are now ignored.
- `goTypeToTS`:
  - handles `*ast.StarExpr` (`T | null`), `(T | null)[]` and `[]byte` → `string`
  - warns for other-package types, generics, funcs and chans
- `collectFields`, `embeddedField` and `resolveFields` flatten embedded structs as encoding/json does:
  - unexported and pointer embeds are included
  - fields promoted through a pointer are optional
  - tagged embeds and embeds of exported non-struct types become named fields
  - the shallowest field wins, then a tagged field; any other collision is dropped with a warning
  - recursive embedding stops
- Untagged fields keep the Go name verbatim, and unexported fields are skipped.
- Tests:
  - `TestGoTypeToTS`, `TestGenerateFields` and `TestGenerateErrors`
  - a golden file for `cmd/gentypes/internal/samplemodel`
  - `TestGoldenMatchesJSON`, which compares against `json.Marshal` of zero and populated values
- `generated.ts` is unchanged.

### Why / Rationale
The generated TypeScript must describe the JSON the backend actually sends. So the rules copy encoding/json behaviour (null vs omitted, promotion and dominance) rather than an approximation. The golden test against `json.Marshal` catches future drift. Warnings make unsupported model constructs visible instead of silently producing `unknown` or dropping fields.
