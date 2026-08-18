# Review Domain: Go Server and Frontend

**Applies to:** Go Web Only, Go CLI + Web, and Go Headless API Service. Category 4 applies wherever an embedded frontend exists, and is skipped for a Headless API Service. Category 3 applies only where a CLI client authenticates against an OAuth provider.

**Skills to load, in full, before running any check below:**
- `[SKILLS_DIR]/go-package-architecture/SKILL.md`
- `[SKILLS_DIR]/go-http-server/SKILL.md`
- `[SKILLS_DIR]/go-oauth-cli/SKILL.md` (Category 3 only)
- `[SKILLS_DIR]/go-embedded-frontend/SKILL.md` (Category 4 only)
- `[SKILLS_DIR]/web-markdown-rendering/SKILL.md` (Category 5 only)
- `[SKILLS_DIR]/web-mermaid-diagrams/SKILL.md` (Category 5 only)

The expected pattern for every check lives in those skills. This file states what to look at and how to look at it.

---

## Pre-check

1. **OAuth:** glob for `internal/auth/` and grep for `golang.org/x/oauth2`. Skip Category 3 when absent.
2. **Frontend:** glob for a `static/` tree under `internal/server/`. Skip Category 4 when absent.
3. **Markdown:** grep the static tree for `marked` and `mermaid`. Skip Category 5 when absent.

---

## Category 1: Package Architecture

| Check | How to verify |
|---|---|
| Organization matches the feature count | Glob `internal/`; count features and compare against how the tree is divided |
| Task packages stay quiet | Read error returns in `internal/` packages that are not commands or handlers, looking for wrapping and logging |
| Boundaries add context | Read the error paths in `cmd/` and in HTTP handlers |
| Server layer imports | Grep `internal/server/` for `utils` imports |
| Storage abstraction is earned | Grep for a `Store` interface; count its implementations |
| State file mode | Grep for `os.WriteFile` calls that write a state or cache file and read the mode argument |
| Config structs | Grep `internal/` for direct flag or environment reads that should have been struct fields |
| HTTP client timeouts | Grep for `http.Client` construction and for `http.DefaultClient` |

## Category 2: HTTP Server

| Check | How to verify |
|---|---|
| Router | Grep server code for gin, chi, echo, gorilla/mux, and for `http.ServeMux` |
| Server struct and lifecycle | Read `internal/server/server.go` for the struct and its constructor and methods |
| Setup separate from Run | Read where the mount errors surface relative to where the process starts listening |
| Static serving wiring | Read `Setup` for the embed directive, the sub-filesystem call, and the prefix stripping |
| Index fallback | Read the pattern the index handler is registered on |
| Headless variant | For a Headless API Service, check that the embed and index pieces are absent rather than present and unused |
| Middleware | Grep for wrapper functions and read where they are applied |
| Server logging | Grep `internal/server/` for the logging calls used |

## Category 3: OAuth

| Check | How to verify |
|---|---|
| Modes offered | Read `internal/auth/` for the flows present and `cmd/login.go` for the flags exposed; check they agree |
| No fallback chain | Read the mode dispatch |
| Device support matches the provider | Read the device auth URL against the provider being used |
| State validation | Read the callback handler for the state comparison, and the manual flow for what it does with state |
| Browser open failure | Read whether the browser launcher returns an error and what the caller does with it |
| Token and directory permissions | Grep for the token write and the config directory creation, and read their mode arguments |
| Refresh on load | Read the authenticated-client entry point |
| Output tiers | Read which printer each URL, code, instruction, and terminal message goes through |

## Category 4: Frontend Assets

| Check | How to verify |
|---|---|
| Directory structure | Glob the static tree |
| Assets are not committed | Run `git ls-files` over the asset directories and read `.gitignore` |
| Versions pinned | Grep the Makefile and any HTML for `@latest` or an unpinned range |
| Fonts present and self-hosted | Glob the fonts directory; grep HTML and CSS for `fonts.googleapis.com` and `fonts.gstatic.com` |
| Font format | Glob the fonts directory for anything that is not woff2 |
| Palette declared once | Grep the HTML for the theme declaration and count how many places define the same colors |
| Tailwind configuration form | Read the HTML for how theme values are declared and check it against the Tailwind version loaded |
| Custom CSS placement | Glob for hand-written `.css` files outside the downloaded set |
| Icon files | Glob the icons directory against the sizes the skill lists |
| PWA completeness | Glob for the manifest and the worker; grep the HTML for the link, the meta tags, and the registration; check they are all present or all absent |

## Category 5: Markdown and Diagrams

| Check | How to verify |
|---|---|
| Renderer overrides | Read the `marked.use` call and the token handlers it installs |
| Post-render call order | Read the render function and trace the order of the parse, the copy buttons, the diagram run, and the icon initialization |
| Copy buttons are idempotent | Read the button injection for its guard against running twice |
| Clipboard fallback | Read the copy handler for the non-secure-context path |
| Mermaid theme mode | Read the `mermaid.initialize` config for its theme key |
| Mermaid render trigger | Read the config for automatic startup and grep for a manual run call |
| Markdown styling | Read the stylesheet for the heading colors, the inline versus block code treatment, and the table rules |

---

## Output Format

```
## Domain: Go Server and Frontend

### [PASS] Category Name

All checks passed.

### [ISSUES] Category Name

1. **[Issue title]** [severity] (skill-name: section)
   - **Where:** file:line
   - **Current:** [what the code does now]
   - **Expected:** [what the cited skill section says]
   - **Fix:** [the specific action]

### [SKIP] Category Name

Not applicable: [reason].
```

End with exactly:

```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```
