import { apiFetch } from '../../apiFetch.js';

/**
 * Der eine Abruf der Statistik (`/api/statistiken`) für Übersicht und Detailseite —
 * mit Reihenfolge-Schutz und Fehlerzustand.
 *
 * Bis zum 21.09.2026 holte jede der beiden Seiten selbst (OFFEN.md 5.8), und beiden
 * fehlte dasselbe: Eine langsame Antwort konnte eine schnellere überholen — Zeitraum
 * gewechselt, und unter der neuen Überschrift standen die Zahlen des alten. Und ein
 * Fehler ergab eine leere Seite: protokolliert im Server (api/stats.go), im Browser
 * nur eine Konsolenzeile, für den Menschen „keine Daten". `laufNr` verwirft, was
 * überholt wurde (wie useStudentProfile); `fehler` gibt der Seite den Zustand, den sie
 * mit LadeFehler zeigt.
 */
export function statistikAbruf() {
	/** @type {any} */
	let daten = $state(null);
	let loading = $state(true);
	let fehler = $state(false);
	let laufNr = 0;

	/** @param {URLSearchParams | string} params */
	async function laden(params) {
		const meine = ++laufNr;
		loading = true;
		fehler = false;
		try {
			const res = await apiFetch(`/api/statistiken?${params}`);
			if (meine !== laufNr) return; // eine jüngere Anfrage ist unterwegs oder da
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			if (meine !== laufNr) return;
			daten = json;
		} catch (err) {
			if (meine !== laufNr) return;
			console.error('Statistik laden fehlgeschlagen:', err);
			daten = null;
			fehler = true;
		} finally {
			if (meine === laufNr) loading = false;
		}
	}

	return {
		get daten() {
			return daten;
		},
		get loading() {
			return loading;
		},
		get fehler() {
			return fehler;
		},
		laden
	};
}
