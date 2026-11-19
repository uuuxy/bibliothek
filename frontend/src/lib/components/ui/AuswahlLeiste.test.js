import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import AuswahlLeiste from './AuswahlLeiste.svelte';

// Die Leiste für markierte Zeilen ist Googles Floating toolbar: M3, Toolbars, Specs: „By default
// all toolbars are 64dp high", „8dp padding between items"; material-components-android
// (FloatingToolbar.md): Fläche colorSurfaceContainer, Form „50% rounded".
describe('AuswahlLeiste: Form nach M3', () => {
	it('ist 64 dp hoch, voll gerundet, auf surface-container und mit 8 dp zwischen den Teilen', () => {
		const { getByRole } = render(AuswahlLeiste, {
			satz: '3 ausgewählt',
			beschriftung: 'Ausgewählte Zeilen',
			onleeren: () => {},
			children: createRawSnippet(() => ({ render: () => '<button>Löschen</button>' }))
		});
		const leiste = /** @type {HTMLElement} */ (
			getByRole('region', { name: 'Ausgewählte Zeilen' }).firstElementChild
		);
		expect(leiste.className.split(/\s+/)).toEqual(
			expect.arrayContaining(['h-16', 'rounded-full', 'bg-surface-container', 'gap-2'])
		);
		expect(getByRole('button', { name: 'Auswahl aufheben' })).toBeTruthy();
	});
});
