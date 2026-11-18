import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Datumsfeld from './Datumsfeld.svelte';

// Das Datumsfeld nach M3 (Docked date picker): tippen ohne Maske, ordentlich schreiben beim
// Verlassen, den Kalender nur über den Knopf, Tastatur und Rollen wie in Angular Material.
describe('Datumsfeld', () => {
	beforeEach(() => {
		vi.useFakeTimers({ toFake: ['Date'] });
		vi.setSystemTime(new Date(2026, 7, 10, 9, 0));
	});
	afterEach(() => vi.useRealTimers());

	/** @param {string} value */
	function zeige(value) {
		const props = $state({ value, etikett: 'Zahlungsfrist' });
		const r = render(Datumsfeld, { props });
		const feld = /** @type {HTMLInputElement} */ (r.getByLabelText('Zahlungsfrist'));
		return { props, feld, ...r };
	}

	/** @param {HTMLElement} feld */
	function beschreibung(feld) {
		return document
			.getElementById(feld.getAttribute('aria-describedby') ?? '')
			?.textContent?.trim();
	}

	it('zeigt TT.MM.JJJJ, nimmt Getipptes an und schreibt es erst beim Verlassen ordentlich', async () => {
		const { props, feld, getByText } = zeige('2026-08-01');
		expect(feld.value).toBe('01.08.2026');
		expect(getByText('TT.MM.JJJJ')).toBeTruthy();
		await fireEvent.input(feld, { target: { value: '3.9.2026' } });
		expect(props.value).toBe('2026-09-03');
		expect(feld.value).toBe('3.9.2026');
		await fireEvent.blur(feld);
		expect(feld.value).toBe('03.09.2026');
		expect(feld.getAttribute('aria-invalid')).not.toBe('true');
	});

	it('meldet eine Eingabe, die kein Datum ist, lässt sie stehen und gibt kein Datum weiter', async () => {
		const { props, feld } = zeige('2026-08-01');
		await fireEvent.input(feld, { target: { value: '31.02.2026' } });
		await fireEvent.blur(feld);
		expect(props.value).toBe('');
		expect(feld.value).toBe('31.02.2026');
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(feld)).toBe('Datum entspricht nicht dem erwarteten Format: TT.MM.JJJJ');
	});

	it('ersetzt das Format durch den Fehlertext, statt ihn anzuhängen', () => {
		const { getByLabelText } = render(Datumsfeld, {
			props: { value: '2026-08-01', etikett: 'Bis', ungueltig: true, hint: 'Ende nach dem Beginn' }
		});
		expect(beschreibung(/** @type {HTMLInputElement} */ (getByLabelText('Bis')))).toBe(
			'Ende nach dem Beginn'
		);
	});

	it('öffnet den Kalender über den Knopf und übernimmt den Tag mit Pfeil und Enter', async () => {
		const { props, getByRole, queryByRole } = zeige('2026-08-01');
		const knopf = getByRole('button', { name: 'Kalender öffnen' });
		expect(queryByRole('dialog')).toBeNull();
		await fireEvent.click(knopf);
		const dialog = getByRole('dialog', { name: 'Datum wählen' });
		expect(getByRole('grid', { name: 'August 2026' })).toBeTruthy();
		await vi.waitFor(() =>
			expect(document.activeElement?.getAttribute('aria-label')).toBe('Samstag, 1. August 2026')
		);
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: 'ArrowRight' });
		await vi.waitFor(() =>
			expect(document.activeElement?.getAttribute('aria-label')).toBe('Sonntag, 2. August 2026')
		);
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: 'Enter' });
		expect(props.value).toBe('2026-08-02');
		expect(dialog.isConnected).toBe(false);
		expect(document.activeElement).toBe(knopf);
	});

	it('schreibt einen Wert von außen ordentlich, nach dem Laden wie beim Zurücksetzen', async () => {
		const { props, feld } = zeige('');
		props.value = '2027-06-24';
		await vi.waitFor(() => expect(feld.value).toBe('24.06.2027'));
		props.value = '';
		await vi.waitFor(() => expect(feld.value).toBe(''));
	});

	it('öffnet mit Alt+↓, springt mit Pos1 und Ende, die Leertaste wählt ohne zu schließen', async () => {
		const { props, feld, getByRole } = zeige('2026-08-12');
		await fireEvent.keyDown(feld, { key: 'ArrowDown', altKey: true });
		const dialog = getByRole('dialog', { name: 'Datum wählen' });
		/** @param {string} name */
		const fokusAuf = (name) =>
			vi.waitFor(() => expect(document.activeElement?.getAttribute('aria-label')).toBe(name));
		await fokusAuf('Mittwoch, 12. August 2026');
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: 'Home' });
		await fokusAuf('Samstag, 1. August 2026');
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: 'End' });
		await fokusAuf('Montag, 31. August 2026');
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: ' ' });
		expect(dialog.isConnected).toBe(true);
		expect(props.value).toBe('2026-08-12');
		await fireEvent.click(getByRole('button', { name: 'OK' }));
		expect(props.value).toBe('2026-08-31');
	});

	it('wählt in der Monatsliste mit Pfeil und Klick', async () => {
		const { getByRole, container } = zeige('2026-08-01');
		await fireEvent.click(getByRole('button', { name: 'Kalender öffnen' }));
		await fireEvent.click(
			/** @type {Element} */ (container.ownerDocument.querySelector('[data-menue="monate"]'))
		);
		expect(getByRole('listbox', { name: 'Monat' })).toBeTruthy();
		await vi.waitFor(() => expect(document.activeElement?.textContent?.trim()).toBe('August'));
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: 'ArrowDown' });
		expect(document.activeElement?.textContent?.trim()).toBe('September');
		await fireEvent.click(/** @type {Element} */ (document.activeElement));
		expect(getByRole('grid', { name: 'September 2026' })).toBeTruthy();
	});

	// Googles Token: außerhalb des Monats nur „unselected outside-month“, on-surface bei 38 %; nicht
	// wählbar, sonst verfehlte der Tag den Kontrast, den WCAG für Bedienelemente verlangt.
	it('lässt Tage außerhalb des Monats nicht wählen, auch nicht den gewählten', async () => {
		const { props, getByRole } = zeige('2026-07-31');
		await fireEvent.click(getByRole('button', { name: 'Kalender öffnen' }));
		await vi.waitFor(() => expect(document.activeElement?.tagName).toBe('BUTTON'));
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: 'PageDown' });
		expect(getByRole('grid', { name: 'August 2026' })).toBeTruthy();
		const juli = /** @type {HTMLButtonElement} */ (
			getByRole('button', { name: 'Freitag, 31. Juli 2026' })
		);
		expect(juli.disabled).toBe(true);
		expect(juli.closest('[role="gridcell"]')?.getAttribute('aria-selected')).toBe('false');
		expect(juli.className).toContain('text-on-surface/38');
		expect(props.value).toBe('2026-07-31');
	});

	it('blättert mit Bild ab einen Monat und mit Umschalt ein Jahr weiter', async () => {
		const { getByRole } = zeige('2026-08-01');
		await fireEvent.click(getByRole('button', { name: 'Kalender öffnen' }));
		await vi.waitFor(() => expect(document.activeElement?.tagName).toBe('BUTTON'));
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), { key: 'PageDown' });
		expect(getByRole('grid', { name: 'September 2026' })).toBeTruthy();
		await vi.waitFor(() =>
			expect(document.activeElement?.getAttribute('aria-label')).toBe('Dienstag, 1. September 2026')
		);
		await fireEvent.keyDown(/** @type {Element} */ (document.activeElement), {
			key: 'PageDown',
			shiftKey: true
		});
		expect(getByRole('grid', { name: 'September 2027' })).toBeTruthy();
	});

	it('schließt mit Escape und mit Abbrechen, ohne den Tag zu übernehmen', async () => {
		const { props, getByRole, queryByRole } = zeige('2026-08-01');
		await fireEvent.click(getByRole('button', { name: 'Kalender öffnen' }));
		await fireEvent.click(getByRole('button', { name: 'Montag, 3. August 2026' }));
		await fireEvent.keyDown(getByRole('dialog'), { key: 'Escape' });
		expect(queryByRole('dialog')).toBeNull();
		expect(props.value).toBe('2026-08-01');
		await fireEvent.click(getByRole('button', { name: 'Kalender öffnen' }));
		await fireEvent.click(getByRole('button', { name: 'Montag, 3. August 2026' }));
		await fireEvent.click(getByRole('button', { name: 'Abbrechen' }));
		expect(props.value).toBe('2026-08-01');
		await fireEvent.click(getByRole('button', { name: 'Kalender öffnen' }));
		await fireEvent.click(getByRole('button', { name: 'Montag, 3. August 2026' }));
		await fireEvent.click(getByRole('button', { name: 'OK' }));
		expect(props.value).toBe('2026-08-03');
	});

	it('markiert heute und den gewählten Tag für den Screenreader', async () => {
		const { getByRole } = zeige('2026-08-01');
		await fireEvent.click(getByRole('button', { name: 'Kalender öffnen' }));
		const heute = getByRole('button', { name: 'Montag, 10. August 2026' });
		expect(heute.getAttribute('aria-current')).toBe('date');
		expect(heute.className, 'Googles Token: heute mit Rand in primary').toContain('border-primary');
		const gewaehlt = getByRole('button', { name: 'Samstag, 1. August 2026' });
		expect(gewaehlt.closest('[role="gridcell"]')?.getAttribute('aria-selected')).toBe('true');
		expect(getByRole('columnheader', { name: 'Montag' })).toBeTruthy();
	});
});
