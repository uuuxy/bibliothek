import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch } from '../../apiFetch.js';
import { sucheSchuelerFuerVormerkung, MAX_TREFFER } from './vormerkungSchuelersuche.js';
import { serverAntwort } from '../../serverAntwort.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

// Der Vormerkungs-Reiter sucht Schüler über die Schülerdatei, nicht über die Theken-Suche
// (Befund-Register, Entscheidung 3 vom 05.09.2026). Hier ohne Browser: welche Tür, welche
// Form, welche Kappung.
describe('Schülersuche des Vormerkungs-Reiters', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	const antwort = (liste, ok = true) =>
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok, json: async () => liste }));

	it('fragt GET /api/schueler?q= (view_students), nie die Theken-Suche', async () => {
		antwort([]);
		await sucheSchuelerFuerVormerkung('  Anna Müller ');
		const [url] = vi.mocked(apiFetch).mock.calls[0];
		expect(url).toBe('/api/schueler?q=Anna%20M%C3%BCller');
		expect(url).not.toContain('/api/search');
	});

	it('bildet die Zeile der Schülerdatei auf die Auswahl des Reiters ab', async () => {
		antwort([{ id: '42', vorname: 'Anna', nachname: 'Müller', klasse: '07A', barcode_id: 'S-1' }]);
		expect(await sucheSchuelerFuerVormerkung('Müller')).toEqual([
			{ id: '42', title: 'Anna Müller', subtitle: '07A · S-1' }
		]);
	});

	it('kappt auf MAX_TREFFER — die Schülerdatei kappt eine Suche nicht', async () => {
		antwort(
			Array.from({ length: MAX_TREFFER + 5 }, (_, i) => ({
				id: String(i),
				vorname: 'V',
				nachname: 'N',
				klasse: '05A',
				barcode_id: `S-${i}`
			}))
		);
		expect(await sucheSchuelerFuerVormerkung('N')).toHaveLength(MAX_TREFFER);
	});

	// Eine leere Auswahl hieße „Keine Schüler gefunden": Lehnt der Server ab, steht sein Grund da.
	it('wirft bei einer Fehlerantwort mit dem Grund des Servers', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
		);
		await expect(sucheSchuelerFuerVormerkung('x')).rejects.toThrow(
			'Keine Berechtigung für diese Aktion'
		);
	});

	it('liefert bei einer Antwort ohne Liste eine leere Auswahl', async () => {
		antwort({ error: 'nicht erlaubt' });
		expect(await sucheSchuelerFuerVormerkung('x')).toEqual([]);
	});
});
