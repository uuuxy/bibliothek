import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../stores/authStore.svelte.js', () => ({
	authStore: { isLoggedIn: true, currentUser: /** @type {any} */ (null) }
}));
vi.mock('../../stores/buchBarcodes.svelte.js', () => ({
	buchBarcodes: { bereitstellen: vi.fn(), stoppeZeitgeber: vi.fn() }
}));
vi.mock('../../stores/leerlaufSperre.svelte.js', () => ({ leerlaufSperre: { gesperrt: false } }));

import ThekenDaten from './ThekenDaten.svelte';
import { authStore } from '../../stores/authStore.svelte.js';
import { buchBarcodes } from '../../stores/buchBarcodes.svelte.js';

// Die Barcode-Liste gehört der Theke, ihre Route verlangt perform_actions. Das Kollegium fragte bei
// jeder Anmeldung trotzdem danach und bekam still 403.
describe('ThekenDaten', () => {
	beforeEach(() => {
		vi.mocked(buchBarcodes.bereitstellen).mockClear();
	});

	it('holt die Liste für den, der an der Theke bucht', async () => {
		authStore.currentUser = { rolle: 'helfer', permissions: ['perform_actions'] };
		render(ThekenDaten);
		await tick();

		expect(buchBarcodes.bereitstellen).toHaveBeenCalledTimes(1);
	});

	it('fragt ohne dieses Recht nicht danach', async () => {
		authStore.currentUser = { rolle: 'kollegium', permissions: ['create_reservations'] };
		render(ThekenDaten);
		await tick();

		expect(buchBarcodes.bereitstellen).not.toHaveBeenCalled();
	});
});
