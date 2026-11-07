import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn(async () => ({ ok: true, status: 200, json: async () => [] }))
}));

import KlassensatzDialog from './KlassensatzDialog.svelte';

// Hat eine gewählte Zielklasse schon einen Klassensatz, ersetzt das Speichern ihn. Der Hinweis
// nennt jede dieser Klassen; nennte er nur die erste, fiele der Satz der zweiten unbemerkt weg.
const vorhandeneGruppen = [
	{ className: '5a', books: [{ id: 1 }] },
	{ className: '5b', books: [{ id: 1 }, { id: 2 }] }
];

/** @param {any} screen @param {string} klasse */
async function waehle(screen, klasse) {
	await fireEvent.click(screen.getByRole('combobox', { name: 'Zielklassen' }));
	await fireEvent.click(await screen.findByRole('option', { name: klasse }));
}

describe('Klassensatz: Hinweis auf ersetzte Klassensätze', () => {
	it('nennt die eine Klasse, deren Satz ersetzt wird', async () => {
		const screen = render(KlassensatzDialog, { props: { vorhandeneGruppen } });
		await waehle(screen, '5a');

		expect(
			await screen.findByText(
				'5a (1 Buch) hat bereits einen Klassensatz. Beim Speichern wird er durch die Auswahl hier ersetzt.'
			)
		).toBeTruthy();
	});

	it('nennt jede Klasse, deren Satz ersetzt wird', async () => {
		const screen = render(KlassensatzDialog, { props: { vorhandeneGruppen } });
		await waehle(screen, '5a');
		await waehle(screen, '5b');

		expect(
			await screen.findByText(
				'5a (1 Buch) und 5b (2 Bücher) haben bereits einen Klassensatz. Beim Speichern werden sie durch die Auswahl hier ersetzt.'
			)
		).toBeTruthy();
	});
});
