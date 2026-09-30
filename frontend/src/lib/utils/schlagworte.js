import { apiFetch, apiPut } from '../apiFetch.js';

/**
 * Die Schlagworte des Bestands als Vorschläge für ein ChipFeld (Migration 138), über
 * GET /api/schlagworte: ohne Suchtext die häufigsten zuerst (höchstens 500), mit Suchtext die
 * Wörter, die ihn enthalten, über alle Wörter (höchstens 50). Die Felder laden sie über
 * erzeugeSchlagwortVorschlaege (schlagwortVorschlaege.svelte.js), das beim Tippen nachfragt.
 *
 * Antwortet der Server nicht, kommt null: Der Aufrufer behält dann, was er hat. Das Feld nimmt
 * weiter freien Text an, es fehlen nur die Vorschläge.
 *
 * @param {string} [suche]
 * @returns {Promise<{ wert: string, beschreibung: string }[] | null>}
 */
export async function ladeSchlagwortVorschlaege(suche = '') {
	try {
		const res = await apiFetch(
			`/api/schlagworte${suche ? `?suche=${encodeURIComponent(suche)}` : ''}`
		);
		if (!res.ok) {
			return null;
		}
		const liste = await res.json();
		if (!Array.isArray(liste)) {
			return null;
		}
		return liste
			.filter((/** @type {any} */ v) => typeof v?.wort === 'string')
			.map((/** @type {{ wort: string, titel: number }} */ v) => ({
				wert: v.wort,
				beschreibung: `${v.titel} Titel`
			}));
	} catch {
		return null;
	}
}

/**
 * Der Schlagwort-Vorschlag der DNB zu einer ISBN (GET /api/schlagworte/dnb-vorschlag, entschieden
 * am 30.09.2026): die Wörter der eigenen Liste, die der DNB-Satz nennt, und die Normdatei-Wörter,
 * die die Liste noch nicht kennt — dieselbe Regel wie beim Anlegen über die Bestellsuche. Nur
 * angeboten, eingetragen wird über das Feld.
 *
 * Ein Fehler wirft: Der Aufrufer sagt „nicht erreichbar" an. Käme hier eine leere Antwort, sähe
 * ein Ausfall der DNB aus wie „die DNB hat nichts".
 *
 * @param {string} isbn
 * @returns {Promise<{ dnbSatz: boolean, liste: string[], neu: string[] }>}
 */
export async function ladeDnbSchlagwortVorschlag(isbn) {
	const res = await apiFetch(`/api/schlagworte/dnb-vorschlag?isbn=${encodeURIComponent(isbn)}`);
	if (!res.ok) {
		throw new Error(`DNB-Vorschlag nicht geladen (${res.status})`);
	}
	const antwort = await res.json();
	const liste = antwort?.schlagwort_vorschlaege;
	const neu = antwort?.schlagwort_vorschlaege_neu;
	return {
		dnbSatz: antwort?.dnb_satz === true,
		liste: Array.isArray(liste) ? liste : [],
		neu: Array.isArray(neu) ? neu : []
	};
}

/**
 * Die Schlagworte EINES Titels. Der Bestellkorb braucht sie, bevor er ändert: Er ersetzt
 * die Menge als Ganzes, und wer die vorhandenen nicht kennt, überschriebe sie mit seinem
 * ersten Wort. Anders als bei den Vorschlägen ist ein Fehler hier deshalb ein Fehler —
 * der Aufrufer sperrt dann das Feld, statt mit einer leeren Liste weiterzumachen.
 *
 * @param {string} titelId
 * @returns {Promise<string[]>}
 */
export async function ladeTitelSchlagworte(titelId) {
	const res = await apiFetch(`/api/buecher/titel/${encodeURIComponent(titelId)}/schlagworte`);
	if (!res.ok) {
		throw new Error(`Schlagworte konnten nicht geladen werden (${res.status})`);
	}
	const antwort = await res.json();
	if (!Array.isArray(antwort?.schlagworte)) {
		throw new Error('Schlagworte: Antwort ohne Liste');
	}
	return antwort.schlagworte;
}

/**
 * Ersetzt die Schlagworte eines Titels (PUT /api/buecher/titel/{id}/schlagworte). Eine
 * leere Liste entfernt alle. Liefert sie in der gespeicherten Schreibweise zurück —
 * „fantasy" kommt als das vorhandene „Fantasy" wieder.
 *
 * @param {string} titelId
 * @param {string[]} schlagworte
 * @returns {Promise<string[]>}
 */
export async function setzeTitelSchlagworte(titelId, schlagworte) {
	const antwort = await apiPut(`/api/buecher/titel/${encodeURIComponent(titelId)}/schlagworte`, {
		schlagworte
	});
	return antwort?.schlagworte ?? schlagworte;
}
