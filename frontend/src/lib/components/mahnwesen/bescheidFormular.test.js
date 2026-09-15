import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../apiFetch.js', () => ({ apiGet: vi.fn(), apiPost: vi.fn() }));

import { apiGet, apiPost } from '../../apiFetch.js';
import { BescheidFormular, zeilenAus } from './bescheidFormular.svelte.js';

// Stufe 2 des Mahnverfahrens: Der Dialog zeigt überfällige Bücher ohne Forderung und
// offene Forderungen in EINER Liste, schickt sie aber getrennt, wie der Server sie kennt
// — die Bücher bucht er als Verlust, die Forderungen ordnet er zu.

const vorschlag = {
	schueler_name: 'Test Verlustkind',
	frist_bis: '2026-10-13',
	fehlende_angaben: [],
	positionen: [
		{
			schadensfall_id: 'f1',
			art: 'beschaedigt',
			titel: 'Mathe 7',
			betrag: 12,
			ist_lernmittel: true
		},
		{ schadensfall_id: 'f2', art: 'beschaedigt', titel: 'Roman', betrag: 5, ist_lernmittel: false }
	],
	ausleihen: [
		{
			ausleihe_id: 'a1',
			titel: 'Physik 8',
			betrag: 24.9,
			ist_lernmittel: true,
			faellig_seit: '2026-08-15'
		}
	]
};

describe('zeilenAus', () => {
	it('führt Bücher und Forderungen mit Schlüssel und Quelle zusammen, Bücher zuerst', () => {
		const zeilen = zeilenAus(vorschlag);
		expect(zeilen.map((z) => [z.key, z.quelle, z.art])).toEqual([
			['a1', 'ausleihe', 'nicht_zurueckgegeben'],
			['f1', 'forderung', 'beschaedigt'],
			['f2', 'forderung', 'beschaedigt']
		]);
	});

	it('kommt mit einem Vorschlag ohne Listen aus', () => {
		expect(zeilenAus(null)).toEqual([]);
		expect(zeilenAus({})).toEqual([]);
	});
});

describe('BescheidFormular', () => {
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiPost).mockReset();
	});

	it('wählt nach dem Laden nur Lernmittel vor und übernimmt Frist und Beträge', async () => {
		vi.mocked(apiGet).mockResolvedValue(vorschlag);
		const f = new BescheidFormular('s1');
		await f.laden();
		expect(f.laedt).toBe(false);
		expect(f.frist).toBe('2026-10-13');
		expect(f.gewaehlt).toEqual({ a1: true, f1: true, f2: false });
		expect(f.betraege).toEqual({ a1: 24.9, f1: 12, f2: 5 });
		expect(f.summe).toBeCloseTo(36.9);
		expect(f.bereit).toBe(true);
	});

	it('trennt im Rumpf Bücher und Forderungen mit dem eingetragenen Betrag', async () => {
		vi.mocked(apiGet).mockResolvedValue(vorschlag);
		const f = new BescheidFormular('s1');
		await f.laden();
		f.betraege.a1 = 20;
		expect(f.rumpf()).toEqual({
			mittel: 'land',
			frist_bis: '2026-10-13',
			positionen: [{ schadensfall_id: 'f1', betrag: 12 }],
			ausleihen: [{ ausleihe_id: 'a1', betrag: 20 }]
		});
	});

	it('ist ohne Auswahl, ohne Frist oder mit fehlenden Angaben nicht bereit', async () => {
		vi.mocked(apiGet).mockResolvedValue({ ...vorschlag, fehlende_angaben: ['Schulnummer'] });
		const f = new BescheidFormular('s1');
		await f.laden();
		expect(f.bereit).toBe(false);

		vi.mocked(apiGet).mockResolvedValue(vorschlag);
		const g = new BescheidFormular('s1');
		await g.laden();
		g.gewaehlt.a1 = false;
		g.gewaehlt.f1 = false;
		expect(g.bereit).toBe(false);
		g.gewaehlt.a1 = true;
		g.frist = '';
		expect(g.bereit).toBe(false);
	});

	it('schickt beim Erstellen den Rumpf und liefert den Brief', async () => {
		vi.mocked(apiGet).mockResolvedValue(vorschlag);
		vi.mocked(apiPost).mockResolvedValue({ id: 'b1', referenznummer: '5830 2026 1234 0001' });
		const f = new BescheidFormular('s1');
		await f.laden();
		const brief = await f.erstellen();
		expect(brief?.referenznummer).toBe('5830 2026 1234 0001');
		expect(vi.mocked(apiPost)).toHaveBeenCalledWith('/api/schueler/s1/bescheide', f.rumpf());
		expect(f.sendet).toBe(false);
	});

	it('liefert null, wenn der Server abweist, und bleibt bedienbar', async () => {
		vi.mocked(apiGet).mockResolvedValue(vorschlag);
		vi.mocked(apiPost).mockRejectedValue(new Error('409'));
		const f = new BescheidFormular('s1');
		await f.laden();
		expect(await f.erstellen()).toBeNull();
		expect(f.sendet).toBe(false);
		expect(f.bereit).toBe(true);
	});
});
