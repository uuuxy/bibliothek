import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import Seitenblatt from './Seitenblatt.svelte';

const inhalt = createRawSnippet(() => ({ render: () => '<p>Spanisch · Jahrgang 7</p>' }));

// Das Blatt trägt die Angaben zu dem, was man angetippt hat: Überschrift und Kreuz oben, darunter
// der Inhalt des Aufrufers. Es heißt wie seine Überschrift, damit Hilfstechnik es findet.
describe('Seitenblatt', () => {
	it('zeigt Überschrift und Inhalt und heißt wie die Überschrift', () => {
		const { getByRole, getByText } = render(Seitenblatt, {
			titel: 'Encuentros 1',
			onschliessen: () => {},
			children: inhalt
		});
		expect(getByRole('complementary', { name: 'Encuentros 1' })).toBeTruthy();
		expect(getByRole('heading', { name: 'Encuentros 1' })).toBeTruthy();
		expect(getByText('Spanisch · Jahrgang 7')).toBeTruthy();
	});

	it('Escape schließt, wie bei den Dialogen', () => {
		const onschliessen = vi.fn();
		render(Seitenblatt, { titel: 'Encuentros 1', onschliessen, children: inhalt });
		window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
		expect(onschliessen).toHaveBeenCalledTimes(1);
	});

	it('das Kreuz schließt', async () => {
		const onschliessen = vi.fn();
		const { getByRole } = render(Seitenblatt, {
			titel: 'Encuentros 1',
			onschliessen,
			children: inhalt
		});
		await fireEvent.click(getByRole('button', { name: 'Schließen' }));
		expect(onschliessen).toHaveBeenCalledTimes(1);
	});
});
