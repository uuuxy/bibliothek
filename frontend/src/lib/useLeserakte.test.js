import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useLeserakte } from './useLeserakte.svelte.js';
import { apiFetch } from './apiFetch.js';
import { snackbars } from './stores/snackbars.svelte.js';

// Die Akte lädt vier Dinge nebeneinander: Stammdaten, Vormerkungen, Gebühren, Bescheide.
//
// Bis zum Rasterdurchgang am 06.09.2026 stand dort dreimal `if (res.ok)` ohne `else` und
// ohne Sequenznummer. Fiel genau eine der drei Anfragen aus — 500, oder 429 vom
// Rate-Limiter, es sind drei parallele Anfragen je Akte —, behielt dieser Teil die Werte
// des VORHER geöffneten Schülers, während Kopf und Ausleihen schon zum neuen gehörten.
//
// Bei den Gebühren ist das nicht nur Anzeige: LeserGebuehrenKarte schreibt auf die
// Fall-ID der Zeile („Zahlung verbucht", „Storno"). Ein Klick auf der Akte von B hätte
// die Zahlung einem Schadensfall von A gutgeschrieben — Kopf, Ausdruck und Audit zeigen B.
vi.mock('./apiFetch.js', async (original) => ({
	// leseApiFehler bleibt die ECHTE Funktion: Der Test unten prüft genau, was sie aus
	// einer Fehlerantwort macht.
	...(await original()),
	apiFetch: vi.fn()
}));

/** @param {any} body */
const ok = (body) => ({ ok: true, json: async () => body });
const fehler = { ok: false, status: 500, json: async () => ({}) };

