import { apiFetch, extractApiError } from '../../apiFetch.js';

/**
 * Die Kandidatensuche des Zusammenführen-Dialogs — eigene Datei wie schuelerSuche.svelte.js:
 * Der Dialog steht an der 200-Zeilen-Ratsche, und die Suche ist das Stück mit eigener
 * Zeitachse (Entprellung, Antwort-Reihenfolge, Fehler).
 *
 * Drei Regeln, alle am 02.09.2026 nachgeholt (Raster-Fund, Frontend-Prüfer):
 *
 *  1. Nur die JÜNGSTE Anfrage schreibt die Liste. Vorher überschrieb die langsame
 *     Antwort auf „Al“ die schnelle auf „Alt123“ — und aus dieser Liste wählt der Admin
 *     den Datensatz, der danach UNUMKEHRBAR gelöscht wird. Vorbild orderStore.svelte.js.
 *  2. Ein Fehler heißt Fehler. 403/500 hießen vorher „Kein anderer Datensatz gefunden“,
 *     und ein Netzwerkfehler flog als unbehandelte Rejection aus dem Timer-Rückruf.
 *  3. Zurücksetzen stoppt auch den laufenden Timer und entwertet die laufende Antwort —
 *     sonst stand beim nächsten Öffnen (anderer Schüler an der Theke) die vorige Liste.
 *
 * @param {() => string} profilId — ID des Datensatzes, aus dessen Akte gesucht wird; er
 *   selbst ist kein Kandidat (der Server schließt ihn aus).
 */
export function erzeugeKandidatenSuche(profilId) {
	let suche = $state('');
	/** @type {any[]} */
	let treffer = $state([]);
	let fehler = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;
	// Laufende Nummer der jüngsten Anfrage; eine Antwort mit älterer Nummer wird verworfen.
	let nr = 0;

	async function suchen() {
		const meine = ++nr;
		const q = suche.trim();
		if (q.length < 2) {
			treffer = [];
			fehler = '';
			return;
		}
		try {
			const res = await apiFetch(
				`/api/schueler/${profilId()}/zusammenfuehren-kandidaten?q=${encodeURIComponent(q)}`
			);
			const liste = res.ok ? ((await res.json()) ?? []) : [];
			const meldung = res.ok ? '' : await extractApiError(res);
			if (meine !== nr) return;
			treffer = liste;
			fehler = meldung;
		} catch {
			if (meine !== nr) return;
			treffer = [];
			fehler = 'Netzwerkfehler — die Suche hat den Server nicht erreicht.';
		}
	}

	return {
		get suche() {
			return suche;
		},
		set suche(wert) {
			suche = wert;
		},
		get treffer() {
			return treffer;
		},
		get fehler() {
			return fehler;
		},

		// 250 ms Entprellung: Der Dialog ist eine Tastatur-Eingabe, kein Scan.
		tippen() {
			clearTimeout(timer);
			timer = setTimeout(suchen, 250);
		},

		zuruecksetzen() {
			clearTimeout(timer);
			nr++;
			suche = '';
			treffer = [];
			fehler = '';
		}
	};
}
