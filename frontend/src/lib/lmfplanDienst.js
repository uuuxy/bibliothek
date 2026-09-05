/** Serverwege des LMF-Plans — Laden, Speichern, Löschen, PDF — und die kleinen
 *  Darstellungsregeln, die Verwaltungsseite und Portal-Reiter teilen.
 *
 *  Der Plan ist die frühere Excel-Tabelle der Schule (Wochentag, Datum, Stunde,
 *  Klasse(n), Besonderheiten). Die Bibliothek pflegt ihn, das Kollegium liest ihn im
 *  Portal — für alle gleich, keine Personalisierung nach Klassenleitung (Peter,
 *  05.09.2026: auch Fachlehrer gehen mit ihren Klassen zum Büchertausch). */
import { apiFetch } from './apiFetch.js';

/** @typedef {{ id?: string, datum: string, stunde: number, art: 'rueckgabe' | 'ausgabe', klassen: string[], vermerk: string }} LmfTermin */

export const ARTEN = /** @type {const} */ ([
	{ wert: 'rueckgabe', label: 'Bücherrückgabe' },
	{ wert: 'ausgabe', label: 'Bücherausgabe' }
]);

export const STUNDEN = Array.from({ length: 12 }, (_, i) => i + 1);

const wochentagFormat = new Intl.DateTimeFormat('de-DE', { weekday: 'long' });
const datumFormat = new Intl.DateTimeFormat('de-DE', {
	day: '2-digit',
	month: '2-digit',
	year: '2-digit'
});

/** @param {string} iso JJJJ-MM-TT */
function alsDatum(iso) {
	const [j, m, t] = iso.split('-').map(Number);
	return new Date(j, m - 1, t);
}

/** @param {string} iso */
export function wochentag(iso) {
	return wochentagFormat.format(alsDatum(iso));
}

/** @param {string} iso */
export function datumKurz(iso) {
	return datumFormat.format(alsDatum(iso));
}

/** @param {number} stunde */
export function stundeText(stunde) {
	return `${stunde}. Std.`;
}

/** @param {string} art */
export function artLabel(art) {
	return ARTEN.find((a) => a.wert === art)?.label ?? art;
}

/** Lädt den Plan ab Schuljahresbeginn (alle = true: auch ältere Termine).
 *  @param {boolean} [alle]
 *  @returns {Promise<{ ab: string, termine: LmfTermin[], ohne_rueckgabe_termin: string[] }>} */
export async function ladePlan(alle = false) {
	const res = await apiFetch(`/api/lmf-termine${alle ? '?alle=1' : ''}`);
	if (!res.ok) throw new Error('LMF-Plan konnte nicht geladen werden');
	return await res.json();
}

/** Legt an (ohne id) oder ändert (mit id). Gibt die Server-Meldung zurück, statt eine
 *  eigene zu formulieren — nur der Server kennt den Grund einer Ablehnung.
 *  @param {LmfTermin} termin
 *  @returns {Promise<{ ok: boolean, termin?: LmfTermin, meldung: string }>} */
export async function speichereTermin(termin) {
	const res = await apiFetch(termin.id ? `/api/lmf-termine/${termin.id}` : '/api/lmf-termine', {
		method: termin.id ? 'PUT' : 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			datum: termin.datum,
			stunde: termin.stunde,
			art: termin.art,
			klassen: termin.klassen,
			vermerk: termin.vermerk
		})
	});
	const json = await res.json().catch(() => ({}));
	return res.ok
		? { ok: true, termin: json, meldung: 'Termin gespeichert.' }
		: { ok: false, meldung: json.error ?? json.message ?? 'Speichern fehlgeschlagen.' };
}

/** @param {string} id @returns {Promise<{ ok: boolean, meldung: string }>} */
export async function loescheTermin(id) {
	const res = await apiFetch(`/api/lmf-termine/${id}`, { method: 'DELETE' });
	if (res.ok) return { ok: true, meldung: 'Termin gelöscht.' };
	const json = await res.json().catch(() => ({}));
	return { ok: false, meldung: json.error ?? 'Löschen fehlgeschlagen.' };
}

/** Lädt den Plan als PDF herunter — dieselbe Auswahl wie die Liste.
 *  @param {boolean} [alle] */
export async function ladePdf(alle = false) {
	const res = await apiFetch(`/api/lmf-termine/pdf${alle ? '?alle=1' : ''}`);
	if (!res.ok) throw new Error('PDF konnte nicht erzeugt werden');
	const url = window.URL.createObjectURL(await res.blob());
	const a = document.createElement('a');
	a.href = url;
	a.download = 'LMF-Plan.pdf';
	document.body.appendChild(a);
	a.click();
	window.URL.revokeObjectURL(url);
	a.remove();
}
