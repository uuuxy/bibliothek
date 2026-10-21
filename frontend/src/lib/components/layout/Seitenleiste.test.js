import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { authStore } from '../../stores/authStore.svelte.js';
import { uiStore } from '../../stores/uiStore.svelte.js';
import Seitenleiste from './Seitenleiste.svelte';

/** Die Antwort des Browsers auf die Breitenabfrage; `matches` heißt „schmaler als 1280 px“. */
class Fensterbreite extends EventTarget {
	matches = false;
	/** @param {boolean} schmal */
	async wird(schmal) {
		this.matches = schmal;
		this.dispatchEvent(new Event('change'));
		await tick();
	}
}

// Unter 1280 px Fensterbreite beginnt die Navigation eingeklappt, damit der Inhalt daneben
// Platz hat (Leserakte: Karte und Ausleihliste nebeneinander). Wer den Doppelpfeil benutzt,
// hat gewählt; die Wahl gilt vor der Fensterbreite.
describe('Seitenleiste: eingeklappt nach Fensterbreite', () => {
	const vorher = window.matchMedia;
	/** @type {Fensterbreite} */
	let fenster;
	/** @type {string[]} */
	let abfragen;

	beforeEach(() => {
		fenster = new Fensterbreite();
		abfragen = [];
		window.matchMedia = /** @type {any} */ (
			(/** @type {string} */ abfrage) => {
				abfragen.push(abfrage);
				return fenster;
			}
		);
		uiStore.sidebarWahl = null;
		authStore.currentUser = {
			vorname: 'Kim',
			nachname: 'Probe',
			rolle: 'mitarbeiter',
			permissions: ['view_books']
		};
	});
	afterEach(() => {
		window.matchMedia = vorher;
		authStore.currentUser = null;
		uiStore.sidebarWahl = null;
	});

	it('steht im breiten Fenster ausgeklappt, mit den Namen der Menüpunkte', () => {
		const screen = render(Seitenleiste);
		expect(screen.getByRole('button', { name: 'Navigation einklappen' })).toBeTruthy();
		expect(screen.getByText('Medienkatalog')).toBeTruthy();
		expect(abfragen).toContain('(width < 80rem)');
	});

	it('beginnt im schmalen Fenster eingeklappt; der Name steht dann am Symbol', () => {
		fenster.matches = true;
		const screen = render(Seitenleiste);
		expect(screen.getByRole('button', { name: 'Navigation ausklappen' })).toBeTruthy();
		expect(screen.queryByText('Medienkatalog')).toBeNull();
		expect(screen.getByTitle('Medienkatalog')).toBeTruthy();
	});

	it('folgt der Fensterbreite, solange niemand gewählt hat', async () => {
		const screen = render(Seitenleiste);
		await fenster.wird(true);
		expect(screen.getByRole('button', { name: 'Navigation ausklappen' })).toBeTruthy();
		await fenster.wird(false);
		expect(screen.getByRole('button', { name: 'Navigation einklappen' })).toBeTruthy();
	});

	it('behält die Wahl am Doppelpfeil, auch wenn das Fenster schmal ist oder wird', async () => {
		fenster.matches = true;
		const screen = render(Seitenleiste);
		await fireEvent.click(screen.getByRole('button', { name: 'Navigation ausklappen' }));
		expect(screen.getByText('Medienkatalog')).toBeTruthy();

		await fenster.wird(false);
		await fenster.wird(true);
		expect(screen.getByRole('button', { name: 'Navigation einklappen' })).toBeTruthy();
	});

	it('behält auch das Einklappen im breiten Fenster', async () => {
		const screen = render(Seitenleiste);
		await fireEvent.click(screen.getByRole('button', { name: 'Navigation einklappen' }));
		expect(screen.queryByText('Medienkatalog')).toBeNull();

		await fenster.wird(true);
		await fenster.wird(false);
		expect(screen.getByRole('button', { name: 'Navigation ausklappen' })).toBeTruthy();
	});

	// Der Knopf „System" nennt den Bereich, den er aufklappt, über dessen Kennung. Beide stehen
	// an zwei Stellen der Datei; der Test hält sie als Paar, damit keine allein umbenannt wird.
	it('nennt am Knopf „System" die Kennung des Bereichs, den er aufklappt', async () => {
		authStore.currentUser = { ...authStore.currentUser, permissions: ['audit_logs'] };
		const screen = render(Seitenleiste);
		const knopf = screen.getByRole('button', { name: 'System' });
		expect(screen.queryByText('System-Logs')).toBeNull();

		await fireEvent.click(knopf);
		expect(knopf.getAttribute('aria-expanded')).toBe('true');
		const bereich = screen.getByText('System-Logs').closest('[id]');
		expect(bereich?.id).toBeTruthy();
		expect(knopf.getAttribute('aria-controls')).toBe(bereich?.id);
	});
});

