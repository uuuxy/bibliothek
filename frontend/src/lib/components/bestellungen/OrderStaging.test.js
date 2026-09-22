import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiPut: vi.fn(async () => ({ ok: true, json: async () => ({}) })),
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => [] })),
	apiClient: { post: vi.fn() }
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../../stores/orderStore.svelte.js', () => ({
	orderStore: { addToCart: vi.fn(), preiseErfassen: true }
}));

import { apiPut, apiFetch } from '../../apiFetch.js';
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

// Die Signatur der Schülerbücherei ist die Regaladresse aus Littera („Sk", „JF", „MANGA").
// Bis zum 22.09.2026 schlug das Fenster „BIB {Kategorie}" aus der DNB-Gattung vor — ein
// Wort, das in keinem Regal der Schule vorkommt; damit entstand neben Litteras Vokabular
// ein zweites. Jetzt bietet das Feld die Adressen AUS DEM BESTAND an und erfindet keine.
describe('OrderStaging: die Signatur kommt aus dem Bestand', () => {
	beforeEach(() => vi.clearAllMocks());

	it('bietet die vorhandenen Regaladressen an und füllt nichts Erfundenes vor', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ signatur: 'JF', titel: 351, exemplare: 400 },
					{ signatur: 'MANGA', titel: 155, exemplare: 160 }
				]
			})
		);
		const screen = fenster({ ...titel, signatur: '' });

		const feld = /** @type {HTMLInputElement} */ (screen.getByLabelText(/Signatur/));
		expect(feld.value, 'ein neuer Titel bekommt keine erfundene Signatur').toBe('');

		await vi.waitFor(() => {
			const optionen = [...screen.container.querySelectorAll('datalist option')].map(
				(o) => /** @type {HTMLOptionElement} */ (o).value
			);
			expect(optionen).toEqual(['JF', 'MANGA']);
		});
		expect(apiFetch).toHaveBeenCalledWith('/api/signaturen');
	});
});
