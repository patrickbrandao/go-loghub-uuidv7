# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`go-loghub-uuidv7` is a dependency-free Go library (package `uuidv7`, module `github.com/patrickbrandao/go-loghub-uuidv7`) dedicated **exclusively to UUIDv7** (RFC 9562) with three levels of embedded temporal precision, in both directions: generating the identifier and converting it back into an instant at the same precision. Only the standard library is used.

It was derived on 2026-09-13 from `go-loghub-uuid` v0.6.0, which covers every RFC 9562 version. Everything not directly tied to generating or processing UUIDv7 was removed: versions 1, 2, 3, 4, 5, 6 and 8, the gregorian clock state, clock sequence and node, the namespaces, the `github.com/google/uuid` compatibility layer and its `tests/compare` module, `Max`/`IsMax`, `UUIDs`, `VersionString`/`VariantString` and `IsInvalidLengthError`. Proposals to bring any of them back belong to `go-loghub-uuid`; the scope decision is in `docs/SPEC.md` §11.3.

**Published.** The first release, `v0.0.1`, was tagged on 2026-09-13 at `github.com/patrickbrandao/go-loghub-uuidv7`. Tags are immutable; follow `docs/RELEASE.md` for the next one.

**Language split (enforce when editing):** code *identifiers* (types, functions, variables, constants, package/file names) are in **English**; *comments*, the documentation under `docs/`, and the spec are in **Brazilian Portuguese**; human-facing *string literals* (error messages, console/report labels) are also Portuguese, with proper accents. No emojis anywhere — formal text only. So a typical function has an English name and a Portuguese doc comment; keep that pattern.

## Commands

```bash
go build ./...                                            # compile library
go vet ./...                                              # static analysis
go test ./tests/ -v                                       # functional tests
go test ./tests/ -run TestRoundTripString                # single test by name
go test ./tests/ -run '^$' -bench Benchmark -benchmem     # benchmarks (no tests)
go run ./tests/benchmark-bulk                             # generate 1,000,000 per level
go test ./ -list Example                                  # list the runnable examples (example_test.go)
golangci-lint run ./...                                   # linter, same config as CI (.golangci.yml, needs v2)
```

```bash
go test ./tests/ ./ -short -coverpkg=github.com/patrickbrandao/go-loghub-uuidv7 -coverprofile=cover.out   # cobertura
go tool cover -func=cover.out                                                                              # por função
```

```bash
go test ./... -race -short                                       # detector de corrida (suíte inteira)
go test ./tests/ -short -run 'Allocations|SingleAllocation' -v   # travas de alocação (sem -race)
go test ./... -short -count 3 -shuffle on                        # independência de ordem e de repetição
go test ./tests/ -run '^$' -fuzz FuzzFromString -fuzztime 60s    # fuzzing do parser estrito
go test ./tests/ -run '^$' -fuzz FuzzParse -fuzztime 60s         # fuzzing do parser permissivo
go test ./tests/ -run '^$' -fuzz FuzzNullUUIDJSON -fuzztime 60s  # fuzzing do JSON de NullUUID
go test ./tests/ -run '^$' -fuzz FuzzInstantArithmetic -fuzztime 60s  # fuzzing das fronteiras e da geração por instante
go test ./tests/ -run '^$' -fuzz FuzzTimeReading -fuzztime 60s   # fuzzing das leituras de tempo (16 bytes crus)
```

Note: tests live in `./tests/` and import the library by its **module path** (as an external consumer would), not as an internal package. Run `go test` against `./tests/`, not the repo root, except for the two root test files: `instant_internal_test.go` covers the pure `splitUnixInstant` decomposition, which cannot be driven from outside the package because the clock is not injectable; `example_test.go` (package `uuidv7_test`) holds the `Example` functions, which godoc only associates with the package when they live in its directory. Examples with `// Output:` use only fixed vectors, never the clock or the PRNG. Use `-short` to skip the 1M mass tests.

**Go 1.22 is enforced by CI, not by the local toolchain.** A newer local `go` accepts standard-library APIs added after 1.22 even with `go 1.22` in `go.mod`. A 1.22.12 toolchain is cached in the module cache; check with `GOTOOLCHAIN=local GOPROXY=off $(go env GOMODCACHE)/golang.org/toolchain@v0.0.1-go1.22.12.darwin-arm64/bin/go test ./... -short` before relying on anything recent.

