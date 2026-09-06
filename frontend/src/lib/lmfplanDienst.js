/** Serverwege des LMF-Plans — Lesen, Vorschau, Speichern, Verwerfen, PDF — und die
 *  kleinen Darstellungsregeln, die Planer und Portal-Reiter teilen.
 *
 *  Der Plan ist eine REIHENFOLGE von Klassen, die der Server auf Schultage × Stunden
 *  gießt (Peter, 05.09.2026, am echten Plan der Schule): Rahmen + Zeilen hin, Plätze
 *  zurück — auch die Vorschau rechnet der Server, damit es keinen JavaScript-Zwilling
 *  der Verteilung gibt. Das Kollegium liest das Ergebnis im Portal, für alle gleich. */
import { apiFetch } from './apiFetch.js';
import { einordnen } from './lmfplanZeilen.js';

/** @typedef {{ id?: string, datum: string, stunde: number, art: 'rueckgabe' | 'ausgabe', klassen: string[], vermerk: string }} LmfTermin */
/** Fest: Datum und Stunde von Hand (die Klasse mit dem Ausflug) — null, wenn die Zeile fließt. */
/** @typedef {{ datum: string, stunde: number }} FesterPlatz */
/** @typedef {{ klassen: string[], vermerk: string, fest?: FesterPlatz | null }} PlanZeile */
/** @typedef {{ datum: string, grund: string }} FreierTag */
/** Der Rahmen hängt an der Art (Migration 101): der Büchertausch ENDET (letzter_tag,
 *  letzte_stunde — Donnerstag vor den Sommerferien, 4. Stunde), die Bücherausgabe BEGINNT
 *  (erster_tag, startstunde). Der Entwurf trägt beide Paare; gesendet wird, was die Art
 *  braucht, der Server verwirft den Rest.
 *  @typedef {{ erster_tag: string, startstunde: number, letzter_tag: string, letzte_stunde: number, stunden_je_tag: number, freie_tage: FreierTag[], zeilen: PlanZeile[], ausgelassen: string[] }} PlanEntwurf */
/** @typedef {{ erster_tag: string, startstunde: number, letzter_tag: string, letzte_stunde: number, stunden_je_tag: number }} RahmenVorgabe */
/** @typedef {{ jahr: number, von: string, bis: string, bekannt: boolean }} Sommerferien */
/** @typedef {{ position: number, datum: string, stunde: number, fest: boolean, klassen: string[], vermerk: string }} PlanPlatz */
/** Ein Werktag im Plan-Zeitraum, an dem der Plan nicht läuft — mit Grund. */
/** @typedef {{ datum: string, grund: string }} Ausfall */
/** veroeffentlicht_am: null = Entwurf (nur im Planer), sonst der Stempel (Migration 100). klassen = Klassen mit
 *  Schülern; eingangsjahrgaenge aus der Einstellung. */
/** @typedef {{ plan: { id: string, art: string, erster_tag: string, startstunde: number, letzter_tag: string, letzte_stunde: number, stunden_je_tag: number, freie_tage: FreierTag[], veroeffentlicht_am?: string | null } | null, zeilen: PlanPlatz[], ausgelassen: string[], vorbei: boolean, vorschlag?: { quelle: 'vorjahr' | 'regel', zeilen: PlanZeile[], ausgelassen: string[], rahmen?: RahmenVorgabe }, klassen: string[], ausgelassen_regel?: string[], eingangsjahrgaenge?: number[], sommerferien?: Sommerferien }} PlanStand */

/** Die zwei Pläne — mit den Worten, die sagen, was passiert (Peter, 06.09.2026: „Rückgabe"
 *  und „Ausgabe" allein waren unklar, das sind zwei verschiedene Dinge zu verschiedenen
 *  Zeiten). Dieselben Titel schreibt das PDF (api/lmf_termine.go, LmfArtTitel). */
export const ARTEN = /** @type {const} */ ([
	{ wert: 'rueckgabe', label: 'Büchertausch vor den Sommerferien' },
	{ wert: 'ausgabe', label: 'Bücherausgabe nach den Sommerferien' }
]);

/** „5 und 7", „5, 7 und 11".
 *  @param {number[] | undefined} jahrgaenge */
