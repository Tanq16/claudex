# Review Domain: Go Backend & Frontend

**Applies to:** Go CLI + Web only
**Skills to load** (paths relative to the plugin root provided in the sub-agent context):
- `../../go-backend/SKILL.md`
- `../../go-frontend/SKILL.md`

---

## Category 8: Backend Architecture (go-backend)

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Package organization | Domain/feature-based by default (`internal/auth/`, `internal/download/`); layered only when 5+ features | Glob `internal/` structure, assess organization |
| Task vs interaction packages | Task packages (internal logic) return errors as-is; Interaction packages (cmd, handlers) wrap errors with context | Read error handling in internal packages vs cmd files |
| HTTP server uses net/http | `net/http` with `http.ServeMux`, NOT third-party routers (gin, chi, echo) | Grep for router imports in server code |
| Server struct pattern | `Server` struct with `port`, `mux` fields; `New()`, `Setup()`, `Run()` methods | Read `internal/server/server.go` |
| Static file serving | `embed.FS` with `fs.Sub()` and `http.StripPrefix` for `/static/` route | Read server setup code |
| Middleware pattern | If middleware exists, uses wrapper function pattern (`func withLogging(next http.HandlerFunc) http.HandlerFunc`) | Grep for middleware patterns |
| Storage interface | If persistence used, `Store` interface with concrete implementations | Grep for storage interface definitions |
| Config struct pattern | Internal packages accept config structs mapped from Cobra flags | Grep for `Config` struct definitions in internal packages |
| No utils import | Internal packages do not import `utils` package (CLI + Web uses `log.Printf` instead) | Grep for `utils` import paths in `internal/` `.go` files |

---

## Category 9: Frontend Assets (go-frontend)

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Directory structure | `internal/server/static/` with `css/`, `fonts/`, `fontawesome/`, `icons/`, `js/` subdirectories | Glob for static asset directories |
| embed.FS directive | `//go:embed static` in server package | Grep for `go:embed` directives |
| Catppuccin Mocha theme | CSS variables defined for all Catppuccin Mocha colors (--rosewater through --crust) | Read HTML files, check for Catppuccin CSS variables |
| Tailwind CSS | Tailwind loaded (via CDN JS or local file), configured with Catppuccin color mappings | Read HTML files for tailwind script and config |
| No custom CSS files | All custom CSS inline in HTML `<style>` blocks; only downloaded CSS in `css/` directory | Glob for `.css` files outside standard downloaded locations |
| Dark theme default | Body uses `bg-base text-text` or equivalent Catppuccin dark background | Read HTML body class |
| Icon links | `<link rel="icon">` tags pointing to `/static/icons/` | Read HTML head section |
| PWA setup (if applicable) | `manifest.json`, `sw.js`, meta tags for theme-color and apple-mobile-web-app | Glob for `manifest.json` and `sw.js` in static directory |
| Icon libraries | Lucide preferred, Font Awesome as fallback, Dev Icons for tech logos | Check which icon libraries are loaded |
| Markdown rendering (if applicable) | Marked.js with custom renderer, Highlight.js for syntax highlighting, `.markdown-body` CSS with Catppuccin heading colors (H1=lavender, H2=mauve, H3=blue), table styling, callout blocks | Read HTML/JS for marked.use(), hljs references, markdown-body styles |
| Mermaid diagrams (if applicable) | `theme: 'base'` with Catppuccin `themeVariables`, `startOnLoad: false`, manual `mermaid.run()`, container CSS on mantle background | Read JS for mermaid.initialize() config, check theme is 'base' not 'dark' |
| Code copy buttons (if applicable) | Copy button on `<pre>` blocks (excluding Mermaid), Lucide copy/check icons, 2-second success state | Read JS for addCopyButtons or equivalent pattern |

---

## Output Format

Report findings in this exact format:

```
## Domain: Go Backend & Frontend

### [PASS] Category Name (source-skill)

All checks passed.

### [ISSUES] Category Name (source-skill)

1. **[Issue title]** (source-skill: section)
   - **Current:** [what the code does now]
   - **Expected:** [what the skill says it should do]
   - **Fix:** [specific action to take]

### [SKIP] Category Name (source-skill)

Not applicable to this project type.
```

End your response with exactly:
```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```

---

## Out of Scope (Hard Boundary)

Do NOT flag any of the following — they are not defined in any loaded skill:

| Category | Specific Examples |
|----------|-------------------|
| Linting & Formatting | No golangci-lint, no gofmt, inconsistent formatting |
| Pre-commit | No pre-commit hooks, no husky |
| Code Quality CI | No lint/format CI steps |
| Documentation beyond README | No godoc, no changelogs, no contributing guide |
| Docker Compose | No docker-compose for development |
| Database | No migrations, no schema files |
| Dependency tooling | No dependabot, no renovate |
| Security scanning | No SAST, no container scanning |
| Code style opinions | Naming conventions not in skills, personal preferences |

**Rule:** If you cannot cite a specific section in a loaded skill for a finding, do not report it.
