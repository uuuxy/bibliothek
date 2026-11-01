import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn(async () => [])
}));

import { apiGet } from '../../apiFetch.js';
import { bestellStore } from '../../stores/bestellStore.svelte.js';
import Wareneingang from './Wareneingang.svelte';

// Der Stand im Kopf einer Bestellung ändert sich außerhalb dieses Reiters. Das Laden des
// Bestellwesens gilt 60 Sekunden als frisch; ohne eigenes Laden stünde hier noch „Wartet auf
// Händler", wenn die Historie die Zusage eben nachgetragen hat.
describe('Wareneingang: Öffnen des Reiters', () => {
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
		bestellStore.incomingShipments = [];
	});

	it('holt den Zulauf frisch', async () => {
		const gruppe = { id: 'b1', supplierName: 'Händler', date: '30.09.2026', items: [] };
		vi.mocked(apiGet).mockResolvedValue([gruppe]);

		render(Wareneingang, { incomingShipments: [], onBack: vi.fn(), onReceived: vi.fn() });

		expect(apiGet).toHaveBeenCalledWith('/api/bestellungen/zulauf');
		await vi.waitFor(() => expect(bestellStore.incomingShipments).toEqual([gruppe]));
	});
});
