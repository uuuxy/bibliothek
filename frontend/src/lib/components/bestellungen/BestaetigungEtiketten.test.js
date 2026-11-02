import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BestaetigungEtiketten from './BestaetigungEtiketten.svelte';

const BESTELLUNG = {
	grosses_etikett: true,
	etiketten_formate: [
		{ id: 'avery-3474', name: 'Avery 3474' },
		{ id: 'herma-4626', name: 'Herma 4626' }
	]
};

afterEach(() => vi.restoreAllMocks());

// Das gewählte Bogenraster gilt nur für die kleinen Etiketten. Das große Lernmittel-Etikett hat
// ein festes Raster und bekäme mit dem Parameter ein Raster, das es nicht gibt.
describe('Etiketten auf der Seite des Lieferanten', () => {
	it('die kleinen Etiketten öffnen mit dem gewählten Raster, die großen ohne', async () => {
		const oeffne = vi.spyOn(window, 'open').mockReturnValue(null);
		const screen = render(BestaetigungEtiketten, {
			bestellung: BESTELLUNG,
			token: 'probe-token',
			formatId: 'herma-4626'
		});

		await fireEvent.click(screen.getByRole('button', { name: /Kleine Etiketten/ }));
		await fireEvent.click(screen.getByRole('button', { name: /Große Lernmittel-Etiketten/ }));

		expect(oeffne.mock.calls.map((aufruf) => aufruf[0])).toEqual([
			'/api/public/bestellung/probe-token/etiketten/klein?format=herma-4626',
			'/api/public/bestellung/probe-token/etiketten/gross'
		]);
	});
});
