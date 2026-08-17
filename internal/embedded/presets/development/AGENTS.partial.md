## Development work

Coding tasks start by running the `develop` skill. It selects and reads the skills that govern the change before any code is written, and checks the diff against them afterward — the conventions only hold if they are in context before the code exists, not recalled after it.

`go-foundations` is in scope for every Go change and `node-foundations` for every Node change, regardless of what else the task touches. The domain skills cover CLI, backend, frontend, concurrency, CI/CD, README, and Chrome extension work.

`review-code` is the separate multi-agent audit of an existing codebase, not a substitute for `develop` on a change you just made.