export function jahrgaengeText(jahrgaenge) {
	const j = (jahrgaenge ?? []).map(String);
	if (j.length <= 1) return j.join('');
	return `${j.slice(0, -1).join(', ')} und ${j[j.length - 1]}`;
}

/** Der eine Satz, der erklärt, was in einem Plan geschieht — im Planer, im Portal und
 *  (gleichlautend) im PDF.
 *  @param {string} art @param {number[] | undefined} eingang */
export function artErklaerung(art, eingang) {
	if (art === 'ausgabe')
		return `Nur die neu gebildeten Klassen (Jahrgang ${jahrgaengeText(eingang)}) bekommen ihre Schulbücher.`;
	return 'Alle Klassen geben die alten Schulbücher ab und bekommen direkt die neuen. „Nur Rückgabe“: Abschlussklassen und Klassen, die zum neuen Schuljahr neu gebildet werden.';
}

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

/** Lädt die Termine veröffentlichter Pläne ab Schuljahresbeginn (alle = true: auch
 *  ältere) — die Tabelle des Portals und der PDF.
 *  @param {boolean} [alle]
 *  @returns {Promise<{ ab: string, termine: LmfTermin[], ohne_rueckgabe_termin: string[], eingangsjahrgaenge?: number[] }>} */
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

/** Der leere Entwurf — der Zustand des Planers vor dem ersten Laden.
 *  @returns {PlanEntwurf} */
export function leererEntwurf() {
	return {
		erster_tag: '',
		startstunde: 1,
		letzter_tag: '',
		letzte_stunde: 4,
		stunden_je_tag: 6,
		freie_tage: [],
		zeilen: [],
		ausgelassen: []
	};
}

/** Baut den bearbeitbaren Entwurf aus dem Serverstand: ein laufender Plan wird
 *  bearbeitet, sonst beginnt der nächste mit dem Vorschlag (Vorjahr oder Regel) und dem
 *  Rahmen aus den Sommerferien (Donnerstag davor, 4. Stunde; erster Schultag danach). Alles
 *  aus dem Vokabular, was in keiner Zeile steht, liegt unter „Nicht im Plan". Feste
 *  Plätze und freie Tage gehören zum laufenden Plan — der Vorschlag fürs nächste Jahr
 *  bringt sie nicht mit (der Ausflug war dieses Jahr).
 *  @param {PlanStand} stand @returns {PlanEntwurf} */
export function entwurfAus(stand) {
	const laufend = stand.plan && !stand.vorbei;
	const quelle = laufend ? stand : (stand.vorschlag ?? { zeilen: [], ausgelassen: [] });
	const zeilen = quelle.zeilen.map((z) => ({
		klassen: [...z.klassen],
		vermerk: z.vermerk ?? '',
		fest: laufend && 'fest' in z && z.fest ? { datum: z.datum, stunde: z.stunde } : null
	}));
	const drin = new Set(zeilen.flatMap((z) => z.klassen.map(normKey)));
	const ausgelassen = [...quelle.ausgelassen];
	for (const k of ausgelassen) drin.add(normKey(k));
	for (const k of stand.klassen ?? []) {
		if (!drin.has(normKey(k))) {
			ausgelassen.push(k);
			drin.add(normKey(k));
		}
	}
	const rahmen = laufend && stand.plan ? stand.plan : (stand.vorschlag?.rahmen ?? leererEntwurf());
	return {
		erster_tag: rahmen.erster_tag,
		startstunde: rahmen.startstunde,
		letzter_tag: rahmen.letzter_tag,
		letzte_stunde: rahmen.letzte_stunde,
		stunden_je_tag: rahmen.stunden_je_tag,
		freie_tage: laufend && stand.plan ? [...(stand.plan.freie_tage ?? [])] : [],
		zeilen,
		ausgelassen: ausgelassen.sort((a, b) => a.localeCompare(b, 'de', { numeric: true }))
	};
}

/** Eine Frage an jede Klasse im Planer (Peter, 06.09.2026: Klassen wechseln mit dem
 *  Schuljahr — mal 3, mal 4, mal 6 je Stufe und Zweig): Hat sie schon Schüler? Ein
 *  „07G6" aus dem Vorjahr oder ein vor dem August-Import getipptes „07G1" hat keine —
 *  es kommt mit dem LUSD-Import oder gehört aus dem Plan. Der Server beantwortet sie,
 *  hier wird nur nachgeschlagen — über den Normschlüssel. („Nur Rückgabe" ist kein
 *  Marker mehr, sondern Text im Vermerk, den der Vorschlag vorbelegt.)
 *  @param {PlanStand | null} stand */
