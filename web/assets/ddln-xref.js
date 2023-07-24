"use strict";

// Decorates some common patterns with links for some DDLN common strings.
// Decorates only HTML elements of `textClass` with pure text content.
function ddlnXref(textClass) {
  // NOTE: All patterns MUST NOT capture any groups, otherwise `.split(SPLIT_PATTERN)` would
  // populate array with all capture groups in addition to the matches themselves!
  const PATTERNS = [
    {pattern: /\b[TD](?:\d+)\b/, handler: phab => `https://p.daedalean.ai/${phab}`},
    {pattern: /\b(?:[A-Fa-f0-9]{7,64})\b/, handler: tkt => `/ticket/${tkt}`},
    {pattern: /\b(?:exp-|prod-)[A-Za-z0-9-]+\b/, handler: repo => `https://gitea.daedalean.ai/daedalean/${repo}`},
    {pattern: /\bhttp(?:s)?:\/\/\S+/, handler: link => link},
  ];

  // SPLIT_PATTERN is the longest one of the matching patterns.
  const SPLIT_PATTERN = new RegExp(`(${PATTERNS.map(p => `(?:${p.pattern.source})`).join('|')})`);

  function replaceSingle(text) {
    for (let p of PATTERNS) {
      if (text.match(p.pattern)) {
        return `<a href="${text.replace(p.pattern, p.handler)}">${text}</a>`;
      }
    }
    return text;
  }

  // Class: textClass
  // No nested HTML elements.
  $(`.${textClass}:not(:has(*))`).each(function() {
    let html = $(this).text().split(SPLIT_PATTERN).map(replaceSingle).join('');
    $(this).html(html);
  });
}

