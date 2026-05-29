/**
 * auth_api.js
 *
 * Enthält gemeinsam genutzte Authentifizierungsfunktionen für die Inventur-App.
 */

export async function loginWithPassword(endpoint, password, errorMessage) {
    const antwort = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ password }),
    });
    if (!antwort.ok) throw new Error(errorMessage);
    return true;
}

export async function holeAuthStatus() {
    const antwort = await fetch("/api/auth/status", {
        credentials: "include",
    });
    if (!antwort.ok) {
        return { authenticated: false, admin: false };
    }
    const daten = await antwort.json().catch(() => ({}));
    return {
        authenticated: Boolean(daten.authenticated),
        admin: Boolean(daten.admin),
    };
}
