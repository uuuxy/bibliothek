import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import NeueFassung from './NeueFassung.svelte';
import { programmstand } from '../../stores/programmstand.svelte.js';

// Der Hinweis steht nur, wenn der Server einen anderen Stand nennt, und er lädt erst auf den
// Knopf neu: An der Theke entscheidet die Person, wann nichts mehr offen ist.
describe('NeueFassung', () => {
	afterEach(() => {
		programmstand.eigener = '';
		programmstand.server = '';
		programmstand.laedt = false;
		vi.restoreAllMocks();
	});

	it('steht nicht da, solange der Server denselben Stand nennt', () => {
		programmstand.eigener = '/assets/index-alt111.js';
		programmstand.server = '/assets/index-alt111.js';
		render(NeueFassung);
		expect(screen.queryByRole('status')).toBeNull();
	});

	it('nennt die neue Fassung und lädt erst auf den Knopf neu', async () => {
		const ladeNeu = vi.spyOn(programmstand, 'ladeNeu').mockImplementation(async () => {});
		programmstand.eigener = '/assets/index-alt111.js';
		programmstand.server = '/assets/index-neu222.js';
		render(NeueFassung);

		expect(screen.getByRole('status').textContent).toContain(
			'Eine neue Fassung des Programms ist eingespielt.'
		);
		expect(ladeNeu).not.toHaveBeenCalled();
		await fireEvent.click(screen.getByRole('button', { name: 'Neu laden' }));
		expect(ladeNeu).toHaveBeenCalledOnce();
	});

	it('zeigt „Wird geladen …“, solange die neue Fassung kommt, und nimmt keinen zweiten Klick an', () => {
		programmstand.eigener = '/assets/index-alt111.js';
		programmstand.server = '/assets/index-neu222.js';
		programmstand.laedt = true;
		render(NeueFassung);
		expect(screen.getByRole('button', { name: 'Wird geladen …' })).toHaveProperty('disabled', true);
	});
});
