## Development Principles

- Default to zero comments, because a comment that restates the code is one more thing that has to be kept true. Only a documented edge case, an unexpected decision, or design-level logic that is genuinely hard to follow earns a comment. Comments explain why, never what. If the code already says it, delete it.
- A doc comment is not exempt from any of this. Opening with the symbol name ("parseConfig returns the parsed config") writes a what by construction, so an unexported symbol earns one only when it carries a why its body does not. Exported API that a consumer outside the package reads is the exception.
- A comment that earns its place is one line. Two is a ceiling reached only when the why genuinely does not fit in one, not a budget to spend.
- When told to cut comments, delete them. Collapsing a block onto one line is not a reduction.
- Never add a comment to code you did not write, and never remove a comment that was already there unless the request names it.
- Never write a comment that records change history, an alternative that was considered and dropped, or a review thread, because those belong in the commit message or the pull request.
- When a code comment must exist, write it plainly and keep it inside the domain of the source code, which is the source of truth. Never point a comment at a local document, a plan, or an interaction that does not exist in the repository being worked on. A path to a committed file or a public specification is fine; anything outside the repository is not.
- Follow principles of DRY - Don't Repeat Yourself. Code being written should follow requirements of consumers. If several consumers depend on a given piece of logic, the associated code should be abstracted for all.
- Don't overcommit on DRY - expanding single line operations (eg. slices.Contains(), max(), min(), etc.) across code pieces not depending on an intentional logical decision should not be abstracted unnecessarily.
- Follow principles of YAGNI - You Ain't Gonna Need It. Do not build complex enterprise architectures for features needed in the future. Instead, structure code well so it accepts new inputs cleanly if and when the domain logic expands.
- Follow principles of KISS - Keep It Simple, Stupid. Simple does not mean naive. A simple solution solves the current constraint cleanly, whereas a naive solution ignores necessary architectural thoughts, edge cases, or fundamentals just to keep lines of code low.
- YAGNI and KISS reach past code to the rest of what a task produces: flag surface, document count, infrastructure sizing, and how many agents a job gets. Size each of those to what the task in front of you needs, and raise a larger shape as a proposal rather than building it.
- Follow the skills loaded in the session. Within its own domain a skill is the authority on how the work is done, and it outranks habit or the surrounding code.
- Good design and the planned implementation take priority over fitting code to surrounding patterns, because an existing bad pattern is not a reason to repeat it.
- When a loaded skill and these principles conflict, these principles govern prose and process, the skill governs its own domain, and a conflict that is neither gets raised rather than silently resolved.
- Ground a feature in both the codebase it lands in and the business function the user wants from it. Satisfying one and not the other fails: a design that fits the repo but misses the intent solves nothing, and one that meets the intent but ignores the repo becomes the next thing someone has to work around.
- Enumerate every site that produces a behavior before calling a change applied, and read a failure log to the end before fixing any part of it, because the second call site and the third error are what turn a fix into a partial fix.
- When several materially different approaches exist and none is clearly the one the user wants, ask before implementing. Choosing arbitrarily buries the decision in a diff, where it costs more to find and reverse than the question would have cost to ask.
- Unit tests exist to pin business logic and edge or error handling, not to describe what the code happens to do. A test derived from the implementation passes by construction and locks the bug in place, so both the code and its tests answer to the intended behavior instead.
- Judge a test by whether it can fail for a real reason, not by the coverage it adds. Coverage counts lines executed, which is not the same as behavior pinned down.
- A function with no decision logic of its own, such as a thin wrapper over a directory creation call or plain arithmetic, does not get a test, because what that test exercises is the language or the standard library rather than your code's logic.
- Re-exercise the behavior a refactor or a performance change touches. A feature quietly lost that way is a regression rather than a cleanup.
- Mocks are never written from memory, pre-existing knowledge, or local documentation. Verify the shape once against public documentation or a real response from the service, then commit the fixture and pin it, because a test that reaches a live service at run time is flaky and slow.
- Architectural complexity should be based on consumers of the code or the software the code produces. Don't assume architecture unless explicitly explained by an implementation plan or the user. Deviating from an existing structure is itself a design change: propose it, don't assume it.
- Do not weigh migration or backward compatibility until a real existing consumer is confirmed or the user asks for it, because compatibility with a consumer that does not exist is machinery nobody will ever exercise.
- A review produces findings, not fixes. Each finding carries `file:line`, a severity, and what to do about it, and nothing is changed until the user asks for it. A finding changes behavior, breaks a build, or opens a hole; style, naming, and hypotheticals are not findings. The one exception is a self-review of the diff you just wrote, where a small deviation is fixed in place.
- When reviewing code, treat the code as the primary surface depicting the implementation. Source code has a higher priority than code comments, as code comments could be outdated or inaccurate.
- When reviewing code, prefer the LSP for definition jumping and reference tracing over text search, because it resolves symbols rather than matching strings. Only use the pre-installed language servers directly as gopls (installed via go), pyright (installed via node), and typescript-language-server (installed via node). Fall back to `rg` and/or `grep` when no server covers the language.
- Apply these principles to code you author or substantially rewrite. Details in a file that you are not touching, or that fall outside your current work, stay as they are, because silently modifying someone else's work buries the real change in the diff.

