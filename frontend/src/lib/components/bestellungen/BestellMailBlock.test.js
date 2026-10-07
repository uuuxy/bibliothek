import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiPost: vi.fn()
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

import { apiPost, FRIST_MAILVERSAND_MS } from '../../apiFetch.js';
import { toastStore } from '../../stores/toastStore.svelte.js';
import BestellMailBlock from './BestellMailBlock.svelte';
import BestellStatusBlock from './BestellStatusBlock.svelte';
import BestellHistorieTabelle from './BestellHistorieTabelle.svelte';

const gescheitert = {
	id: 'b1',
	bestelldatum: '2026-10-07T08:00:00Z',
	lieferant_name: 'Buchhandlung Nord',
	lieferant_email: 'handel@example.org',
	kundennummer: 'K-1',
	mittel: 'land',
	anzahl_exemplare: 2,
	gesamtbetrag: 0,
	mit_bestaetigung: true,
	link_aktiv: true,
	mail_gescheitert_am: '2026-10-07T08:00:05Z'
};

// Scheitert die Mail an den Lieferanten, steht es an der Bestellung, und sie lässt sich von
// dort erneut senden.
describe('Bestellung: gescheiterter Versand der Bestellmail', () => {
	beforeEach(() => vi.clearAllMocks());

	it('nennt den gescheiterten Versand und sendet erneut, mit der Frist der Mail-Aufrufe', async () => {
		vi.mocked(apiPost).mockResolvedValue({ status: 'success', message: 'Bestellung gesendet.' });
		const neuLaden = vi.fn(async () => {});
		const screen = render(BestellMailBlock, {
			b: gescheitert,
			darfSenden: true,
			onAktualisieren: neuLaden
		});

		expect(screen.getByText('Die Bestellmail ist nicht rausgegangen')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut senden' }));

		await waitFor(() => expect(neuLaden).toHaveBeenCalledTimes(1));
		expect(apiPost).toHaveBeenCalledWith('/api/bestellungen/b1/mail', null, {
			timeoutMs: FRIST_MAILVERSAND_MS
		});
		expect(toastStore.addToast).toHaveBeenCalledWith('Bestellung gesendet.', 'success');
	});

	it('lädt die Bestellung auch nach einem gescheiterten Versuch neu', async () => {
		vi.mocked(apiPost).mockRejectedValue(new Error('Versand gescheitert'));
		const neuLaden = vi.fn(async () => {});
		const screen = render(BestellMailBlock, {
			b: gescheitert,
			darfSenden: true,
			onAktualisieren: neuLaden
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut senden' }));
		await waitFor(() => expect(neuLaden).toHaveBeenCalledTimes(1));
		expect(toastStore.addToast).not.toHaveBeenCalled();
	});

	it('zeigt ohne das Recht zu bestellen den Hinweis ohne Knopf', () => {
		const screen = render(BestellMailBlock, {
			b: gescheitert,
			darfSenden: false,
			onAktualisieren: async () => {}
		});

		expect(screen.getByText('Die Bestellmail ist nicht rausgegangen')).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'Erneut senden' })).toBeNull();
	});

	// Der Block der Bestätigung behauptete nach einem gescheiterten Versand weiter, der Link
	// sei mit der Bestellmail rausgegangen.
	it('der Block der Bestätigung behauptet nicht, der Link sei mit der Mail rausgegangen', () => {
		const screen = render(BestellStatusBlock, { b: gescheitert, onAktualisieren: async () => {} });

		expect(screen.queryByText(/ging mit der Bestellmail raus/)).toBeNull();
		expect(screen.getByText(/Der Händler hat den Bestätigungs-Link nicht erhalten/)).toBeTruthy();
	});

	it('die Zeile der Bestellhistorie nennt den gescheiterten Versand', () => {
		const zeilen = [gescheitert, { ...gescheitert, id: 'b2', mail_gescheitert_am: undefined }];
		const screen = render(BestellHistorieTabelle, {
			bestellungen: zeilen,
			euro: String,
			datum: () => '07.10.2026',
			kurzdatum: () => '07.10.',
			onOeffnen: () => {}
		});

		expect(screen.getAllByText('Mail nicht versendet')).toHaveLength(1);
		expect(screen.getAllByText('Wartet auf Händler')).toHaveLength(1);
	});
});
