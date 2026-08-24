import { apiFetch } from '../../apiFetch.js';
import { uiStore } from '../../stores/uiStore.svelte.js';

/**
 * Die Serversuche der Schülerdatei. Eigene Datei wie ausweisdruck.svelte.js:
 * StudentDirectory steht an der Größen-Ratsche, und die Suche ist das Stück, das
 * nichts mit dem Führen der Liste zu tun hat.
 *
 * Gesucht wird auf dem SERVER. Vorher filterte die Ansicht im Browser über die
 * gelieferte Liste — und die ist bei 500 Zeilen gekappt. Bei 875 Schülern waren 375
 * über die Suche schlicht nicht erreichbar, welche genau hing an der alphabetischen
 * Reihenfolge der Klassennamen. Für den Benutzer sah das nach Zufall aus.
 *
 * Nebeneffekt, der den Ausschlag gab: Die Serversuche ist dieselbe wie an der Theke
 * (suchnorm) — "Muller" findet Müller, "Hoffmann Lena" dasselbe wie "Lena Hoffmann".
 * Der Browser-Filter konnte beides nicht.
 *
 * @param {() => void} nachKlassenDruck — läuft, wenn ein aus dem Druck-Center
 *   angeforderter Klassen-Stapeldruck fertig geladen ist (Treffer markieren).
 */
export function erzeugeSchuelerSuche(nachKlassenDruck) {
	/** @type {any[]} */
	let students = $state.raw([]);
	let laedt = $state(false);
	let sucheLaeuft = $state(false);
	let query = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;

	/** Muss zu ListStudentsWithStatsLimit im Backend passen: Erreicht die ungefilterte
	 *  Liste diese Länge, ist sie gekappt und die Ansicht sagt das auch. */
	const LISTEN_GRENZE = 500;

	// Nur die JÜNGSTE Anfrage darf die Liste schreiben: Beim Sprung aus dem
	// Druck-Center laufen die ungefilterte Mount-Ladung und die Klassensuche
	// gleichzeitig — welche Antwort zuletzt eintrifft, entschiede sonst der Server.
	let ladeNr = 0;

	async function lade() {
		const nr = ++ladeNr;
		laedt = true;
		try {
			const q = query.trim();
			const res = await apiFetch(`/api/schueler${q ? `?q=${encodeURIComponent(q)}` : ''}`);
			if (res.ok && nr === ladeNr) {
				students = (await res.json()) || [];
			}
		} catch (err) {
			console.error('Fehler beim Laden des Schülerverzeichnisses:', err);
		} finally {
			if (nr === ladeNr) {
				laedt = false;
				sucheLaeuft = false;
			}
		}
	}

	// Erste Ladung sofort (wie erzeugeAusweisdruck sein Design lädt) — AUSSER ein
	// Klassen-Stapeldruck ist angefordert: Dann lädt gleich der Effekt unten mit der
	// Klasse im Suchfeld. Eine ungefilterte Parallel-Ladung daneben hieße, dass die
	// Antwort-Reihenfolge entscheidet, ob der Rückruf eine leere Liste markiert —
	// genau so ist es beim ersten Bau passiert (der Effekt läuft vor onMount).
	if (!uiStore.requestedKlassenDruck) lade();

	// Aus dem Druck-Center angeforderter Klassen-Stapeldruck (gleiche Mechanik wie
	// requestedStudentId): Klasse suchen, dann markiert der Aufrufer die Treffer.
	// Gedruckt wird in der Schülerdatei, hinter der Aktionsleiste — ihre Warnungen
	// (fehlendes Ablaufdatum, Etiketten-Startposition) stehen damit auch auf diesem
	// Weg vor dem Stapel.
	$effect(() => {
		const klasse = uiStore.requestedKlassenDruck;
		if (!klasse) return;
		uiStore.requestedKlassenDruck = null;
		query = klasse;
		lade().then(nachKlassenDruck);
	});

	return {
		get students() {
			return students;
		},
		get beschaeftigt() {
			return laedt || sucheLaeuft;
		},
		get query() {
			return query;
		},
		set query(wert) {
			query = wert;
		},
		get suchend() {
			return query.trim().length > 0;
		},
		get gekuerzt() {
			return !query.trim() && students.length >= LISTEN_GRENZE;
		},
		lade,

		// Tippen wird entprellt, damit nicht jeder Tastendruck eine Abfrage auslöst.
		// 300 ms wie in der Omnibox — dieselbe Eingabegeschwindigkeit, dieselbe Wartezeit.
		angestossen() {
			sucheLaeuft = true;
			clearTimeout(timer);
			timer = setTimeout(lade, 300);
		}
	};
}
