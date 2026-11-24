---
name: mutation-testing
description: >-
  Check that the unit tests actually catch bugs, using Gremlins. Use after writing or
  changing ANY test, before calling the test work done. Loading this is unconditional;
  not every change needs an actual run: code that only wires up Cobra flags or Console
  calls doesn't. Triggers: "mutation testing", "are these tests enough", "gremlins",
  changed a *_test.go.
---

# Mutation Testing

Coverage proves a line ran. It doesn't prove a test fails when that line is wrong.
Gremlins changes the code in small ways ("mutants") and reruns the tests. A mutant that
survives is a place where the tests assert too little.

## Run it

Only the lines your branch changed (run from the repo root, keep the `.`):

```bash
GOFLAGS=-count=1 go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0 unleash --diff origin/main .
```

A whole package:

```bash
GOFLAGS=-count=1 go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0 unleash ./internal/changes
```

- Keep `GOFLAGS=-count=1`. Gremlins takes its timeout from a first test run. A cached
  run is too fast, and then every mutant shows as `TIMED OUT`.
- `--diff` with a package path skips every mutant. Use it with `.` only.
- Settings are in `.gremlins.yaml`. It hides killed mutants and turns on the mutant
  types that are off by default.
- It mutates a copy of the repo, never your working tree.
- It is not part of CI. Running it is on the PR author.

## What to run it on

Run it on code that decides something: branching, parsing, validation, diffing, retries,
error handling. Most of `internal/` is this.

Skip wiring: Cobra command and flag setup in `cmd/`, a function that builds one request
and returns the response, type definitions. A mutant there changes data, not behaviour,
so every survivor is noise.

If you can't name a behaviour the function decides, skip it.

## Reading the results

Read the mutants, not the score. Each line gives a status, a mutant type and
`file:line:column`. Open that position to see what was changed.

| Status        | Meaning                                             |
| ------------- | --------------------------------------------------- |
| `LIVED`       | Tests ran the mutated code and still passed         |
| `NOT COVERED` | No test runs this code                              |
| `TIMED OUT`   | Tests hung. Counts as caught, unless all of them do |

| Mutant type             | Change                                         |
| ----------------------- | ---------------------------------------------- |
| `CONDITIONALS_NEGATION` | `==` ↔ `!=`, `<` ↔ `>=`, `>` ↔ `<=`            |
| `CONDITIONALS_BOUNDARY` | `<` ↔ `<=`, `>` ↔ `>=`                         |
| `ARITHMETIC_BASE`       | `+` ↔ `-`, `*` ↔ `/`, `%` → `*`                |
| `INCREMENT_DECREMENT`   | `++` ↔ `--`                                    |
| `INVERT_NEGATIVES`      | `-x` → `+x`                                    |
| `INVERT_LOGICAL`        | `&&` ↔ `\|\|`                                  |
| `INVERT_LOOPCTRL`       | `break` ↔ `continue`                           |
| `INVERT_ASSIGNMENTS`    | `+=` ↔ `-=`, `*=` ↔ `/=`                       |
| `REMOVE_SELF_ASSIGNMENTS` | `+=`, `-=`, `*=`, `&=` and similar → `=`     |
| `INVERT_BWASSIGN`       | `&=` ↔ `\|=`, `<<=` ↔ `>>=`, `^=` → `&=`       |

`INVERT_BITWISE` stays off: it also mutates the address-of `&x`, which never compiles.

For each `LIVED` or `NOT COVERED` mutant in code you wrote or changed:

- **Real gap:** the behaviour differs and a test should have caught it. Add or strengthen
  an assertion until the mutant is killed.
- **Equivalent or irrelevant:** nothing observable changes (a bound nothing reaches, a
  log line, a fast path). Leave it. Don't add an assertion only to kill it.

Don't chase 100%. Equivalent mutants can't be killed.

Gremlins never runs a mutant inside a `case` expression of a `switch`: Go coverage
doesn't count `case` expressions, so it shows as `NOT COVERED`. Reaching the `case`
body isn't enough. Check that tests assert the expected behaviour for both outcomes
of the condition. If unsure, apply the mutation by hand and confirm a test fails.

## Example

```
LIVED CONDITIONALS_NEGATION at changes.go:132:10
```

`internal/changes/changes.go:132`:

```go
err := console.MetadateUpdate(cnx, c, &ds.DS, managedFrom)
if err != nil {
	return err
}
```

With `!=` flipped to `==`, the loop stops after the first successful update and ignores
a failed one. The tests still pass: they update one data structure and the request never
fails. Add a test where the metadata request fails and assert the error is returned.
