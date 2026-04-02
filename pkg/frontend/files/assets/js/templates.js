// templates.js - loads Mustache template files from assets/templates/
"use strict";

const BASE = "assets/templates/";

const TEMPLATE_FILES = [
  "_footer",
  "_title",
  "_topnav-links",
  "error",
  "loading",
  "page-highscore",
  "page-stats",
  "page-player",
  "page-player-profile",
  "toc-link",
  "toc-list",
  "toc-loading",
];

// Cache: template name -> template string
const _cache = new Map();

// Fetches all template files in parallel and stores them in the cache.
// Must be called once at startup before get() is used.
export async function loadTemplates() {
  await Promise.all(
    TEMPLATE_FILES.map(async (name) => {
      const res = await fetch(`${BASE}${name}.mustache`);

      if (!res.ok) {
        throw new Error(
          `Failed to load template "${name}": ${res.status} - ${res.url}`,
        );
      }

      _cache.set(name, await res.text());
    }),
  );
}

// Returns the template string for the given name.
// Throws if the template was not loaded.
export function get(name) {
  if (!_cache.has(name)) {
    const available = [..._cache.keys()].join(", ");
    throw new Error(`Template "${name}" not loaded. Available: ${available}`);
  }

  return _cache.get(name);
}
