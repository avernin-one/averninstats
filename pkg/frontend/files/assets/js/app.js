// app.js - entry point
"use strict";

import { on, start } from "./router.js";
import { initI18n } from "./i18n.js";
import { loadTemplates } from "./templates.js";
import {
  renderIndex,
  renderStats,
  renderPlayers,
  renderHighscore,
} from "./views.js";

const ANNOYING_BANNER = `
                                ▀▀               ██         ██          
   ▀▀█▄ ██ ██ ▄█▀█▄ ████▄ ████▄ ██  ████▄ ▄█▀▀▀ ▀██▀▀ ▀▀█▄ ▀██▀▀ ▄█▀▀▀  
  ▄█▀██ ██▄██ ██▄█▀ ██ ▀▀ ██ ██ ██  ██ ██ ▀███▄  ██  ▄█▀██  ██   ▀███▄  
  ▀█▄██  ▀█▀  ▀█▄▄▄ ██    ██ ██ ██▄ ██ ██ ▄▄▄█▀  ██  ▀█▄██  ██   ▄▄▄█▀  
                                                                        
`;

async function bootstrap() {
  console.log(ANNOYING_BANNER);

  // Load templates and translations in parallel before the router starts.
  try {
    await Promise.all([loadTemplates(), initI18n()]);
  } catch (err) {
    console.error("Bootstrap failed:", err);
    document.getElementById("app").innerHTML =
      `<div style="padding:2rem;color:#e94560">⚠️ Failed to load: ${err.message}</div>`;
    return;
  }

  renderIndex();

  // Register routes.
  on("/highscore", () => renderHighscore(null));
  on("/highscore/:stat", ({ stat }) => renderHighscore(stat));
  on("/block", () => renderStats("block", null));
  on("/block/:stat", ({ stat }) => renderStats("block", stat));
  on("/item", () => renderStats("item", null));
  on("/item/:stat", ({ stat }) => renderStats("item", stat));
  on("/entity", () => renderStats("entity", null));
  on("/entity/:stat", ({ stat }) => renderStats("entity", stat));
  on("/player", () => renderPlayers());
  on("/player/:name", ({ name }) => renderPlayers(name));

  start();
}

await bootstrap();
