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

/**
 * Drop-in replacement for `fetch()` with automatic CSRF and credentials.
 *
 * @param {string | URL} url - The URL to fetch
 * @param {RequestInit} [options] - Standard fetch options
 * @returns {Promise<Response>}
 */
export function apiFetch(url, options = {}) {
  // Always include credentials (cookies)
  options.credentials = "include";

  const method = (options.method || "GET").toUpperCase();

  // Inject CSRF header on mutating requests
  if (MUTATION_METHODS.has(method)) {
    const token = readCsrfToken();
    if (token) {
      options.headers = {
        .../** @type {Record<string, string>} */ (options.headers || {}),
        "X-CSRF-Token": token,
      };
    }
  }

  return fetch(url, options);
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
};
