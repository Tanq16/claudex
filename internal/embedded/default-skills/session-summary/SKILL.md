---
name: session-summary
description: User-invoked session handoff. `/session-summary pause` writes a structured resume file capturing what the session was doing, what changed, what was decided, and what is unfinished; `/session-summary resume` reads that file back and continues from it. Invoked explicitly by name only — never write or read the resume file on your own initiative, and never at the end of an ordinary task.
user-invocable: true
---

# session-summary

**Carry one session's context across a restart. `pause` writes the handoff; `resume` reads it.**

This skill is explicit-invoke only. A session ending, a task finishing, or a context window filling up are not triggers on their own.

## Invocation

```
/session-summary pause     # write the handoff
/session-summary resume    # read it back and continue
```

An invocation with no argument, or an unrecognized one, is a request to say which of the two was meant rather than to guess — writing a handoff when the user wanted to read one destroys the file they were asking for.

## The file

```
.agents/session-resume/resume.md
```

One file, overwritten by every `pause`, because the newest state of the work is the only one worth resuming into. Create the directories if they are missing.

`.agents/` is already excluded from git in a project `claudex apply` has been run in, so the file never reaches a commit. Outside such a project, write it anyway and say in one line that the path is not excluded, so the user can decide before it shows up in `git status`.

## pause

### How long it should be

Up to about 5,000 tokens, and shorter whenever shorter is enough. The whole budget is available and using it is fine; padding to reach it is not.

What sets the length is whether the work is finished:

- **Finished** — the outcome, where it landed, and the decisions worth not re-litigating. Usually a few hundred words. A completed feature does not need its own history retold.
- **Unfinished** — everything a fresh session would otherwise have to rediscover: the exact state of the last thing tried, the commands that worked, the paths in flight, the approaches already ruled out. This is where the budget goes, because rediscovering it costs far more than writing it down.
- **Blocked** — as above, plus what the block is and what would clear it.

### Structure

```markdown
# Session resume — <one line naming the work>

## State
<complete | in progress | blocked> — one sentence.

## What we set out to do
The task as the user framed it, including constraints they stated.

## Where it landed
Branch, PR, files changed, commands that worked. Exact names and paths.

## Decisions
What was decided and why. Include what was considered and rejected, and
anything the user overruled, with their reason.

## Open
Only when the state is not complete: what is unfinished, what has been
tried, what the next step is.

## Specifics to carry forward
Paths, commands, IDs, URLs, flags, versions — verbatim.
```

Drop a section that has nothing real in it. An empty heading reads as a gap in the work rather than as an omission in the summary.

### Rules

Record only what happened. An invented detail is worse than a missing one, because the next session acts on it without knowing to check.

Quote commands, paths, and identifiers verbatim rather than describing them. A resumed session will paste them, and a paraphrased command fails in a way that is slow to diagnose.

Keep rejected approaches and the reason they were rejected. Re-deriving a discarded approach is the most common way a resumed session wastes its first half hour.

Keep decisions the user overruled, in their words. Those are the ones a fresh session is most likely to walk back into.

Write the state of the work, not the story of the session. The order things were tried in rarely matters; what is true now always does.

Say what is uncertain when something is uncertain, rather than presenting a guess as settled. A summary that reads as more resolved than the work actually is sends the next session off confidently in the wrong direction.

### Reporting

After writing, report the path and the state in one line. The file's content does not need repeating back — the user is about to restart, not read it in this session.

## resume

Read `.agents/session-resume/resume.md`, then state in one line what was picked up and what state it was in. Continue the work from there without waiting for confirmation, unless the file says the work is complete, in which case ask what to do next.

When the file is missing, say so and stop. There is nothing to reconstruct, and inferring the previous session from the repository produces a confident summary of the wrong thing.
