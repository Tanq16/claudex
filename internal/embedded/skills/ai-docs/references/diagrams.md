# AI-docs diagrams

Architecture and flow diagrams are hand-authored inline `<svg>`, wrapped in `<figure class="diagram">`.
This file is the source of truth for the **house diagram style** — the dense, ReactFlow-looking
architecture panels with color-coded planes and hover tooltips. Follow it whenever a doc needs an
architecture, topology, data-flow, or sequence picture. Do **not** reach for Mermaid, ASCII art, or
image files; they don't match the viewer and don't carry tooltips.

Author diagrams the way you'd author the doc: conclusion-first. The diagram should make the topology
legible at a glance, with detail tucked into per-element tooltips for the reader who hovers.

## The canvas

```html
<figure class="diagram">
  <svg viewBox="0 0 960 460" width="100%" xmlns="http://www.w3.org/2000/svg"
       fill="#cdd6f4" stroke="none" role="img"
       aria-label="One-sentence description of the whole diagram.">
    <rect x="0" y="0" width="960" height="460" fill="#1e1e2e"/>
    <!-- groups go here -->
  </svg>
  <figcaption>Short diagram title</figcaption>
</figure>
```

Non-negotiables:

- **House size is `viewBox="0 0 960 460"`** (a ~2:1 panel). Keep this aspect; if you need more room,
  grow height to 540/620 but keep width 960. The viewer scales the whole thing down to fit, so 960 is
  coordinate space, not pixels — always pair it with `width="100%"` and never hardcode pixel width/height.
- **First child is a full-canvas background rect** filling the whole viewBox: `fill="#1e1e2e"` (base),
  `#181825` (mantle) for a slightly darker panel, or `#11111b` (crust) for the darkest. This solid fill
  is what gives the diagram its framed-panel look — without it the diagram looks unbounded.
- **Set `fill="#cdd6f4"` on the `<svg>`** as the default text color so you don't repeat it on every label.
- One `aria-label` on the svg summarizing the whole picture.

## Every element is a `<g>` with a `<title>`

This is the signature of the style. Each logical thing — a node, a boundary, a connector — is its own
group whose first child is a `<title>` giving a one-line hover tooltip. The tooltip is where detail
lives: name the element and state its role/contract in plain language.

```html
<g><title>Access broker — mints short-lived creds after JWT + VPCE checks</title>
  <rect x="270" y="115" width="190" height="32" rx="4" fill="#313244" stroke="#89b4fa" stroke-width="1"/>
  <text x="365" y="135" text-anchor="middle" font-size="11">access broker (/broker)</text>
</g>
```

Write a real, informative `<title>` for every group. A reader should be able to hover any box or line
and learn what it is and why it's there. Empty or generic titles ("box1") defeat the pattern.

## Nodes

A node is a `rect` + one or more `text` lines inside a `<g><title>`.

- Rect: `rx="6"`–`"8"`, `fill` a surface (`#313244` surface0, `#181825` mantle, `#45475a` surface1,
  `#11111b` crust), `stroke` = the node's **semantic color** (see below), `stroke-width="1.5"`.
- Heading line: `font-size="12"`–`"14"`, `font-weight="600"`, `fill="{semantic color}"`.
- Detail lines: `font-size="9"`–`"11"`, `fill="#cdd6f4"` (primary) or `#a6adc8` / `#bac2de` (muted),
  stacked ~14–18px apart, all `text-anchor="middle"` at the rect's horizontal center.
- Size the rect to fit its longest line; don't let text overflow the box.

```html
<g><title>Engineer VM — single AMI, cloud-init dispatches tier; one per (engineer, tenant)</title>
  <rect x="120" y="280" width="280" height="120" rx="8" fill="#313244" stroke="#a6e3a1" stroke-width="1.5"/>
  <text x="260" y="302" text-anchor="middle" font-size="12" font-weight="600" fill="#a6e3a1">EngineerVM (pet)</text>
  <text x="260" y="320" text-anchor="middle" font-size="10">single AMI, cloud-init tier dispatch</text>
  <text x="260" y="338" text-anchor="middle" font-size="10" fill="#bac2de">t3.large / t3.xlarge / m5.2xlarge</text>
  <text x="260" y="356" text-anchor="middle" font-size="10" fill="#bac2de">gp3 100GB, KMS-CMK per stack</text>
</g>
```

## Semantic colors (use role → color consistently)

Pick the stroke/heading color by what the node *is*, and keep it consistent across the whole diagram
(and ideally across diagrams in the same doc). This color-coding is what makes the panels readable.