describe('useLeserakte.ladeAkte', () => {
	// Mit Klammern, ohne Rückgabe: mockReset() gibt das Mock zurück, und Vitest ruft einen
	// zurückgegebenen Funktionswert nach dem Test als Aufräumer auf — apiFetch() ohne
	// Argumente. Solange jede Implementierung auflöste, fiel das nicht auf; die erste
	// ablehnende (Netzwerkfehler-Fall, 15.09.2026) ließ den Aufräumer werfen.
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('lässt die Gebühren des vorigen Schülers nicht stehen', async () => {
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			if (u.includes('schadensfaelle')) return /** @type {any} */ (ok([{ id: 'f-A' }]));
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: 'A', vorname: 'Anna' }));
		});
		const st = useLeserakte();
		await st.ladeAkte('A');
		expect(st.gebuehren).toEqual([{ id: 'f-A' }]);

		// Schüler B: die Gebühren-Anfrage scheitert.
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			if (u.includes('schadensfaelle')) return /** @type {any} */ (fehler);
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: 'B', vorname: 'Ben' }));
		});
		await st.ladeAkte('B');
		expect(st.akte?.id).toBe('B');
		expect(st.gebuehren, 'Gebühren von A stehen unter dem Namen von B').toEqual([]);
	});

	// Dieselbe Klasse für die Bescheide (seit 12.09.2026 in der Akte): Ein Bescheid nennt
	// Referenznummer, Betrag und Frist. Der von A unter dem Namen von B wäre die Auskunft,
	// mit der Eltern in der Bibliothek stehen.
	it('lässt die Bescheide des vorigen Schülers nicht stehen', async () => {
		/** @param {string} id */
		const antworten = (id, bescheideOk) =>
			vi.mocked(apiFetch).mockImplementation(async (url) => {
				const u = String(url);
				if (u.includes('bescheide')) {
					return /** @type {any} */ (bescheideOk ? ok([{ id: 'bescheid-' + id }]) : fehler);
				}
				if (u.includes('schadensfaelle')) return /** @type {any} */ (ok([]));
				if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
				return /** @type {any} */ (ok({ id, vorname: id }));
			});

		const st = useLeserakte();
		antworten('A', true);
		await st.ladeAkte('A');
		expect(st.bescheide).toEqual([{ id: 'bescheid-A' }]);

		antworten('B', false);
		await st.ladeAkte('B');
		expect(st.akte?.id).toBe('B');
		expect(st.bescheide, 'der Bescheid von A steht unter dem Namen von B').toEqual([]);
	});

	// Ein gescheiterter Abruf ist kein „kein Bescheid" (Review 14.09.2026, OFFEN.md 1.5): Eltern
	// stehen mit dem Brief in der Bibliothek, GET …/bescheide läuft in 503 — die Akte zeigte
	// keine Karte, und die Auskunft lautete „bei uns liegt kein Bescheid vor". Seit dem
	// 15.09.2026 steht der Ausfall in fehlendeListen, wie in der Buch-Akte (useBuchakte).
	/** @param {Record<string, any>} je Antwort je Teil-URL; ohne Eintrag: ok und leer */
	const antwortenJe = (je) =>
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			for (const [teil, antwort] of Object.entries(je)) if (u.includes(teil)) return antwort;
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			if (u.includes('schadensfaelle') || u.includes('bescheide'))
				return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: 'A', vorname: 'Anna' }));
		});

	it('vermerkt einen gescheiterten Abruf, statt ihn als „nichts da" zu zeigen', async () => {
		antwortenJe({ bescheide: { ok: false, status: 503, json: async () => ({}) } });
		const st = useLeserakte();
		await st.ladeAkte('A');
		expect(st.bescheide).toEqual([]);
		expect(st.fehlendeListen, 'der Ausfall muss in der Akte sichtbar sein').toEqual(['Bescheide']);

		antwortenJe({
			schadensfaelle: fehler,
			vormerkungen: { ok: false, status: 504, json: async () => ({}) }
		});
		await st.ladeAkte('A');
		expect(st.fehlendeListen).toEqual(['Vormerkungen', 'Gebühren']);
	});

	it('403 (Rolle ohne Einsicht) und eine leere Liste sind kein Ladefehler', async () => {
		antwortenJe({ bescheide: { ok: false, status: 403, json: async () => ({}) } });
		const st = useLeserakte();
		await st.ladeAkte('A');
		expect(st.bescheide).toEqual([]);
		expect(st.fehlendeListen).toEqual([]);

		antwortenJe({});
		await st.ladeAkte('A');
		expect(st.fehlendeListen).toEqual([]);
	});

	it('ein Netzwerkfehler lässt keinen Vermerk des vorigen Laufs stehen und setzt alle', async () => {
		antwortenJe({ bescheide: { ok: false, status: 503, json: async () => ({}) } });
		const st = useLeserakte();
		await st.ladeAkte('A');
		// Der Netzwerkfehler wird im Hook protokolliert (console.error) — hier stumm, wie in liveEvents.test.js.
		vi.spyOn(console, 'error').mockImplementation(() => {});
		vi.mocked(apiFetch).mockRejectedValue(new TypeError('Failed to fetch'));
		await st.ladeAkte('B');
		expect(st.fehlendeListen).toEqual(['Vormerkungen', 'Gebühren', 'Bescheide']);
	});

	it('lässt die überholte Antwort nicht gewinnen', async () => {
		/** @type {() => void} */
		let loesenA = () => {};
		const aKam = new Promise((r) => (loesenA = /** @type {any} */ (r)));
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			const istA = u.includes('/A');
			if (istA) await aKam;
			if (u.includes('schadensfaelle')) return /** @type {any} */ (ok([]));
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: istA ? 'A' : 'B', vorname: istA ? 'Anna' : 'Ben' }));
		});
		const st = useLeserakte();
		const langsam = st.ladeAkte('A');
		await st.ladeAkte('B');
		expect(st.akte?.id).toBe('B');
		loesenA();
		await langsam;
		expect(st.akte?.id, 'die ältere Antwort hat die jüngere überschrieben').toBe('B');
		expect(st.laedt).toBe(false);
	});

	// Auch das Lesen eines Rumpfs dauert: Die Antworten zu A können schon da sein, während der
	// Rumpf ihrer Schadensfälle erst nach der ganzen Akte von B gelesen ist.
	it('mischt keine Schadensfälle des vorigen Lesers in die Akte, deren Rumpf zuletzt gelesen ist', async () => {
		/** @type {(daten: unknown) => void} */
		let gibFaelleA = () => {};
		const faelleA = vi.fn(() => new Promise((ja) => (gibFaelleA = ja)));
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			const istA = u.includes('/A');
			if (u.includes('schadensfaelle'))
				return /** @type {any} */ (istA ? { ok: true, json: faelleA } : ok([{ id: 'fall-B' }]));
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			if (u.includes('bescheide')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: istA ? 'A' : 'B' }));
		});
		const st = useLeserakte();
		const langsam = st.ladeAkte('A');
		await vi.waitFor(() => expect(faelleA).toHaveBeenCalled());
		await st.ladeAkte('B');
		gibFaelleA([{ id: 'fall-A' }]);
		await langsam;

		expect(st.akte?.id).toBe('B');
		expect(
			st.gebuehren.map((/** @type {any} */ f) => f.id),
			'„Zahlung verbucht" schriebe auf den Schadensfall von A'
		).toEqual(['fall-B']);
	});

	it('schließt beim Schülerwechsel jedes offene Blatt', () => {
		const st = useLeserakte();
		st.bearbeitenOffen = true;
		st.schadenOffen = true;
		st.sperreOffen = true;
		st.loeschfrageOffen = true;
		st.kameraOffen = true;
		st.schliesseAlleBlaetter();
		expect([
			st.bearbeitenOffen,
			st.schadenOffen,
			st.sperreOffen,
			st.loeschfrageOffen,
			st.kameraOffen
		]).toEqual([false, false, false, false, false]);
	});
});

// Der Mensch an der Theke liest den Satz des Servers, nicht seinen JSON-Rumpf.
//
// Fund (OFFEN.md 5.12): Die Ersatzforderung antwortet mit {"error": "…"}, und der Toast
// zeigte `Error: {"error":"…"}` — mit Klammern und Anführungszeichen. Wer an der Theke
// steht, soll lesen können, was los ist.
describe('useLeserakte: Fehlermeldungen', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		snackbars.toasts.forEach((t) => snackbars.removeToast(t.id));
	});

	it('zeigt den Satz des Servers statt seines JSON-Rumpfs', async () => {
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			if (u.includes('/api/print/rechnung/')) {
				return /** @type {any} */ ({
					ok: false,
					status: 404,
					text: async () => JSON.stringify({ error: 'Keine offene Forderung ohne Bescheid.' })
				});
			}
			return /** @type {any} */ ({ ok: true, json: async () => ({ id: 'A', vorname: 'Anna' }) });
		});

		const hook = useLeserakte();
		await hook.ladeAkte('A');
		await hook.ladeRechnungHerunter();

		// Über die Meldungen der Anwendung, nicht über eine eigene Fläche der Akte.
		const meldung = snackbars.toasts.at(-1);
		expect(meldung?.type).toBe('error');
		expect(meldung?.message).toBe('Keine offene Forderung ohne Bescheid.');
	});
});
