import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn()
}));

import { apiGet } from '../../apiFetch.js';
import Signaturen from './Signaturen.svelte';

// Die Suche über die Signaturen liest Umlaute, Akzente und groß und klein wie die übrigen
// Listen (suchnorm), hält aber die Folge: Eine Signatur ist eine Regaladresse.
describe('Signaturen: Suche', () => {
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockImplementation(async (/** @type {string} */ pfad) =>
			pfad === '/api/signaturen'
				? [
						{ signatur: 'D Mül', exemplare: 2 },
						{ signatur: 'LMF Deu 7', exemplare: 30 },
						{ signatur: 'LMF Deu 17', exemplare: 4 }
					]
				: []
		);
	});

	/** @param {string} wert */
	async function suche(wert) {
		render(Signaturen);
		await screen.findByText('LMF Deu 7');
		await fireEvent.input(screen.getByLabelText('Signatur suchen'), { target: { value: wert } });
	}

	it('findet „Mül“ mit ausgeschriebenem Umlaut', async () => {
		await suche('d muel');
		expect(screen.getByText('D Mül')).toBeTruthy();
		expect(screen.queryByText('LMF Deu 7')).toBeNull();
	});

	it('trifft mit „LMF Deu 7“ nicht „LMF Deu 17“', async () => {
		await suche('lmf deu 7');
		expect(screen.getByText('LMF Deu 7')).toBeTruthy();
		expect(screen.queryByText('LMF Deu 17')).toBeNull();
	});
});
