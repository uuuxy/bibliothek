// Die drei Wege der Sperre zum Server (internal/auth/sperre.go): sperren, aufschließen, den Zustand lesen.
// Alle laufen unter /api/auth/: Ein 401 dort meldet den Arbeitsplatz nicht ab (apiFetch).

import { apiFetch } from '../apiFetch.js';

/** Abstand der Anläufe, solange die Sperre den Server nicht erreicht; wie die Warteschlange. */
const NEUER_ANLAUF_MS = 60_000;

// Sperren und Aufschließen gehen nacheinander zum Server. Kommt eine Sperre dort nach dem
// Aufschließen an, sperrt er erneut, und der Sperrbildschirm steht mit der nächsten Anfrage wieder da.
/** @type {Promise<unknown>} */
let reihe = Promise.resolve();

/**
 * @template T
 * @param {() => Promise<T>} schritt
 * @returns {Promise<T>}
 */
function nacheinander(schritt) {
	const lauf = reihe.then(schritt);
	reihe = lauf.catch(() => {});
	return lauf;
}

/**
 * Sperrt die Anmeldung dieses Browsers am Server. Ein Aufschließen, das gerade läuft, wartet sie
 * ab: Hat es geöffnet, ist sie nicht mehr fällig und geht nicht hinaus.
 *
 * `nichtSperrbar`: Der Server führt zu dieser Anmeldung keinen Prüfwert des Passworts und
 * sperrt sie deshalb nicht — sie ließe sich bei einem Ausfall des Anmeldeservers nicht aufschließen.
 * `beendet`: Die Anmeldung gibt es am Server nicht mehr, zu sperren ist nichts.
 * @param {() => boolean} faellig
 * @returns {Promise<'gesperrt' | 'nichtSperrbar' | 'beendet' | 'unerreicht'>}
 */
async function sperreAmServer(faellig) {
	try {
		// In der Reihe steht nur die Anfrage: Ihre Frist begrenzt, wie lange ein Aufschließen wartet.
		const res = await nacheinander(async () =>
			faellig() ? apiFetch('/api/auth/sperren', { method: 'POST' }) : null
		);
		if (!res) return 'unerreicht';
		if (res.status === 401) return 'beendet';
		if (!res.ok) return 'unerreicht';
		const antwort = await res.json();
		return antwort?.gesperrt === true ? 'gesperrt' : 'nichtSperrbar';
	} catch {
		return 'unerreicht';
	}
}

/**
 * Sperrt am Server und fragt jede Minute noch einmal, solange er nicht erreicht ist und die
 * Sperre noch fällig ist. Bis er sie bestätigt, verdeckt sie nur das Fenster, und Neuladen
 * öffnete die Anwendung ohne Passwort.
 * @param {() => boolean} faellig
 */
export async function sperreBisZumServer(faellig) {
	let ergebnis = await sperreAmServer(faellig);
	while (ergebnis === 'unerreicht' && faellig()) {
		await new Promise((weiter) => setTimeout(weiter, NEUER_ANLAUF_MS));
		ergebnis = await sperreAmServer(faellig);
	}
	return ergebnis;
}

/**
 * Schickt das Passwort zum Aufschließen, sobald eine Sperre, die unterwegs ist, beantwortet ist.
 * Die Frist liegt über dem Standard: Der Server fragt Mailserver oder Verzeichnisdienst und wartet
 * bei dessen Ausfall, bevor er selbst prüft.
 * @param {string} passwort
 * @param {(konto: any) => void} schliesseAuf  öffnet das Fenster, bevor die nächste Sperre an der Reihe ist
 * @returns {Promise<Response>}
 */
export function entsperreAmServer(passwort, schliesseAuf) {
	return nacheinander(async () => {
		const res = await apiFetch('/api/auth/entsperren', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ password: passwort }),
			timeoutMs: 30000
		});
		// Abgelehnt: Den Grund liest der Aufrufer aus der Antwort.
		if (!res.ok) return res;
		schliesseAuf(await res.json());
		return res;
	});
}

/**
 * Fragt den Server, wie es um die Anmeldung steht. Gesperrt nennt er den Grund: nach
 * Inaktivität oder zehn Stunden nach der letzten Passworteingabe (internal/auth/sitzungen.go).
 * @returns {Promise<{ zustand: 'offen', konto: any } | { zustand: 'gesperrt', grund: SperrGrund } | { zustand: 'beendet' | 'unbekannt' }>}
 */
export async function leseSperrzustand() {
	try {
		const res = await apiFetch('/api/auth/me');
		if (res.ok) return { zustand: 'offen', konto: await res.json() };
		if (res.status === 423) return { zustand: 'gesperrt', grund: await leseSperrGrund(res) };
		if (res.status === 401) return { zustand: 'beendet' };
	} catch {
		/* kein Netz */
	}
	return { zustand: 'unbekannt' };
}

/** @typedef {'inaktivitaet' | 'hoechstdauer'} SperrGrund */

/**
 * Ohne lesbaren Grund gilt die Inaktivität, der häufige Fall.
 * @param {Response} res
 * @returns {Promise<SperrGrund>}
 */
async function leseSperrGrund(res) {
	try {
		const antwort = await res.json();
		return antwort?.grund === 'hoechstdauer' ? 'hoechstdauer' : 'inaktivitaet';
	} catch {
		return 'inaktivitaet';
	}
}
