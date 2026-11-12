import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import LmfPlanReihenfolge from './LmfPlanReihenfolge.svelte';

// Die Besonderheiten einer Planer-Zeile stehen als Text, ein Klick öffnet an ihrer Stelle das
// Feld (LmfPlanVermerkZelle). Die Zeilen kommen wie im Planer aus einem $state: Aus einem
// schlichten Objekt sähe die Zeile das Tippen nicht.
/** @param {any[]} anfang */
function zeige(anfang) {
	const zeilen = $state(anfang);
	return render(LmfPlanReihenfolge, {
		zeilen,
		plaetze: anfang.map((_, i) => ({ datum: '2027-06-28', stunde: i + 1 })),
		marker: { ohneSchueler: () => false },
		bereit: true,
		ausgelassen: [],
		draussen: () => false,
		markiert: null,
		onklasseraus: vi.fn(),
		onhinein: vi.fn(),
		ontausch: vi.fn()
	});
}

describe('LmfPlanVermerkZelle', () => {
	it('zeigt die Besonderheiten als Text; ein Klick öffnet das Feld, Escape lässt den alten Text', async () => {
		const { container, getByRole, getByLabelText, queryByLabelText } = zeige([
			{ klassen: ['10R1'], vermerk: 'nur Rückgabe' },
			{ klassen: ['10R2'], vermerk: '' }
		]);
		// Kein Eingabefeld, solange niemand schreibt: sechzig leere Kästen wären ein Formular.
		expect(container.querySelectorAll('input[id^="lmf-zeile-vermerk-"]')).toHaveLength(0);
		const text = getByRole('button', { name: /^Besonderheiten Zeile 1:/ });
		expect(text.textContent).toContain('nur Rückgabe');

		await fireEvent.click(text);
		await tick();
		const feld = getByLabelText('Besonderheiten Zeile 1');
		expect(document.activeElement).toBe(feld);
		await fireEvent.input(feld, { target: { value: 'nur Rückgabe, Raum 12' } });
		await fireEvent.keyDown(feld, { key: 'Escape' });
		expect(queryByLabelText('Besonderheiten Zeile 1')).toBeNull();
		const danach = getByRole('button', { name: /^Besonderheiten Zeile 1:/ }).textContent;
		expect(danach).toContain('nur Rückgabe');
		expect(danach).not.toContain('Raum 12');
	});

	it('übernimmt die Besonderheiten mit Enter und wenn der Fokus das Feld verlässt', async () => {
		const { getByRole, getByLabelText, queryByLabelText } = zeige([
			{ klassen: ['10R1'], vermerk: '' }
		]);
		await fireEvent.click(getByRole('button', { name: /^Besonderheiten Zeile 1:/ }));
		let feld = getByLabelText('Besonderheiten Zeile 1');
		await fireEvent.input(feld, { target: { value: 'Raum 12' } });
		await fireEvent.keyDown(feld, { key: 'Enter' });
		expect(queryByLabelText('Besonderheiten Zeile 1')).toBeNull();
		expect(getByRole('button', { name: /^Besonderheiten Zeile 1:/ }).textContent).toContain(
			'Raum 12'
		);

		await fireEvent.click(getByRole('button', { name: /^Besonderheiten Zeile 1:/ }));
		feld = getByLabelText('Besonderheiten Zeile 1');
		await fireEvent.input(feld, { target: { value: 'Raum 14' } });
		await fireEvent.blur(feld);
		expect(queryByLabelText('Besonderheiten Zeile 1')).toBeNull();
		expect(getByRole('button', { name: /^Besonderheiten Zeile 1:/ }).textContent).toContain(
			'Raum 14'
		);
	});

	it('lässt das Feld einer Zeile ohne Klasse offen und rot, solange ihr Text fehlt', async () => {
		const { getByLabelText } = zeige([{ klassen: [], vermerk: '' }]);
		const feld = getByLabelText('Besonderheiten Zeile 1');
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		// Beim Tippen bleibt das Feld stehen, auch wenn der erste Buchstabe die Pflicht erfüllt.
		await fireEvent.focus(feld);
		await fireEvent.input(feld, { target: { value: 'N' } });
		expect(getByLabelText('Besonderheiten Zeile 1')).toBe(feld);
		expect(feld.getAttribute('aria-invalid')).toBeNull();
	});
	it('kürzt einen langen Text auf eine Zeile; ganz steht er beim Zeigen und im Feld', async () => {
		// Die Zeile bleibt 36 px hoch wie das Feld; gekürzt wird nur mit Weg zum Rest.
		const lang = 'Nur Rückgabe — Abschlussklasse, Bücher bitte gebündelt in die Bibliothek';
		const { getByRole, getByLabelText } = zeige([{ klassen: ['10R1'], vermerk: lang }]);
		const knopf = getByRole('button', { name: /^Besonderheiten Zeile 1:/ });
		expect(knopf.getAttribute('title')).toBe(lang);
		expect(knopf.querySelector('.truncate')?.textContent).toBe(lang);
		await fireEvent.click(knopf);
		expect(/** @type {HTMLInputElement} */ (getByLabelText('Besonderheiten Zeile 1')).value).toBe(
			lang
		);
	});

	it('zeigt einen Text nur aus Leerzeichen wie eine leere Zelle', () => {
		const { getByRole } = zeige([{ klassen: ['10R1'], vermerk: '   ' }]);
		const knopf = getByRole('button', { name: /^Besonderheiten Zeile 1:/ });
		expect(knopf.textContent).toContain('Besonderheit eintragen');
		expect(knopf.hasAttribute('title')).toBe(false);
	});
});