| Role | Color | Hex |
|------|-------|-----|
| Client / user / browser (the origin) | lavender | `#b4befe` |
| Control plane / orchestrator boundary | mauve | `#cba6f7` |
| Generic sub-service / endpoints | blue | `#89b4fa` |
| Primary connection plane (WS / gateway) | teal | `#74c7ec` / `#89dceb` |
| Secondary / native path (SSM, escape hatch) | sky-green | `#94e2d5` |
| Compute / VM / the protected workload | green | `#a6e3a1` |
| Network egress / NAT / internet path | peach | `#fab387` |
| Future / planned / deferred | yellow | `#f9e2af` (always with a dashed stroke) |
| Risky / rejected / public surface / rule violation | red / pink | `#f38ba8` / `#eba0ac` |
| Audit / metrics side-flow | pink | `#f5c2e7` |
| Neutral / internet / out-of-scope box | overlay | `#7f849c` / `#9399b2` |

## Boundaries and planes (containers)

Group related nodes inside a labeled boundary box: a tenant/VPC perimeter, a trust zone, a "future"
region. The boundary is a `fill="none"` rect with a **dashed** stroke and a label anchored at its
top-left, inside.

```html
<g><title>Tenant boundary — dedicated engineer-VM VPC, private subnets only</title>
  <rect x="80" y="240" width="600" height="190" rx="10" fill="none" stroke="#f9e2af" stroke-width="1.5" stroke-dasharray="6 4"/>
  <text x="100" y="260" font-size="11" font-weight="600" fill="#f9e2af">tenant boundary (engineer-VM VPC, private subnets)</text>
</g>
```

Place the contained nodes (as their own `<g><title>` groups) at coordinates inside the boundary rect.

## Connectors (edges)

A connector is a `<path>` colored by the **flow** it represents (reuse the semantic color of what
travels the edge). Add a label near it; for diagonals, rotate the label to run along the line.

- Solid for the primary path; `stroke-dasharray="6 3"` (or `"4 3"`) for secondary / escape / optional.
- Arrowheads: prefer one shared `<marker>` in `<defs>` reused via `marker-end`. Inline `<polygon>`
  tips at the path end are equivalent and are what the checkpoints use — either is fine, just be
  consistent within a diagram.

```html
<defs>
  <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
    <path d="M 0 0 L 10 5 L 0 10 z" fill="#9399b2"/>
  </marker>
</defs>

<g><title>Browser path — engineer JWT → ws-authorizer → WS gateway → in-VM agent</title>
  <path d="M 440 54 L 280 180" stroke="#74c7ec" stroke-width="2" fill="none" marker-end="url(#arrow)"/>
  <text x="320" y="120" font-size="10" fill="#74c7ec" transform="rotate(-30 320 120)">WS + JWT</text>
</g>
```

(If you use a single neutral marker color like `#9399b2` for all arrowheads, keep the path itself the
semantic color — the tip color can stay neutral.)

## Layout discipline

Lay the topology out so flow reads top-to-bottom (or left-to-right) without crossing lines where
avoidable:

- **Top:** the origin (client/browser).
- **Upper band:** control plane — one wide mauve rect grouping several blue sub-service rects.
- **Middle:** connection-plane nodes (teal primary, sky-green secondary).
- **Lower:** a dashed tenant/trust boundary containing the compute node, VPC endpoints, NAT.
- **Edge/corner:** internet (neutral) and any **future** nodes (yellow, dashed) as a legend strip.

Keep ~10–20px gutters, align rects to a rough grid, and size each rect to its content. Don't let
labels collide or spill outside their box.

## Worked example (copyable skeleton)

A compact but faithful instance of the style — background panel, default text fill, `g`+`title` on
every element, semantic colors, a dashed boundary, a dashed future node, and a labeled connector.

