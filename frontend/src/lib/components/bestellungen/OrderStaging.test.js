import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiPut: vi.fn(async () => ({ ok: true, json: async () => ({}) })),
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() }
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../../stores/orderStore.svelte.js', () => ({
	orderStore: { addToCart: vi.fn(), preiseErfassen: true }
}));

import { apiPut } from '../../apiFetch.js';
import { orderStore } from '../../stores/orderStore.svelte.js';
import OrderStaging from './OrderStaging.svelte';

// Das offene Gate aus #596: „DNB-Titel mit Haken entsteht als Lernmittel".
//
// Ein über die DNB-Bestellsuche neu angelegter Titel entsteht OHNE Kennzeichen. Das
// Staging-Fenster ist der Moment, in dem jemand ihn ohnehin ansieht, und die Antwort auf
// „Lernmittel?" entscheidet zweierlei: das Kennzeichen am Titel (Frist, Katalog,
// Bestellbedarf) und den Topf, den der Warenkorb für die Bestellung vorschlägt — Land
// oder Schulträger. Bleibt sie liegen, bestellt die Schule ein Schulbuch aus dem falschen
// Haushalt, und auf dem Anschreiben steht der falsche Vermerk.
//
// Der Server-Teil ist in api/titel_lernmittel_pg_test.go belegt; hier fehlte die
// Oberfläche.

const titel = { id: 't-1', titel: 'Mathematik 7', autor: 'Autor', isbn: '978', signatur: 'MA 7' };

/** @param {any} buch */
function fenster(buch = titel) {
	return render(OrderStaging, { book: buch, onDone: () => {} });
}

describe('OrderStaging: die Frage „Lernmittel?"', () => {
	beforeEach(() => vi.clearAllMocks());

	it('schreibt den gesetzten Haken an den Titel und gibt ihn dem Warenkorb mit', async () => {
		const screen = fenster();

		screen.getByLabelText(/Lernmittel/).click();
		screen.getByRole('button', { name: 'In den Warenkorb' }).click();
		await vi.waitFor(() => expect(orderStore.addToCart).toHaveBeenCalled());

		expect(apiPut).toHaveBeenCalledWith('/api/buecher/titel/t-1/lernmittel', {
			ist_lernmittel: true
		});
		const [uebergeben] = vi.mocked(orderStore.addToCart).mock.calls[0];
		expect(
			uebergeben.ist_lernmittel,
			'der Warenkorb bekäme den Treffer statt des geprüften Standes — und schlüge den falschen Topf vor'
		).toBe(true);
	});

	// Ein unangetasteter Haken löst keinen Request aus: Das Fenster steht bei JEDER
	// Bestellposition offen, und ein PUT je Treffer wäre Schreiblast ohne Änderung.
	it('schweigt, wenn niemand den Haken anfasst', async () => {
		const screen = fenster();

		screen.getByRole('button', { name: 'In den Warenkorb' }).click();
		await vi.waitFor(() => expect(orderStore.addToCart).toHaveBeenCalled());

		expect(apiPut).not.toHaveBeenCalledWith('/api/buecher/titel/t-1/lernmittel', expect.anything());
	});

	// Die Gegenrichtung: ein falsch gekennzeichnetes Schulbuch lässt sich hier auch
	// zurücknehmen.
	it('nimmt das Kennzeichen auch wieder zurück', async () => {
		const screen = fenster({ ...titel, ist_lernmittel: true });

		screen.getByLabelText(/Lernmittel/).click();
		screen.getByRole('button', { name: 'In den Warenkorb' }).click();
		await vi.waitFor(() => expect(orderStore.addToCart).toHaveBeenCalled());

		expect(apiPut).toHaveBeenCalledWith('/api/buecher/titel/t-1/lernmittel', {
			ist_lernmittel: false
		});
	});
});
