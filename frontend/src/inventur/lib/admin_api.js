import { apiFetch, apiClient } from "../../lib/apiFetch.js";
import { appState } from "./store.svelte.js";

export async function holeBuecherListe() {
    const suchParameter = appState.searchQuery
        ? `?q=${encodeURIComponent(appState.searchQuery)}`
        : "";
    const res = await apiFetch(`/api/books${suchParameter}`, {
        credentials: "include",
    });
    if (!res.ok) {
        if (res.status === 401) {
            appState.adminAuthenticated = false;
            throw new Error("UNAUTHORIZED");
        }
        throw new Error("Fehler beim Laden der Bücher");
    }
    const json = await res.json();
    return json.data || [];
}

/** @param {File} datei */
export async function importiereExcel(datei) {
    const formData = new FormData();
    formData.append("file", datei);
    const res = await apiFetch("/api/books/import", {
        method: "POST",
        credentials: "include",
        headers: {
        },
        body: formData,
    });
    if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson.error || errJson.message || "Import fehlgeschlagen");
    }
    return true;
}

/** @param {string[]} ids */
export async function loescheBuecher(ids) {
    const res = await apiFetch("/api/books", {
        method: "DELETE",
        credentials: "include",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ ids }),
    });
    if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson.error || "Löschen fehlgeschlagen");
    }
    return true;
}

export async function holeExterneCover() {
    const res = await apiFetch("/api/admin/books/external-covers", {
        credentials: "include",
    });
    if (!res.ok) throw new Error("Externe Cover konnten nicht geladen werden");
    const json = await res.json();
    return json.data || [];
}

/** @param {string[]} ids */
export async function retryExterneCover(ids = []) {
    const res = await apiFetch("/api/admin/books/retry-covers", {
        method: "POST",
        credentials: "include",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ ids, limit: 300 }),
    });
    if (!res.ok) throw new Error("Cover-Retry fehlgeschlagen");
    const json = await res.json();
    return json.data;
}
