// views.js - page rendering via Mustache + i18n
"use strict";

import Mustache from "https://unpkg.com/mustache@4.2.0/mustache.mjs";
import { fetchJSON, formatValue, titleCase } from "./utils.js";
import { translate } from "./i18n.js";
import * as T from "./templates.js";
import { alignTables } from "./align-tables.js";

const SCROLL_OPTIONS = {
  behavior: "smooth",
  block: "center",
};

let tocs = [];
let title = "";

// Waits for the browser to paint before measuring table widths.
function scheduleAlign(scope) {
  requestAnimationFrame(() => alignTables(scope));
}

// ---------------------------------------------------------------------------
// Page-level helpers
// ---------------------------------------------------------------------------

function setActiveNav(id) {
  const links = document.querySelectorAll(".nav-link");
  for (const link of links) {
    link.classList.toggle("active", link.dataset.id === id);
  }
}

function setActiveToc(id) {
  const links = document.querySelectorAll(".toc-link");
  for (const link of links) {
    let isActive = link.classList.toggle("active", link.dataset.id === id);
    if (isActive) {
      link.scrollIntoView(SCROLL_OPTIONS);
    }
  }
}

function scrollToSection(id) {
  const topNav = document.querySelector("nav.topnav");
  const main = document.querySelector("main");
  const section = document.getElementById(id);

  let offset = topNav.offsetHeight + main.style.paddingBottom;

  if (section) {
    section.style.scrollMarginTop = `${offset}px`;
    section.scrollIntoView(SCROLL_OPTIONS);
  }
}

function setTitle(page) {
  document.title = page ? `${title} - ${page}` : title;
}

export function renderIndex() {
  // Title
  document
    .querySelector("head")
    .insertAdjacentHTML("afterbegin", Mustache.render(T.get("_title"), {}));

  title = document.querySelector("head meta[name=custom_title]").content;

  document.querySelector("nav.topnav .logo").innerHTML = title;

  // Topnav external links
  document.querySelector("nav.topnav .ext-links").innerHTML = Mustache.render(
    T.get("_topnav-links"),
  );

  // Footer
  document.querySelector("footer").innerHTML = Mustache.render(
    T.get("_footer"),
    {},
  );

  const darkButton = "🌑";
  const lightButton = "🌕";
  const lastModeKey = "lastMode";
  const toggleButton = document.querySelector("#toggleButton");

  function applyMode(isLight) {
    document.body.classList.toggle("light", isLight);
    toggleButton.innerText = isLight ? darkButton : lightButton;
    localStorage.setItem(lastModeKey, isLight ? "light" : "dark");
  }

  toggleButton.addEventListener("click", () => {
    const isLight = !document.body.classList.contains("light");
    applyMode(isLight);
  });

  window.addEventListener("DOMContentLoaded", () => {
    const stored = localStorage.getItem(lastModeKey);

    const isLight =
      stored === "light" ||
      (!stored && window.matchMedia?.("(prefers-color-scheme: light)").matches);

    applyMode(isLight);
  });
}

function render(html) {
  document.getElementById("main").innerHTML = html;
}

function renderLoading() {
  render(Mustache.render(T.get("loading"), {}));
}

function renderError(message) {
  render(Mustache.render(T.get("error"), { message }));
}

async function renderToc(category, doTranslate = true) {
  const toc = document.getElementById("toc-list");

  // Load manifest if not cached
  if (!tocs[category]) {
    try {
      const manifest = await fetchJSON(`${category}/_manifest.json`);
      const tocItems = manifest
        .map((name) => ({
          id: name,
          name: doTranslate ? translate(name) : name,
          href: `#/${category}/${name}`,
        }))
        .sort((a, b) => a.name.localeCompare(b.name));

      tocs[category] = tocItems;
    } catch (err) {
      renderError(`Could not load ${category} manifest.`);
      console.error(err);
      return;
    }
  }

  // Update TOC if category changed or not rendered yet
  if (toc.dataset.id != category) {
    toc.dataset.id = category;
    toc.innerHTML = Mustache.render(T.get("toc-list"), tocs[category]);
  }

  // Attach search event listener
  const searchInput = document.getElementById("toc-search");
  searchInput.value = "";
  searchInput.addEventListener("input", () => {
    const query = searchInput.value.toLowerCase();
    const filteredItems = tocs[category].filter((item) =>
      item.name.toLowerCase().includes(query),
    );
    toc.innerHTML = Mustache.render(T.get("toc-list"), filteredItems);
  });

  return tocs[category];
}

