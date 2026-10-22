import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, melde: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', async (importOriginal) => ({
	...(await importOriginal()),
	verwerfenBestaetigen: vi.fn()
}));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { verwerfenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
import { authStore } from '../../../../lib/stores/authStore.svelte.js';
import { uiStore } from '../../../../lib/stores/uiStore.svelte.js';
import { etikettenStore } from '../../../../lib/stores/etiketten.svelte.js';
import { druckWarteschlange } from '../../../../lib/stores/druckWarteschlange.svelte.js';
import BuchFormular from './BuchFormular.svelte';
import { leeresBuchFormular } from './buchFormularRegeln.js';

// „Barcodes drucken" in der Maske eines Titels druckt im Druck-Center, auf dem Bogen, der dort
// gewählt ist, und über dieselbe Tür wie jedes andere Etikett (POST /api/print/labels).

/** Rendert die Maske eines vorhandenen Titels und liefert den Knopf (oder null). */
function knopf() {
	const formular = $state({
		...leeresBuchFormular(),
		id: 't-1',
		title: 'Natur und Technik',
		author: 'Anna Autorin'
	});
	const screen = render(BuchFormular, {
		formular,
		onClose: () => {},
		onSave: () => {},
		onCoverUpload: () => {},
		onCoverNeuHolen: () => {},
		onAssignClass: () => {}
	});
	return screen.queryByRole('button', { name: 'Barcodes drucken' });
}

beforeEach(() => {
	vi.mocked(apiFetch).mockImplementation(
		async (url) =>
			/** @type {any} */ ({
				ok: true,
				status: 200,
				json: async () =>
					String(url).endsWith('/exemplare') ? [{ barcode_id: 'B-1' }, { barcode_id: 'B-2' }] : []
			})
	);
	vi.mocked(verwerfenBestaetigen).mockReset();
	uiStore.verlassenSperre = null;
	uiStore.blockierterWechsel = null;
	uiStore.activeTab = 'medienkatalog';
	uiStore.requestedDruckCenterTab = null;
});

describe('Barcodes drucken in der Maske eines Titels', () => {
	it('öffnet das Druck-Center mit dem Titel und seinen Exemplaren', async () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books', 'view_books'] };
		// Ein Auftrag, der noch in der Warteschlange liegt, ginge sonst statt des Titels auf den Bogen.
		druckWarteschlange.copies = [{ barcode_id: 'B-ALT', titel: 'Anderes Buch' }];

		await fireEvent.click(/** @type {HTMLElement} */ (knopf()));

		await vi.waitFor(() => expect(uiStore.activeTab).toBe('druck-center'));
		expect(uiStore.requestedDruckCenterTab).toBe('labels');
		expect(druckWarteschlange.copies).toBeNull();
		expect(etikettenStore.gewaehlterTitel).toMatchObject({
			id: 't-1',
			titel: 'Natur und Technik',
			autor: 'Anna Autorin'
		});
		await waitFor(() =>
			expect(etikettenStore.bogen.map((l) => l.barcode_id)).toEqual(['B-1', 'B-2'])
		);
	});

	// Hält die Maske Ungespeichertes, fragt der Knopf zuerst. Wer bleibt, findet das Druck-Center,
	// wie es war; wer verwirft, kommt mit dem Titel hinein.
	it('fragt vor dem Wechsel und ändert nach „Bleiben" nichts im Druck-Center', async () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books', 'view_books'] };
		const wartend = [{ barcode_id: 'B-ALT', titel: 'Anderes Buch' }];
		druckWarteschlange.copies = wartend;
		uiStore.verlassenSperre = () => true;
		const bleiben = /** @type {HTMLElement} */ (knopf());

		vi.mocked(verwerfenBestaetigen).mockResolvedValueOnce(false);
		await fireEvent.click(bleiben);
		await new Promise((fertig) => setTimeout(fertig, 0));

		expect(druckWarteschlange.copies).toEqual(wartend);
		expect(uiStore.requestedDruckCenterTab).toBeNull();
		expect(uiStore.activeTab).toBe('medienkatalog');
		expect(verwerfenBestaetigen).toHaveBeenCalledTimes(1);

		vi.mocked(verwerfenBestaetigen).mockResolvedValueOnce(true);
		await fireEvent.click(bleiben);

		await vi.waitFor(() => expect(uiStore.activeTab).toBe('druck-center'));
		expect(uiStore.requestedDruckCenterTab).toBe('labels');
		expect(druckWarteschlange.copies).toBeNull();
		expect(etikettenStore.gewaehlterTitel).toMatchObject({ id: 't-1' });
	});

	it('bietet den Knopf nicht an, wer die Buch-Etiketten nicht öffnen darf', () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books'] };

		expect(knopf()).toBeNull();
	});
});