**Linter.** `.golangci.yml` (v2 format) enables `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, `gosec`, `errorlint`, `revive` (exported-comment rule) and `nolintlint`; `misspell` is off because the comments are Portuguese. `gosec` G115 (integer truncation) is excluded globally: byte packing by shift-and-truncate is the whole library, and the hot path must not gain masks to silence it. Every `//nolint` needs a linter name and a reason; the tests that compare a sentinel with `==` on purpose (`ErrInvalidFormat` from `FromString`, `ErrNotV7`, the `ErrEntropySource` panic value) carry `//nolint:errorlint` with that reason. Do not touch `uuid.go`/`conversion.go`/`import.go` to satisfy a lint finding without benchmarking before and after.

**Coverage is 100% of statements, with no documented exception.** Measured with `-coverpkg` because the suite is a separate package; CI (stable only) prints `go tool cover -func`, uploads the HTML as an artifact and fails below 100% (`COVERAGE_MIN` in `ci.yml`). The `crypto/rand` failure branches that the predecessor excluded went away with the versions that had them; the one entropy failure that remains (`NewGeneratorWithReader` on an exhausted reader) is exercised. If `go tool cover` reports an uncovered block, it is a test gap — add the test, never lower the threshold.

**100% coverage is necessary, not sufficient.** On 2026-09-13 a directed mutation campaign (45 one-line defects injected one at a time) found 34 passing the whole suite at 100% coverage; the tests added then catch all 45. When adding or changing behavior, ask which one-line defect would still pass and lock the exact value: a constant entropy source hides which word feeds which field, a round trip hides a parser that accepts too much, a monotonicity check hides where saturation starts, a validity check hides where the bits came from, and a location name hides local time on a `TZ=UTC` host.

**Allocation locks run under `-race` too.** The default generator reads `math/rand/v2`'s runtime source and has no state to rebuild, so the detector does not change the count. CI still measures the locks in a separate step without the detector, since that is the reference measurement. The time readings are locked both accepting and refusing: `ErrNotV7` is a preallocated value and a refusal must never start building an error per call. Do not "fix" a future failure here by relaxing the locks.

**CI.** `.github/workflows/ci.yml` has three jobs. `test` (Linux, Go 1.22 and stable) runs gofmt, vet, build, cross-compile plus vet for `windows/amd64`, `darwin/arm64` and `linux/arm64`, golangci-lint, `-race -short`, the allocation locks, a short `-benchmem` benchmark and the coverage report (the last three tool-dependent steps on stable only). `test-os` runs build, vet and `go test ./... -short` on `windows-latest` and `macos-latest`, but only on pull requests, tags, the weekly schedule and manual dispatch, to save the pricier runners. `deep` (weekly and on dispatch) runs the full suite and 60 s of fuzzing on each of the five targets with `continue-on-error`, uploads `tests/testdata/fuzz/` as the `fuzz-corpus` artifact whenever a campaign fails, and fails the job afterwards; reproduction steps are in `docs/TEST-AND-BENCHMARK.md`. Every `actions/*` step is pinned to an explicit major and must stay on a version whose `action.yml` says `using: node24` — check the action's own repo before bumping, never copy a version number out of a doc. `setup-go` passes `cache: false` in all three jobs: with no dependencies there is no `go.sum` and nothing to cache, and without the flag every run prints a "Dependencies file is not found" warning. Do not silence it by committing an empty `go.sum` instead — that would be a lying file, and the root stays minimal. `go list -m all` at the root must print only this module. Never tag a release without a green `test` job on that commit; the procedure is in `docs/RELEASE.md`. Keep `go.mod` at `go 1.22` unless a newer API is genuinely needed; CI on 1.22 is what enforces that.

**Ordering has no monotonic counter — decided, not merely absent.** A counter was prototyped and rejected on 2026-09-11 (+8.5% serial, 32× worse in parallel); see `docs/SPEC.md` §3.4 for the normative rule and §11.1 for the record. Ordering is chronological *at the level's resolution*, with a **random** tie-break inside the same embedded instant. Generating a UUID is faster than most hosts' clock step, so consecutive UUIDs routinely tie (always, at Level1, whose resolution is the millisecond). Never write an ordering test that counts "regressions in a tight loop against a tolerated threshold" — that measures the host clock, not the library, and fails permanently on microsecond-clock hosts such as macOS. The clock-independent invariant lives in `TestOrderingFollowsEmbeddedTime`; `TestTieRateReport` reports the tie rate as a diagnostic; `TestMonotonicity` sleeps between generations so the embedded instant genuinely advances.

## Architecture

