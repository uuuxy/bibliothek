import { apiFetch } from '../apiFetch.js';

/**
 * Logik der globalen Suchleiste („Springen"): entprellt, Antwort-Reihenfolge per
 * Sequenznummer (wie orderStore/omnibox), Fehlerzustand statt leerer Liste. Enter
 * entscheidet: exakter Scan-Treffer (Exemplar, Ausweis) → springen; genau ein Titel bei
 * ISBN-Eingabe → springen; sonst bleibt die Liste offen. Sie BUCHT nie — GET /api/search.
 * Rechte-Kante: /api/search hängt an perform_actions und liefert dem Helfer Schüler, die
 * Schülerdatei aber braucht view_students — der Router stellt gesperrte Ziele still zurück.
 * Darum bietet die Leiste nur an, was der Klick auch öffnen darf (darfSchueler/darfBuecher).
 * @param {{ zuBuch: (titelId: string) => void, zuSchueler: (id: string) => void, darfSchueler?: () => boolean, darfBuecher?: () => boolean, holen?: (url: string) => Promise<{ ok: boolean, json: () => Promise<any> }> }} ziele
 */
export function erzeugeGlobalSuche({
	zuBuch,
	zuSchueler,
	darfSchueler = () => true,
	darfBuecher = () => true,
	holen = apiFetch
}) {
	let suche = $state('');
	/** @type {any[]} */
	let schueler = $state.raw([]);
	/** @type {any[]} */
	let buecher = $state.raw([]);
	let fehler = $state('');
	let offen = $state(false);
	let lauf = 0;
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;

	function leeren() {
		clearTimeout(timer);
		lauf++;
		suche = '';
		schueler = [];
		buecher = [];
		fehler = '';
		offen = false;
	}

	/** @param {string} q */
	async function abfragen(q) {
		const seq = ++lauf;
		try {
			const res = await holen(`/api/search?q=${encodeURIComponent(q)}`);
			if (seq !== lauf) return null;
			if (!res.ok) {
				fehler = 'Suche nicht möglich.';
				return null;
			}
			const roh = await res.json();
			if (seq !== lauf) return null;
			const data = nurErlaubtes(roh);
			schueler = data.students;
			buecher = data.books;
			offen = schueler.length > 0 || buecher.length > 0;
			const verborgen = !offen && ((roh.students ?? []).length > 0 || (roh.books ?? []).length > 0);
			fehler = verborgen ? 'Treffer nur in Bereichen, die dir nicht freigegeben sind.' : '';
			return data;
		} catch {
			if (seq === lauf) fehler = 'Suche nicht möglich.';
			return null;
		}
	}

	/** Was der Klick nicht öffnen darf, wird gar nicht erst angeboten. @param {any} roh */
	function nurErlaubtes(roh) {
		const t = roh.treffer;
		const trefferErlaubt =
			t && ((t.typ === 'schueler' && darfSchueler()) || (t.typ !== 'schueler' && darfBuecher()));
		return {
			students: darfSchueler() ? (roh.students ?? []) : [],
			books: darfBuecher() ? (roh.books ?? []) : [],
			treffer: trefferErlaubt ? t : undefined
		};
	}

	function tippen() {
		clearTimeout(timer);
		const q = suche.trim();
		if (q.length < 2) {
			schueler = [];
			buecher = [];
			offen = false;
			return;
		}
		timer = setTimeout(() => abfragen(q), 250);
	}

	/** Enter oder Scanner-Abschluss: exakt springen, sonst Liste zeigen. */
	async function bestaetigen() {
		clearTimeout(timer);
		const q = suche.trim();
		if (!q) return;
		const data = await abfragen(q);
		if (!data) return;
		springeWennEindeutig(q, data);
	}

	/** @param {string} q @param {any} data */
	function springeWennEindeutig(q, data) {
		const t = data.treffer;
		if (t?.typ === 'exemplar' && t.titel_id) return wechsle(() => zuBuch(t.titel_id));
		if (t?.typ === 'schueler') return wechsle(() => zuSchueler(t.id));
		const isbn = q.replace(/-/g, '');
		const b = data.books ?? [];
		if (/^\d{10}(\d{3})?$/.test(isbn) && b.length === 1) return wechsle(() => zuBuch(b[0].id));
		if (b.length === 1 && (data.students ?? []).length === 0) return wechsle(() => zuBuch(b[0].id));
		if ((data.students ?? []).length === 1 && b.length === 0)
			return wechsle(() => zuSchueler(data.students[0].id));
	}

	/** @param {() => void} ziel */
	function wechsle(ziel) {
		leeren();
		ziel();
	}

	return {
		get suche() {
			return suche;
		},
		set suche(v) {
			suche = v;
		},
		get schueler() {
			return schueler;
		},
		get buecher() {
			return buecher;
		},
		get fehler() {
			return fehler;
		},
		get offen() {
			return offen;
		},
		tippen,
		bestaetigen,
		leeren,
		/** @param {any} s */
		waehleSchueler: (s) => wechsle(() => zuSchueler(s.id)),
		/** @param {any} b */
		waehleBuch: (b) => wechsle(() => zuBuch(b.id))
	};
}
