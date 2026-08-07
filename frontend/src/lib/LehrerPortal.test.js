import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LehrerPortal from './LehrerPortal.svelte';
import { apiFetch } from './apiFetch.js';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));

/**
 * Eine Lehrkraft bestellt einen Klassensatz für 8a — und danach denselben Titel für 8b.
 *
 * Genau das ging nicht: Nach dem Absenden ersetzte das Badge „✓ Gesendet" den Knopf
 * dauerhaft. Aufgeräumt wird der Formularzustand aber ausgerechnet in toggleForm, und
 * das hing an diesem Knopf — der einzige Rückweg war ein Seitenreload. Kein Fehler, kein
 * Statuscode, nichts zu sehen: die Anfrage war ja erfolgreich.
 *
 * Der Test hält deshalb nicht das Badge fest, sondern die Handlungsfähigkeit danach.
 */
const TITEL = 'Seydlitz Geographie';

function suchtreffer() {
	return {
		ok: true,
		json: async () => ({ books: [{ id: 'titel-1', titel: TITEL, autor: 'Klaus Berger' }] })
	};
}

/** Sucht wie das Portal: debounced, deshalb über findBy* abwarten. */
async function sucheUndOeffneFormular(screen) {
	await fireEvent.input(screen.getByPlaceholderText(/Titel, Autor oder ISBN suchen/), {
		target: { value: 'Seydlitz' }
	});
	const knopf = await screen.findByRole('button', { name: 'Klassensatz reservieren' });
	await fireEvent.click(knopf);
	await fireEvent.input(await screen.findByLabelText('Klasse *'), { target: { value: '08a' } });
	await fireEvent.click(screen.getByRole('button', { name: /Anfrage senden/ }));
}

describe('LehrerPortal', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/search')
						? suchtreffer()
						: { ok: true, text: async () => '', json: async () => ({}) }
			)
		);
	});

	it('lässt nach einer gesendeten Anfrage sofort die nächste Klasse zu', async () => {
		const screen = render(LehrerPortal, { user: { klasse: '' } });

		await sucheUndOeffneFormular(screen);

		expect(await screen.findByText('✓ Gesendet')).toBeTruthy();

		// Der Kern: Der Weg zur nächsten Reservierung ist offen — ohne Reload.
		const erneut = await screen.findByRole('button', { name: 'Weitere Klasse reservieren' });
		expect(erneut.hasAttribute('disabled')).toBe(false);

		await fireEvent.click(erneut);
		expect(await screen.findByLabelText('Klasse *')).toBeTruthy();
	});

	it('meldet einen abgelehnten Versuch und blockiert den Knopf nicht', async () => {
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/search')
						? suchtreffer()
						: { ok: false, text: async () => 'Titel ist gesperrt.' }
			)
		);

		const screen = render(LehrerPortal, { user: { klasse: '' } });
		await sucheUndOeffneFormular(screen);

		expect(await screen.findByText('Titel ist gesperrt.')).toBeTruthy();
		expect(screen.queryByText('✓ Gesendet')).toBeNull();
		expect(screen.getByRole('button', { name: /Anfrage senden/ }).hasAttribute('disabled')).toBe(
			false
		);
	});
});
