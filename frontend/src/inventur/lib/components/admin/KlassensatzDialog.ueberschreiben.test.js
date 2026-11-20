import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn(async () => ({ ok: true, status: 200, json: async () => [] }))
}));

import KlassensatzDialog from './KlassensatzDialog.svelte';

// Nimmt „Bücher verwalten" eine Zielklasse dazu, die schon einen Klassensatz hat, ersetzt das
// Speichern ihn. Der Hinweis nennt jede dieser Klassen; nennte er nur die erste, fiele der Satz der
// zweiten unbemerkt weg. „Klassensatz anlegen" ergänzt nur und warnt deshalb nicht.
const verwaltet = { className: '5c', books: [{ id: 9 }] };
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
		const screen = render(KlassensatzDialog, {
			props: { vorhandeneGruppen, initialGroup: verwaltet }
		});
		await waehle(screen, '5a');

		expect(
			await screen.findByText(
				'5a mit 1 Buch hat bereits einen Klassensatz. Beim Speichern wird er durch die Auswahl hier ersetzt.'
			)
		).toBeTruthy();
	});

	it('nennt jede Klasse, deren Satz ersetzt wird', async () => {
		const screen = render(KlassensatzDialog, {
			props: { vorhandeneGruppen, initialGroup: verwaltet }
		});
		await waehle(screen, '5a');
		await waehle(screen, '5b');

		expect(
			await screen.findByText(
				'5a mit 1 Buch und 5b mit 2 Büchern haben bereits einen Klassensatz. Beim Speichern werden sie durch die Auswahl hier ersetzt.'
			)
		).toBeTruthy();
	});
});

// „Klassensatz anlegen" schickt an /api/admin/class-books/add, und das ergänzt (ON CONFLICT DO
// NOTHING). Der Hinweis, der Satz werde ersetzt, stimmte dort nicht.
describe('Klassensatz anlegen: kein Hinweis auf Ersetzen', () => {
	it('warnt nicht, wenn die Klasse schon einen Satz hat', async () => {
		const screen = render(KlassensatzDialog, { props: { vorhandeneGruppen } });
		await waehle(screen, '5a');

		expect(screen.queryByText(/Beim Speichern wird er durch die Auswahl hier ersetzt/)).toBeNull();
	});
});

// Wer in „Bücher verwalten" die Klasse entfernt, um eine andere zu wählen, behält die Auswahl der
// Bücher. Vorher leerte ein $effect sie, sobald keine Klasse mehr gewählt war.
describe('Bücher verwalten: Klasse wechseln', () => {
	it('behält die Auswahl, wenn die einzige Klasse entfernt wird', async () => {
		const screen = render(KlassensatzDialog, {
			props: { vorhandeneGruppen, initialGroup: verwaltet }
		});
		await vi.waitFor(() => expect(screen.getByText('1', { selector: 'div' })).toBeTruthy());

		await fireEvent.click(screen.getByRole('button', { name: 'Klasse 5c entfernen' }));

		expect(screen.getByText('1', { selector: 'div' })).toBeTruthy();
	});
});
