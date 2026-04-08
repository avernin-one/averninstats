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

// Waits for the browser to draw before measuring table widths.
function scheduleAlign(scope) {
  requestAnimationFrame(() => alignTables(scope));
}

// -----------------------------------------------------------------------------
// Page-level helpers
// -----------------------------------------------------------------------------

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

function setTitle(...s) {
  document.title = s.length > 0 ? `${title} ▸ ${s.join(" ▸ ")}` : title;
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

function renderLoading(info = "") {
  render(Mustache.render(T.get("loading"), { info: info }));
}

function renderError(message) {
  render(Mustache.render(T.get("error"), { message }));
}

async function renderToc(category, doTranslate = true) {
  const toc = document.getElementById("toc-list");

  // Load index if not cached
  if (!tocs[category]) {
    try {
      const index = await fetchJSON(`${category}/.index.json`);
      const tocItems = index
        .map((name) => ({
          id: name,
          name: doTranslate ? translate(name) : name,
          href: `#/${category}/${name}`,
        }))
        .sort((a, b) => a.name.localeCompare(b.name));

      tocs[category] = tocItems;
    } catch (err) {
      renderError(`Could not load ${category} index.`);
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
  const handleSearch = (event) => {
    const query = event.target.value.toLowerCase();
    const filteredItems = tocs[category].filter((item) =>
      item.name.toLowerCase().includes(query),
    );
    toc.innerHTML = Mustache.render(T.get("toc-list"), filteredItems);
  };

  const searchInput = document.getElementById("toc-search");

  searchInput.value = "";
  searchInput.removeEventListener("input", handleSearch);
  searchInput.addEventListener("input", handleSearch);

  return tocs[category];
}

// -----------------------------------------------------------------------------
// Highscore
// -----------------------------------------------------------------------------
export async function renderHighscore(stat = null) {
  setTitle("Highscore");
  setActiveNav("highscore");

  let index = await renderToc("highscore");
  let site = document.querySelector(`.stat-detail[data-id="highscore"]`);

  if (site == null) {
    let data;
    try {
      renderLoading("highscore");
      data = await fetchJSON(`highscore/highscore.json`);
      data = formatHighscoreStats(data);
    } catch (err) {
      console.error(`Failed to fetch highscore data`, err);
    }

    render(Mustache.render(T.get("page-highscore"), data));
    scheduleAlign(".stat-detail");
  }

  let url = `#/highscore`;
  if (stat) {
    setActiveToc(stat);
    url = `#/highscore/${stat}`;
  }

  scrollToSection(stat ?? index[0].id);

  globalThis.history.replaceState(this, "", url);
}

// -----------------------------------------------------------------------------
// Stats
// -----------------------------------------------------------------------------
export async function renderStats(category, statName) {
  let index = await renderToc(category);

  if (statName == null) {
    statName = index[0].id;
  }

  setTitle(titleCase(category), translate(statName));
  setActiveNav(category);
  setActiveToc(statName);

  renderLoading(translate(statName));

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
  for (const elem in data) {
    let entry = {
      id: elem,
      name: translate(elem),
      scores: formatScores(data[elem], elem),
    };

    sections.push(entry);
  }

  render(
    Mustache.render(T.get("page-stats"), {
      title: translate(statName),
      sections,
    }),
  );

  scheduleAlign(".stat-detail");
  const url = `#/${category}/${statName}`;
  globalThis.history.replaceState(this, "", url);
}

// -----------------------------------------------------------------------------
// Players
// -----------------------------------------------------------------------------
export async function renderPlayers(playerName = null) {
  setActiveNav("player");

  let index = await renderToc("player", false);

  if (playerName == null) {
    setTitle("Player");

    render(
      Mustache.render(T.get("page-player"), {
        players: index.sort(() => Math.random() - 0.5),
      }),
    );

    return;
  }

  setTitle("Player", playerName);
  renderLoading(playerName);

  let data;
  try {
    data = await fetchJSON(`player/${playerName}.json`);
  } catch {
    renderError(`Player "${playerName}" not found.`);
    return;
  }

  data.stats = formatPlayerStats(data.stats);
  data.scores = formatPlayerScores(data.scores);

  render(Mustache.render(T.get("page-player-profile"), data));
  scheduleAlign(".player-profile");
  setActiveToc(playerName);
}

function formatHighscoreStats(data) {
  return Object.entries(data)
    .sort(([a], [b]) => a.localeCompare(b)) // sort alphabetic ascending
    .map(([id, scores]) => ({
      id: id,
      title: translate(id),
      scores: formatScores(scores, id),
    }));
}

/*
  Formats the input data from:

  {
    "2": [
      "PlayerA"
    ],
    "3": [
      "PlayerB"
    ],
    "8": [
      "PlayerB",
      "PlayerC"
    ]
  }

  to:

  [
    {
      "rank": 1,
      "score": "8",
      "players": [
        "PlayerA"
      ]
    },
    {
      "rank": 2,
      "score": "3",
      "players": [
        "PlayerB"
      ]
    },
    {
      "rank": 3,
      "score": "2",
      "players": [
        "PlayerB",
        "PlayerC"
      ]
    }
  ]
*/
function formatScores(data, id) {
  return Object.entries(data)
    .sort(([a], [b]) => Number(b) - Number(a))
    .map(([score, players], index) => ({
      rank: index + 1,
      score: formatValue(id, score),
      players: players,
    }));
}

function formatPlayerStats(data) {
  return Object.entries(data)
    .map(([key, value]) => ({
      key: translate(key),
      value: formatValue(key, value),
    }))
    .sort((a, b) => a.key.localeCompare(b.key));
}

function formatPlayerScores(data) {
  return Object.entries(data)
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
}
