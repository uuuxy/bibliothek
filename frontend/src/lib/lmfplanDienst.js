/** Serverwege des LMF-Plans — Lesen, Vorschau, Speichern, Verwerfen, PDF — und die
 *  kleinen Darstellungsregeln, die Planer und Portal-Reiter teilen.
 *
 *  Der Plan ist eine REIHENFOLGE von Klassen, die der Server auf Schultage × Stunden
 *  gießt (Peter, 05.09.2026, am echten Plan der Schule): Rahmen + Zeilen hin, Plätze
 *  zurück — auch die Vorschau rechnet der Server, damit es keinen JavaScript-Zwilling
 *  der Verteilung gibt. Das Kollegium liest das Ergebnis im Portal, für alle gleich. */
import { apiFetch } from './apiFetch.js';

/** @typedef {{ id?: string, datum: string, stunde: number, art: 'rueckgabe' | 'ausgabe', klassen: string[], vermerk: string }} LmfTermin */
/** @typedef {{ klassen: string[], vermerk: string }} PlanZeile */
/** @typedef {{ erster_tag: string, startstunde: number, stunden_je_tag: number, zeilen: PlanZeile[], ausgelassen: string[] }} PlanEntwurf */
/** @typedef {{ position: number, datum: string, stunde: number, klassen: string[], vermerk: string }} PlanPlatz */
/** @typedef {{ plan: { id: string, art: string, erster_tag: string, startstunde: number, stunden_je_tag: number } | null, zeilen: PlanPlatz[], ausgelassen: string[], vorbei: boolean, vorschlag?: { quelle: 'vorjahr' | 'regel', zeilen: PlanZeile[], ausgelassen: string[] }, klassen: string[] }} PlanStand */

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

/** Lädt die Termine ab Schuljahresbeginn (alle = true: auch ältere) — die Tabelle des
 *  Portals und der PDF.
 *  @param {boolean} [alle]
 *  @returns {Promise<{ ab: string, termine: LmfTermin[], ohne_rueckgabe_termin: string[] }>} */
export async function ladePlan(alle = false) {
	const res = await apiFetch(`/api/lmf-termine${alle ? '?alle=1' : ''}`);
	if (!res.ok) throw new Error('LMF-Plan konnte nicht geladen werden');
	return await res.json();
}

/** Der neueste Plan einer Art samt Vorschlag und Klassenliste.
 *  @param {string} art @returns {Promise<PlanStand>} */
export async function ladeStand(art) {
	const res = await apiFetch(`/api/lmf-plan/${art}`);
	if (!res.ok) throw new Error('LMF-Plan konnte nicht geladen werden');
	return await res.json();
}

/** Baut den bearbeitbaren Entwurf aus dem Serverstand: ein laufender Plan wird
 *  bearbeitet, sonst beginnt der nächste mit dem Vorschlag (Vorjahr oder Regel). Alles
 *  aus dem Vokabular, was in keiner Zeile steht, liegt unter „Nicht im Plan".
 *  @param {PlanStand} stand @returns {PlanEntwurf} */
export function entwurfAus(stand) {
	const laufend = stand.plan && !stand.vorbei;
	const quelle = laufend ? stand : (stand.vorschlag ?? { zeilen: [], ausgelassen: [] });
	const zeilen = quelle.zeilen.map((z) => ({ klassen: [...z.klassen], vermerk: z.vermerk ?? '' }));
	const drin = new Set(zeilen.flatMap((z) => z.klassen.map(normKey)));
	const ausgelassen = [...quelle.ausgelassen];
	for (const k of ausgelassen) drin.add(normKey(k));
	for (const k of stand.klassen ?? []) {
		if (!drin.has(normKey(k))) {
			ausgelassen.push(k);
			drin.add(normKey(k));
		}
	}
	return {
		erster_tag: laufend && stand.plan ? stand.plan.erster_tag : '',
		startstunde: laufend && stand.plan ? stand.plan.startstunde : 1,
		stunden_je_tag: laufend && stand.plan ? stand.plan.stunden_je_tag : 6,
		zeilen,
		ausgelassen: ausgelassen.sort((a, b) => a.localeCompare(b, 'de', { numeric: true }))
	};
}

/** Der Vergleichsschlüssel des Vokabulars (klassen_normkey): klein, ohne Leerzeichen,
 *  ohne führende Nullen — „05F1" und „5f1" sind dieselbe Klasse.
 *  @param {string} k */
export function normKey(k) {
	return k
		.replace(/\s+/g, '')
		.toLowerCase()
		.replace(/^0+(\d)/, '$1');
}

/** @param {string} art @param {PlanEntwurf} entwurf @param {boolean} vorschau */
async function sende(art, entwurf, vorschau) {
	const res = await apiFetch(`/api/lmf-plan/${art}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ ...entwurf, vorschau })
	});
	const json = await res.json().catch(() => ({}));
	return { res, json };
}

/** Rechnet die Plätze, ohne zu speichern. Leer, wenn der Rahmen noch unvollständig ist.
 *  @param {string} art @param {PlanEntwurf} entwurf @returns {Promise<PlanPlatz[]>} */
export async function rechneVorschau(art, entwurf) {
	if (!entwurf.erster_tag) return [];
	const { res, json } = await sende(art, entwurf, true);
	if (!res.ok) throw new Error(json.error ?? json.message ?? 'Vorschau fehlgeschlagen');
	return json.zeilen ?? [];
}

/** Speichert den Plan. Gibt die Server-Meldung zurück — nur der Server kennt den Grund
 *  einer Ablehnung und die Zahl der Ausleihen, deren Frist dem Plan gefolgt ist.
 *  @param {string} art @param {PlanEntwurf} entwurf
 *  @returns {Promise<{ ok: boolean, meldung: string }>} */
export async function speicherePlan(art, entwurf) {
	const { res, json } = await sende(art, entwurf, false);
	if (!res.ok)
		return { ok: false, meldung: json.error ?? json.message ?? 'Speichern fehlgeschlagen.' };
	const n = Number(json.fristen_angepasst ?? 0);
	return {
		ok: true,
		meldung: n > 0 ? `Plan gespeichert · Frist von ${n} Ausleihen angepasst.` : 'Plan gespeichert.'
	};
}

/** @param {string} art @returns {Promise<{ ok: boolean, meldung: string }>} */
export async function verwerfePlan(art) {
	const res = await apiFetch(`/api/lmf-plan/${art}`, { method: 'DELETE' });
	const json = await res.json().catch(() => ({}));
	if (!res.ok) return { ok: false, meldung: json.error ?? 'Verwerfen fehlgeschlagen.' };
	const n = Number(json.fristen_angepasst ?? 0);
	return {
		ok: true,
		meldung:
			n > 0
				? `Plan verworfen · Frist von ${n} Ausleihen auf den Stichtag zurückgesetzt.`
				: 'Plan verworfen.'
	};
}

/** Lädt das PDF und öffnet den Download — Verwaltung und Portal gleich.
 *  @param {boolean} [alle] */
export async function ladePdf(alle = false) {
	const res = await apiFetch(`/api/lmf-termine/pdf${alle ? '?alle=1' : ''}`);
	if (!res.ok) throw new Error('PDF konnte nicht erzeugt werden');
	const blob = await res.blob();
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = 'LMF-Plan.pdf';
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}