**Production code lives only in the repo root**, alongside `go.mod`, `README.md`, `STARTHERE.md`, `CHANGELOG.md`, `LICENSE`, `SECURITY.md` and `CONTRIBUTING.md` (GitHub reads both from root), this file, the two root test files `instant_internal_test.go` and `example_test.go`, the tool configuration `.golangci.yml` and `.gitignore`, and `.github/` (which GitHub requires at root). Everything else — `docs/`, `docs/SPEC.md`, `tests/` — is intentionally kept out of root so the production surface stays minimal. Preserve this separation: do not add other non-production files to root. `tasks/` is the local scratch folder of the problem-generator skill and is gitignored. `CHANGELOG.md` is the project history; every behavior change gets an entry under "Não publicado" with the file it touched.

Each source file is a distinct concern.

**Core — the hot path. Do not add cost here.**
- `uuid.go` — types (`Level`, `UUID [16]byte`, `Generator`, level constants), generation logic, package-level default generator, `Version`/`Variant`.
- `conversion.go` — `String()` / `FromString()` and their aliases (`BinaryToString`, `StringToBinary`).
- `import.go` — `ErrNotV7`, the `Time` struct and `Import` / `ImportBinary` (extract time fields back out of a UUIDv7; both refuse anything `IsValid` refuses).
- `version7.go` — the generation names: `GenerateV7` (the RFC-standard UUIDv7, i.e. `Generate(Level1)`) and `GenerateV7Level1`/`GenerateV7Level2`/`GenerateV7Level3` (the per-level names for `Generate(LevelN)`; `GenerateV7Level1` is the same function as `GenerateV7`), each on the `Generator` and as a package function. One-line wrappers the compiler inlines; the implementation stays in `uuid.go`, so this file never carries logic, never gains a level parameter (the level is in the name) and never gains a string form. They were kept by an explicit decision when the project was created, for reasons that do not depend on the removed versions (`docs/SPEC.md` §11.3). `TestGenerateV7LevelNamesMatchLevels` regenerates each name's output by instant at the declared level and requires identical bytes, so a name wired to the wrong level fails. That test injects entropy, so it only covers the methods; the package functions use the default generator, which has no injection point by decision (`docs/SPEC.md` §11.2). `TestEveryGenerationFormWritesItsLevel` in `tests/robustness_test.go` proves the level of every generation form — package and method, binary and text, clock and instant, unknown levels, and the four names — by the statistical signature of the free bits (`observedLevel`). Add any new generation form to it.

**Support API.**
- `inspect.go` — `Timestamp` and `TimestampWithLevel`: the instant as `time.Time`, both refusing non-v7 with `false`.
- `construct.go` — building a UUIDv7 from an **explicit instant** instead of
  the clock: `saturatedInstant` (the parameter-side decomposition, which
  saturates at both ends), `packV7` (the shared byte packing) and
  `GenerateAt`/`GenerateAtString`, on the `Generator` and as package
  functions. See `docs/SPEC.md` §3.6.
- `bounds.go` — `MinAt`, `MaxAt`, `RangeAt`: the UUIDv7 that delimits an
  instant, for range queries off the primary key index. It is `packV7`
  with the free bits held constant instead of drawn. See `docs/SPEC.md` §3.5.
- `parse.go` — lenient `Parse`/`ParseBytes` (four formats, generic over `string`/`[]byte` to stay allocation-free), `Validate`, `FromBytes`, `MustParse`, `Must`, and the wrapped format errors.
- `values.go` — `Nil`, `IsZero`, `IsValid`, `Compare`, `URN`, `Bytes`.
- `encoding.go` — `encodeHex`, `AppendTo`/`AppendText`/`AppendBinary`, plus the text and binary marshalers.
- `sql.go` — `Scan`, `Value`, `NullUUID`, `BinaryUUID`, `NullBinaryUUID`.
- `entropy.go` — `NewGeneratorWithReader`, `NewCryptoGenerator`, `ErrEntropySource`.

**Three invariants worth stating explicitly.** First, `String()` and `encodeHex` duplicate the same eight lines on purpose: `String()` is the hot path and must not pay a call. Second, `packV7` in `construct.go` duplicates the byte layout of `Generate` for the same reason; folding the two into one function would put a call or a branch in the hot path. Note the scope: **everything off the hot path shares `packV7`** — bounds and generation-at-instant both go through it — and only `Generate` keeps a private copy. Third, `FromString` returns the bare `ErrInvalidFormat` sentinel, so `err == ErrInvalidFormat` works; only `Parse` returns the wrapped, more specific errors. All three are recorded in `docs/SPEC.md` §11 — do not "fix" any of them as a DRY or consistency finding.

