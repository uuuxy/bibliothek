import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import BestellDetailPositionen from './BestellDetailPositionen.svelte';

// Zwei Positionen mit gleichem Titel ohne ISBN (zwei Ausgaben eines Arbeitshefts) gaben mit Titel
// und ISBN denselben Schlüssel, und Svelte brach die Bestellung mit each_key_duplicate ab. Jede
// Position hat ihre Kennung.
describe('Positionen einer Bestellung', () => {
	it('zeigt zwei Positionen mit gleichem Titel ohne ISBN', () => {
		const position = {
			titel_name: 'Lesebuch 5',
			isbn: '',
			menge: 1,
			einzelpreis: 1000,
			gesamtpreis: 1000,
			etiketten_offen: 0
		};
		const screen = render(BestellDetailPositionen, {
			positionen: [
				{ ...position, id: 'p1' },
				{ ...position, id: 'p2', menge: 2 }
			],
			euro: (/** @type {number} */ n) => `${n} €`,
			onNachdruck: vi.fn(),
			onTitel: vi.fn()
		});

		expect(screen.getAllByText('Lesebuch 5')).toHaveLength(2);
	});
});
