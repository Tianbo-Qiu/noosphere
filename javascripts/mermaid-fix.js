// Material styles Mermaid via themeCSS, but some selectors target Mermaid 10
// element ids. Append rules that match Mermaid 11 sequence diagrams.
(function () {
  if (typeof mermaid === "undefined") return
  const extra = `
    marker[id$="-arrowhead"] path,
    marker[id$="-crosshead"] path,
    marker[id$="-filled-head"] path {
      fill: var(--md-mermaid-sequence-message-line-color);
      stroke: none;
    }
    .actor-line {
      stroke: var(--md-mermaid-sequence-actor-line-color);
    }
  `
  const initialize = mermaid.initialize.bind(mermaid)
  mermaid.initialize = config =>
    initialize({ ...config, themeCSS: (config.themeCSS || "") + extra })
})()
