# AI-docs writing style

This governs how a doc reads, not what it contains. Same information as a default markdown answer — reorganized and trimmed, never dropped. Depth and exhaust move into `Raw material`; they don't get deleted.

## Prose

- Short sentences. Plain words. No hedging, no preamble, no marketing voice.
- Lead with the conclusion. The "what" first, the "why" after.
- Lists over paragraphs when there are more than two items. Tables for comparisons. Steps for sequences.
- Anchor every claim to a file:line, a number, or a concrete artifact. No "various places", no "many".
- One idea per paragraph. If a paragraph runs past 3 sentences, it's two paragraphs.
- Cut filler: "Note that...", "It's worth mentioning...", "As we discussed...", "In summary...", "Going forward...", "It is important to...".
- Don't narrate the process of writing the doc. Don't restate the title in the first sentence.
- Don't write summary / README / CHANGELOG / INDEX docs unless asked.

## What changes vs. a normal answer

The substance of the agent's result does not change. Only the representation does:

- The same findings, structured into the spine (see `./document-template.md`).
- Verbose explanation compressed; the reader gets the answer without commentary.
- Supporting evidence, long logs, full command output, and tangents go into a collapsible `Raw material` section at the end — present but out of the way.

If you would have written 6 paragraphs of markdown, write the conclusion + the 2 that matter, and put the other 4 (or their evidence) under `<details>` in Raw material. Nothing is lost; it's just ranked.

## Lean mode (opt-in)

The rules above are the **default ("tight")** density. *Lean mode* is a further squeeze, applied **only when the user asks** ("make this lean", "terse it up", "lean docs") — never automatically. It is caveman-inspired information density, but the house styling and the nothing-dropped rule still hold: lean *ranks and compresses*, it never *deletes*.

What changes in lean mode:

- **No transitional or framing prose.** Cut the connective tissue ("To do this…", "With that in place…", "It's worth noting…"). State the point, move on.
- **Fragments allowed in lists, labels, table cells, and `▲` annotations** — drop articles and copulas where meaning survives (`token expiry check uses < not <=`).
- **Compress Process** to one line or a tight `steps` list; the blow-by-blow goes to Raw material.
- **Push harder under fold.** Anything skippable on a first pass goes into `<details>` in Raw material.

What stays, even in lean mode (the floor — a doc is re-read later, maybe by someone else):

- **Details prose stays grammatical, standalone sentences.** Lean ≠ telegraphic. Never the wenyan/ultra register — a paragraph must still read as English a week from now.
- **Nothing is deleted.** Evidence, caveats, and depth move to Raw material; they don't vanish.
- **Structure, components, diagrams, and anchors remain.** Lean trims words, not the file:line anchors, tables, or diagrams that carry the substance.

Rule of thumb: tight is what you write unasked; lean is tight with the connective prose stripped and more pushed under fold — still a document, just denser.
