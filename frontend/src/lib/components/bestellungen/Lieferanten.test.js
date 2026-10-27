import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import Lieferanten from './Lieferanten.svelte';

// Lehnt der Server einen neuen Lieferanten ab, bleibt die Eingabe zum Berichtigen stehen; eine
// abgelehnte Änderung lässt die Zeile mit den Änderungen offen. Die Meldung zeigt apiFetch,
// der Store gibt den Fehler weiter (bestellStore.test.js).

const LIEFERANT = {
	id: 's1',
	name: 'Buchhandlung Alt',
	email: 'alt@example.org',
	customerNumber: 'K-1',
	kundennummer_schultraeger: '',
	ist_hauptlieferant: false
};

/** @param {Partial<Record<string, any>>} [props] */
function baue(props = {}) {
	const ansicht = render(Lieferanten, {
		suppliers: [],
		onAddSupplier: vi.fn(),
		onEditSupplier: vi.fn(),
		onRemoveSupplier: vi.fn(),
		...props
	});
	const feld = (/** @type {string} */ etikett) =>
		/** @type {HTMLInputElement} */ (ansicht.getByLabelText(etikett));
	return { ...ansicht, feld };
}

/** @param {ReturnType<typeof baue>} ansicht */
async function fuelleNeuenLieferanten(ansicht) {
	await fireEvent.input(ansicht.feld('Name'), { target: { value: 'Buchhandlung Neu' } });
	await fireEvent.input(ansicht.feld('E-Mail'), { target: { value: 'neu@example.org' } });
	await fireEvent.input(ansicht.feld('Kundennummer'), { target: { value: 'K-9' } });
}

describe('Lieferanten', () => {
	it('behält die Eingabe, wenn das Anlegen scheitert, und leert sie nach dem Anlegen', async () => {
		const onAddSupplier = vi
			.fn()
			.mockRejectedValueOnce(new Error('abgelehnt'))
			.mockResolvedValueOnce(undefined);
		const ansicht = baue({ onAddSupplier });
		await fuelleNeuenLieferanten(ansicht);
		const speichern = ansicht.getByRole('button', { name: 'Lieferanten speichern' });

		await fireEvent.click(speichern);
		await tick();
		expect(onAddSupplier).toHaveBeenCalledTimes(1);
		expect(ansicht.feld('Name').value, 'abgelehnt: die Eingabe bleibt').toBe('Buchhandlung Neu');

		await fireEvent.click(speichern);
		await tick();
		expect(onAddSupplier).toHaveBeenCalledTimes(2);
		expect(ansicht.feld('Name').value, 'angelegt: das Formular ist leer').toBe('');
	});

	it('schickt einen zweiten Klick nicht, solange das Anlegen läuft', async () => {
		/** @type {(wert?: unknown) => void} */
		let fertig = () => {};
		const onAddSupplier = vi.fn(() => new Promise((aufloesen) => (fertig = aufloesen)));
		const ansicht = baue({ onAddSupplier });
		await fuelleNeuenLieferanten(ansicht);

		await fireEvent.click(ansicht.getByRole('button', { name: 'Lieferanten speichern' }));
		const laeuft = ansicht.getByRole('button', { name: 'Wird gespeichert …' });
		expect(laeuft.hasAttribute('disabled')).toBe(true);
		await fireEvent.submit(/** @type {HTMLFormElement} */ (laeuft.closest('form')));
		expect(onAddSupplier).toHaveBeenCalledTimes(1);

		fertig();
		await tick();
		await tick();
		expect(ansicht.getByRole('button', { name: 'Lieferanten speichern' })).toBeTruthy();
	});

	it('lässt die Zeile offen, wenn die Änderung abgelehnt wird', async () => {
		const onEditSupplier = vi.fn().mockRejectedValueOnce(new Error('abgelehnt'));
		const ansicht = baue({ suppliers: [LIEFERANT], onEditSupplier });

		await fireEvent.click(ansicht.getByRole('button', { name: /bearbeiten/i }));
		const name = /** @type {HTMLInputElement} */ (ansicht.getByDisplayValue('Buchhandlung Alt'));
		await fireEvent.input(name, { target: { value: 'Buchhandlung Umbenannt' } });
		await fireEvent.click(
			ansicht.getByRole('button', { name: 'Änderungen für Lieferant Buchhandlung Alt speichern' })
		);
		await tick();

		expect(onEditSupplier).toHaveBeenCalledWith('s1', { name: 'Buchhandlung Umbenannt' });
		expect(
			ansicht.getByDisplayValue('Buchhandlung Umbenannt'),
			'die Zeile ist noch in Bearbeitung'
		).toBeTruthy();
	});
});

// Ein zweiter Klick auf „Löschen" vor der Antwort schickte ein zweites DELETE, und nach dem
// Löschen stand „Lieferant nicht gefunden", in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('Lieferanten: Löschen', () => {
	it('schickt einen zweiten Klick nicht, solange das Löschen läuft', async () => {
		/** @type {(wert?: unknown) => void} */
		let fertig = () => {};
		const onRemoveSupplier = vi.fn(() => new Promise((aufloesen) => (fertig = aufloesen)));
		const ansicht = baue({ suppliers: [LIEFERANT], onRemoveSupplier });
		const knopf = ansicht.getByRole('button', { name: 'Lieferant Buchhandlung Alt löschen' });

		await fireEvent.click(knopf);
		expect(knopf.hasAttribute('disabled'), 'gesperrt, solange das Löschen läuft').toBe(true);
		await fireEvent.click(knopf);
		expect(onRemoveSupplier, 'ein zweites DELETE vor der Antwort').toHaveBeenCalledTimes(1);

		fertig();
		await tick();
		await tick();
		expect(knopf.hasAttribute('disabled')).toBe(false);
	});
});
