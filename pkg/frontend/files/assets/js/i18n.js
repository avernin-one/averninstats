// i18n.js - translation loading and lookup
//
// Translation files live at: i18n/<lang>.json
// They map stripped Minecraft keys to display names:
//   "stone"   -> "Stone"
//   "creeper" -> "Creeper"
//
// Stat labels (play_time, deaths, ...) are NOT in these files.
// They come from minecraft:custom and are handled by translate() falling
// back to titleCase() from utils.js.
"use strict";

import { fetchJSON, titleCase } from "./utils.js";

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

let _currentLang = "en-gb";
let _manifest = null;

// Loaded translations: langCode -> Map<key, displayName>
const _cache = new Map();

// Callbacks fired when the language changes.
const _changeListeners = [];

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Loads the i18n manifest and the default language.
// Must be called once at startup before translate() is used.
export async function initI18n() {
  try {
    _manifest = await fetchJSON("i18n/_manifest.json");
  } catch {
    _manifest = { languages: [{ code: "en-gb" }] };
  }

  _currentLang = "en-gb";
  await _load(_currentLang);

  return _manifest;
}

// Returns the display name for a Minecraft key in the current language.
// Falls back to title-casing the key if no translation exists.
export function translate(key) {
  const map = _cache.get(_currentLang);

  if (map && map.has(key)) {
    return map.get(key);
  }

  return titleCase(key);
}

// Returns the current language code, e.g. "en-gb".
export function currentLang() {
  return _currentLang;
}

// Returns the language list from the manifest.
export function availableLanguages() {
  if (_manifest && _manifest.languages) {
    return _manifest.languages;
  }
  return [];
}

// Switches to a new language, loading its file if not yet cached.
// Fires all registered change listeners afterwards.
export async function setLanguage(code) {
  if (code === _currentLang) {
    return;
  }

  await _load(code);
  _currentLang = code;
  localStorage.setItem("lang", code);

  for (const fn of _changeListeners) {
    fn(code);
  }
}

// Registers a callback that fires whenever the language changes.
// Use this to re-render the current view.
export function onLanguageChange(fn) {
  _changeListeners.push(fn);
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

async function _load(code) {
  if (_cache.has(code)) {
    return;
  }

  try {
    const raw = await fetchJSON(`i18n/${code}.json`);
    _cache.set(code, new Map(Object.entries(raw)));
  } catch {
    // File missing - store empty map so translate() falls back to titleCase.
    _cache.set(code, new Map());
  }
}
