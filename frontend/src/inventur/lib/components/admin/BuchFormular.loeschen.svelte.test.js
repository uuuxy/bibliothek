import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn(async () => ({ ok: true, status: 200, json: async () => [] }))
}));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, melde: vi.fn() }));

import BuchFormular from './BuchFormular.svelte';
import { leeresBuchFormular } from './buchFormularRegeln.js';

// „Titel löschen" in der Maske: Ein zweiter Auftrag vor der Antwort schickte ein zweites DELETE,
// und nach „Titel gelöscht." stand „Titel nicht gefunden", in Chromium, Firefox und WebKit
// (OFFEN.md 5.106).
describe('BuchFormular: Titel löschen', () => {
	it('nimmt keinen zweiten Auftrag an, solange der erste läuft', async () => {
		/** @type {() => void} */
		let fertig = () => {};
		const onDelete = vi.fn(() => new Promise((weiter) => (fertig = () => weiter(undefined))));
		const screen = render(BuchFormular, {
			formular: { ...leeresBuchFormular(), id: 't-1', title: 'Natura' },
			onClose: () => {},
			onSave: () => {},
			onCoverUpload: () => {},
			onCoverNeuHolen: () => {},
			onAssignClass: () => {},
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
