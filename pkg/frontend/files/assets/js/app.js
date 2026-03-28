// app.js - entry point
"use strict";

import { on, start } from "./router.js";
import { initI18n } from "./i18n.js";
import { loadTemplates } from "./templates.js";
import {
  renderHighscores,
  renderStatList,
  renderStatDetail,
  renderPlayerList,
  renderPlayerDetail,
} from "./views.js";

async function bootstrap() {
  // Load templates and translations in parallel before the router starts.
  try {
    await Promise.all([loadTemplates(), initI18n()]);
  } catch (err) {
    console.error("Bootstrap failed:", err);
    document.getElementById("app").innerHTML =
      `<div style="padding:2rem;color:#e94560">⚠️ Failed to load: ${err.message}</div>`;
    return;
  }

  // Register routes.
  on("/highscores", () => renderHighscores());
  on("/block", () => renderStatList("block"));
  on("/block/:stat", ({ stat }) => renderStatDetail("block", stat));
  on("/item", () => renderStatList("item"));
  on("/item/:stat", ({ stat }) => renderStatDetail("item", stat));
  on("/entity", () => renderStatList("entity"));
  on("/entity/:stat", ({ stat }) => renderStatDetail("entity", stat));
  on("/player", () => renderPlayerList());
  on("/player/:name", ({ name }) => renderPlayerDetail(name));

  start();
}

bootstrap();
