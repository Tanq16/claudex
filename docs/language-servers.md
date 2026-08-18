# Language servers

The global plugin's `.lsp.json` gives Claude go-to-definition, find-references, and type errors after every edit, in every session, with no build step. ClaudeX wires the servers up and you install the binaries.

| Language | Binary | Install |
|---|---|---|
| Go | `gopls` | `go install golang.org/x/tools/gopls@latest` |
| Python | `pyright-langserver` | `npm install -g pyright` |
| TypeScript / JS | `typescript-language-server` | `npm install -g typescript-language-server typescript@5` |

A server whose binary is missing is skipped and the rest still start, so a partial install is fine. `/plugin` reports a missing one under its Errors tab.

`typescript` is a second package rather than a typo. The server drives `tsserver` and ships without it, resolving it from the project's `node_modules` first and falling back to the global install. The `@5` pin matters: `typescript@latest` is 7.x, which no longer ships a `tsserver` binary, and the language server refuses to start against it.

`claude --bare` is what skips the plugin, and with it the language servers.
