# AI-docs writing style

This governs how a doc reads, not what it contains. Same information as a default markdown answer —
reorganized and trimmed, never dropped. Depth and exhaust move into `Raw material`; they don't get
deleted.

## Prose

- Short sentences. Plain words. No hedging, no preamble, no marketing voice.
- Lead with the conclusion. The "what" first, the "why" after.
- Lists over paragraphs when there are more than two items. Tables for comparisons. Steps for sequences.
- Anchor every claim to a file:line, a number, or a concrete artifact. No "various places", no "many".
- One idea per paragraph. If a paragraph runs past 3 sentences, it's two paragraphs.
- Cut filler: "Note that...", "It's worth mentioning...", "As we discussed...", "In summary...",
  "Going forward...", "It is important to...".
- Don't narrate the process of writing the doc. Don't restate the title in the first sentence.
- Don't write summary / README / CHANGELOG / INDEX docs unless asked.

## What changes vs. a normal answer

The substance of the agent's result does not change. Only the representation does:

- The same findings, structured into the spine (see `./document-template.md`).
- Verbose explanation compressed; the reader gets the answer without commentary.
- Supporting evidence, long logs, full command output, and tangents go into a collapsible
  `Raw material` section at the end — present but out of the way.

If you would have written 6 paragraphs of markdown, write the conclusion + the 2 that matter, and put
the other 4 (or their evidence) under `<details>` in Raw material. Nothing is lost; it's just ranked.
