# Keeping another repo's agent files out of your clone

`claudex apply` writes its four paths into `.git/info/exclude`, which is local to your clone and never pushed. That only ever affects untracked files, so it cannot keep a `GEMINI.md` or a `.cursor/rules/` that the repository itself tracks out of your working tree.

Sparse-checkout can, on a fresh clone or one you already have:

```bash
git sparse-checkout set --no-cone '/*' '!/GEMINI.md' '!/.cursor/' '!/.github/copilot-instructions.md'
```

Those paths stop being written to your working tree. `git pull` still updates everything else, `git status` stays clean with the ClaudeX layout in place, and `git sparse-checkout disable` puts them back.

Non-cone mode is deprecated by git and is also the only mode that can exclude a single file rather than a whole directory. The `git update-index --skip-worktree` equivalent is not deprecated, but it does not survive a pull that touches the file, so it is not a substitute.

What none of this covers is a path the repository tracks that ClaudeX also writes. Git clears the skip bit the moment a file reappears there, so a repo shipping its own `AGENTS.md` reports it as modified after `apply`.