// ---------------------------------------------------------------------------
// Highscore
// ---------------------------------------------------------------------------
export async function renderHighscore(stat = null) {
  setTitle("Highscore");
  setActiveNav("highscore");

  let manifest = await renderToc("highscore");
  let site = document.querySelector(`.stat-detail[data-id="highscore"]`);

  if (site == null) {
    renderLoading();

    const sections = [];
    for (const name of manifest) {
      let data;
      try {
        data = await fetchJSON(`highscore/${name.id}.json`);

        data.scores = Object.entries(data.scores)
          .sort(([a], [b]) => Number(b) - Number(a))
          .map(([score, players], index) => ({
            rank: index + 1,
            score: formatValue(data.name, score),
            players: players,
          }));

        data.id = data.name;
        data.name = translate(name.id);
      } catch (err) {
        console.error(`Failed to fetch highscore data for ${name.id}:`, err);
        continue;
      }

      sections.push(data);
    }

    render(Mustache.render(T.get("page-highscore"), { sections }));
    scheduleAlign(".stat-detail");
  }

  let url = `#/highscore`;
  if (stat) {
    setActiveToc(stat);
    url = `#/highscore/${stat}`;
  }

  scrollToSection(stat ?? manifest[0].id);

  globalThis.history.replaceState(this, "", url);
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------
export async function renderStats(category, statName) {
  let manifest = await renderToc(category);

  if (statName == null) {
    statName = manifest[0].id;
  }

  setTitle(`${titleCase(category)} - ${translate(statName)}`);
  setActiveNav(category);
  setActiveToc(statName);

  renderLoading();

  let data = {};
  try {
    data = await fetchJSON(`${category}/${statName}.json`);
  } catch {
    render(
      Mustache.render(T.get("error"), {
        message: `Could not load "${translate(statName)}".`,
      }),
    );
    return;
  }

  let sections = [];
  for (const elem in data.scores) {
    let entry = {
      id: elem,
      name: translate(elem),
      scores: Object.entries(data.scores[elem])
        .sort(([a], [b]) => Number(b) - Number(a))
        .map(([score, players], index) => ({
          rank: index + 1,
          score: formatValue(data.name, score),
          players: players,
        })),
    };

    sections.push(entry);
  }

  render(
    Mustache.render(T.get("page-stats"), {
      title: translate(statName),
      sections,
    }),
  );

  const url = `#/${category}/${statName}`;
  globalThis.history.replaceState(this, "", url);
}

// ---------------------------------------------------------------------------
// Players
// ---------------------------------------------------------------------------
export async function renderPlayers(playerName = null) {
  setActiveNav("player");

  let manifest = await renderToc("player", false);

  if (playerName == null) {
    setTitle("Player");

    render(
      Mustache.render(T.get("page-player"), {
        players: manifest.sort(() => Math.random() - 0.5),
      }),
    );

    return;
  }

  setTitle(`Player - ${playerName}`);

  let data;
  try {
    data = await fetchJSON(`player/${playerName}.json`);
  } catch {
    renderError(`Player "${playerName}" not found.`);
    return;
  }

  data.stats = Object.entries(data.stats)
    .map(([key, value]) => ({
      key: translate(key),
      value: formatValue(key, value),
    }))
    .sort((a, b) => a.key.localeCompare(b.key));

  data.scores = Object.entries(data.scores)
    .map(([cat, actions]) => ({
      category: translate(cat),
      actions: Object.entries(actions)
        .map(([action, scoreList]) => ({
          action: translate(action),
          scores: Object.entries(scoreList)
            .sort(([a], [b]) => Number(b) - Number(a))
            .map(([score, names], index) => ({
              index: index + 1,
              score: formatValue(action, score),
              names: names.map((name) => translate(name)),
            })),
        }))
        .sort((a, b) => a.action.localeCompare(b.action)),
    }))
    .sort((a, b) => a.category.localeCompare(b.category));

  render(Mustache.render(T.get("page-player-profile"), data));
  scheduleAlign(".player-profile");
  setActiveToc(playerName);
}
