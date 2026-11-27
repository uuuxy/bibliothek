import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn()
}));

import { apiGet } from '../../apiFetch.js';
import Signaturen from './Signaturen.svelte';

// Die Suche über die Signaturen liest wie die übrigen Listen und die Titelsuche im Katalog:
// Umlaute, Akzente und groß und klein zählen nicht, jedes Wort für sich, in beliebiger Folge.
describe('Signaturen: Suche', () => {
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockImplementation(async (/** @type {string} */ pfad) =>
			pfad === '/api/signaturen'
				? [
						{ signatur: 'D Mül', exemplare: 2 },
						{ signatur: 'LMF Deu 7', exemplare: 30 },
						{ signatur: 'LMF Deu 7 / Bie', exemplare: 28 },
						{ signatur: 'LMF Bio 7', exemplare: 25 },
						{ signatur: 'LMF Deu 8', exemplare: 30 }
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

	it('findet mit „LMF 7“ alle Regale des 7. Jahrgangs', async () => {
		await suche('lmf 7');
		expect(screen.getByText('LMF Deu 7')).toBeTruthy();
		expect(screen.getByText('LMF Bio 7')).toBeTruthy();
		expect(screen.queryByText('LMF Deu 8')).toBeNull();
	});

	it('findet die Wörter in jeder Folge', async () => {
		await suche('bie deu');
		expect(screen.getByText('LMF Deu 7 / Bie')).toBeTruthy();
		expect(screen.queryByText('LMF Deu 7')).toBeNull();
	});
});
