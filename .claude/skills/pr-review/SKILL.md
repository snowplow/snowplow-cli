---
name: pr-review
description: >-
  Review a pull request of this repository against the team standard (CLAUDE.md) and
  post one short, plain-language review comment. The Claude Code Review workflow runs it
  when a PR becomes ready for review and when someone comments "@claude review" on the
  PR. Run it locally with /pr-review <PR number> before asking a teammate for a review.
---

# PR Review

Reviews a pull request the way the team does: against the rules this repository already
writes down, in as few words as the findings need.

## Arguments

- `/pr-review 123` or a PR URL: review that PR.
- `/pr-review`: review the PR of the current branch (`gh pr view --json number`).

## Tooling

`gh` is the only GitHub path this skill uses, so it behaves the same in CI and locally.
Locally, check `gh auth status` first; if it fails, stop and ask for `gh auth login`.

## Step 1: Gather the PR

```bash
gh pr view <N> --json title,body,headRefOid,baseRefName,files
gh pr diff <N>
```

Read the PR body once. It tells you what the author intended and which checks they ran.
Do not restate it in the review.

Skip `go.sum`. For `go.mod` the only check is whether a new dependency is needed at all.

## Step 2: Load the standard

Read `CLAUDE.md`. It is the team standard: layout, conventions for commands, config,
logging, exit codes and comments. If the diff overlaps an open spec under `docs/specs/`,
read that spec too.

## Step 3: Check

Read every changed file in full, not only the hunks. In order of importance:

1. Bugs: wrong logic, nil dereferences, errors dropped or swallowed, unclosed response
   bodies or files, a `context.Context` that is not passed on, goroutine leaks and data
   races, a command that breaks on an edge-case flag or input.
2. Team rules: anything `CLAUDE.md` forbids or requires. The common ones: business logic
   in `cmd/` instead of `internal/`, a Console API call that does not go through
   `internal/console`, command code writing to stdout or stderr instead of
   `internal/logging`, `os.Exit` in leaf command logic, a new config key missing from
   `internal/config` or `env.example`, a new Go file without the license header.
3. Tests: changed behaviour with no test, tests that assert nothing.
4. Security: credentials or tokens logged, or written to a file with a mode wider than
   `0600`, user input used in a file path or URL without checks.
5. Readability, only when it hides a bug or breaks a convention above.

CI already runs `go test`, `golangci-lint` and `zizmor`. Do not report what they catch, and do not
suggest performance tweaks unless the code is on a path where they matter.

Verify before you write. Open the file, follow the call, confirm the claim. Do not report
what you did not check. Do not report a pre-existing problem outside the diff unless the
diff makes it worse.

## Step 4: Write the comment

The reader is the author. They need to know what to change, where, and why. Nothing else.

```markdown
Review of `<short head sha>`

<verdict>
Confidence Score: <1 to 5>/5

### Must fix

- `internal/path/file.go:42` <What is wrong>. <Why it matters>. <What to do>. Rule: "<quote>" (`CLAUDE.md`).

### Should fix

- ...

### Nits

- ...

New pushes are not reviewed automatically. Comment `@claude review` to review the latest commit.
```

The verdict is one line: `Nothing to report.`, `No blocking issues.` or
`<N> issues to fix before merge.`

The score is one line, `Confidence Score: <N>/5`, and says how safe the PR is to merge as
it stands. Count the findings you are about to post, then apply this table. Nits never
change the score.

| Findings                                           | Score |
| -------------------------------------------------- | ----- |
| No Must fix and no Should fix                      | 5/5   |
| No Must fix, one or more Should fix                | 4/5   |
| One or two Must fix                                | 3/5   |
| Three to five Must fix                             | 2/5   |
| Six or more Must fix, or any Must fix for security | 1/5   |

Findings you left out because of the 10-finding cap still count. The score follows the
findings, never the other way round: do not add or drop a finding to reach a score.

Severity:

- Must fix: a bug, a security issue, a missing test for changed behaviour, or a violation
  of a rule in `CLAUDE.md`.
- Should fix: a likely bug you could not confirm, or a convention `CLAUDE.md` recommends
  but does not require.
- Nits: small readability points. Three at most.

Style rules, all binding:

- One finding per bullet: location, problem, fix. Two or three sentences, each under 20
  words.
- Plain words. If a new teammate would need to look a word up, use a simpler one.
- Quote the rule when a finding comes from `CLAUDE.md`, so the author sees a team
  decision, not an opinion.
- Group repeats: "Same in `a.go:10` and `b.go:22`."
- At most 10 findings. Keep the most important and say how many you left out.
- Drop empty sections. With no findings, post the first line, the verdict, the score and
  the last line only.
- No praise, no summary of the PR, no restating the diff, no "overall", no closing
  remarks. Ask the author a question only when a finding needs the answer.
- No em dashes, no en dashes, no arrows. Use a comma or a full stop.
- No emojis, no bold, no nested bullets, no tables, no diagrams, no headings other than
  the three above.
- Code only for identifiers and paths, or for a fix that is shorter as code than as words.
  A code block stays under eight lines.
- Under 300 words in total. Blocking findings may need more words. Filler never does.

Read the comment back once. Remove every sentence that does not change what the author
does next.

## Step 5: Post

One comment, posted once:

```bash
gh pr comment <N> --body-file - <<'REVIEW'
<the comment>
REVIEW
```

Never split the review across comments, never post inline review comments and never use
`gh pr review`. The single comment is the whole review.

## When this runs in CI

The `Claude Code Review` workflow (`.github/workflows/claude-code-review.yml`) runs this
skill:

- once, when a PR is opened ready for review or is marked "Ready for review". Draft PRs
  and PRs from forks are skipped. Pushes after that do not start a new review.
- again only when a repository owner, member or collaborator writes `@claude review` in a
  top-level PR comment.

Every other `@claude` comment goes to the general `Claude Code` workflow
(`.github/workflows/claude.yml`), which does not run this skill.
