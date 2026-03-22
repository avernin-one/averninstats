// views.js - page rendering via Mustache + i18n
"use strict";

import Mustache from "https://unpkg.com/mustache@4.2.0/mustache.mjs";
import {
  fetchJSON,
  flattenScoreList,
  flattenActionScores,
  formatValue,
  headImgPath,
  bodyImgPath,
  titleCase,
} from "./utils.js";
import { translate } from "./i18n.js";
import * as T from "./templates.js";

// ---------------------------------------------------------------------------
// Page-level helpers
// ---------------------------------------------------------------------------

function setActiveNav(section) {
  const links = document.querySelectorAll(".nav-link");
  for (const link of links) {
    link.classList.toggle("active", link.dataset.section === section);
  }
}

function setTitle(page) {
  document.title = page ? `averninstats - ${page}` : "averninstats";
}

function render(html) {
  document.getElementById("app").innerHTML = html;
}

function renderLoading() {
  render(Mustache.render(T.get("loading"), {}));
}

function renderError(message) {
  render(Mustache.render(T.get("error"), { message }));
}

// ---------------------------------------------------------------------------
// TOC helpers
// ---------------------------------------------------------------------------

// Replaces the TOC list with a loading spinner.
function showTocSpinner() {
  const list = document.getElementById("toc-list");
  if (list) {
    list.innerHTML = Mustache.render(T.get("toc-loading"), {});
  }
}

