import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import StudentGebuehrenCard from './StudentGebuehrenCard.svelte';
import { apiClient } from './apiFetch.js';

vi.mock('./apiFetch.js', () => ({ apiClient: { post: vi.fn() } }));

/**
 * Die zwei Erledigungs-Wege einer Gebühr: Der Storno darf NIE ohne Grund abgehen
 * (das Backend lehnt ihn mit 400 ab, aber die UI soll es gar nicht erst anbieten),
 * und ein erledigter Fall darf keine Aktionsknöpfe mehr zeigen — sonst kassiert
 * die zweite Kollegin am Mehrplatz-System einen nackten Fehler statt einer Antwort.
 */

const OFFEN = {
	id: 'sf-1',
	beschreibung: 'Wasserschaden',
	betrag: 12.5,
	ist_bezahlt: false,
	erstellt_am: '2026-08-01T10:00:00Z',
	storniert_am: null,
	stornierungsgrund: null,
	titel: 'Mathematik 8',
	barcode_id: 'B-1'
};

const STORNIERT = {
	...OFFEN,
	id: 'sf-2',
	ist_bezahlt: true,
	storniert_am: '2026-08-02T10:00:00Z',
	stornierungsgrund: 'Buch wiedergefunden'
};

beforeEach(() => {
	vi.mocked(apiClient.post).mockReset();
});

describe('StudentGebuehrenCard', () => {
	it('zeigt Aktionen nur am offenen Fall — und nur mit Bearbeitungsrecht', () => {
		const screen = render(StudentGebuehrenCard, {
			props: { gebuehren: [OFFEN, STORNIERT], canEdit: true, onChanged: () => {} }
		});
		// Ein offener Fall: je ein Bezahlt-/Storno-Knopf; der stornierte Fall trägt keine.
		expect(screen.getAllByRole('button', { name: /Bezahlt/ })).toHaveLength(1);
		expect(screen.getAllByRole('button', { name: /Stornieren/ })).toHaveLength(1);
		expect(screen.getByText('storniert')).toBeTruthy();
		expect(screen.getByText('Grund: Buch wiedergefunden')).toBeTruthy();
		screen.unmount();

		const ohneRecht = render(StudentGebuehrenCard, {
			props: { gebuehren: [OFFEN], canEdit: false, onChanged: () => {} }
		});
		expect(ohneRecht.queryByRole('button', { name: /Bezahlt/ })).toBeNull();
	});

	it('Storno geht nur mit Grund raus — und der Grund bleibt nicht im Modal stehen', async () => {
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({}) })
		);
		const onChanged = vi.fn();
		const screen = render(StudentGebuehrenCard, {
			props: { gebuehren: [OFFEN], canEdit: true, onChanged }
		});

		await fireEvent.click(screen.getByRole('button', { name: /^Stornieren$/ }));
		const bestaetigen = /** @type {HTMLButtonElement | undefined} */ (
			screen.getAllByRole('button', { name: /Stornieren/ }).find((b) => b.closest('.fixed'))
		);
		if (!bestaetigen) throw new Error('Bestätigen-Knopf im Modal nicht gefunden');
		// Ohne Grund ist der Bestätigen-Knopf gesperrt — kein Request.
		expect(bestaetigen.disabled).toBe(true);

		await fireEvent.input(screen.getByLabelText('Stornierungsgrund'), {
			target: { value: '  Kulanz  ' }
		});
		expect(bestaetigen.disabled).toBe(false);
		await fireEvent.click(bestaetigen);

		expect(apiClient.post).toHaveBeenCalledWith('/api/schadensfaelle/sf-1/storno', {
			grund: 'Kulanz'
		});
		expect(onChanged).toHaveBeenCalled();
		// Modal zu, Grund geleert — beim nächsten Fall darf nichts vorausgefüllt sein.
		expect(screen.queryByLabelText('Stornierungsgrund')).toBeNull();
	});

	it('Bezahlt schickt keinen Body-Betrag — der kommt aus der Datenbank', async () => {
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({}) })
		);
		const screen = render(StudentGebuehrenCard, {
			props: { gebuehren: [OFFEN], canEdit: true, onChanged: () => {} }
		});
		await fireEvent.click(screen.getByRole('button', { name: /Bezahlt/ }));
		expect(apiClient.post).toHaveBeenCalledWith('/api/schadensfaelle/sf-1/bezahlt', {});
	});

	it('zeigt die Fehlermeldung des Servers (409-Konflikt), nicht nur "fehlgeschlagen"', async () => {
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				json: async () => ({ error: 'Schadensfall wurde bereits bezahlt oder storniert' })
			})
		);
		const onChanged = vi.fn();
		const screen = render(StudentGebuehrenCard, {
			props: { gebuehren: [OFFEN], canEdit: true, onChanged }
		});
		await fireEvent.click(screen.getByRole('button', { name: /Bezahlt/ }));
		expect(onChanged).not.toHaveBeenCalled();
	});
});
