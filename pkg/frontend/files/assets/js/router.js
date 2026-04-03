// router.js - lightweight hash-based router
"use strict";

const routes = [];

// Registers a route pattern.
// Patterns may contain named segments like :name, e.g. "/player/:name".
export function on(pattern, handler) {
  const keys = [];

  const regexStr = pattern
    .replaceAll(/:([^/]+)/g, (_, key) => {
      keys.push(key);
      return "([^/]+)";
    })
    .replaceAll(/\*/g, ".*");

  routes.push({
    regex: new RegExp(`^${regexStr}$`),
    keys,
    handler,
  });
}

// Navigates to a new hash route without reloading the page.
export function navigate(path) {
  globalThis.location.hash = path;
}

// Returns the current route path, stripping the leading #.
export function currentPath() {
  const hash = globalThis.location.hash.replace(/^#/, "");
  return hash || "/highscore";
}

export function dispatch() {
  const path = currentPath();

  for (const route of routes) {
    const match = route.regex.exec(path);

    if (match) {
      const params = {};
      route.keys.forEach((key, i) => {
        params[key] = decodeURIComponent(match[i + 1]);
      });
      route.handler(params);
      return;
    }
  }

  // No route matched - fall back to highscores.
  navigate("/highscore");
}

// Starts the router: dispatches the current hash and listens for changes.
export function start() {
  globalThis.addEventListener("hashchange", dispatch);
  dispatch();
}
