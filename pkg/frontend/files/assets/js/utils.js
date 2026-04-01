// utils.js - shared helpers
"use strict";

// ---------------------------------------------------------------------------
// Type checks
// ---------------------------------------------------------------------------

// Returns true if the stat key holds a distance value in centimetres.
export function isDistance(key) {
  return key.endsWith("_one_cm");
}

// Returns true if the stat key holds a value in game ticks (20 ticks/sec).
export function isTicks(key) {
  return key.startsWith("time_") || key.endsWith("_time");
}

// ---------------------------------------------------------------------------
// Value formatting
// ---------------------------------------------------------------------------

// Formats a raw Minecraft stat value into a human-readable string.
// Distances are stored as centimetres, time values as ticks.
export function formatValue(key, raw) {
  if (isDistance(key)) {
    return formatDistance(raw);
  }

  if (isTicks(key)) {
    return formatTicks(raw);
  }

  return Number(raw).toLocaleString();
}

// Converts cm to "Xkm Ym Zcm".
// Zero parts are omitted, e.g. 150 cm -> "1m 50cm".
export function formatDistance(cm) {
  if (cm === 0) {
    return "0m";
  }

  const km = Math.floor(cm / 100000);
  const m = Math.floor((cm % 100000) / 100);
  const rem = cm % 100;

  const parts = [];
  if (km > 0) parts.push(`${km}km`);
  if (m > 0) parts.push(`${m}m`);
  if (rem > 0) parts.push(`${rem}cm`);

  return parts.join(" ");
}

// Converts game ticks (20 per second) to "Xd Yh Zm".
// Always shows at least minutes, e.g. 0 ticks -> "0m".
export function formatTicks(ticks) {
  const totalSeconds = Math.floor(ticks / 20);
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);

  const parts = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0 || parts.length === 0) parts.push(`${minutes}m`);

  return parts.join(" ");
}

// ---------------------------------------------------------------------------
// String helpers
// ---------------------------------------------------------------------------

// Converts a snake_case or kebab-case key to Title Case.
export function titleCase(key) {
  return String(key)
    .replaceAll(/[-_]/g, " ")
    .replaceAll(/\b\w/g, (c) => c.toUpperCase());
}

// ---------------------------------------------------------------------------
// Data fetching
// ---------------------------------------------------------------------------

// Fetches a JSON file. Throws a descriptive error on HTTP failure.
export async function fetchJSON(path) {
  const res = await fetch(path);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${path}`);
  }

  return res.json();
}
