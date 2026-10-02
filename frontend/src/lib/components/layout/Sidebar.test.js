import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { authStore } from '../../stores/authStore.svelte.js';
import { uiStore } from '../../stores/uiStore.svelte.js';
import Sidebar from './Sidebar.svelte';

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
		const screen = render(Sidebar);
		expect(screen.getByRole('button', { name: 'Navigation einklappen' })).toBeTruthy();
		expect(screen.getByText('Medienkatalog')).toBeTruthy();
		expect(abfragen).toContain('(width < 80rem)');
	});

	it('beginnt im schmalen Fenster eingeklappt; der Name steht dann am Symbol', () => {
		fenster.matches = true;
		const screen = render(Sidebar);
		expect(screen.getByRole('button', { name: 'Navigation ausklappen' })).toBeTruthy();
		expect(screen.queryByText('Medienkatalog')).toBeNull();
		expect(screen.getByTitle('Medienkatalog')).toBeTruthy();
	});

	it('folgt der Fensterbreite, solange niemand gewählt hat', async () => {
		const screen = render(Sidebar);
		await fenster.wird(true);
		expect(screen.getByRole('button', { name: 'Navigation ausklappen' })).toBeTruthy();
		await fenster.wird(false);
		expect(screen.getByRole('button', { name: 'Navigation einklappen' })).toBeTruthy();
	});

	it('behält die Wahl am Doppelpfeil, auch wenn das Fenster schmal ist oder wird', async () => {
		fenster.matches = true;
		const screen = render(Sidebar);
		await fireEvent.click(screen.getByRole('button', { name: 'Navigation ausklappen' }));
		expect(screen.getByText('Medienkatalog')).toBeTruthy();

		await fenster.wird(false);
		await fenster.wird(true);
		expect(screen.getByRole('button', { name: 'Navigation einklappen' })).toBeTruthy();
	});

	it('behält auch das Einklappen im breiten Fenster', async () => {
		const screen = render(Sidebar);
		await fireEvent.click(screen.getByRole('button', { name: 'Navigation einklappen' }));
		expect(screen.queryByText('Medienkatalog')).toBeNull();

		await fenster.wird(true);
		await fenster.wird(false);
		expect(screen.getByRole('button', { name: 'Navigation ausklappen' })).toBeTruthy();
	});
});
