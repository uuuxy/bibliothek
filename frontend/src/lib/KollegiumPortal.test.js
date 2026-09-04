import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import KollegiumPortal from './KollegiumPortal.svelte';
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

/**
 * Antwort des OPAC — ein nacktes Array, kein `{books: …}`-Umschlag, und mit den
 * Bestandszahlen. Beides muss hier stimmen: Das Portal hat lange `/api/search`
 * befragt, das die Felder `verfuegbar`/`gesamt` gar nicht kennt.
 */
function suchtreffer(verfuegbar = 12, gesamt = 30) {
	return {
		ok: true,
		json: async () => [{ id: 'titel-1', titel: TITEL, autor: 'Klaus Berger', verfuegbar, gesamt }]
	};
}

/** Sucht wie das Portal: debounced, deshalb über findBy* abwarten. */
async function sucheUndOeffneFormular(screen) {
	await fireEvent.input(
		screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
		{
			target: { value: 'Seydlitz' }
		}
	);
	const knopf = await screen.findByRole('button', { name: 'Klassensatz reservieren' });
	await fireEvent.click(knopf);
	await fireEvent.input(await screen.findByLabelText('Klasse *'), { target: { value: '08a' } });
	await fireEvent.click(screen.getByRole('button', { name: /Anfrage senden/ }));
}

describe('KollegiumPortal', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/public/opac/suche')
						? suchtreffer()
						: { ok: true, text: async () => '', json: async () => ({}) }
			)
		);
	});

	it('lässt nach einer gesendeten Anfrage sofort die nächste Klasse zu', async () => {
		const screen = render(KollegiumPortal, { user: { klasse: '' } });

		await sucheUndOeffneFormular(screen);

		expect(await screen.findByText('✓ Gesendet')).toBeTruthy();

		// Der Kern: Der Weg zur nächsten Reservierung ist offen — ohne Reload.
		const erneut = await screen.findByRole('button', { name: 'Weitere Klasse reservieren' });
		expect(erneut.hasAttribute('disabled')).toBe(false);

		await fireEvent.click(erneut);
		expect(await screen.findByLabelText('Klasse *')).toBeTruthy();
	});

	/**
	 * Der Bestand MUSS am Treffer stehen — sonst kann eine Lehrkraft nicht entscheiden,
	 * ob ein Klassensatz für ihre Gruppe überhaupt reicht.
	 *
	 * Das war lange kaputt und völlig unsichtbar: Das Portal fragte `/api/search`, das
	 * `BookTitle` ohne Bestandsfeld liefert. Das Abzeichen hängt an `{#if book.verfuegbar
	 * != null}` — ein Wächter, der still übersprang. Kein Fehler, keine Lücke im Layout,
	 * die Zahl fehlte einfach. Deshalb prüft dieser Test den TEXT, nicht das Vorhandensein
	 * eines Elements.
	 */
	it('zeigt am Treffer, wie viele Exemplare frei sind und wie viele es gibt', async () => {
		const screen = render(KollegiumPortal, { user: { klasse: '' } });

		await fireEvent.input(
			screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
			{
				target: { value: 'Seydlitz' }
			}
		);

		expect(await screen.findByText('12 von 30 verfügbar')).toBeTruthy();
	});

	it('nennt bei vergriffenem Titel trotzdem den Gesamtbestand', async () => {
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/public/opac/suche')
						? suchtreffer(0, 30)
						: { ok: true, text: async () => '', json: async () => ({}) }
			)
		);

		const screen = render(KollegiumPortal, { user: { klasse: '' } });
		await fireEvent.input(
			screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
			{
				target: { value: 'Seydlitz' }
			}
		);

		// „nicht verfügbar" allein hieße für die Lehrkraft: gibt es hier gar nicht.
		expect(await screen.findByText('nicht verfügbar (30 im Bestand)')).toBeTruthy();
	});

	it('meldet einen abgelehnten Versuch und blockiert den Knopf nicht', async () => {
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/public/opac/suche')
						? suchtreffer()
						: { ok: false, text: async () => 'Titel ist gesperrt.' }
			)
		);

		const screen = render(KollegiumPortal, { user: { klasse: '' } });
		await sucheUndOeffneFormular(screen);

		expect(await screen.findByText('Titel ist gesperrt.')).toBeTruthy();
		expect(screen.queryByText('✓ Gesendet')).toBeNull();
		expect(screen.getByRole('button', { name: /Anfrage senden/ }).hasAttribute('disabled')).toBe(
			false
		);
	});
});

/**
 * Das Warteschlangen-Modell (16.08.2026): Reservieren sperrt nichts — wer denselben
 * Titel reserviert, stellt sich an. Das Portal muss beides leisten: die bestehende
 * Reservierung VOR dem Klick zeigen und nach dem Absenden sagen, hinter wem man steht.
 */
it('zeigt die Warteschlange am Treffer und nennt nach dem Absenden den Vordermann', async () => {
	vi.mocked(apiFetch).mockImplementation(
		/** @type {any} */ (
			async (/** @type {string} */ url, /** @type {any} */ opts) => {
				if (url.startsWith('/api/public/opac/suche')) return suchtreffer();
				if (url.startsWith('/api/reservierungen/klassensatz/offen')) {
					return {
						ok: true,
						json: async () => [
							{ titel_id: 'titel-1', klasse: '8a', anzahl: 28, erstellt_am: '10.08.2026' }
						]
					};
				}
				if (url === '/api/reservierungen/klassensatz' && opts?.method === 'POST') {
					return { ok: true, text: async () => '', json: async () => ({ id: 'neu' }) };
				}
				return { ok: true, text: async () => '', json: async () => ({}) };
			}
		)
	);
	const screen = render(KollegiumPortal, { user: { klasse: '' } });

	await fireEvent.input(
		screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
		{ target: { value: 'Seydlitz' } }
	);

	// Die Warteschlange steht am Treffer, BEVOR reserviert wird.
	expect(await screen.findByText('28 reserviert für 8a (seit 10.08.2026)')).toBeTruthy();
	// Und verrechnet: 12 im Regal minus 28 vorgemerkt — die Lehrkraft muss nicht rechnen.
	expect(screen.getByText('28 vorgemerkt · 0 rechnerisch frei')).toBeTruthy();

	await fireEvent.click(screen.getByRole('button', { name: 'Klassensatz reservieren' }));
	// Die Warnung steht im Formular, bevor irgendetwas gesendet wird.
	expect((await screen.findByRole('status')).textContent?.replace(/\s+/g, ' ').trim()).toBe(
		'Reicht aktuell nicht: 0 rechnerisch frei — du stellst dich hinter 8a an.'
	);
	await fireEvent.input(await screen.findByLabelText('Klasse *'), { target: { value: '9b' } });
	await fireEvent.click(screen.getByRole('button', { name: /Anfrage senden/ }));

	// Die Bestätigung sagt, hinter wem der eigene Satz an der Reihe ist.
	expect(await screen.findByTitle(/dein Satz ist nach 8a an der Reihe/)).toBeTruthy();
});
