import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ChipFeld from './ChipFeld.svelte';

/**
 * Das ChipFeld nimmt mehrere freie Werte (Schlagworte am Titel, Migration 138). Diese
 * Tests halten fest, wann ein Wert übernommen wird — und wann nicht: Getippter Text wird
 * erst mit Enter, Komma, Vorschlags-Auswahl oder beim Verlassen des Feldes zum Chip,
 * nie Buchstabe für Buchstabe.
 */
const aufbau = (/** @type {Record<string, any>} */ props = {}) =>
	render(ChipFeld, { label: 'Schlagworte', ...props });

/** Getippter Text: Der Browser meldet ihn als insertText, nicht als Auswahl. */
const tippe = (/** @type {HTMLElement} */ feld, /** @type {string} */ text) =>
	fireEvent.input(feld, { target: { value: text }, inputType: 'insertText' });

const chips = (/** @type {HTMLElement} */ container) =>
	[...container.querySelectorAll('li')].map((li) => (li.textContent || '').trim());

describe('ChipFeld', () => {
	it('übernimmt mit Enter und leert das Feld', async () => {
		const { getByLabelText, container } = aufbau();
		const feld = /** @type {HTMLInputElement} */ (getByLabelText('Schlagworte'));
		await tippe(feld, 'Fantasy');
		expect(chips(container)).toEqual([]);
		await fireEvent.keyDown(feld, { key: 'Enter' });
		expect(chips(container)).toEqual(['Fantasy']);
		expect(feld.value).toBe('');
	});

	it('übernimmt mit Komma, und Kommas trennen auch beim Verlassen des Feldes', async () => {
		const { getByLabelText, container } = aufbau();
		const feld = getByLabelText('Schlagworte');
		await tippe(feld, 'Krimi');
		await fireEvent.keyDown(feld, { key: ',' });
		await tippe(feld, 'Pferde,  Freundschaft ');
		await fireEvent.blur(feld);
		expect(chips(container)).toEqual(['Krimi', 'Pferde', 'Freundschaft']);
	});

	it('übernimmt eine Auswahl aus der Vorschlagsliste sofort, getippten Text nicht', async () => {
		const { getByLabelText, container } = aufbau({ vorschlaege: [{ wert: 'Fantasy' }] });
		const feld = getByLabelText('Schlagworte');
		await tippe(feld, 'Fantasy');
		expect(chips(container)).toEqual([]);
		await fireEvent.input(feld, {
			target: { value: 'Fantasy' },
			inputType: 'insertReplacementText'
		});
		expect(chips(container)).toEqual(['Fantasy']);
	});

	it('zählt Doppelte ohne Groß- und Kleinschreibung, und ein Vorschlag gibt die Schreibweise vor', async () => {
		const { getByLabelText, container } = aufbau({
			werte: ['Fantasy'],
			vorschlaege: [{ wert: 'Science-Fiction', beschreibung: '4 Titel' }]
		});
		const feld = getByLabelText('Schlagworte');
		await tippe(feld, 'fantasy');
		await fireEvent.keyDown(feld, { key: 'Enter' });
		await tippe(feld, 'science-fiction');
		await fireEvent.keyDown(feld, { key: 'Enter' });
		expect(chips(container)).toEqual(['Fantasy', 'Science-Fiction']);
	});

	it('entfernt mit dem × und gibt den Fokus ans Feld zurück', async () => {
		const { getByLabelText, getByRole, container } = aufbau({ werte: ['Krimi', 'Pferde'] });
		await fireEvent.click(getByRole('button', { name: '„Krimi“ entfernen' }));
		expect(chips(container)).toEqual(['Pferde']);
		expect(document.activeElement).toBe(getByLabelText('Schlagworte'));
	});

	it('meldet die Grenze im Fehlerzustand, statt still nichts zu tun', async () => {
		const { getByLabelText, getByText, container } = aufbau({ werte: ['A', 'B'], max: 2 });
		const feld = getByLabelText('Schlagworte');
		await tippe(feld, 'C');
		await fireEvent.keyDown(feld, { key: 'Enter' });
		expect(chips(container)).toEqual(['A', 'B']);
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		expect(getByText(/Höchstens 2/)).toBeTruthy();
	});

	it('hat die Chip-Form des Hauses: 32 px, Radius 8 px, secondary-container, × auf 32 × 32 px', () => {
		const { getByRole, container } = aufbau({ werte: ['Krimi'] });
		const chip = (container.querySelector('li')?.getAttribute('class') || '').split(/\s+/);
		expect(chip).toEqual(
			expect.arrayContaining([
				'h-8',
				'rounded-md',
				'bg-secondary-container',
				'text-on-secondary-container'
			])
		);
		const knopf = (
			getByRole('button', { name: '„Krimi“ entfernen' }).getAttribute('class') || ''
		).split(/\s+/);
		expect(knopf).toEqual(expect.arrayContaining(['h-8', 'w-8']));
	});
});

