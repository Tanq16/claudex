---
name: web-mermaid-diagrams
description: Catppuccin Mocha themed Mermaid diagrams in a browser page. Use when a frontend renders flowcharts, sequence diagrams, gantt charts, pie charts, git graphs, state diagrams, timelines, or mindmaps, or when Mermaid output does not match the page theme. Triggers on mermaid.initialize, mermaid.run, themeVariables, startOnLoad, a mermaid code fence, and diagrams rendering with default colors on a dark page.
user-invocable: false
---

# Web Mermaid Diagrams

**One config object themes every Mermaid diagram type in Catppuccin Mocha.**

Mermaid is vendored and pinned at `mermaid@11.16.1`, served from the page's own origin.

```html
<script src="/static/js/mermaid.min.js"></script>
```

## Rules

`theme: 'base'` is what makes `themeVariables` authoritative. Any built-in theme name ignores most of the overrides and leaves diagrams half-themed, which reads as a rendering bug rather than a config choice.

`startOnLoad: false` with a manual `mermaid.run()` puts rendering under your control. Automatic rendering fires once at load and misses every diagram inserted afterwards, which is all of them on a page that renders Markdown.

```javascript
mermaid.initialize(mermaidConfig);
mermaid.run({ nodes: container.querySelectorAll('.mermaid') });
```

Passing an explicit node list re-renders only the diagrams in the container that changed, rather than rescanning the document.

A `mermaid` code fence becomes a `<div class="mermaid">` during Markdown parsing, and `mermaid.run()` takes over those elements afterwards.

`fontFamily: 'Inter'` matches the body text, so diagram labels do not arrive in a different typeface from the paragraph above them.

## Config

```javascript
const mermaidConfig = {
    startOnLoad: false,
    theme: 'base',
    fontFamily: 'Inter',
    themeVariables: {
        darkMode: true,
        background: '#1e1e2e',
        mainBkg: '#1e1e2e',

        primaryColor: '#313244',
        primaryTextColor: '#cdd6f4',
        primaryBorderColor: '#89b4fa',
        secondaryColor: '#45475a',
        secondaryTextColor: '#cdd6f4',
        secondaryBorderColor: '#7f849c',
        tertiaryColor: '#313244',
        tertiaryTextColor: '#cdd6f4',
        tertiaryBorderColor: '#585b70',

        lineColor: '#89b4fa',
        arrowheadColor: '#89b4fa',
        textColor: '#cdd6f4',
        titleColor: '#cba6f7',
        noteBkgColor: '#45475a',
        noteTextColor: '#f9e2af',
        noteBorderColor: '#585b70',

        // Flowchart
        nodeBkg: '#313244',
        nodeBorder: '#89b4fa',
        clusterBkg: '#181825',
        clusterBorder: '#585b70',
        defaultLinkColor: '#89b4fa',
        edgeLabelBackground: '#313244',
        nodeTextColor: '#cdd6f4',

        // Sequence
        actorBkg: '#313244',
        actorBorder: '#89b4fa',
        actorTextColor: '#cdd6f4',
        actorLineColor: '#585b70',
        signalColor: '#f5c2e7',
        signalTextColor: '#cdd6f4',
        labelBoxBkgColor: '#45475a',
        labelBoxBorderColor: '#585b70',
        labelTextColor: '#cdd6f4',
        loopTextColor: '#f9e2af',
        activationBorderColor: '#cba6f7',
        activationBkgColor: '#45475a',
        sequenceNumberColor: '#1e1e2e',

        // Gantt
        sectionBkgColor: '#181825',
        altSectionBkgColor: '#1e1e2e',
        sectionBkgColor2: '#11111b',
        taskBkgColor: '#89b4fa',
        taskBorderColor: '#b4befe',
        taskTextColor: '#1e1e2e',
        taskTextLightColor: '#1e1e2e',
        taskTextDarkColor: '#cdd6f4',
        taskTextOutsideColor: '#cdd6f4',
        taskTextClickableColor: '#89dceb',
        activeTaskBkgColor: '#cba6f7',
        activeTaskBorderColor: '#f5c2e7',
        doneTaskBkgColor: '#45475a',
        doneTaskBorderColor: '#585b70',
        critBkgColor: '#f38ba8',
        critBorderColor: '#eba0ac',
        gridColor: '#313244',
        todayLineColor: '#f38ba8',

        // Pie, twelve distinct hues
        pie1: '#cba6f7', pie2: '#89b4fa', pie3: '#a6e3a1', pie4: '#f9e2af',
        pie5: '#f38ba8', pie6: '#94e2d5', pie7: '#fab387', pie8: '#89dceb',
        pie9: '#f5c2e7', pie10: '#74c7ec', pie11: '#eba0ac', pie12: '#b4befe',
        pieTitleTextColor: '#cdd6f4',
        pieSectionTextColor: '#1e1e2e',
        pieLegendTextColor: '#cdd6f4',
        pieStrokeColor: '#1e1e2e',
        pieOuterStrokeColor: '#313244',

        // Git graph, eight branch colors
        git0: '#89b4fa', git1: '#cba6f7', git2: '#a6e3a1', git3: '#f9e2af',
        git4: '#f38ba8', git5: '#94e2d5', git6: '#fab387', git7: '#74c7ec',
        gitInv0: '#1e1e2e', gitInv1: '#1e1e2e', gitInv2: '#1e1e2e', gitInv3: '#1e1e2e',
        gitInv4: '#1e1e2e', gitInv5: '#1e1e2e', gitInv6: '#1e1e2e', gitInv7: '#1e1e2e',
        commitLabelColor: '#bac2de',
        commitLabelBackground: '#1e1e2e',
        tagLabelColor: '#1e1e2e',
        tagLabelBackground: '#f9e2af',
        tagLabelBorder: '#fab387',

        // State diagram
        labelBackgroundColor: '#313244',

        // Color scale, used by mindmaps and timelines
        cScale0: '#313244', cScale1: '#89b4fa', cScale2: '#cba6f7',  cScale3: '#a6e3a1',
        cScale4: '#f9e2af', cScale5: '#f38ba8', cScale6: '#94e2d5',  cScale7: '#fab387',
        cScale8: '#89dceb', cScale9: '#f5c2e7', cScale10: '#74c7ec', cScale11: '#b4befe',
    },
};
```

