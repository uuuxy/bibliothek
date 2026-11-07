/**
 * Titel löschen: der eine Weg für Buchakte, Buchmaske und die Mehrfachauswahl der
 * Bestandstabelle. Erst liest er, was geschieht (GET /api/buecher/titel/loeschfolgen), dann fragt
 * der Dialog nach dem Grund, vorgewählt „Alter Bestand", und nennt die verliehenen Exemplare mit
 * ihrem Leser. Sie bleiben beim Leser, bis jemand ihr eigenes Kästchen ankreuzt; nur dann gehen
 * ihre Kennungen mit (DELETE /api/books), und der Server löscht genau diese.
 */
import { apiFetch, leseApiFehler } from '../../apiFetch.js';
import { loeschenMitGrund } from '../../stores/bestaetigung.svelte.js';

/** Mehr Namen stehen nicht im Dialog; der Rest heißt „und n weitere". */
const MAX_NAMEN = 5;

/**
 * @typedef {{ exemplar_id: string, barcode: string, titel: string, leser: string, klasse: string }} Verliehenes
 * @typedef {{ titel: number, im_haus: number, verliehen: Verliehenes[] }} Loeschfolgen
 * @typedef {{ geloescht: string[], geblieben: string[] }} Loeschergebnis
 */

/** @param {number} n */
const exemplare = (n) => (n === 1 ? '1 Exemplar' : `${n} Exemplare`);

/**
 * Die Überschrift der Rückfrage: die Zahl der Titel, sonst der Name des einen.
 * @param {number} anzahlTitel @param {string | undefined} name
 */
function frageTitel(anzahlTitel, name) {
	if (anzahlTitel > 1) return `${anzahlTitel} Titel löschen?`;
	return name ? `„${name}“ löschen?` : 'Titel löschen?';
}

/**
 * Was bleibt, bis das Verliehene zurück ist: von mehreren Titeln der des einen Exemplars oder
 * die der verliehenen, sonst der eine Titel.
 * @param {number} anzahlTitel @param {number} anzahlVerliehen
 */
function bleibendeTitel(anzahlTitel, anzahlVerliehen) {
	if (anzahlTitel > 1) return anzahlVerliehen === 1 ? 'sein Titel bleibt' : 'ihre Titel bleiben';
	return 'der Titel bleibt';
}

/**
 * Baut die Rückfrage aus der Vorschau des Servers.
 * @param {number} anzahlTitel
 * @param {string | undefined} name der Titel, wenn es einer ist
 * @param {Loeschfolgen} folgen
 * @returns {import('../../stores/bestaetigung.svelte.js').Rueckfrage}
 */
export function baueLoeschfrage(anzahlTitel, name, folgen) {
	const imHaus = folgen.im_haus;
	const verliehen = folgen.verliehen;
	const titel = frageTitel(anzahlTitel, name);
	let text = 'Das lässt sich nicht rückgängig machen.';
	if (imHaus === 1) text = 'Das Exemplar im Haus kommt mit diesem Grund ins Abgangsbuch.';
	else if (imHaus > 1)
		text = `Die ${imHaus} Exemplare im Haus kommen mit diesem Grund ins Abgangsbuch.`;
	else if (verliehen.length > 0) text = 'Im Haus ist kein Exemplar.';
	if (verliehen.length === 0) return { titel, text };

	const eines = verliehen.length === 1;
	const bleibt = bleibendeTitel(anzahlTitel, verliehen.length);
	const namen = verliehen
		.slice(0, MAX_NAMEN)
		.map((v) => (v.klasse ? `${v.leser} (${v.klasse})` : v.leser));
	if (verliehen.length > MAX_NAMEN) namen.push(`und ${verliehen.length - MAX_NAMEN} weitere`);
	return {
		titel,
		text,
		aktion: imHaus > 0 ? `${exemplare(imHaus)} löschen` : 'Löschen',
		zusatz: {
			hinweis: eines
				? 'Ein Exemplar ist verliehen:'
				: `${verliehen.length} Exemplare sind verliehen:`,
			liste: namen,
			nachsatz: eines
				? `Es bleibt beim Leser, und ${bleibt}, bis es zurück ist.`
				: `Sie bleiben bei den Lesern, und ${bleibt}, bis sie zurück sind.`,
			kaestchen: eines
				? 'Auch dieses Exemplar löschen und aus dem Leserkonto austragen'
				: `Auch diese ${verliehen.length} Exemplare löschen und aus den Leserkonten austragen`,
			aktionMit:
				imHaus + verliehen.length === 1
					? 'Das Exemplar löschen'
					: `Alle ${imHaus + verliehen.length} Exemplare löschen`,
			noetig: imHaus === 0
		}
	};
}

/**
 * Die Meldung nach dem Löschen, eine Zeile (M3, Snackbar: „Keep the snackbar text label to one
 * line long when possible").
 * @param {Loeschergebnis} ergebnis
 */
export function meldungZumErgebnis(ergebnis) {
	const g = ergebnis.geloescht.length;
	const b = ergebnis.geblieben.length;
	if (b === 0) return g === 1 ? 'Titel gelöscht' : `${g} Titel gelöscht`;
	const rest = 'bis die verliehenen Exemplare zurück sind';
	const mehrere = `${b} Titel bleiben`;
	if (g === 0) return `${b > 1 ? mehrere : 'Der Titel bleibt'}, ${rest}`;
	return `${g} Titel gelöscht. ${b > 1 ? mehrere : '1 Titel bleibt'}, ${rest}`;
}

/**
 * Löscht Titel nach der Rückfrage.
 * @param {string[]} ids
 * @param {string} [name] der Titel, wenn es einer ist
 * @returns {Promise<Loeschergebnis | null>} null: abgebrochen
 * @throws {Error} mit der Meldung des Servers
 */
export async function loescheTitelMitRueckfrage(ids, name) {
	const vorschau = await apiFetch(
		`/api/buecher/titel/loeschfolgen?ids=${ids.map(encodeURIComponent).join(',')}`,
		{ credentials: 'include' }
	);
	if (!vorschau.ok)
		throw new Error(
			await leseApiFehler(vorschau, 'Die Vorschau zum Löschen ließ sich nicht laden.')
		);
	/** @type {Loeschfolgen} */
	const folgen = await vorschau.json();

	const wahl = await loeschenMitGrund(baueLoeschfrage(ids.length, name, folgen));
	if (!wahl) return null;
	const verliehene = wahl.zusatz ? folgen.verliehen.map((v) => v.exemplar_id) : [];
	const res = await apiFetch('/api/books', {
		method: 'DELETE',
		credentials: 'include',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ ids, grund: wahl.grund, verliehene })
	});
	if (!res.ok) throw new Error(await leseApiFehler(res, 'Der Titel ließ sich nicht löschen.'));
	return await res.json();
}
