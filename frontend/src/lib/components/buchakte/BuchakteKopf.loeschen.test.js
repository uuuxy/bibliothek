import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import BuchakteKopf from './BuchakteKopf.svelte';

// Ein zweiter Auftrag zum Löschen vor der Antwort schickte ein zweites DELETE: Nach „Titel
// erfolgreich gelöscht" stand „Titel nicht gefunden", in Chromium, Firefox und WebKit
// (OFFEN.md 5.106).
describe('Buchakte, Kopf: Titel löschen', () => {
	it('nimmt keinen zweiten Auftrag an, solange der erste läuft', async () => {
		/** @type {() => void} */
		let fertig = () => {};
		const onDelete = vi.fn(() => new Promise((weiter) => (fertig = () => weiter(undefined))));
		const screen = render(BuchakteKopf, {
			titel: { id: 't-1', title: 'Natura' },
			ausleiher: [],
			exemplare: [],
			onDelete
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Titel löschen' })
		);

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);

		expect(onDelete, 'ein zweiter Auftrag vor der Antwort').toHaveBeenCalledTimes(1);
		expect(knopf.disabled, 'gesperrt, solange das Löschen läuft').toBe(true);

		fertig();
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});
