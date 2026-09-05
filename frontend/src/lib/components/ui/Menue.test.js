import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
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
});
