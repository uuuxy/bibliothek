import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import BestellBerichte from './BestellBerichte.svelte';
import { apiGet } from '../../apiFetch.js';

// Der orderStore holt beides über apiGet (stores/orderStore.svelte.js). Ein Mock, der nur
// apiFetch ersetzt, macht apiGet zu `undefined` — der Aufruf fliegt dann in den
// catch-Block des Stores, die Liste bleibt leer und der Test wäre rot, ohne dass an der
// Ansicht etwas falsch ist. Also die Türen mocken, die der Store wirklich benutzt.
vi.mock('../../apiFetch.js', () => ({
	apiGet: vi.fn(),
	apiPost: vi.fn(),
	apiPut: vi.fn(),
	apiDelete: vi.fn(),
	apiFetch: vi.fn()
}));

/**
 * Die Bestellberichte sind am 17.09.2026 vom Reiter im Bestellwesen zu einem eigenen
 * Bildschirm unter „Berichte" geworden. Genau eine Sache kann dabei brechen: Die Ansicht
 * bekam Lieferantenliste und Preis-Schalter vorher vom Bestellwesen gereicht
 * (`suppliers`-Prop, Laden im Workspace). Steht sie allein, muss sie beides selbst holen —
 * sonst bleibt die Lieferantenauswahl dauerhaft leer und der Schalter „Preise im
 * Bestellwesen" wirkungslos, ohne dass irgendetwas rot wird.
 *
 * Die Adresse des PDFs prüft weiterhin bestellberichte.test.js an der reinen Funktion;
 * hier steht nur, was der Umzug angefasst hat.
 *
 * Der Dateiname ist bewusst nicht „BestellBerichte.test.js": Auf einem
 * case-insensitiven Dateisystem ist das derselbe Name wie bestellberichte.test.js,
 * und die Datei würde beim Anlegen die andere überschreiben.
 */
/** @type {any} */
const getMock = apiGet;

beforeEach(() => {
	getMock.mockReset();
	getMock.mockImplementation(async (/** @type {string} */ url) => {
		if (url.startsWith('/api/lieferanten')) {
			return [{ id: 'l-1', name: 'E2E-Haendler', email: 'h@test.invalid' }];
		}
		if (url.startsWith('/api/bestellungen/konfiguration')) {
			return { preise_erfassen: false, bestelllink_ohne_adresse: false };
		}
		return null;
	});
});

describe('Bestellberichte als eigener Bildschirm', () => {
	it('holt die Lieferanten selbst und bietet sie in der Abrechnung an', async () => {
		render(BestellBerichte);

		await waitFor(() =>
			expect(getMock.mock.calls.some((/** @type {any[]} */ c) => c[0] === '/api/lieferanten')).toBe(
				true
			)
		);

		// Die dritte Berichtsart ist die einzige mit Lieferantenauswahl.
		await fireEvent.click(await screen.findByRole('radio', { name: /Lieferanten/ }));
		const auswahl = await screen.findByLabelText('Lieferant');
		await waitFor(() => expect(auswahl.textContent).toContain('E2E-Haendler'));

		// Gegenprobe zum „leer ist auch grün": Ohne geladene Liste stünde hier der Satz
		// „Keine Lieferanten vorhanden." — dann wäre der Umzug halb erledigt.
		expect(screen.queryByText('Keine Lieferanten vorhanden.')).toBeNull();
	});

	it('holt den Preis-Schalter selbst — ohne Preise heißt das Blatt Übersicht', async () => {
		render(BestellBerichte);

		// preise_erfassen: false steht im Mock oben. Ohne eigenes Laden bliebe die Vorgabe
		// `true` stehen und das Blatt hieße „Lieferantenabrechnung" — abgerechnet wird aber
		// nichts, es listet Mengen.
		expect(await screen.findByText('Lieferantenübersicht')).toBeTruthy();
		expect(screen.queryByText('Lieferantenabrechnung')).toBeNull();
	});
});
