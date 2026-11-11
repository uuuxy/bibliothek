import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import Dialog from './Dialog.svelte';

/** @param {string} html */
const schnipsel = (html) => createRawSnippet(() => ({ render: () => html }));

// Vier Dialoge übergaben ihren Titel nur als header („Leser anlegen", zweimal „Endgültig
// löschen?", der Versand der Mahnungen) und hatten keinen Namen: Ein Bildschirmleser sagte nur
// „Dialog", und axe meldet „aria-dialog-name".
describe('Dialog: Name', () => {
	it('benennt sich nach dem header, wenn kein Name übergeben wird', () => {
		const screen = render(Dialog, {
			open: true,
			onclose: vi.fn(),
			header: schnipsel('<h3 class="dialog-titel">Endgültig löschen?</h3>'),
			children: schnipsel('<p>Inhalt</p>')
		});
		expect(screen.getByRole('dialog', { name: 'Endgültig löschen?' })).toBeTruthy();
	});

	it('nimmt beschriftung und beschriftetDurch vor dem header (Gegenprobe)', () => {
		const a = render(Dialog, {
			open: true,
			beschriftung: 'Eigener Name',
			header: schnipsel('<h3>Kopf</h3>'),
			children: schnipsel('<p>Inhalt</p>')
		});
		expect(a.getByRole('dialog', { name: 'Eigener Name' })).toBeTruthy();
		a.unmount();

		const b = render(Dialog, {
			open: true,
			beschriftetDurch: 'fremder-titel',
			children: schnipsel('<h2 id="fremder-titel">Fremder Titel</h2>')
		});
		expect(b.getByRole('dialog', { name: 'Fremder Titel' })).toBeTruthy();
	});
});
