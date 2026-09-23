import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import FilterChips from './FilterChips.svelte';

/**
 * Die Filter-Chips (M3) — zuerst für die Schlagworte in „Mein Portal" (docs/OFFEN.md 4.20).
 * Die Tests halten fest, was sie von den Nachbarn unterscheidet: Ein Chip lässt sich
 * zurücknehmen (Segmente nicht), die Auswahl steht auch ohne Farbe fest (aria-pressed,
 * Häkchen), und die Bauform folgt den M3-Token (32 px, Ecke 8 px, secondary-container).
 */
const optionen = [
	{ wert: 'a', text: 'Fantasy' },
	{ wert: 'b', text: 'Krimi' }
];

const aufbau = (/** @type {string | null} */ wert, onwahl = vi.fn()) => ({
	onwahl,
	...render(FilterChips, { optionen, wert, onwahl, etikett: 'Nach Schlagwort filtern' })
});

describe('FilterChips', () => {
	it('wählt einen Chip und nimmt den gewählten mit einem zweiten Klick zurück', async () => {
		const { getByRole, onwahl } = aufbau('a');
		await fireEvent.click(getByRole('button', { name: 'Krimi' }));
		expect(onwahl).toHaveBeenLastCalledWith('b');
		await fireEvent.click(getByRole('button', { name: 'Fantasy' }));
		expect(onwahl).toHaveBeenLastCalledWith(null);
	});

	it('meldet die Auswahl über aria-pressed und das Häkchen, nicht nur über die Farbe', () => {
		const { getByRole, container } = aufbau('b');
		expect(getByRole('button', { name: 'Krimi' }).getAttribute('aria-pressed')).toBe('true');
		expect(getByRole('button', { name: 'Fantasy' }).getAttribute('aria-pressed')).toBe('false');
		expect(container.querySelectorAll('svg').length).toBe(1);
		expect(getByRole('group', { name: 'Nach Schlagwort filtern' })).toBeTruthy();
	});

	it('trägt die M3-Bauform: 32 px, Ecke 8 px, gewählt secondary-container statt Primärfarbe', () => {
		const { getByRole } = aufbau('a');
		const klassen = (/** @type {string} */ name) =>
			(getByRole('button', { name }).getAttribute('class') || '').split(/\s+/);
		for (const name of ['Fantasy', 'Krimi']) {
			expect(klassen(name)).toContain('h-8');
			expect(klassen(name)).toContain('rounded-md');
		}
		expect(klassen('Fantasy')).toContain('bg-secondary-container');
		expect(klassen('Fantasy')).toContain('text-on-secondary-container');
		expect(klassen('Fantasy')).not.toContain('bg-primary');
		expect(klassen('Krimi')).toContain('border-outline');
		expect(klassen('Krimi')).toContain('text-on-surface-variant');
	});
});