// Renders the TOC list and wires up the search input.
// Items with hrefs not starting with "#/" are treated as scroll anchors
// and get a click handler instead of hash navigation.
function buildToc(items) {
  const list = document.getElementById("toc-list");
  if (!list) {
    return;
  }

  function paint(filter) {
    const lc = filter ? filter.toLowerCase() : "";

    const filtered = lc
      ? items.filter((item) => item.label.toLowerCase().includes(lc))
      : items;

    list.innerHTML = Mustache.render(T.get("toc-list"), { items: filtered });

    // Attach scroll handlers to anchor links (e.g. highscore section jumps).
    for (const link of list.querySelectorAll(".toc-link")) {
      const href = link.getAttribute("href");

      if (href && !href.startsWith("#/")) {
        link.addEventListener("click", (e) => {
          e.preventDefault();
          const id = href.replace(/^#/, "");
          const target = document.getElementById(id);
          if (target) {
            target.scrollIntoView({ behavior: "smooth", block: "start" });
          }
        });
      }
    }
  }

  paint("");

  const searchInput = document.getElementById("toc-search");
  if (searchInput) {
    searchInput.value = "";
    searchInput.addEventListener("input", (e) => paint(e.target.value));
  }
}

// Marks the active TOC link and scrolls it into view.
function setActiveTocLink(href) {
  for (const link of document.querySelectorAll(".toc-link")) {
    link.classList.toggle("toc-active", link.getAttribute("href") === href);
  }
  const active = document.querySelector(".toc-link.toc-active");
  if (active) {
    active.scrollIntoView({ block: "nearest" });
  }
}

// ---------------------------------------------------------------------------
// Highscores
// ---------------------------------------------------------------------------

export async function renderHighscores() {
  setActiveNav("highscores");
  setTitle("Highscores");
  renderLoading();
  showTocSpinner();

  let manifest;
  try {
    manifest = await fetchJSON("highscore/_manifest.json");
  } catch {
    renderError("Could not load highscore index.");
    return;
  }

  render(Mustache.render(T.get("page-highscores"), {}));

  const content = document.querySelector("#stat-content .main-inner");
  const tocItems = [];

  for (const name of manifest.stats) {
    let data;
    try {
      data = await fetchJSON(`highscore/${name}.json`);
    } catch {
      continue;
    }

    const label = translate(data.name);
    const entries = flattenScoreList(data.scores).map((entry, index) => ({
      rank: index + 1,
      name: entry.name,
      headImg: headImgPath(entry.name),
      displayValue: formatValue(data.name, entry.score),
    }));

    if (entries.length === 0) {
      continue;
    }

    tocItems.push({ label, href: `#${data.name}` });
    content.insertAdjacentHTML(
      "beforeend",
      Mustache.render(T.get("highscore-section"), {
        anchor: data.name,
        label,
        entries,
      }),
    );
  }

  buildToc(tocItems);
}

// ---------------------------------------------------------------------------
// Stat list pages (block / item / entity)
// ---------------------------------------------------------------------------

export async function renderStatList(category) {
  setActiveNav(category);
  setTitle(titleCase(category));
  renderLoading();
  showTocSpinner();

  let manifest;
  try {
    manifest = await fetchJSON(`${category}/_manifest.json`);
  } catch {
    renderError(`Could not load ${category} index.`);
    return;
  }

  render(Mustache.render(T.get("page-stat-list"), {}));

  const tocItems = manifest.stats
    .map((name) => ({
      label: translate(name),
      href: `#/${category}/${name}`,
    }))
    .sort((a, b) => a.label.localeCompare(b.label));

  buildToc(tocItems);
}

export async function renderStatDetail(category, statName) {
  setActiveNav(category);
  setTitle(`${translate(statName)} - ${titleCase(category)}`);

  if (!document.getElementById("stat-content")) {
    await renderStatList(category);
  }

  const target = document.querySelector("#stat-content .main-inner");
  if (!target) {
    return;
  }

  target.innerHTML = Mustache.render(T.get("loading"), {});

  let data;
  try {
    data = await fetchJSON(`${category}/${statName}.json`);
  } catch {
    target.innerHTML = Mustache.render(T.get("error"), {
      message: `Could not load "${translate(statName)}".`,
    });
    return;
  }

  const sections = flattenActionScores(data.scores, translate)
    .map(({ label, entries }) => ({
      label,
      entries: entries.map((entry, index) => ({
        rank: index + 1,
        name: entry.name,
        headImg: headImgPath(entry.name),
        displayValue: entry.score.toLocaleString(),
      })),
    }))
    .filter((section) => section.entries.length > 0);

  target.innerHTML = Mustache.render(T.get("stat-detail"), {
    title: translate(statName),
    sections,
  });

  setActiveTocLink(`#/${category}/${statName}`);
}

// ---------------------------------------------------------------------------
// Players
// ---------------------------------------------------------------------------

export async function renderPlayerList() {
  setActiveNav("players");
  setTitle("Players");
  renderLoading();
  showTocSpinner();

  let manifest;
  try {
    manifest = await fetchJSON("player/_manifest.json");
  } catch {
    renderError("Could not load player index.");
    return;
  }

  render(Mustache.render(T.get("page-players"), {}));

  const tocItems = manifest.players
    .sort((a, b) => a.localeCompare(b, undefined, { sensitivity: "base" }))
    .map((name) => ({ label: name, href: `#/player/${name}` }));

  buildToc(tocItems);
}

export async function renderPlayerDetail(playerName) {
  setActiveNav("players");
  setTitle(`${playerName} - Players`);

  if (!document.getElementById("player-content")) {
    await renderPlayerList();
  }

  const content = document.querySelector("#player-content .main-inner");
  if (!content) {
    return;
  }

  content.innerHTML = Mustache.render(T.get("loading"), {});

  let data;
  try {
    data = await fetchJSON(`player/${playerName}.json`);
  } catch {
    content.innerHTML = Mustache.render(T.get("error"), {
      message: `Player "${playerName}" not found.`,
    });
    return;
  }

  // Build the stats table rows.
  const statsRows = Object.entries(data.stats ?? {})
    .map(([key, value]) => ({
      label: translate(key),
      displayValue: formatValue(key, value),
    }))
    .sort((a, b) => a.label.localeCompare(b.label));

  // Build per-category score sections.
  const scoreCategories = [];

  for (const [cat, actionScores] of Object.entries(data.scores ?? {})) {
    titleCase(cat);

    const actions = flattenActionScores(actionScores, translate)
      .map(({ label, entries }) => ({
        label,
        typeLabel: categoryLabel,
        entries: entries.map((entry, index) => ({
          rank: index + 1,
          name: translate(entry.name),
          displayValue: entry.score.toLocaleString(),
        })),
      }))
      .filter((action) => action.entries.length > 0);

    if (actions.length > 0) {
      scoreCategories.push({ categoryLabel, actions });
    }
  }

  content.innerHTML = Mustache.render(T.get("player-profile"), {
    name: data.name,
    bodyImg: bodyImgPath(data.name),
    gold: data.medals?.gold || 0,
    silver: data.medals?.silver || 0,
    bronze: data.medals?.bronze || 0,
    statsRows,
    scoreCategories,
  });

  setActiveTocLink(`#/player/${playerName}`);
}
