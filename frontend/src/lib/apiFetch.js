/**
 * apiFetch.js — Centralized fetch wrapper for the Bibliothek main API.
 *
 * Responsibilities:
 *   1. Always sends `credentials: "include"` so cookies are attached.
 *   2. On mutating methods (POST/PUT/PATCH/DELETE), automatically reads the
 *      `csrf_token` cookie and sends it as the `X-CSRF-Token` header.
 *   3. Provides a consistent API surface across all components.
 *
 * Usage:
 *   import { apiFetch, apiClient } from "$lib/../lib/apiFetch.js";
 *   const res = await apiFetch("/api/benutzer", {
 *     method: "POST",
 *     headers: { "Content-Type": "application/json" },
 *     body: JSON.stringify(payload),
 *   });
 */

/**
 * Reads the global CSRF token from the `csrf_token` cookie.
 * Returns an empty string if the cookie is missing (e.g. first request).
 * @returns {string}
 */
function readCsrfToken() {
  if (typeof document === "undefined") return "";
  const entries = document.cookie ? document.cookie.split("; ") : [];
  for (const entry of entries) {
    if (entry.startsWith("csrf_token=")) {
      return decodeURIComponent(entry.slice("csrf_token=".length));
    }
  }
  return "";
}

/** HTTP methods that require CSRF token validation */
const MUTATION_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

/** @type {Promise<string> | null} Laufender Bootstrap — verhindert parallele Doppel-Requests */
let csrfBootstrap = null;

/**
 * Liefert das CSRF-Token und holt es beim allerersten Mal vom Bootstrap-Endpoint.
 * Ohne das lief die erste Mutation direkt nach dem Login ohne Token in einen 403
 * (Cookie wird sonst erst durch irgendein vorheriges API-GET gesetzt).
 * @returns {Promise<string>}
 */
async function ensureCsrfToken() {
  const existing = readCsrfToken();
  if (existing) return existing;

  csrfBootstrap ??= fetch("/api/csrf-token", { credentials: "include" })
    .then(async (res) => {
      if (!res.ok) return "";
      const data = await res.json();
      return data.csrf_token || "";
    })
    .catch(() => "")
    .finally(() => { csrfBootstrap = null; });

  const fetched = await csrfBootstrap;
  return readCsrfToken() || fetched;
}

/**
 * Drop-in replacement for `fetch()` with automatic CSRF and credentials.
 *
 * @param {string | URL} url - The URL to fetch
 * @param {RequestInit} [options] - Standard fetch options
 * @returns {Promise<Response>}
 */
export async function apiFetch(url, options = {}) {
  // Always include credentials (cookies)
  options.credentials = "include";

  const method = (options.method || "GET").toUpperCase();

  // Inject CSRF header on mutating requests
  if (MUTATION_METHODS.has(method)) {
    const token = await ensureCsrfToken();
    if (token) {
      options.headers = {
        .../** @type {Record<string, string>} */ (options.headers || {}),
        "X-CSRF-Token": token,
      };
    }
  }

  // Set up AbortController for network timeouts
  const timeoutMs = MUTATION_METHODS.has(method) ? 10000 : 15000;
  const controller = new AbortController();
  const id = setTimeout(() => controller.abort(), timeoutMs);

  // If the user already provided a signal, link them (simplified handling: we just overwrite with our timeout, 
  // or if they passed one we could use `AbortSignal.any` but standard AbortController is enough for our use-case)
  options.signal = controller.signal;

  try {
    const response = await fetch(url, options);
    clearTimeout(id);
    return response;
  } catch (error) {
    clearTimeout(id);
    if (error instanceof Error && error.name === 'AbortError') {
      throw new Error("Netzwerk-Timeout: Die Anfrage hat zu lange gedauert.");
    }
    throw error;
  }
}

/**
 * apiClient provides convenience methods for common API operations.
 * It automatically serializes objects to JSON and sets the Content-Type header.
 */
export const apiClient = {
  get(url, options = {}) {
    return apiFetch(url, { ...options, method: "GET" });
  },
  post(url, data, options = {}) {
    return apiFetch(url, {
      ...options,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
      body: JSON.stringify(data),
    });
  },
  put(url, data, options = {}) {
    return apiFetch(url, {
      ...options,
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
      body: JSON.stringify(data),
    });
  },
  delete(url, options = {}) {
    return apiFetch(url, { ...options, method: "DELETE" });
  },
  patch(url, data, options = {}) {
    return apiFetch(url, {
      ...options,
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
      body: JSON.stringify(data),
    });
  },
};

import { toastStore } from "./stores/toastStore.svelte.js";

async function handleSmartResponse(res) {
  if (res.ok) {
    if (res.status === 204) return null;
    const text = await res.text();
    if (!text) return null;
    try {
      return JSON.parse(text);
    } catch {
      return text;
    }
  } else {
    const errorText = (await res.text()) || `Fehler ${res.status}`;
    toastStore.addToast(errorText, "error");
    throw new Error(errorText);
  }
}

export async function apiGet(url, options = {}) {
  const res = await apiFetch(url, { ...options, method: "GET" });
  return handleSmartResponse(res);
}

export async function apiPost(url, data, options = {}) {
  const res = await apiFetch(url, {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...options.headers },
    body: data ? JSON.stringify(data) : undefined,
  });
  return handleSmartResponse(res);
}

export async function apiPut(url, data, options = {}) {
  const res = await apiFetch(url, {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...options.headers },
    body: data ? JSON.stringify(data) : undefined,
  });
  return handleSmartResponse(res);
}

export async function apiDelete(url, options = {}) {
  const res = await apiFetch(url, { ...options, method: "DELETE" });
  return handleSmartResponse(res);
}
