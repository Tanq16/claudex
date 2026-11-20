# Review Domain: Go Server and Frontend

**Applies to:** Go Web Only, Go CLI + Web, and Go Headless API Service. Category 4 applies wherever an embedded frontend exists, and is skipped for a Headless API Service. Category 3 applies only where a CLI client authenticates against an OAuth provider.

**Skills and references to load, in full, before running any check below:**
- `[SKILLS_DIR]/go-package-architecture/SKILL.md`
- `[SKILLS_DIR]/go-http-server/SKILL.md`
- `[SKILLS_DIR]/go-oauth-cli/SKILL.md` (Category 3 only)
- `[SKILLS_DIR]/go-embedded-frontend/SKILL.md` (Category 4 only)
- `[SKILLS_DIR]/go-embedded-frontend/references/foundations.md` and `kit-core.md` (Category 4 only)
- Each other `[SKILLS_DIR]/go-embedded-frontend/references/` file whose part the frontend builds, as the skill's "Start here" list maps them (Category 4 only)
- `[SKILLS_DIR]/web-markdown-rendering/SKILL.md` (Category 5 only)
- `[SKILLS_DIR]/web-markdown-rendering/references/document-view.md` (Category 5, when the page displays a document)
- `[SKILLS_DIR]/web-mermaid-diagrams/SKILL.md` (Category 5 only)

The expected pattern for every check lives in those files. This file states what to look at and how to look at it.

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
| Server logging | Grep `internal/server/` for the logging calls used; zerolog in every project type, never the standard library logger |
| Event stream | Grep for `text/event-stream`; when present, read the handler for the `Last-Event-ID` and `?after=` cursor, the `reset` path, the heartbeat, flushing through `http.NewResponseController`, and the return on the request context |

## Category 3: OAuth

| Check | How to verify |
|---|---|
| One login flow | Read `internal/auth/` for a single flow that prints the URL and reads the pasted redirect address through `PromptInput`, and `cmd/login.go` for no mode flags |
| Pasted address parsed | Read `extractCode` for `url.Parse` accepting both a full redirect address and a bare code |
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
| Nerd Font is earned | Grep the page for a private-use-area glyph or a Powerline separator when the nerd variant is downloaded |
| Palette declared once | Grep the HTML for the theme declaration and count how many places define the same colors |
| Tailwind configuration form | Read the HTML for how theme values are declared and check it against the Tailwind version loaded |
| Color mapping is inline | Read the `@theme` block mapping the palette for the `inline` keyword |
| Light mode is opt-in | When a light mode exists, read for a `.light` class on `<html>` carrying the full palette and a pre-paint script after the `theme-color` meta |
| Page ground | Read the `body` background and check it against the layering table |
| Layer steps | Read the backgrounds of nested surfaces and check that each is one rung from its parent |
| Fills step forward | Read the background of every pill track, footer, and nested control and check none takes a token behind the layer it sits on |
| Component kit | Grep views outside `ui.js` for hand-built control class strings such as `bg-surface0 px-4 py-2.5` |
| Control states | Read the kit for row hover at `surface0/50`, selected rows at `surface0`, and the `surface1` step for anything inside a `surface0` control |
| Shadows | Grep for `shadow-` utilities and check each sits on a floating layer and takes the `shade` token |
| Hover-revealed actions | Grep for `opacity-0` beside `group-hover:` and check each carries `ui.cls.reveal` |
| Multi-button fields | Read each `ui.field` wrapping a pill track, a stepper, or a tag field for `group: true` |
| Coarse-pointer text size | Grep every input, textarea, and editable control for `pointer-coarse:text-[16px]` |
| Row text on touch | Read the main text of list rows and table cells for `ui.cls.rowText` |
| Form submission | Read each form modal for a `<form novalidate>`, a submit button naming it through `form`, and validation through `ui.invalid` |
| Empty states | Read each `ui.empty` that follows a label or heading for `fill: false` |
| Mono ligatures | Read the `@theme` block for `--font-mono--font-feature-settings` |
| Scroll model | Read the `body` for `h-dvh overflow-hidden`, grep scroll regions for `data-scroll`, and check native scrollbars are hidden |
| Scroll lanes | Read each `data-scroll` region's content: one lining up against its right edge, such as a table, a right-aligned column, or a chat transcript, carries `data-scroll="lane"`, and every other region carries a bare `data-scroll` |
| PDF refit | When a PDF reader exists, read its refit for a `ResizeObserver` on the container rather than a window `resize` listener |
| Text ramp | Read the default body color and the heading colors and check they are not the same token |
| Less text | Read page heads, panel headers, and modals for explanatory subtitles, and status chips for a label that only one state can show |
| Shallow markup | Read view templates for a `div` that carries no class or role |
| Project logo | Glob the icons directory for `logo.svg` and read the header for the `img` showing it |
| Structure is neutral | Grep border and background utilities for accent colors |
| Utilities over CSS | Read every hand-written rule and check it against the named exceptions rather than for a utility that would do it |
| Custom CSS placement | Glob for hand-written `.css` files outside the downloaded set |
| Icon files | Glob the icons directory against the sizes the skill lists |
| PWA completeness | Glob for the manifest and the worker; grep the HTML for the link, the meta tags, and the registration; check they are all present or all absent |

## Category 5: Markdown and Diagrams

| Check | How to verify |
|---|---|
| Renderer overrides | Read the `marked.use` call and the token handlers it installs |
| Post-render call order | Read the render function and trace the order of the sanitized parse, the diagram run, and the icon initialization |
| Output is sanitized | Read the render call for `DOMPurify.sanitize` wrapping `marked.parse` |
| Copy buttons | Read the code renderer for the kit's icon button and a single delegated copy action, and every copy handler for the button passed to `ui.copy` rather than a confirmation of its own |
| Clipboard fallback | Read the copy handler for the non-secure-context path |
| Mermaid theme mode | Read the `mermaid.initialize` config for its theme key |
| Mermaid render trigger | Read the config for automatic startup and grep for a manual run call |
| Diagrams wait for fonts | Read the run call for `document.fonts.ready` |
| Diagrams follow the theme | Read for the source kept in `data-src`, a re-run on `themechange`, and `darkMode` read from the page |
| Markdown styling | Read the stylesheet for the heading colors, the inline versus block code treatment, and the table rules |
| Colors follow the palette | Grep the Markdown stylesheet, the `.hljs` rules, and the Mermaid config for hex literals rather than palette reads, and the page for `github-dark.min.css` |
| Document view | When the page displays a document, read for the reading column, the scroll-following outline, the Contents menu, and the Rendered and Source toggle |

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
