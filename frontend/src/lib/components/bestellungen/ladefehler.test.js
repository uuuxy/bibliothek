import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

// Kommt eine Liste des Bestellwesens nicht, steht ihr Grund da statt eines Leerzustands:
// „Bestände ausreichend", „Keine Positionen im Zulauf" und „Keine Lieferanten angelegt" wären
// falsche Auskünfte, während nur der Abruf scheiterte. apiGet zeigt den Grund außerdem als
// Meldung, die nach Sekunden verschwindet.
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn(async () => [])
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiGet } from '../../apiFetch.js';
import { bestellStore } from '../../stores/bestellStore.svelte.js';
import Bestellbedarf from './Bestellbedarf.svelte';
import WareneingangTabelle from './WareneingangTabelle.svelte';
import Lieferanten from './Lieferanten.svelte';
import BestellBerichte from './BestellBerichte.svelte';

const GRUND = 'Die Datenbank antwortet nicht';
const scheitert = () => vi.mocked(apiGet).mockRejectedValue(new Error(GRUND));

describe('Bestellwesen: eine Liste kommt nicht', () => {
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockResolvedValue([]);
		bestellStore.nichtGeladen = {};
		bestellStore.recommendations = [];
		bestellStore.incomingShipments = [];
		bestellStore.suppliers = [];
	});

	it('der Store merkt sich je Liste den Grund und behält, was schon da war', async () => {
		bestellStore.recommendations = [{ id: 't1' }];
		scheitert();
		expect(await bestellStore.loadRecommendations()).toBe(false);
		expect(await bestellStore.loadIncomingShipments()).toBe(false);
		expect(await bestellStore.loadSuppliers()).toBe(false);
		expect(bestellStore.nichtGeladen).toEqual({ bedarf: GRUND, zulauf: GRUND, lieferanten: GRUND });
		expect(bestellStore.recommendations).toEqual([{ id: 't1' }]);

		vi.mocked(apiGet).mockResolvedValue([{ id: 't2' }]);
		expect(await bestellStore.loadRecommendations()).toBe(true);
		expect(bestellStore.nichtGeladen.bedarf).toBe('');
		expect(bestellStore.recommendations).toEqual([{ id: 't2' }]);
	});

	it('der Bestellbedarf zeigt den Grund statt „Bestände ausreichend" und lädt neu', async () => {
		scheitert();
		await bestellStore.loadRecommendations();
		render(Bestellbedarf, { props: { vorschlaege: [], onAddToCart: vi.fn() } });

		expect(screen.getByText('Bestellbedarf nicht geladen')).toBeTruthy();
		expect(screen.getByText(GRUND)).toBeTruthy();
		expect(screen.queryByText('Bestände ausreichend')).toBeNull();

		vi.mocked(apiGet).mockResolvedValue([]);
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await vi.waitFor(() => expect(screen.getByText('Bestände ausreichend')).toBeTruthy());
	});

	it('der Zulauf zeigt den Grund statt „Keine Positionen im Zulauf"', async () => {
		scheitert();
		await bestellStore.loadIncomingShipments();
		render(WareneingangTabelle, { props: { incomingShipments: [], totalItems: 0 } });

		expect(screen.getByText('Zulauf nicht geladen')).toBeTruthy();
		expect(screen.getByText(GRUND)).toBeTruthy();
		expect(screen.queryByText('Keine Positionen im Zulauf.')).toBeNull();
	});

	it('die Lieferanten in den Einstellungen zeigen den Grund statt „Keine Lieferanten angelegt"', () => {
		render(Lieferanten, {
			props: {
				suppliers: [],
				ladefehler: GRUND,
				onErneut: vi.fn(),
				onAddSupplier: vi.fn(),
				onEditSupplier: vi.fn(),
				onRemoveSupplier: vi.fn()
			}
		});

		expect(screen.getByText('Lieferanten nicht geladen')).toBeTruthy();
		expect(screen.getByText(GRUND)).toBeTruthy();
		expect(screen.queryByText('Keine Lieferanten angelegt.')).toBeNull();
	});

	// Der Store frischt im Hintergrund auf (init nach 60 Sekunden). Scheitert das, ist die Liste
	// von vorhin noch brauchbar; den Grund meldet apiGet. Der Kasten steht nur statt einer leeren.
	it('scheitert nur das Auffrischen, bleiben die Listen stehen', async () => {
		const titel = { id: 't1', titel: 'Faust', isbn: '9783150000014', verfuegbar: 0, bestand: 3 };
		const lieferung = {
			supplierName: 'Hahn',
			date: '01.10.2026',
			items: [{ titel: 'Faust', exemplar_ids: ['e1'], menge: 1 }]
		};
		bestellStore.recommendations = [titel];
		bestellStore.incomingShipments = [lieferung];
		bestellStore.suppliers = [
			{ id: 'l1', name: 'Hahn', email: 'hahn@test.local', customerNumber: '' }
		];
		scheitert();
		await bestellStore.loadRecommendations();
		await bestellStore.loadIncomingShipments();

		const bedarf = render(Bestellbedarf, {
			props: { vorschlaege: bestellStore.recommendations, onAddToCart: vi.fn() }
		});
		expect(bedarf.queryByText('Bestellbedarf nicht geladen')).toBeNull();
		expect(bedarf.getByText('Faust')).toBeTruthy();
		bedarf.unmount();

		const zulauf = render(WareneingangTabelle, {
			props: { incomingShipments: bestellStore.incomingShipments, totalItems: 1 }
		});
		expect(zulauf.queryByText('Zulauf nicht geladen')).toBeNull();
		expect(zulauf.getByText('Faust')).toBeTruthy();
		zulauf.unmount();

		const lieferanten = render(Lieferanten, {
			props: {
				suppliers: bestellStore.suppliers,
				ladefehler: GRUND,
				onErneut: vi.fn(),
				onAddSupplier: vi.fn(),
				onEditSupplier: vi.fn(),
				onRemoveSupplier: vi.fn()
			}
		});
		expect(lieferanten.queryByText('Lieferanten nicht geladen')).toBeNull();
		expect(lieferanten.getByText('Hahn')).toBeTruthy();
		lieferanten.unmount();

		const berichte = render(BestellBerichte);
		await fireEvent.click(
			berichte.getByRole('radio', { name: /^Lieferanten(abrechnung|übersicht)/ })
		);
		await vi.waitFor(() => expect(bestellStore.nichtGeladen.lieferanten).toBe(GRUND));
		expect(berichte.queryByText('Lieferanten nicht geladen')).toBeNull();
		expect(bestellStore.suppliers).toHaveLength(1);
	});

	it('die Berichte sagen beim Lieferanten, dass die Liste nicht kam', async () => {
		scheitert();
		render(BestellBerichte);
		await fireEvent.click(
			screen.getByRole('radio', { name: /^Lieferanten(abrechnung|übersicht)/ })
		);

		await vi.waitFor(() => expect(screen.getByText('Lieferanten nicht geladen')).toBeTruthy());
		expect(screen.queryByText('Keine Lieferanten vorhanden.')).toBeNull();
	});
});
