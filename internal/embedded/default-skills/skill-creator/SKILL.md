---
name: skill-creator
description: User-invoked skill author. Writes a new skill — frontmatter, body, and any reference files — in the Agent Skills format and sized to fit the per-skill context budget. Invoked explicitly as `/skill-creator <what the skill should cover>`. It is not for editing an existing SKILL.md and does not decide whether an oversized skill should be split.
user-invocable: true
---

# skill-creator

**Write one new skill that conforms to the Agent Skills format and fits the context budget. You are the author: settle the scope, write `SKILL.md`, add references only where they are needed, then report the size.**

This skill is explicit-invoke only (`/skill-creator`). Editing an existing skill is ordinary work and does not go through it.

## Invocation

```
/skill-creator <what the skill should cover>
```

The argument is the subject. Everything else — the name, the structure, the size — is worked out below.

## Where the skill goes

`.agents/skills/<name>/` in the current directory, always. Create the directories if they are missing.

## Workflow

### Step 1: Settle the name and the scope

State in one line what the skill is for and when an agent should reach for it. That line becomes the seed of the `description`, so it is worth getting right before any body text exists.

The name has to satisfy the format: 1–64 characters, lowercase letters, digits and single hyphens, no leading or trailing hyphen, and identical to the directory it lives in. A mismatch between `name` and the directory makes the skill invalid rather than merely untidy.

### Step 2: Write the frontmatter

```yaml
---
name: <matches the directory>
description: <what it does, when to use it, and how it is invoked>
user-invocable: <true when the user calls it by name, false when the model routes to it>
---
```

| Field | Required | Constraint |
|---|---|---|
| `name` | yes | ≤64 chars, lowercase alphanumeric and hyphens, matches the directory |
| `description` | yes | ≤1024 chars, states both what the skill does and when to use it |
| `license` | no | License name, or the name of a bundled license file |
| `compatibility` | no | ≤500 chars; only when the skill needs specific tools or an environment |
| `metadata` | no | String-to-string map for anything outside the spec |
| `allowed-tools` | no | Space-separated pre-approved tools; experimental, support varies |

The `description` is the only part loaded before activation, so it is doing routing work rather than summary work. Name the concrete triggers — the file types, commands, or phrasings that should pull the skill in — because a description that only says what the skill is about leaves the model guessing about when.

A skill the user calls by name says so in the description (`Invoked explicitly as /<name> ...`) and states what should not trigger it. Without that, incidental mentions of the subject activate it.

### Step 3: Write the body

Structure it as: what the skill does, when it applies, the rules, then the examples. Put rules before examples, because a skill that overruns its budget is truncated from the end and the examples are the cheaper half to lose.

Every behavioral rule carries the reason it exists, in the same sentence or the next one. A rule with a reason attached is followed far more often than a bare instruction, and the reason is what lets an agent extend the rule to a case the skill never anticipated.

Write rules as descriptive statements about how the work is done, not as commands. Imperative, shouted instructions get resisted and sometimes read as injected text, while the flat descriptive form of the same rule is simply followed.

One worked example per rule is enough, and it should be the positive case. Describing what success looks like beats enumerating failure, and a prohibition against a mistake the model was not going to make can anchor it toward that mistake.

An example anchors format and never scope. A template, a file shape, or a sample of the output the rule produces is what an example is for; a list of the cases the rule covers is a boundary rather than an example, and it silently becomes the whole of the rule. State the rule as the test it applies, then let the example show what the answer looks like.

Separate facts from rules. A closed set the machine already has, such as the installed language servers or the environment paths, is a fact and belongs in a table where it can be checked and corrected. A criterion is a rule and is written as the test it applies, because an inventory standing in for a criterion becomes the scope of that rule.

Write a prohibition only for a failure that has actually been observed. Rules invented in anticipation of a failure are the ones that get ignored, and they cost budget that a real rule needs.

Leave `MUST`, `CRITICAL`, ALL-CAPS and `!!` out of the body. Emphasis belongs in the `description`, where it does routing work; in the body, emphasis without an adjacent reason reads as anxiety, and an anxious prompt produces a hedging agent.

### Step 4: Add references only where they earn it

Default to a self-contained `SKILL.md`. Reference files are never auto-loaded — the agent has to choose to read one, and often doesn't — so a rule that lives in a reference is a rule that may never be seen.

A reference earns its place when the material is bulk that would otherwise blow the budget (a long template, a table of domain checks) and something other than a prose suggestion forces the read — an explicit step in the workflow, or a sub-agent handed the path directly.

When references are used:

- They live in `references/` beside `SKILL.md` and are cited as `./references/<file>.md`, one level deep. Relative paths work in every client; deep chains do not.
- Every reference file is listed under a `## Start here — required reading` section in the body, marked as read-always or read-before-a-named-sub-task. A reference no one is told to read is dead weight.

```markdown
## Start here — required reading

**Always:**
- `./references/<patterns>.md` — the patterns every task in this skill follows

**When scaffolding a new command:**
- `./references/<templates>.md` — full file templates
```

### Step 5: Check the size and report

The body is loaded in full on activation, and after a compaction each skill re-attaches at its first 5,000 tokens under a shared 25,000-token budget. Past that cap the tail is dropped silently, so a skill that overruns loses content without saying so.

Keep `SKILL.md` under roughly 4,500 tokens (about 18,000 characters, well under 500 lines), which leaves headroom under the re-attach cap.

When the finished draft is over the limit, report the size and what the largest sections are, and stop there. Whether an oversized skill is trimmed or split into two is the user's call — splitting on your own produces skills nobody asked for and a name the user has to live with.

Close by reporting the path written, the token or character count, and the reference files created, if any.

## Worked example

A request for a skill covering the project's database migration process:

```markdown
---
name: db-migrations
description: Writes and reviews database migrations for this project — file naming, the up/down pair, and the backfill rules for a live table. Use when adding a migration, changing a schema, or reviewing a migration in a diff.
user-invocable: false
---

# DB Migrations

**How schema changes are made in this project.**

## When to Use

Use this skill when adding or reviewing a migration, or changing a table definition.

## Rules

Migrations are additive in one deploy and destructive in the next, never both at once,
because a rollback of a single deploy must not lose data that the previous version wrote.

...
```

Written to `.agents/skills/db-migrations/SKILL.md`, 2,100 characters, no references needed.