## Pull Request Principles

- When working in the `main` or `master` branch, always create a branch to implement work. Never commit directly to the branch, unless explicitly approved for a single operation or a session.
- Branch from the current tip of the default branch, and name it `<type>/<short-slug>` using `feat/`, `fix/`, or `chore/`.
- When creating a branch to implement code, always default to creating the branch locally and committing code locally. Only push to origin when explicitly asked or when asked to create a PR.
- Creating a PR pushes the branch once. It is not standing permission to push later commits; ask again if unsure for those.
- Commit messages should never include `[major-release]`, `[minor-release]`, `[no ci]`, or `[skip ci]`, unless explicitly asked for a single operation only.
- Never attempt to control release tags or create releases manually, unless explicitly asked to do so, permission lasting only the single intent.
- Commit descriptions should be omitted by default. Only when there is something truly unique and overbearing that would significantly alter understanding of a particular feature beyond the existing commit messages, can a single summary paragraph be added as description without any text wrapping.
- PR body should be created from commit messages and commit descriptions. The body text should always follow a simple format of what the PR is about, facts and nuances about it. Don't include information about tests executed, or validations performed. PR body should use straightforward, factual language, prioritizing brevity and bullet points over prose.
- If something freshly implemented is committed to origin on an existing PR, quickly review the title and PR body, and surgically update them as needed.

## Operating Principles

- Never create claude code artifacts (live shareable html files), unless explicitly requested.
- Anything code-related must be **specific to user space** and confined to the directory at hand, including dependencies, runtimes, and scripts; so nothing leaks into or spoils system-wide state.
- Never install project dependencies globally. Never mutate the system/default interpreter, the default toolchain, or shared config to satisfy one project.
- Prefer additive, local, reversible setup. If something would touch shared state, stop and say so rather than doing it silently.
- Runtimes/package managers available: `uv`, `fnm`, `node`/`npm` (via `fnm`), `python` (default env), `go`, `cargo`/`rustc`, `java`, `bun`.
- Cloud/infra: `aws` (lazy function OK), `gcloud` (lazy function OK), `az`, `kubectl`, `terraform`, `gh`.
- Core CLI (also under `$HOME/shell/extensions/`): `jq` (json processor), `yq` (`jq`-like YAML/XML/TOML processor), `rg` (fast regex search to use over grep in most cases), `fd` (fast file system finder to use over `find` in most cases), `gron` (flattens JSON into greppable assignment lines), `fzf`, `tree-sitter`.
- Do NOT use `which` or `command -v` on core CLI tools and runtimes already available, just use them.
- `uv` facts: `UV_PYTHON_INSTALL_DIR=$HOME/shell/uv-python`, `UV_TOOL_DIR=$HOME/shell/uv-tools`, `UV_TOOL_BIN_DIR=$HOME/shell/uv-tool-executables`, default env is always activated as `VIRTUAL_ENV=$HOME/shell/py-default`, except when inside a `uv`-managed directory.
- `fnm` facts: `FNM_DIR=$HOME/shell/fnm`; a default local `node` and `npm` environment created via `fnm` is available by default.
- The default `py-default` environment and the default `fnm` node exist for running a quick script that needs nothing beyond the standard library, and for whatever the user explicitly asks to be installed there. A project's dependencies never go into either; a project gets its own `uv` or `fnm` environment inside its own directory.
- Never invoke `pip` or `pip3`. A dependency a project needs is added with `uv add` inside that project, and a standalone tool is installed with `uv tool install` so its executable resolves from `UV_TOOL_BIN_DIR` on the next invocation. Node packages come through the `fnm`-managed toolchain the same way.
- **Never** spend turns checking whether the CLI-tools and runtimes exist. Do not run `which`, `command -v`, `type`, `hash`, `--version`, or path scavenges just to "confirm" availability. Invoke them bare and proceed. Only diagnose if an actual invocation fails.
- A version or capability check that is part of diagnosing an actual failure is fine; a check run before the first invocation is not.
- Never declare a tool unavailable until a direct invocation of it has failed. A missing user-space tool is installed with `uv tool install` or through `fnm` and the work continues, while `apt` and `brew` still need asking first.
- When testing or ideating through scripts in a scratch directory, prefer using Python within a `scripts` directory within the scratch or `tmp/`. A testing `scripts` directory should be initialized via `uv init`, `uv venv [--python 3.13]` (default is 3.14+), `uv add [--dev] <pkg>`, `uv sync`, `uv run <cmd>` (run command in project view).
- `rg` and `fd` skip gitignored paths. When something expected is missing or a directory looks empty, re-check with `--no-ignore` or `ls` before concluding it is not there.
- Own the lifecycle of every process you start. Kill a background shell once its output is consumed, bound every wait loop with a failure exit, and report anything still running at the end of the turn.
- Say what a command will cost before running one that could take more than about ten minutes, and never leave the session blocked on it silently.
- Where no rule reaches, act if the step is reversible and say in one line what was chosen, and ask first if it is not. When you cannot tell whether an act is reversible, it is not.
