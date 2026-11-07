import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({
	loeschenMitGrund: vi.fn(async () => null)
}));
vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import BuchakteExemplare from './BuchakteExemplare.svelte';
import { authStore } from '../../stores/authStore.svelte.js';
import { loeschenMitGrund } from '../../stores/bestaetigung.svelte.js';

// Ist genau ein Exemplar markiert, nennt die Rückfrage seinen Barcode. Sie muss das markierte
// nennen und nicht ein anderes aus der Liste: Bestätigt wird, was dort steht.
describe('Buchakte: die Rückfrage zum Löschen der Markierten', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(loeschenMitGrund).mockClear();
		authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
	});

	it('nennt das eine markierte Exemplar', async () => {
		const exemplare = ['e1', 'e2', 'e3'].map((id, i) => ({
			id,
			barcode_id: `B-${i + 1}`,
			ist_ausleihbar: true,
			ist_verfuegbar: true,
			im_bestand: true
		}));
		const screen = render(BuchakteExemplare, {
			props: { exemplare, titel: { id: 't-1', title: 'Momo' }, ladeAlles: () => {} }
		});
		await fireEvent.click(screen.getByLabelText('Exemplar B-2 auswählen'));
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));

		await waitFor(() => expect(loeschenMitGrund).toHaveBeenCalledOnce());
		expect(vi.mocked(loeschenMitGrund).mock.calls[0][0]).toMatchObject({
			titel: 'Exemplar B-2 löschen?'
		});
		expect(apiFetch, 'abgebrochen: kein DELETE').not.toHaveBeenCalled();
	});
});
