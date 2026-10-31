import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Buchakte from './Buchakte.svelte';
import { apiFetch } from '../../apiFetch.js';
import { appState } from '../../../inventur/lib/store.svelte.js';
import { authStore } from '../../stores/authStore.svelte.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, status: 200, json: async () => [] })
	);
	// Der Kopf kommt aus der Auswahl der Titel-Verwaltung; die Listen antworten leer.
	appState.selectedBook = /** @type {any} */ ({ id: 'A', title: 'Momo', authors: 'Ende' });
	authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
});

// Die Reiter der Akte kommen aus ui/Reiter wie in Bestellwesen und Bestandsbüchern: eine
// benannte Leiste mit der Rolle tablist, ein Reiter je Bereich.
describe('Buchakte: Reiter', () => {
	it('stehen in einer benannten Reiterleiste und wechseln mit einem Klick', async () => {
		const screen = render(Buchakte, { titelId: 'A', onBack: vi.fn() });

		const leiste = await screen.findByRole('tablist', { name: 'Bereiche der Buchakte' });
		const namen = [...leiste.querySelectorAll('[role="tab"]')].map((r) => r.textContent?.trim());
		expect(namen).toEqual(['Ausleiher (0)', 'Exemplare (0)', 'Vormerkungen (0)', 'Historie']);

		const historie = screen.getByRole('tab', { name: 'Historie' });
		await fireEvent.click(historie);
		expect(historie.getAttribute('aria-selected')).toBe('true');
		expect(screen.getByText('Noch keine Ausleihen in der Datenbank vorhanden.')).toBeTruthy();
	});
});

// Ein Helfer sieht die Exemplare: „habt ihr Band 3 noch da?". Ausleiher nennen Personen.
describe('Buchakte: ohne das Recht auf Personen', () => {
	it('zeigt keine Reiter, keinen Zähler der Ausleiher und keine Meldung', async () => {
		authStore.currentUser = /** @type {any} */ ({
			rolle: 'helfer',
			permissions: ['perform_actions', 'view_books']
		});
		const screen = render(Buchakte, { titelId: 'A', onBack: vi.fn() });

		await screen.findByText('verfügbar');
		expect(screen.queryByRole('tablist')).toBeNull();
		expect(screen.queryByText('Ausleiher')).toBeNull();
		expect(screen.queryByRole('alert')).toBeNull();
		const pfade = vi.mocked(apiFetch).mock.calls.map(([pfad]) => String(pfad));
		expect(pfade).toEqual(['/api/buecher/titel/A/exemplare']);
	});

	it('bietet mit dem Recht der Warteliste die Reiter Exemplare und Vormerkungen an', async () => {
		authStore.currentUser = /** @type {any} */ ({
			rolle: 'helfer',
			permissions: ['view_books', 'manage_vormerkungen']
		});
		const screen = render(Buchakte, { titelId: 'A', onBack: vi.fn() });

		const leiste = await screen.findByRole('tablist', { name: 'Bereiche der Buchakte' });
		const namen = [...leiste.querySelectorAll('[role="tab"]')].map((r) => r.textContent?.trim());
		expect(namen).toEqual(['Exemplare (0)', 'Vormerkungen (0)']);
		expect(screen.getByRole('tab', { name: 'Exemplare (0)' }).getAttribute('aria-selected')).toBe(
			'true'
		);
	});
});

// „1 von 2 verfügbar" nennt den Bestand; eine Zahl daneben, die auch bestellte und
// ausgesonderte Exemplare zählte, widersprach ihr („4 Exemplare").
describe('Buchakte: Zahlen zu den Exemplaren', () => {
	const exemplare = [
		{ id: 'e1', barcode_id: '1', ist_ausleihbar: true, ist_verfuegbar: true, im_bestand: true },
		{ id: 'e2', barcode_id: '2', ist_ausleihbar: true, ist_verfuegbar: false, im_bestand: true },
		{ id: 'e3', barcode_id: '3', ist_ausleihbar: false, ist_verfuegbar: true, im_bestand: false },
		{
			id: 'e4',
			barcode_id: '4',
			ist_ausleihbar: false,
			ist_verfuegbar: true,
			im_bestand: false,
			ist_ausgesondert: true
		}
	];

	/** @param {any[]} liste */
	function antworteMitExemplaren(liste) {
		vi.mocked(apiFetch).mockImplementation(
			async (url) =>
				/** @type {any} */ ({
					ok: true,
					status: 200,
					json: async () => (String(url).endsWith('/exemplare') ? liste : [])
				})
		);
	}

	it('der Reiter zählt den Bestand, der Kopf nennt die bestellten', async () => {
		antworteMitExemplaren(exemplare);
		const screen = render(Buchakte, { titelId: 'A', onBack: vi.fn() });

		expect(await screen.findByRole('tab', { name: 'Exemplare (2)' })).toBeTruthy();
		const bestellt = screen.getByText('bestellt', { selector: 'dt' });
		expect(bestellt.parentElement?.querySelector('dd')?.textContent?.trim()).toBe('1');
		expect(screen.queryByText('Exemplare', { selector: 'dt' })).toBeNull();
	});

	it('ohne bestelltes Exemplar steht im Kopf keine dritte Zahl', async () => {
		antworteMitExemplaren(exemplare.filter((ex) => ex.im_bestand));
		const screen = render(Buchakte, { titelId: 'A', onBack: vi.fn() });

		expect(await screen.findByRole('tab', { name: 'Exemplare (2)' })).toBeTruthy();
		expect(screen.queryByText('bestellt', { selector: 'dt' })).toBeNull();
		expect(screen.queryByText('Exemplare', { selector: 'dt' })).toBeNull();
	});
});
