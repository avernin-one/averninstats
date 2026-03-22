// align-tables.js - measures all score tables within a container,
// finds the widest one, and sets all of them to that width.
// The container uses overflow-x: auto so the tables stay aligned
// even when the window is narrower than the computed width.
"use strict";

export function alignTables(scope) {
  const container = scope ? document.querySelector(scope) : document;
  if (!container) {
    return;
  }

  const tables = [...container.querySelectorAll(".score-table")];
  if (tables.length === 0) {
    return;
  }

  // Reset any previously set width so we measure the natural size.
  for (const table of tables) {
    table.style.width = "";
  }

  // Find the widest table.
  let maxWidth = 0;
  for (const table of tables) {
    const width = table.getBoundingClientRect().width;
    if (width > maxWidth) {
      maxWidth = width;
    }
  }

  if (maxWidth === 0) {
    return;
  }

  // Set all tables to the same width.
  for (const table of tables) {
    table.style.width = `${Math.ceil(maxWidth)}px`;
  }
}