describe('ChipFeld ohne geladene Werte', () => {
	// null kommt aus der Katalogliste („nicht geladen"). Das Feld zeigt dann nichts und
	// bricht nicht ab; ob es offen ist, entscheidet der Aufrufer über disabled.
	it('zeigt bei werte = null nichts an und bricht nicht ab', () => {
		const { container, getByLabelText } = aufbau({ werte: null, disabled: true });
		expect(chips(container)).toEqual([]);
		expect(/** @type {HTMLInputElement} */ (getByLabelText('Schlagworte')).disabled).toBe(true);
	});
});

// Angebote (M3 Suggestion chips): Werte zum Anklicken unter den Chips — zuerst der
// Schlagwort-Vorschlag aus der DNB beim Bestellen per ISBN (docs/OFFEN.md 4.20). Eingetragen
// ist nichts, bis jemand klickt; und ein Angebot unterliegt denselben Regeln wie Getipptes.
describe('ChipFeld mit Angeboten', () => {
	const angebot = (/** @type {any} */ screen, /** @type {string} */ wert) =>
		screen.queryByRole('button', { name: `„${wert}“ übernehmen` });

	it('übernimmt ein Angebot mit einem Klick und nimmt es aus der Zeile', async () => {
		const screen = aufbau({ angebote: ['Krieg', 'Erste Liebe'], angeboteEtikett: 'Aus der DNB' });
		expect(chips(screen.container)).toEqual([]);
		expect(screen.getByRole('group', { name: 'Aus der DNB' })).toBeTruthy();
		await fireEvent.click(angebot(screen, 'Krieg'));
		expect(chips(screen.container)).toEqual(['Krieg']);
		expect(angebot(screen, 'Krieg')).toBeNull();
		expect(angebot(screen, 'Erste Liebe')).toBeTruthy();
	});

	it('bietet nicht an, was schon gewählt ist — ohne Rücksicht auf Groß- und Kleinschreibung', () => {
		const screen = aufbau({ werte: ['krieg'], angebote: ['Krieg'] });
		expect(angebot(screen, 'Krieg')).toBeNull();
		expect(screen.queryByRole('group')).toBeNull();
	});

	it('hält die Obergrenze auch für Angebote und sperrt sie mit dem Feld', async () => {
		const voll = aufbau({ werte: ['A', 'B'], max: 2, angebote: ['C'] });
		await fireEvent.click(angebot(voll, 'C'));
		expect(chips(voll.container)).toEqual(['A', 'B']);
		expect(voll.getByText(/Höchstens 2/)).toBeTruthy();
		voll.unmount();

		const gesperrt = aufbau({ angebote: ['C'], disabled: true });
		expect(/** @type {HTMLButtonElement} */ (angebot(gesperrt, 'C')).disabled).toBe(true);
	});
});
