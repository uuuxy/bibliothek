import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn(async () => []),
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ({}) }))
}));

import { uiStore } from '../../stores/uiStore.svelte.js';
import Bestellwesen from './Bestellwesen.svelte';

// Die Seitenleiste zählt an „Bestellungen“ die offenen Meldungen der Lehrkräfte mit
// (SeitenleisteZiele, wartendeArbeit). Der Reiter „Meldungen“ trägt dieselbe Zahl; ohne sie
// führte die Zahl in der Seitenleiste ins Leere.
describe('Bestellwesen: die Zahlen an den Reitern', () => {
	it('zeigt die offenen Meldungen am Reiter „Meldungen“', async () => {
		uiStore.offeneAnliegen = 2;
		uiStore.pendingReservierungen = 0;
		render(Bestellwesen);
		const reiter = await screen.findByRole('tab', { name: /^Meldungen/ });
		expect(reiter.textContent).toMatch(/2/);
	});
});
