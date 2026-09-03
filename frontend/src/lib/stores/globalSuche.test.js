import { describe, it, expect, vi } from 'vitest';
import { erzeugeGlobalSuche } from './globalSuche.svelte.js';

// Die Sprung-Entscheidung der globalen Suchleiste: exakter Scan → Akte; ISBN mit genau
// einem Titel → Buchakte; Name → Liste bleibt offen; späte Antwort überholt nicht.
const antwort = (body, ok = true) => Promise.resolve({ ok, json: () => Promise.resolve(body) });

describe('globale Suche', () => {
	it('springt bei Exemplar-Treffer in die Buchakte und leert das Feld', async () => {
		const zuBuch = vi.fn();
		const s = erzeugeGlobalSuche({
			zuBuch,
			zuSchueler: vi.fn(),
			holen: () =>
				antwort({ students: [], books: [], treffer: { typ: 'exemplar', id: 'e1', titel_id: 't1' } })
		});
		s.suche = 'B-1';
		await s.bestaetigen();
		expect(zuBuch).toHaveBeenCalledWith('t1');
		expect(s.suche).toBe('');
		expect(s.offen).toBe(false);
	});
	it('springt bei Ausweis in die Schülerakte', async () => {
		const zuSchueler = vi.fn();
		const s = erzeugeGlobalSuche({
			zuBuch: vi.fn(),
			zuSchueler,
			holen: () =>
				antwort({ students: [{ id: 's1' }], books: [], treffer: { typ: 'schueler', id: 's1' } })
		});
		s.suche = 'S-1';
		await s.bestaetigen();
		expect(zuSchueler).toHaveBeenCalledWith('s1');
	});
	it('lässt bei einem Namen mit mehreren Treffern die Liste offen', async () => {
		const zuSchueler = vi.fn();
		const s = erzeugeGlobalSuche({
			zuBuch: vi.fn(),
			zuSchueler,
			holen: () => antwort({ students: [{ id: 'a' }, { id: 'b' }], books: [] })
		});
		s.suche = 'Müller';
		await s.bestaetigen();
		expect(zuSchueler).not.toHaveBeenCalled();
		expect(s.offen).toBe(true);
		expect(s.schueler).toHaveLength(2);
	});
	it('verwirft eine verspätete Antwort', async () => {
		/** @type {(v: any) => void} */
		let erste = () => {};
		const holen = vi
			.fn()
			.mockImplementationOnce(() => new Promise((r) => (erste = r)))
			// zwei Treffer: kein Sprung, die Liste bleibt stehen
			.mockImplementationOnce(() =>
				antwort({ students: [{ id: 'neu' }, { id: 'neu2' }], books: [] })
			);
		const s = erzeugeGlobalSuche({ zuBuch: vi.fn(), zuSchueler: vi.fn(), holen });
		s.suche = 'Al';
		const p1 = s.bestaetigen();
		s.suche = 'Alt';
		await s.bestaetigen();
		erste({ ok: true, json: () => Promise.resolve({ students: [{ id: 'alt' }], books: [] }) });
		await p1;
		expect(s.schueler.map((x) => x.id)).toEqual(['neu', 'neu2']);
	});
	it('meldet einen Fehler statt einer leeren Liste', async () => {
		const s = erzeugeGlobalSuche({
			zuBuch: vi.fn(),
			zuSchueler: vi.fn(),
			holen: () => antwort({}, false)
		});
		s.suche = 'xy';
		await s.bestaetigen();
		expect(s.fehler).toContain('nicht möglich');
	});
});

// Rechte-Kante: Was die Leiste anbietet, muss der Klick auch öffnen dürfen. Der Router
// stellt einen gesperrten Bildschirm still auf den ersten erlaubten zurück — ein
// Schüler-Treffer für den Helfer (view_books ohne view_students) wäre eine tote Tür.
describe('globale Suche — Rechte', () => {
	it('zeigt dem Helfer keine Schüler und springt bei Ausweis-Scan nicht', async () => {
		const zuSchueler = vi.fn();
		const s = erzeugeGlobalSuche({
			zuBuch: vi.fn(),
			zuSchueler,
			darfSchueler: () => false,
			holen: () =>
				antwort({ students: [{ id: 's1' }], books: [], treffer: { typ: 'schueler', id: 's1' } })
		});
		s.suche = 'S-1';
		await s.bestaetigen();
		expect(zuSchueler).not.toHaveBeenCalled();
		expect(s.schueler).toHaveLength(0);
		expect(s.offen).toBe(false);
		expect(s.fehler).toMatch(/freigegeben/);
	});
	it('lässt dem Helfer die Bücher und springt bei genau einem Titel', async () => {
		const zuBuch = vi.fn();
		const s = erzeugeGlobalSuche({
			zuBuch,
			zuSchueler: vi.fn(),
			darfSchueler: () => false,
			holen: () => antwort({ students: [{ id: 's1' }], books: [{ id: 'b1' }] })
		});
		s.suche = 'Meier';
		await s.bestaetigen();
		expect(zuBuch).toHaveBeenCalledWith('b1');
	});
});

// Rasterdurchgang 03.09.2026, Frage 6 (Zeit): Der „zu kurz"-Zweig von tippen() leerte die
// Listen, verwarf aber laufende Antworten nicht. Wer „Meier" tippt und sofort auf „M"
// zurücklöscht, bekam die Treffer zu „Meier" in die eben geleerte Liste zurück — und die
// Liste klappte wieder auf, zu einer Eingabe, die nicht mehr im Feld steht.
describe('globale Suche — späte Antwort nach dem Leeren', () => {
	it('füllt die Liste nicht mehr, wenn das Feld inzwischen zu kurz ist', async () => {
		/** @type {(v: any) => void} */
		let antworte = () => {};
		const s = erzeugeGlobalSuche({
			zuBuch: vi.fn(),
			zuSchueler: vi.fn(),
			holen: () => new Promise((r) => (antworte = r))
		});
		s.suche = 'Meier';
		const unterwegs = s.bestaetigen();
		s.suche = 'M';
		s.tippen();
		antworte({
			ok: true,
			json: () => Promise.resolve({ students: [{ id: 'a' }, { id: 'b' }], books: [] })
		});
		await unterwegs;
		expect(s.schueler).toHaveLength(0);
		expect(s.offen).toBe(false);
		expect(s.fehler).toBe('');
	});
});
