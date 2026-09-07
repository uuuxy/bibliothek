import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import Menue from './Menue.svelte';

// Das eine Menü des Hauses: öffnet am Knopf, meldet die Wahl, schließt mit Escape und
// gibt den Fokus zurück, Pfeiltasten wandern, gesperrte Einträge wählen nichts.
const EINTRAEGE = [
	{ id: 'a', text: 'Erster Eintrag' },
	{ id: 'b', text: 'Zweiter Eintrag', disabled: true },
	{ id: 'c', text: 'Dritter Eintrag', trennerDavor: true }
];

describe('Menue', () => {
	it('öffnet, wählt und schließt — Escape gibt den Fokus an den Knopf zurück', async () => {
		const wahl = vi.fn();
		const { getByRole, queryByRole, getAllByRole } = render(Menue, {
			etikett: 'Aktionen',
			eintraege: EINTRAEGE,
			onwahl: wahl
		});
		const knopf = getByRole('button', { name: 'Aktionen' });
		expect(queryByRole('menu')).toBeNull();

		await fireEvent.click(knopf);
		expect(getByRole('menu', { name: 'Aktionen' })).toBeTruthy();
		expect(getAllByRole('menuitem')).toHaveLength(3);
		expect(getByRole('separator')).toBeTruthy();

		await fireEvent.click(getByRole('menuitem', { name: 'Zweiter Eintrag' }));
		expect(wahl).not.toHaveBeenCalled(); // gesperrt

		await fireEvent.click(getByRole('menuitem', { name: 'Dritter Eintrag' }));
		expect(wahl).toHaveBeenCalledWith('c');
		expect(queryByRole('menu')).toBeNull();

		await fireEvent.click(knopf);
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(queryByRole('menu')).toBeNull();
		expect(document.activeElement).toBe(knopf);
	});

	it('lässt die Pfeiltasten über die Einträge wandern', async () => {
		const { getByRole, getAllByRole } = render(Menue, {
			etikett: 'Aktionen',
			eintraege: EINTRAEGE,
			onwahl: vi.fn()
		});
		await fireEvent.click(getByRole('button', { name: 'Aktionen' }));
		const menu = getByRole('menu');
		const items = getAllByRole('menuitem');
		await fireEvent.keyDown(menu, { key: 'ArrowDown' });
		expect(document.activeElement).toBe(items[1]);
		await fireEvent.keyDown(menu, { key: 'End' });
		expect(document.activeElement).toBe(items[2]);
		await fireEvent.keyDown(menu, { key: 'ArrowDown' });
		expect(document.activeElement).toBe(items[0]);
	});

	// Seit 07.09.2026: eigener Auslöser (Split-Button) und ein Kopf mit Bedienelement —
	// die zwei Fähigkeiten, die das Mahnwesen-Menü und der Ausweis-Knopf brauchen.
	it('öffnet über einen eigenen Auslöser und gibt den Fokus an dessen Chevron zurück', async () => {
		const ausloeser = createRawSnippet(
			(/** @type {() => { offen: boolean, umschalten: () => void }} */ args) => ({
				render: () =>
					`<span><button type="button">Hauptaktion</button><button type="button" aria-haspopup="menu" aria-label="Mehr wählen">▾</button></span>`,
				setup: (el) => {
					const chevron = /** @type {HTMLButtonElement} */ (el.querySelector('[aria-haspopup]'));
					chevron.onclick = () => args().umschalten();
				}
			})
		);
		const { getByRole, queryByRole } = render(Menue, {
			etikett: 'Mehr',
			eintraege: EINTRAEGE,
			onwahl: vi.fn(),
			ausloeser
		});
		expect(queryByRole('button', { name: 'Mehr' })).toBeNull(); // kein ⋮-Knopf mehr
		const chevron = getByRole('button', { name: 'Mehr wählen' });
		await fireEvent.click(chevron);
		expect(getByRole('menu')).toBeTruthy();
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(queryByRole('menu')).toBeNull();
		expect(document.activeElement, 'Fokus gehört dem Chevron, nicht der Hauptaktion').toBe(chevron);
	});

	it('trägt einen Kopf mit Bedienelement: Tab wandert hinein, Pfeile bleiben dem Feld', async () => {
		const kopf = createRawSnippet(() => ({
			render: () => `<div><label>Klasse <input aria-label="Klasse" /></label></div>`
		}));
		const wahl = vi.fn();
		const { getByRole, getAllByRole, queryByRole, getByLabelText } = render(Menue, {
			etikett: 'Aktionen',
			eintraege: [{ id: 'a', text: 'Erster Eintrag', ueberschriftDavor: 'Weitere' }],
			onwahl: wahl,
			kopf
		});
		await fireEvent.click(getByRole('button', { name: 'Aktionen' }));
		const menu = getByRole('menu');
		expect(getByLabelText('Klasse')).toBeTruthy();
		expect(menu.textContent).toContain('Weitere');

		// Fokus ins Feld im Kopf: Das Menü bleibt offen …
		const feld = getByLabelText('Klasse');
		feld.focus();
		await fireEvent.focusOut(menu, { relatedTarget: feld });
		expect(queryByRole('menu'), 'Fokus im Kopf darf das Menü nicht schließen').toBeTruthy();
		// … und ArrowDown im Feld springt NICHT auf einen Eintrag.
		await fireEvent.keyDown(feld, { key: 'ArrowDown' });
		expect(document.activeElement).toBe(feld);
		expect(getAllByRole('menuitem')).toHaveLength(1);

		// Fokus ganz hinaus: zu.
		const draussen = document.createElement('button');
		document.body.appendChild(draussen);
		await fireEvent.focusOut(menu, { relatedTarget: draussen });
		expect(queryByRole('menu')).toBeNull();
		draussen.remove();
	});
});