export function klassenMarker(stand) {
	const mitSchuelern = new Set((stand?.klassen ?? []).map(normKey));
	return {
		/** @param {string} k */ ohneSchueler: (k) => !mitSchuelern.has(normKey(k))
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

/** Hat der Entwurf den Anker seiner Art — das Ende beim Büchertausch, den Beginn bei
 *  der Bücherausgabe? Erst dann gibt es Plätze zu rechnen und etwas zu speichern.
 *  @param {string} art @param {PlanEntwurf} e */
export function ankerGesetzt(art, e) {
	return Boolean(art === 'rueckgabe' ? e.letzter_tag : e.erster_tag);
}

/** Rechnet die Plätze und die Ausfälle (Feiertage, freie Tage) ohne zu speichern, dazu
 *  den gerechneten Beginn (beim Büchertausch: wo der Plan anfängt). Leer, wenn der
 *  Anker noch fehlt.
 *  @param {string} art @param {PlanEntwurf} entwurf
 *  @returns {Promise<{ plaetze: PlanPlatz[], ausfaelle: Ausfall[], beginn: { datum: string, stunde: number } | null }>} */
export async function rechneVorschau(art, entwurf) {
	if (!ankerGesetzt(art, entwurf)) return { plaetze: [], ausfaelle: [], beginn: null };
	const { res, json } = await sende(art, entwurf, true);
	if (!res.ok) throw new Error(json.error ?? json.message ?? 'Vorschau fehlgeschlagen');
	const beginn = json.plan?.erster_tag
		? { datum: json.plan.erster_tag, stunde: json.plan.startstunde }
		: null;
	return { plaetze: json.zeilen ?? [], ausfaelle: json.ausfaelle ?? [], beginn };
}

/** Alles am Entwurf, wovon die Plätze abhängen — als EIN Text, den die Vorschau
 *  beobachtet: Rahmen, freie Tage, Anzahl der Zeilen und je Zeile der feste Platz.
 *  Klassen und Vermerke stehen nicht drin: Sie ändern keinen Platz. Ein fester Platz
 *  dagegen verschiebt die Zeilen um ihn herum, und sein Wechsel von Zeile zu Zeile
 *  (Umsortieren) ebenfalls.
 *  @param {PlanEntwurf} e */
export function vorschauSchluessel(e) {
	return [
		e.erster_tag,
		e.startstunde,
		e.letzter_tag,
		e.letzte_stunde,
		e.stunden_je_tag,
		e.freie_tage.map((t) => t.datum).join(','),
		e.zeilen.map((z) => (z.fest ? `${z.fest.datum}/${z.fest.stunde}` : '-')).join('|')
	].join(';');
}

/** Die Plätze der Zeilen dürfen erst gespeichert werden, wenn jeder feste Platz ein
 *  Datum hat — ein fester Termin ohne Tag ist kein „fließt eben".
 *  @param {PlanZeile[]} zeilen */
export function festePlaetzeVollstaendig(zeilen) {
	return zeilen.every((z) => !z.fest || (Boolean(z.fest.datum) && z.fest.stunde >= 1));
}

/** Speichert den Plan. Gibt die Server-Meldung zurück — nur der Server kennt den Grund
 *  einer Ablehnung und die Zahl der Ausleihen, deren Frist dem Plan gefolgt ist. Ein
 *  unveröffentlichter Plan bleibt Entwurf (Migration 100); die Meldung sagt es.
 *  @param {string} art @param {PlanEntwurf} entwurf
 *  @returns {Promise<{ ok: boolean, meldung: string }>} */
export async function speicherePlan(art, entwurf) {
	const { res, json } = await sende(art, entwurf, false);
	if (!res.ok)
		return { ok: false, meldung: json.error ?? json.message ?? 'Speichern fehlgeschlagen.' };
	if (!json.plan?.veroeffentlicht_am) return { ok: true, meldung: 'Entwurf gespeichert.' };
	const n = Number(json.fristen_angepasst ?? 0);
	return {
		ok: true,
		meldung: n > 0 ? `Plan gespeichert · Frist von ${n} Ausleihen angepasst.` : 'Plan gespeichert.'
	};
}

/** Veröffentlicht den gespeicherten Plan der Art: ab jetzt sehen ihn Portal und PDF, und
 *  bei einem Rückgabe-Plan folgen die Fristen der Klassen (die Meldung nennt die Zahl).
 *  @param {string} art @returns {Promise<{ ok: boolean, meldung: string }>} */
export async function veroeffentlichePlan(art) {
	const res = await apiFetch(`/api/lmf-plan/${art}/veroeffentlichen`, { method: 'POST' });
	const json = await res.json().catch(() => ({}));
	if (!res.ok) return { ok: false, meldung: json.error ?? 'Veröffentlichen fehlgeschlagen.' };
	const n = Number(json.fristen_angepasst ?? 0);
	return {
		ok: true,
		meldung:
			n > 0 ? `Plan veröffentlicht · Frist von ${n} Ausleihen angepasst.` : 'Plan veröffentlicht.'
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

/** Lädt das PDF und öffnet den Download — Verwaltung und Portal gleich. entwurf = true
 *  (nur der Planer, edit_books) nimmt den unveröffentlichten Entwurf mit: das PDF für
 *  die Abnahme durch die Schulleitung.
 *  @param {boolean} [alle] @param {boolean} [entwurf] */
export async function ladePdf(alle = false, entwurf = false) {
	const pfad = entwurf ? '/api/lmf-termine/entwurf/pdf' : '/api/lmf-termine/pdf';
	const res = await apiFetch(`${pfad}${alle ? '?alle=1' : ''}`);
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

/** Welche Klassen im Planer eingeklappt unter „bleiben draußen" stehen: die, die der
 *  gespeicherte Plan (laufend) oder der Vorschlag (Vorjahr, Regel) bewusst auslässt,
 *  UND die, die die Regel der Art auslässt (Server: ausgelassen_regel — die Oberstufe,
 *  beim Ausgabe-Plan alles außer den Eingangsjahrgängen). Letzteres auch bei einem
 *  laufenden Plan, der die Klassen nie kannte: Ein Plan mit alten Klassennamen bot
 *  sonst 60 Chips offen an (06.09.2026). Was übrig bleibt, hat keine Regel und steht
 *  offen unter „Noch nicht im Plan": die neue Klasse nach dem LUSD-Import.
 *  @param {PlanStand | null} stand @returns {(klasse: string) => boolean} */
export function bewusstDraussen(stand) {
	const laufend = stand?.plan && !stand.vorbei;
	const liste = laufend ? (stand?.ausgelassen ?? []) : (stand?.vorschlag?.ausgelassen ?? []);
	const menge = new Set([...liste, ...(stand?.ausgelassen_regel ?? [])].map(normKey));
	return (k) => menge.has(normKey(k));
}

/** Nimmt eine Klasse in den Entwurf: aus „Nicht im Plan" heraus und — wenn sie in
 *  keiner Zeile steht — als eigene Zeile hinein, vor `vor` (Ziehen auf eine Zeile) oder
 *  nach der Nachbar-Regel (lmfplanZeilen.einordnen). `index` ist die neue Zeile, oder
 *  null, wenn die Klasse schon im Plan stand.
 *  @param {PlanEntwurf} e @param {string} k @param {number} [vor]
 *  @returns {{ entwurf: PlanEntwurf, index: number | null }} */
export function klasseHinein(e, k, vor) {
	const ausgelassen = e.ausgelassen.filter((x) => normKey(x) !== normKey(k));
	if (e.zeilen.some((z) => z.klassen.some((x) => normKey(x) === normKey(k))))
		return { entwurf: { ...e, ausgelassen }, index: null };
	const { zeilen, index } = einordnen(e.zeilen, k, vor);
	return { entwurf: { ...e, ausgelassen, zeilen }, index };
}

/** Merkt eine Klasse als ausgelassen (die Zeile nimmt LmfPlanReihenfolge selbst weg).
 *  @param {PlanEntwurf} e @param {string} k @returns {PlanEntwurf} */
export function klasseRaus(e, k) {
	if (e.ausgelassen.some((x) => normKey(x) === normKey(k))) return e;
	return {
		...e,
		ausgelassen: [...e.ausgelassen, k].sort((a, b) => a.localeCompare(b, 'de', { numeric: true }))
	};
}
