import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import { tick } from 'svelte';
import SignaturCoverVorschau from './SignaturCoverVorschau.svelte';
import { authStore } from '../../stores/authStore.svelte.js';
import { ladeCoverZurSignatur } from '../../utils/signaturen.js';

vi.mock('../../utils/signaturen.js', () => ({
	ladeCoverZurSignatur: vi.fn(() => Promise.resolve([]))
}));

// Die Cover kommen aus GET /api/signaturen/buecher mit view_books. Das Menü der Signatur steht auch
// Konten offen, die nur bestellen dürfen; ohne die Prüfung bekämen sie je Zeigen eine Ablehnung.
describe('SignaturCoverVorschau', () => {
	beforeEach(() => {
		vi.mocked(ladeCoverZurSignatur).mockClear();
	});

	it('fragt ohne view_books nicht nach Covern', async () => {
		authStore.currentUser = /** @type {any} */ ({
			rolle: 'mitarbeiter',
			permissions: ['create_orders']
		});
		render(SignaturCoverVorschau, { signatur: 'LMF Deu 7' });
		await tick();
		expect(ladeCoverZurSignatur).not.toHaveBeenCalled();
	});

	it('fragt mit view_books nach den Covern der Signatur', async () => {
		authStore.currentUser = /** @type {any} */ ({
			rolle: 'mitarbeiter',
			permissions: ['view_books']
		});
		render(SignaturCoverVorschau, { signatur: 'LMF Deu 7' });
		await tick();
		expect(ladeCoverZurSignatur).toHaveBeenCalledWith('LMF Deu 7');
	});
});