**Settled decisions live in `docs/SPEC.md` §11.** Before proposing a change to the default entropy source, a monotonic counter, an injectable clock or default generator, the three invariants above, the v7-only scope, the non-v7 refusal, the decimal sub-millisecond encoding (not RFC 9562 Method 3), the saturation above 48 bits (not the RFC §6.1 truncation), the reader byte order, or a v1.0.0 tag, read that section: each was decided with a reason and a stated bar for reopening. Record every new design decision there — including refusals — plus a `CHANGELOG.md` entry. An unrecorded refusal comes back at the next audit.

### The three levels (core concept)

A UUIDv7 carries a 48-bit millisecond timestamp in its top bytes; the lower bits (`rand_a` = 12 bits, `rand_b` = 62 bits) are normally random. This library optionally overwrites those random bits with sub-millisecond precision while **always preserving version (7) and variant (RFC `0b10`)**, so output is always a valid UUIDv7:

| Level    | `rand_a` (12 bits) | top 10 bits of `rand_b` | rest of `rand_b` |
|----------|--------------------|-------------------------|------------------|
| `Level1` | random             | random                  | random           |
| `Level2` | microseconds 0–999 | random                  | random           |
| `Level3` | microseconds 0–999 | nanoseconds 0–999       | random (52 bits) |

Because the precision bits sit immediately after the milliseconds, **lexicographic string order stays chronological** across all levels (subject to the tie-break caveat above). Unknown `Level` values fall back to `Level1`. The exact byte layout is documented in the `Generate` doc comment in [uuid.go](uuid.go), in [STARTHERE.md](STARTHERE.md) §4 and in `docs/SPEC.md` §3.1, and it is **written a second time in code** by `packV7` in [construct.go](construct.go), which mirrors `Generate` field by field with the entropy arriving as parameters — keep all four in sync if the bit layout ever changes. That duplication is deliberate and recorded in `docs/SPEC.md` §11.2: the hot path must not pay a call to share it. `Generate` carries no pointer back, so a change there has to be followed into `construct.go` by hand. `TestGenerateAtMatchesGenerateLayout` is what catches the omission: it generates off the clock, reads the embedded instant back and regenerates for it, so the two copies must agree byte for byte.

**Entropy draws per level.** Level2/Level3 put the microseconds in `rand_a`, so they consume **one** 64-bit word; only Level1 (and unknown levels, which behave as Level1) consumes **two**. `tests/robustness_test.go` locks this in — it matters for callers who supply `crypto/rand` through `NewGeneratorWith`.

**Where each word goes is locked too, in `tests/entropy_test.go`.** Counting draws is not enough. With distinct words (a constant source cannot tell `r1` from `r2`) it locks `r1` → `rand_a` and `r2` → `rand_b` at Level1, and the single word → `rand_b` at Level2/Level3, on the clock path and on the instant path; the reader contract of `docs/SPEC.md` §5.3 (big-endian words, `r1` first, short reads completed, 8 bytes per word) with `iotest.OneByteReader`; that `GenerateString`/`GenerateAtString` use the receiver's source; that `NewCryptoGenerator` reads `crypto/rand.Reader`, by swapping that global only during construction; and, statistically, that the default generator draws two independent words. Because of that swap, **no test in `./tests/` may call `t.Parallel`** (recorded in `docs/SPEC.md` §11.2); `TestSuiteHasNoParallelTests` scans the package's test files and fails if one does.

### Reading the instant back

