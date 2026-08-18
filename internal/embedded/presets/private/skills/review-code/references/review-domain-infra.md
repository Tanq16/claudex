# Review Domain: Infrastructure

**Applies to:** every project type. Each category names the types it covers.

**Skills to load, in full, before running any check below:**
- `[SKILLS_DIR]/go-makefile/SKILL.md` (Go and Chrome extension projects)
- `[SKILLS_DIR]/node-makefile/SKILL.md` (Node projects)
- `[SKILLS_DIR]/github-release-workflow/SKILL.md`
- `[SKILLS_DIR]/dockerize/SKILL.md` (Category 3 only)
- `[SKILLS_DIR]/node-release-artifacts/SKILL.md` (Node projects, Category 4 only)
- `[SKILLS_DIR]/project-readme/SKILL.md`
- `[SKILLS_DIR]/chrome-extension/SKILL.md` (Chrome extension projects only)

The expected pattern for every check lives in those skills. This file states what to look at and how to look at it.

---

## Pre-check

1. **Container:** glob for a `Dockerfile`. Skip Category 3 when absent, unless the project type calls for one, in which case its absence is the finding.
2. **Node artifact:** skip Category 4 for anything that is not a Node project.
3. **Extension:** skip Category 6 for anything without a `manifest.json` carrying `manifest_version`.

---

## Category 1: Makefile

**Applies to:** every project type.

| Check | How to verify |
|---|---|
| Makefile exists | Glob the project root |
| Target set | Read the target list against what the project type calls for |
| No lint or format targets | Read the target list |
| Assets downloaded, not committed | Read the asset target; run `git ls-files` over the directories it writes; read `.gitignore` |
| Every claimed asset is downloaded | Cross-reference what the frontend loads against what the asset target fetches |
| Versions pinned | Grep the Makefile for `latest` and for unpinned ranges in download URLs |
| Fonts | Read the font targets for the three fonts, their sources, and the format they end up in |
| Version calculation defined once | Read the version target; grep the workflow for a reimplementation of the same logic |
| Version injection path | Read the build target's linker flags against where the version variable is declared |
| Docker targets | Read the target list against what the project type calls for |

## Category 2: Release Workflow

**Applies to:** every project type.

| Check | How to verify |
|---|---|
| Workflow exists | Glob `.github/workflows/` |
| Only a release workflow | List the workflow files |
| Trigger | Read the `on:` block |
| Test gate | Read the job graph and check what the release job depends on |
| Version source | Read the version step and check whether it calls the Makefile or reimplements the calculation |
| History depth | Read the checkout step in the version job |
| Draft-then-publish | Read the release creation, the uploads, and the publish step and check their order |
| Cleanup guard | Read the cleanup job's condition |
| Action versions | Read every `uses:` line and check each tag against that action's own releases rather than against memory |
| Asset step present where needed | Read the build and test jobs for the asset target, and compare against whether the project embeds a frontend |
| Matrix | Read the matrix against the platforms the skill names |

## Category 3: Container

| Check | How to verify |
|---|---|
| Stage count | Read the `FROM` lines |
| Cache cleared in the creating layer | Read each package-manager `RUN` |
| Dependency layer before source | Read the order of the copies |
| Base image choice | Read the final `FROM` and trace whether anything copied into it is linked against glibc |
| Runtime packaging | Read how the runtime reaches the final stage |
| Non-root user | Read for user and group creation, the `USER` instruction, and where it sits relative to the last privileged `RUN` |
| Fixed UID and GID | Read the user creation for literal numeric IDs |
| Ownership of copies and volumes | Read the `COPY` instructions and any volume directory creation |
| Entrypoint and command split | Read the `ENTRYPOINT` and `CMD` lines |
| Runtime packages | Read the final stage's installed packages against what the process needs |

## Category 4: Node Release Artifact

**Applies to:** Node Web Only.

| Check | How to verify |
|---|---|
| One artifact shape | Read the Makefile and workflow for whether both a binary and a tarball are produced |
| Shape matches the addon situation | Grep `package.json` and the tree for a native addon, then compare against the shape chosen |
| Bundle layout | Read the bundle script against the layout the skill describes |
| Launcher | Read the launcher for how it resolves its own path, whether it execs, and how it forwards arguments |
| Self-containment proven | Read the smoke test for whether it clears the environment or merely edits the path |
| Addon compiled from source | Read the install steps for the build-from-source flag |
| Native runners | Read the matrix runner assignments against the architectures being built |

## Category 5: README

**Applies to:** every project type.

| Check | How to verify |
|---|---|
| README exists | Glob the project root |
| Header | Read the top of the file |
| Badges | Read the badge URLs against what the project actually publishes |
| Navigation links resolve | Compare each anchor against the headings present |
| Section order | Compare the section order against what the project type calls for |
| Install paths match reality | Compare the documented install methods against what the release workflow publishes |
| Config documented | Compare the documented keys against the loader's defaults |
| Container user documented | When the project ships an image, read the README for the UID and GID and the volume ownership note |
| Extension disclaimer | For an extension, compare the permissions requested against whether the disclaimer is present |

## Category 6: Chrome Extension

**Applies to:** Chrome extensions.

| Check | How to verify |
|---|---|
| Manifest version | Read `manifest.json` |
| Required fields | Read `manifest.json` |
| No empty declarations | Read `manifest.json` for declared scripts and permissions that nothing uses |
| Permissions minimal | Read the permission list and, for each, find the API call that needs it |
| Host permissions scoped | Read the match patterns |
| Icon set | Glob `icons/` against the sizes the skill lists |
| Directory separation | Glob for the popup, content, and background directories |
| Popup theme | Read `popup.css` for the palette and the width constraints |
| Message channel handling | Read each `onMessage` listener for whether it keeps the channel open when it responds asynchronously |
| Service worker state | Grep the worker for module-level variables holding state across events |
| Storage API | Grep for `localStorage` |

---

## Output Format

```
## Domain: Infrastructure

### [PASS] Category Name

All checks passed.

### [ISSUES] Category Name

1. **[Issue title]** (skill-name: section)
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
