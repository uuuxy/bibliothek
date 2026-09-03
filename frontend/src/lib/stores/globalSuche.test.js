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
