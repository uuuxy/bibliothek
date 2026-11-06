import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import VorschlagFeld from './VorschlagFeld.svelte';

// Ein Textfeld mit Vorschlägen aus dem Bestand (Signatur, Standort, Ziel beim Zusammenführen).
// Geprüft wird, was die Liste des Browsers vorher von allein leistete: freie Eingabe, Enter an
// das Formular, Escape zuerst an die Liste, ein Klick in die Liste ohne Fokusverlust. Dazu, was
// sie nicht konnte: keine Liste bei leerem Feld ohne Wunsch, der Zusatz rechts im Eintrag.
// „Wsd“ steht vorn: Nur so zeigt sich, dass vorn Passendes beim Tippen nach vorn rückt.
const vorschlaege = [
	{ wert: 'Wsd', zusatz: '1 Titel' },
	{ wert: 'Sd', zusatz: '21 Titel' },
	{ wert: 'Sdk 2', zusatz: '4 Titel' },
	{ wert: 'Pg', zusatz: '464 Titel' }
];

/** @param {Record<string, any>} [props] */
function feld(props = {}) {
	const screen = render(VorschlagFeld, {
		value: '',
		vorschlaege,
		'aria-label': 'Signatur',
		...props
	});
	const eingabe = /** @type {HTMLInputElement} */ (screen.getByRole('combobox'));
	return { screen, eingabe };
}

/** @param {import('@testing-library/svelte').RenderResult<any>} screen */
const zeilen = (screen) =>
	screen.queryAllByRole('option').map((o) => o.textContent?.replace(/\s+/g, ' ').trim());

/** @param {HTMLInputElement} eingabe @param {string} text */
const tippe = (eingabe, text) => fireEvent.input(eingabe, { target: { value: text } });

describe('VorschlagFeld', () => {
	it('zeigt bei leerem Feld keine Liste, beim Tippen die passenden Werte, die vorn passenden zuerst', async () => {
		const { screen, eingabe } = feld();
		await fireEvent.focus(eingabe);
		expect(screen.queryByRole('listbox'), 'kein Aufklappen beim Hineinklicken').toBeNull();

		await tippe(eingabe, 'sd');
		expect(zeilen(screen)).toEqual(['Sd 21 Titel', 'Sdk 2 4 Titel', 'Wsd 1 Titel']);
		expect(eingabe.getAttribute('aria-expanded')).toBe('true');
		expect(eingabe.getAttribute('aria-controls')).toBe(screen.getByRole('listbox').id);
	});

	it('übernimmt mit Pfeil abwärts und Enter den markierten Wert und schließt die Liste', async () => {
		const { screen, eingabe } = feld();
		await tippe(eingabe, 's');
		await fireEvent.keyDown(eingabe, { key: 'ArrowDown' });
		await fireEvent.keyDown(eingabe, { key: 'ArrowDown' });
		const aktiv = eingabe.getAttribute('aria-activedescendant');
		expect(document.getElementById(/** @type {string} */ (aktiv))?.textContent).toContain('Sdk 2');

		const weiter = await fireEvent.keyDown(eingabe, { key: 'Enter' });
		expect(weiter, 'das Enter gehört der Liste, nicht dem Formular').toBe(false);
		expect(eingabe.value).toBe('Sdk 2');
		expect(screen.queryByRole('listbox')).toBeNull();
	});

	it('lässt Enter ohne markierten Vorschlag an das Formular gehen und behält den getippten Text', async () => {
		const { screen, eingabe } = feld();
		await tippe(eingabe, 'S');
		expect(screen.getByRole('listbox'), 'die Liste ist offen, nur nichts markiert').toBeTruthy();
		expect(await fireEvent.keyDown(eingabe, { key: 'Enter' })).toBe(true);
		expect(eingabe.value).toBe('S');
	});

	it('schließt mit Escape nur die Liste; das zweite Escape gehört dem Dialog', async () => {
		const { screen, eingabe } = feld();
		await tippe(eingabe, 's');
		expect(await fireEvent.keyDown(eingabe, { key: 'Escape' })).toBe(false);
		expect(screen.queryByRole('listbox')).toBeNull();
		expect(await fireEvent.keyDown(eingabe, { key: 'Escape' })).toBe(true);
	});

	it('zeigt über den Pfeil im Feld alle Werte; ein Klick übernimmt, und der Fokus bleibt im Feld', async () => {
		const { screen, eingabe } = feld();
		eingabe.focus();
		await fireEvent.click(screen.getByRole('button', { name: 'Vorschläge zeigen' }));
		expect(zeilen(screen)).toHaveLength(4);

		const pg = screen.getByRole('option', { name: /Pg/ });
		expect(await fireEvent.mouseDown(pg), 'sonst verlöre das Feld den Fokus').toBe(false);
		await fireEvent.click(pg);
		expect(eingabe.value).toBe('Pg');
		expect(document.activeElement).toBe(eingabe);
		expect(screen.queryByRole('listbox')).toBeNull();
	});

	it('hebt die Markierung auf, wenn neue Vorschläge ankommen', async () => {
		const { screen, eingabe } = feld();
		await tippe(eingabe, 's');
		await fireEvent.keyDown(eingabe, { key: 'ArrowDown' });
		expect(eingabe.getAttribute('aria-activedescendant')).toBeTruthy();

		await screen.rerender({ vorschlaege: [{ wert: 'Sz' }, ...vorschlaege] });
		expect(
			eingabe.getAttribute('aria-activedescendant'),
			'sonst zeigte sie auf eine andere Zeile'
		).toBeNull();
		expect(await fireEvent.keyDown(eingabe, { key: 'Enter' })).toBe(true);
	});

	it('schließt die Liste mit Tab, ohne die Taste zu verhindern', async () => {
		const { screen, eingabe } = feld();
		await tippe(eingabe, 's');
		expect(await fireEvent.keyDown(eingabe, { key: 'Tab' })).toBe(true);
		expect(screen.queryByRole('listbox')).toBeNull();
	});

	it('ruft das oninput des Aufrufers weiter auf, etwa für die Suche am Server', async () => {
		const oninput = vi.fn();
		const { eingabe } = feld({ oninput });
		await tippe(eingabe, 'W');
		expect(oninput).toHaveBeenCalledTimes(1);
	});

	it('zeigt ohne Vorschläge keinen Pfeil und keine Liste', async () => {
		const { screen, eingabe } = feld({ vorschlaege: [] });
		expect(screen.queryByRole('button', { name: 'Vorschläge zeigen' })).toBeNull();
		await tippe(eingabe, 's');
		expect(screen.queryByRole('listbox')).toBeNull();
		expect(eingabe.getAttribute('aria-expanded')).toBe('false');
	});
});
