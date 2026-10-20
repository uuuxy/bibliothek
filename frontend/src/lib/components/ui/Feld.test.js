import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import Feld from './Feld.svelte';

// Der Fehlerzustand nach Material 3, Text fields: „It's strongly recommended to show an error
// icon when the text field is in the error state". Die Lage des Symbols misst
// e2e/feld-fehlersymbol.spec.js im Browser; hier stehen Aufbau und Verhalten.
const symbol = (/** @type {HTMLElement} */ wurzel) =>
	wurzel.querySelector('svg.lucide-circle-alert');

const knopf = createRawSnippet(() => ({
	render: () => '<button type="button" aria-label="Zurücksetzen">↺</button>'
}));

describe('Feld: Fehlerzustand', () => {
	it('zeigt das Fehlersymbol und färbt die Beschriftung, nur solange der Fehler steht', async () => {
		const k = render(Feld, { etikett: 'Kürzel', ungueltig: false, hint: '' });
		const beschriftung = k.getByText('Kürzel');
		expect(symbol(k.container)).toBeNull();
		expect(beschriftung.className).toContain('text-on-surface-variant');

		await k.rerender({ ungueltig: true, hint: 'Bitte ein Kürzel eintragen.' });
		expect(symbol(k.container)).not.toBeNull();
		expect(symbol(k.container)?.getAttribute('aria-hidden')).toBe('true');
		expect(beschriftung.className).toContain('text-error');
		expect(k.getByLabelText('Kürzel').getAttribute('aria-invalid')).toBe('true');

		await k.rerender({ ungueltig: false, hint: '' });
		expect(symbol(k.container)).toBeNull();
	});

	it('baut das Feld nicht neu, wenn der Fehler kommt und geht: Element und Fokus bleiben', async () => {
		const k = render(Feld, { 'aria-label': 'E-Mail', ungueltig: false });
		const feld = k.getByLabelText('E-Mail');
		feld.focus();

		await k.rerender({ ungueltig: true });
		expect(k.getByLabelText('E-Mail')).toBe(feld);
		expect(document.activeElement).toBe(feld);

		await k.rerender({ ungueltig: false });
		expect(k.getByLabelText('E-Mail')).toBe(feld);
		expect(document.activeElement).toBe(feld);
	});

	it('lässt nachlaufende Knöpfe stehen und setzt das Symbol davor', () => {
		const k = render(Feld, { 'aria-label': 'Jahr', ungueltig: true, nachlaufend: knopf });
		const zurueck = k.getByRole('button', { name: 'Zurücksetzen' });
		const zeichen = symbol(k.container);
		expect(zeichen).not.toBeNull();
		expect(
			(zeichen?.compareDocumentPosition(zurueck) ?? 0) & Node.DOCUMENT_POSITION_FOLLOWING
		).toBeTruthy();
	});

	it('legt um ein Feld ohne Fehlerzustand und ohne Symbole keine Hülle', () => {
		const k = render(Feld, { 'aria-label': 'Suche' });
		expect(k.getByLabelText('Suche').parentElement).toBe(k.container);
	});
});