**Time readings refuse non-v7; parsing does not.** `IsValid` is true iff the version nibble is 7 **and** the variant is `0b10` (`Nil` and all-ones are refused). `Import`/`ImportBinary` return exactly `ErrNotV7` with a zeroed `Time`, `Timestamp`/`TimestampWithLevel` return `false` with the zero instant, for everything `IsValid` refuses. `ErrNotV7` does **not** wrap `ErrInvalidFormat`: the text was accepted. `FromString`, `Parse`, `Scan` and the unmarshalers stay format-only on purpose — storing and logging an identifier does not depend on its version; callers who need v7 at the edge call `IsValid`. The rule is normative in `docs/SPEC.md` §4. `TestTimeReadingsRejectNonV7` sweeps all 64 version×variant combinations, so a check on the version alone (the predecessor's `TimestampWithLevel` bug) fails there. The check costs ~0.6 ns on `ImportBinary` (1.7 → 2.2 ns); a single-compare form was measured and was slower, so keep the readable `u[6]>>4 == 7 && u[8]>>6 == 0b10`.

**`ImportBinary` is level-blind; `TimestampWithLevel` discards.** Once the UUID is accepted as v7, `ImportBinary` always reads `rand_a` as microseconds and the top 10 bits of `rand_b` as nanoseconds, treating random bits as if they were precise time — it cannot know which level produced a UUID. `TimestampWithLevel` is the deliberate opposite: it takes the level and **discards** any sub-millisecond field outside 0..999, and at Level3 both fields fall together (an out-of-range `rand_a` proves the top of `rand_b` is noise too). Both policies are normative in `docs/SPEC.md` §7 and recorded in §11.3; do not unify them, and do not "fix" either as an inconsistency.

**`Scan` treats empty text as absent.** `""` and `[]byte{}` set `Nil` without error (and `Valid=false` on `NullUUID`); code reading `DEFAULT ''` columns depends on it.

**JSON wire format.** `MarshalText` makes `encoding/json` write the canonical string (not a 16-number array); `gob` uses `MarshalBinary`. `NullUUID.UnmarshalJSON` delegates to `encoding/json` only when the string contains a backslash; the no-escape path stays allocation-free, and the escape path must keep `Parse`'s specific error (length, brackets), locked by the escape cases in `TestErrorTaxonomy`. When writing a test input that contains a JSON backslash-u escape, check the bytes on disk afterwards: some editing tools decode the escape into the character itself, and the test then silently stops reaching the escape path (this happened on 2026-09-13).

**Golden vectors.** `tests/golden_test.go` holds the multilevel vectors from `docs/SPEC.md` §10 case 12, produced via `MinAt`/`MaxAt`; `tests/timestamp_test.go` reads the RFC 9562 appendix A.6 UUIDv7 example (case 5), the only time-reading vector computed outside this project, and uses the RFC examples of the other versions as refusal inputs (case 7). Never fill a vector with this library's own output: the published multilevel values were computed from the formulas by an independent implementation, and so were the vectors added on 2026-09-13 — the sequence-source and reader vectors in `tests/entropy_test.go` and the top-of-range table in `TestBoundsExactValuesAtTheTopOfTheRange` (`docs/SPEC.md` §10 case 10). The RFC vectors were checked with Python's `uuid` module before being transcribed: version and variant of each, the A.6 timestamp decoded, the A.1 instant matched to A.6, and A.2, A.4 and B.2 recomputed from their name-based inputs (B.1 was left out because its inputs cannot be recomputed).

### Concurrency / performance design

`Generator` is built once at boot and shared across goroutines. `NewGenerator()` draws entropy from `math/rand/v2`'s package functions, which since Go 1.22 read the runtime's generator: one ChaCha8 instance per thread, seeded by the OS — no shared lock, no pool, no state for the library to keep. `NewGeneratorWith(source)` lets callers swap in custom entropy (e.g. full `crypto/rand`); the supplied function **must be concurrency-safe**. Package-level `Generate`/`GenerateString` use an internal default `Generator`. The library has no other mutable global state, which is why the suite must stay green under `-count 3 -shuffle on`.

Hot paths avoid allocations: binary generation does zero allocs; `String()` writes into a fixed `[36]byte` buffer; every time reading is zero-alloc. When modifying generation, conversion or reading, preserve the zero/low-allocation property — `benchmark_test.go`, `alloc_test.go` and `-benchmem` are the guardrails.

## Reference docs

- [STARTHERE.md](STARTHERE.md) — full project map and public API listing.
- [CHANGELOG.md](CHANGELOG.md) — history per version and rejected proposals with reasons. Check it before re-proposing any of those.
- [docs/SPEC.md](docs/SPEC.md) — language-agnostic specification sufficient to reimplement the entire library from scratch (generation, time reading and the non-v7 refusal, parsing, concurrency, serialization). **§11 is the settled-decisions register**: read it before proposing any design change, and add to it whenever a new decision is taken.
- [docs/RELEASE.md](docs/RELEASE.md) — release procedure: prerequisites, tag and `gh release`, post-publication check, and why tags are never moved.
- [SECURITY.md](SECURITY.md) — private vulnerability reporting and the documented threat model; [CONTRIBUTING.md](CONTRIBUTING.md) — the subset of these rules that applies to external contributors.
- [docs/](docs/) — quick use, full use, testing/benchmark guides.
