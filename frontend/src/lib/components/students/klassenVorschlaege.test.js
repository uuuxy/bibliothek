import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugeKlassenVorschlaege } from './klassenVorschlaege.svelte.js';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Die Auswahl im Anlegen-Dialog las bis zum 15.09.2026 die Tabelle lesergruppen, die kein
// Schreibweg füllt: Angeboten wurde nur „Manuell eingeben…". Die Klassen der Schule liefert
// GET /api/klassen — dieselbe Quelle wie Druck-Center und LMF-Verlängerung.

beforeEach(() => {
	vi.clearAllMocks();
});

describe('erzeugeKlassenVorschlaege', () => {
	it('bietet die Klassen der Schule an', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ['05F1', '07H', '10R'] })
		);

		const vorschlaege = erzeugeKlassenVorschlaege();
		await vorschlaege.lade();

		expect(apiFetch).toHaveBeenCalledWith('/api/klassen');
		expect(vorschlaege.liste).toEqual(['05F1', '07H', '10R']);
	});

	it('bleibt leer, wenn der Abruf scheitert — die Klasse lässt sich von Hand eintragen', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: false, json: async () => ({}) })
		);

		const vorschlaege = erzeugeKlassenVorschlaege();
		await vorschlaege.lade();

		expect(vorschlaege.liste).toEqual([]);
	});
});