// Eingeklappt klappt die Leiste wie bei Gmail auf, solange die Maus über ihr steht: erst nach
// 200 ms, nur für die Maus, und ohne dass die Wahl am Doppelpfeil sich ändert.
describe('Seitenleiste: klappt beim Überfahren über dem Inhalt auf', () => {
	const vorher = window.matchMedia;

	beforeEach(() => {
		vi.useFakeTimers();
		window.matchMedia = /** @type {any} */ (
			() => Object.assign(new EventTarget(), { matches: true })
		);
		uiStore.sidebarWahl = null;
		authStore.currentUser = {
			vorname: 'Kim',
			nachname: 'Probe',
			rolle: 'mitarbeiter',
			permissions: ['view_books']
		};
	});
	afterEach(() => {
		vi.useRealTimers();
		window.matchMedia = vorher;
		authStore.currentUser = null;
		uiStore.sidebarWahl = null;
	});

	/** @param {import('@testing-library/svelte').RenderResult<any>} screen @param {number} ms */
	async function zeigeMitMaus(screen, ms = 200) {
		const leiste = /** @type {HTMLElement} */ (screen.container.querySelector('aside'));
		await fireEvent.pointerEnter(leiste, { pointerType: 'mouse' });
		vi.advanceTimersByTime(ms);
		await tick();
		return leiste;
	}

	it('zeigt die Namen erst nach 200 ms und nimmt sie beim Verlassen wieder weg', async () => {
		const screen = render(Seitenleiste);
		const leiste = await zeigeMitMaus(screen, 199);
		expect(screen.queryByText('Medienkatalog')).toBeNull();

		vi.advanceTimersByTime(1);
		await tick();
		expect(screen.getByText('Medienkatalog')).toBeTruthy();
		// Die Wahl bleibt „eingeklappt": Der Doppelpfeil bietet weiter das Ausklappen an.
		expect(screen.getByRole('button', { name: 'Navigation ausklappen' })).toBeTruthy();

		await fireEvent.pointerLeave(leiste, { pointerType: 'mouse' });
		expect(screen.queryByText('Medienkatalog')).toBeNull();
	});

	it('klappt bei Touch und Stift nicht auf', async () => {
		const screen = render(Seitenleiste);
		const leiste = /** @type {HTMLElement} */ (screen.container.querySelector('aside'));
		for (const pointerType of ['touch', 'pen']) {
			await fireEvent.pointerEnter(leiste, { pointerType });
			vi.advanceTimersByTime(500);
			await tick();
			expect(screen.queryByText('Medienkatalog'), pointerType).toBeNull();
		}
	});

	it('klappt nach der Wahl eines Ziels wieder ein, obwohl die Maus noch darüber steht', async () => {
		const screen = render(Seitenleiste);
		await zeigeMitMaus(screen);
		await fireEvent.click(screen.getByText('Medienkatalog'));
		expect(uiStore.activeTab).toBe('media_catalog');
		vi.advanceTimersByTime(500);
		await tick();
		expect(screen.queryByText('Medienkatalog')).toBeNull();
	});

	it('hält die Wahl am Doppelpfeil; nach dem Einklappen bleibt die Leiste unter der Maus schmal', async () => {
		const screen = render(Seitenleiste);
		await zeigeMitMaus(screen);
		await fireEvent.click(screen.getByRole('button', { name: 'Navigation ausklappen' }));
		expect(uiStore.sidebarWahl).toBe(false);
		expect(screen.getByText('Medienkatalog')).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: 'Navigation einklappen' }));
		vi.advanceTimersByTime(500);
		await tick();
		expect(screen.queryByText('Medienkatalog')).toBeNull();
	});
});
