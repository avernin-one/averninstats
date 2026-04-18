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

let current = "";
let index = [];

const cache = new Map();

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Loads the i18n index and the default language.
// Must be called once at startup before translate() is used.
export async function initI18n() {
  try {
    index = await fetchJSON("/i18n/.index.json");
  } catch (err) {
    index = [];
    console.error(err);
  }

  index.sort();
  current = localStorage.getItem("lang");
  if (!index.includes(current)) {
    current = index[0] || null;
  }

  await load(current);

  const langSelector = document.querySelector("#language");
  const languageSwitcher = document.querySelector("#language-switcher");

  const translate = new Intl.DisplayNames(current ?? [], {
    type: "language",
    style: "long",
    fallback: "code",
  });

  const getDisplayName = (code) => {
    try {
      return translate.of(code) || code;
    } catch {
      return code;
    }
  };

  index
    .map((code) => ({ id: code, name: getDisplayName(code) }))
    .sort((a, b) => a.name.localeCompare(b.name))
    .forEach(({ id, name }) => {
      const li = document.createElement("li");
      li.dataset.lang = id;
      li.textContent = name;

      li.addEventListener("click", (e) => {
        setLanguage(e.target.dataset.lang);
        hideSelector();
      });

      langSelector.appendChild(li);
    });

  const showSelector = () => {
    langSelector.style.display = "block";
  };

  const hideSelector = () => {
    langSelector.style.display = "none";
  };

  languageSwitcher.addEventListener("click", () => {
    langSelector.style.display === "block" ? hideSelector() : showSelector();
  });

  document.addEventListener("mousedown", (e) => {
    const clickedOutside =
      !langSelector.contains(e.target) && e.target !== languageSwitcher;
    if (clickedOutside) hideSelector();
  });
}

// Returns the display name for a Minecraft key in the current language.
// Falls back to title-casing the key if no translation exists.
export function translate(key) {
  const map = cache.get(current);

  if (map?.has(key)) {
    return titleCase(map.get(key));
  }

  return titleCase(key);
}

// Returns the current language code, e.g. "en-gb".
export function currentLang() {
  return current;
}

// Returns the language list from the index.
export function availableLanguages() {
  if (index?.languages) {
    return index.languages;
  }
  return [];
}

// Switches to a new language, loading its file if not yet cached.
// Fires all registered change listeners afterwards.
export async function setLanguage(code) {
  if (code === current) {
    return;
  }

  await load(code);
  current = code;
  localStorage.setItem("lang", code);
  globalThis.location.reload();
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

async function load(language) {
  if (cache.has(language)) {
    return;
  }

  try {
    const raw = await fetchJSON(`i18n/${language}.json`);
    cache.set(language, new Map(Object.entries(raw)));
  } catch (err) {
    console.error(err);
    // File missing - store empty map so translate() falls back to titleCase.
    cache.set(language, new Map());
  }
}
