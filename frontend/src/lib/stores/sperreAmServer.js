// Die drei Wege der Sperre nach Inaktivität zum Server (auth/handlers_sperre.go).
// Alle laufen unter /api/auth/: Ein 401 dort meldet den Arbeitsplatz nicht ab (apiFetch).

import { apiFetch } from '../apiFetch.js';

/**
 * Sperrt die Anmeldung dieses Browsers am Server.
 *
 * `nichtSperrbar`: Der Server führt zu dieser Anmeldung keinen Prüfwert des Passworts und
 * sperrt sie deshalb nicht — sie ließe sich bei einem Ausfall des Mailservers nicht aufschließen.
 * @returns {Promise<'gesperrt' | 'nichtSperrbar' | 'unerreicht'>}
 */
export async function sperreAmServer() {
	try {
		const res = await apiFetch('/api/auth/sperren', { method: 'POST' });
		if (!res.ok) return 'unerreicht';
		const antwort = await res.json();
		return antwort?.gesperrt === true ? 'gesperrt' : 'nichtSperrbar';
	} catch {
		return 'unerreicht';
	}
}

/**
 * Schickt das Passwort zum Aufschließen. Die Frist liegt über dem Standard: Der Server fragt
 * den Mailserver der Schule und wartet bei dessen Ausfall, bevor er selbst prüft.
 * @param {string} passwort
 * @returns {Promise<Response>}
 */
export function entsperreAmServer(passwort) {
	return apiFetch('/api/auth/entsperren', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ password: passwort }),
		timeoutMs: 30000
	});
}

/**
 * Fragt den Server, wie es um die Anmeldung steht.
 * @returns {Promise<{ zustand: 'offen', konto: any } | { zustand: 'gesperrt' | 'beendet' | 'unbekannt' }>}
 */
export async function leseSperrzustand() {
	try {
		const res = await apiFetch('/api/auth/me');
		if (res.ok) return { zustand: 'offen', konto: await res.json() };
		if (res.status === 423) return { zustand: 'gesperrt' };
		if (res.status === 401) return { zustand: 'beendet' };
	} catch {
		/* kein Netz */
	}
	return { zustand: 'unbekannt' };
}