The pie, git, and colour-scale series each cycle through the full Catppuccin accent range rather than shades of one hue, so adjacent slices and branches stay distinguishable.

## Container

```css
.mermaid {
    background-color: #181825;
    padding: 1rem;
    border-radius: 0.5rem;
    margin: 1.5rem 0;
    text-align: center;
    overflow-x: auto;
}
```

`overflow-x: auto` keeps a wide diagram inside its own scroll region instead of stretching the page.

## Gantt Overrides

Gantt charts read several colors from stylesheet rules rather than from `themeVariables`, so they need CSS to finish the job. These are the only diagram type that does.

```css
.mermaid .grid .tick line {
    stroke: #313244 !important;
    stroke-width: 0.5px !important;
    opacity: 0.5 !important;
}
.mermaid .grid .tick text { fill: #a6adc8 !important; }
.mermaid .grid path { stroke: #313244 !important; stroke-width: 0.5px !important; }

.mermaid .section { stroke: none !important; }
.mermaid .section0, .mermaid .section2 { fill: #181825 !important; }
.mermaid .section1, .mermaid .section3 { fill: #1e1e2e !important; }

.mermaid .today { stroke: #f38ba8 !important; stroke-width: 2px !important; }

.mermaid .taskText { fill: #1e1e2e !important; font-size: 0.75em !important; }
.mermaid .taskTextOutsideRight,
.mermaid .taskTextOutsideLeft { fill: #cdd6f4 !important; }

.mermaid .sectionTitle { fill: #cba6f7 !important; }
.mermaid .titleText { fill: #cba6f7 !important; }
```

`!important` is load-bearing here, because Mermaid writes its own inline and generated styles onto these elements after the page stylesheet has been applied.