```html
<figure class="diagram">
  <svg viewBox="0 0 760 380" width="100%" xmlns="http://www.w3.org/2000/svg"
       fill="#cdd6f4" stroke="none" role="img"
       aria-label="Browser hits the control plane, which brokers access into a VM inside the tenant boundary; VM egresses through NAT.">
    <rect x="0" y="0" width="760" height="380" fill="#1e1e2e"/>

    <defs>
      <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
        <path d="M 0 0 L 10 5 L 0 10 z" fill="#9399b2"/>
      </marker>
    </defs>

    <g><title>Engineer browser — the only client surface</title>
      <rect x="300" y="16" width="160" height="40" rx="6" fill="#313244" stroke="#b4befe" stroke-width="1.5"/>
      <text x="380" y="41" text-anchor="middle" font-size="13" font-weight="600" fill="#b4befe">Engineer (browser)</text>
    </g>

    <g><title>Future surface — deferred, shares the lifecycle controller</title>
      <rect x="600" y="16" width="140" height="44" rx="6" fill="none" stroke="#f9e2af" stroke-width="1" stroke-dasharray="4 3"/>
      <text x="670" y="35" text-anchor="middle" font-size="10" fill="#f9e2af">C2 (future)</text>
      <text x="670" y="50" text-anchor="middle" font-size="9" fill="#a6adc8">shared lifecycle</text>
    </g>

    <g><title>Control plane — UI, access broker, lifecycle controller</title>
      <rect x="80" y="86" width="600" height="74" rx="8" fill="#181825" stroke="#cba6f7" stroke-width="1.5"/>
      <text x="380" y="106" text-anchor="middle" font-size="12" font-weight="600" fill="#cba6f7">control plane</text>
      <g><title>UI / API gateway — entry point for the browser</title>
        <rect x="100" y="116" width="170" height="32" rx="4" fill="#313244" stroke="#89b4fa" stroke-width="1"/>
        <text x="185" y="136" text-anchor="middle" font-size="11">UI / API gateway</text>
      </g>
      <g><title>Access broker — mints short-lived creds after auth checks</title>
        <rect x="295" y="116" width="170" height="32" rx="4" fill="#313244" stroke="#89b4fa" stroke-width="1"/>
        <text x="380" y="136" text-anchor="middle" font-size="11">access broker</text>
      </g>
      <g><title>Lifecycle controller — launch / pause / resume / expire sweep</title>
        <rect x="490" y="116" width="170" height="32" rx="4" fill="#313244" stroke="#89b4fa" stroke-width="1"/>
        <text x="575" y="136" text-anchor="middle" font-size="11">lifecycle controller</text>
      </g>
    </g>

    <g><title>Browser → control plane over authenticated WS</title>
      <path d="M 380 56 L 380 86" stroke="#74c7ec" stroke-width="2" fill="none" marker-end="url(#arrow)"/>
      <text x="392" y="76" font-size="10" fill="#74c7ec">WS + JWT</text>
    </g>

    <g><title>Tenant boundary — dedicated VPC, private subnets only</title>
      <rect x="80" y="190" width="470" height="160" rx="10" fill="none" stroke="#f9e2af" stroke-width="1.5" stroke-dasharray="6 4"/>
      <text x="100" y="210" font-size="11" font-weight="600" fill="#f9e2af">tenant boundary (private subnets)</text>

      <g><title>Engineer VM — single AMI, one per (engineer, tenant)</title>
        <rect x="110" y="226" width="240" height="104" rx="8" fill="#313244" stroke="#a6e3a1" stroke-width="1.5"/>
        <text x="230" y="248" text-anchor="middle" font-size="12" font-weight="600" fill="#a6e3a1">Engineer VM (pet)</text>
        <text x="230" y="266" text-anchor="middle" font-size="10">single AMI, cloud-init dispatch</text>
        <text x="230" y="282" text-anchor="middle" font-size="10" fill="#bac2de">in-VM agent (WS) + tmux</text>
        <text x="230" y="298" text-anchor="middle" font-size="10" fill="#bac2de">SSM agent (SSH), per-VM SG</text>
      </g>

      <g><title>NAT gateway — single static EIP, attribution surface</title>
        <rect x="380" y="246" width="150" height="64" rx="6" fill="#181825" stroke="#fab387" stroke-width="1.5"/>
        <text x="455" y="272" text-anchor="middle" font-size="11" font-weight="600" fill="#fab387">NAT + static EIP</text>
        <text x="455" y="290" text-anchor="middle" font-size="9" fill="#a6adc8">no allowlist, AWS pool</text>
      </g>
    </g>

    <g><title>Broker → VM: short-lived creds delivered at launch</title>
      <path d="M 380 160 L 230 226" stroke="#a6e3a1" stroke-width="2" fill="none" stroke-dasharray="6 3" marker-end="url(#arrow)"/>
      <text x="270" y="196" font-size="10" fill="#a6e3a1" transform="rotate(-24 270 196)">creds (in-VM)</text>
    </g>

    <g><title>VM egress through NAT to the internet</title>
      <path d="M 350 278 L 380 278" stroke="#fab387" stroke-width="2" fill="none" marker-end="url(#arrow)"/>
    </g>

    <g><title>Internet — VM egress exits via the NAT EIP</title>
      <rect x="600" y="246" width="120" height="64" rx="6" fill="#11111b" stroke="#9399b2" stroke-width="1"/>
      <text x="660" y="282" text-anchor="middle" font-size="11" fill="#a6adc8">Internet</text>
    </g>
    <g><title>NAT → internet egress</title>
      <path d="M 530 278 L 600 278" stroke="#fab387" stroke-width="2" fill="none" marker-end="url(#arrow)"/>
    </g>
  </svg>
  <figcaption>Engineer-VM access path: browser → broker → VM, egress via NAT</figcaption>
</figure>
```

## Checklist before shipping a diagram

- [ ] Wrapped in `<figure class="diagram">` with a `viewBox="0 0 960 460"`-shaped canvas.
- [ ] Full-canvas background `<rect>` as the first child; `fill="#cdd6f4"` default on the `<svg>`.
- [ ] Every node, boundary, and connector is a `<g>` whose first child is a meaningful `<title>`.
- [ ] Colors assigned by role, consistently (client=lavender, control=mauve, compute=green,
      egress=peach, future=yellow+dashed, risky=red).
- [ ] Boundaries are dashed `fill="none"` rects with a top-left label.
- [ ] Connectors carry the flow's color; secondary paths dashed; arrowheads consistent.
- [ ] Topology reads top-to-bottom; no overflowing text, no needless line crossings.
- [ ] An `aria-label` on the svg and a `figcaption` title under it.
