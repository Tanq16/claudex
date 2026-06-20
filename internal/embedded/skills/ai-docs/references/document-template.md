# AI-docs document template

The default spine. It's a starting shape, not a cage — add, reorder, or nest subsections as the
content needs. The right-hand TOC is built from `<h2>`/`<h3>`, so use headings freely.

Every file starts with `<!-- curated -->`. Body fragment only (no page tags).

## The spine

1. **Header** — `doc-header` with `h1` and a one-sentence `lede`. Optional `eyebrow` kicker.
2. **Details** — the substance. What the thing is / what was found. The bulk of the doc. Break into
   `<h2>` sections as needed (use tables, cols, facts, steps, code from the vocabulary).
3. **Process** — what was actually done to produce this (steps taken, files touched, how it was
   verified). Short. Use `<ol class="steps">` when it's a sequence.
4. **Raw material** (optional) — a final `<h2>` holding supporting evidence, full logs, long quotes,
   command output, and anything too detailed for the main flow. Wrap each chunk in
   `<details class="expand">` so it's collapsed by default.

Drop sections that don't apply. A pure factual inventory may have no Process; a quick capture may have
no Raw material.

## Skeleton

```html
<!-- curated -->
<header class="doc-header">
  <div class="eyebrow">[kind · topic]</div>
  <h1>[Title]</h1>
  <p class="lede">[One sentence: the headline finding or what this doc is.]</p>
</header>

<h2>Details</h2>
<p>[The substance. Lead with the conclusion.]</p>

<h3>[Subsection]</h3>
<p>[...]</p>

<h2>Process</h2>
<ol class="steps">
  <li><strong>[Action].</strong> [What happened.]</li>
</ol>

<h2>Raw material</h2>
<details class="expand">
  <summary>[What's inside]</summary>
  <pre><code>[logs / full output / long quote]</code></pre>
</details>
```

## Worked example

```html
<!-- curated -->
<header class="doc-header">
  <div class="eyebrow">comparison · queueing</div>
  <h1>SQS vs. Kafka for the ingest path</h1>
  <p class="lede">SQS covers current volume with no ops burden; Kafka only pays off past ~50k msg/s.</p>
</header>

<aside class="tldr">
  <div class="label">TL;DR</div>
  <ul>
    <li>Peak measured load is 3.2k msg/s — 15x below the Kafka break-even.</li>
    <li>SQS FIFO gives ordering per group; we only need per-tenant ordering.</li>
  </ul>
</aside>

<h2>Details</h2>
<div class="cols-2">
  <div class="col"><h4>SQS</h4><ul><li>Zero brokers to run.</li><li>FIFO ordering per group.</li></ul></div>
  <div class="col"><h4>Kafka</h4><ul><li>Replay + retention.</li><li>Needs a cluster + on-call.</li></ul></div>
</div>
<p>At current volume the deciding factor is operational cost, not throughput.</p>

<h2>Process</h2>
<ol class="steps">
  <li><strong>Measured load.</strong> Pulled 30d CloudWatch <code class="kbd">NumberOfMessagesSent</code>.</li>
  <li><strong>Compared.</strong> Mapped both against the measured peak.</li>
</ol>

<h2>Raw material</h2>
<details class="expand">
  <summary>30-day throughput query + output</summary>
  <pre><code>aws cloudwatch get-metric-statistics ...</code></pre>
</details>
```
