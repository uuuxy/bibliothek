import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LmfPlanReihenfolge from './LmfPlanReihenfolge.svelte';

// Zwei Klassen dürfen sich eine Stunde teilen — im Plan der Schule stehen „10R1/10R2"
// und „6F1/6F2" so (Peter, 05.09.2026: „das muss alles super flexibel ablaufen").
// Geprüft wird das Umformen der Reihenfolge selbst: zusammenlegen, wieder trennen,
// schieben, eine Klasse herausnehmen. Die Plätze kommen vom Server; hier steht, WAS
// verteilt wird.
const PLAETZE = [
	{ datum: '2027-06-28', stunde: 3 },
	{ datum: '2027-06-28', stunde: 4 },
	{ datum: '2027-06-28', stunde: 5 },
	{ datum: '2027-06-28', stunde: 6 }
];

/** @param {any} zeilen */
function zeige(zeilen, onklasseraus = vi.fn()) {
	return render(LmfPlanReihenfolge, { zeilen, plaetze: PLAETZE, onklasseraus });
}

/** Die Klassen je Zeile, wie die Tabelle sie zeigt. */
function klassenJeZeile(container) {
	return [...container.querySelectorAll('tbody tr')].map((tr) =>
		[...tr.querySelectorAll('td:nth-child(5) span')].map((s) => s.textContent.trim()).join('/')
	);
}

describe('LmfPlanReihenfolge', () => {
	const start = [
		{ klassen: ['10R1'], vermerk: '' },
		{ klassen: ['10R2'], vermerk: '' },
		{ klassen: ['10R3'], vermerk: '' }
	];

	it('legt zwei Klassen in eine Stunde zusammen — und trennt sie wieder', async () => {
		const { container, getByLabelText } = zeige(start.map((z) => ({ ...z })));
		await fireEvent.click(getByLabelText('Zeile 2 mit voriger zusammenlegen'));
		expect(klassenJeZeile(container)).toEqual(['10R1/10R2', '10R3']);
		// Die zusammengelegte Zeile steht in der Stunde der ersten, 10R3 rückt eine hoch.
		const zeilen = container.querySelectorAll('tbody tr');
		expect(zeilen[0].textContent).toContain('3. Std.');
		expect(zeilen[1].textContent).toContain('4. Std.');

		await fireEvent.click(getByLabelText('Zeile 1 trennen'));
		expect(klassenJeZeile(container)).toEqual(['10R1', '10R2', '10R3']);
	});

	it('schiebt eine Zeile nach unten', async () => {
		const { container, getByLabelText } = zeige(start.map((z) => ({ ...z })));
		await fireEvent.click(getByLabelText('Zeile 1 nach unten'));
		expect(klassenJeZeile(container)).toEqual(['10R2', '10R1', '10R3']);
	});

	it('nimmt eine Klasse aus einer geteilten Stunde und meldet sie zurück', async () => {
		const zurueck = vi.fn();
		const { container, getByTitle } = zeige(
			[{ klassen: ['10R1', '10R2'], vermerk: 'zusammen' }],
			zurueck
		);
		await fireEvent.click(getByTitle('10R2 aus dem Plan nehmen'));
		expect(zurueck).toHaveBeenCalledWith('10R2');
		expect(klassenJeZeile(container)).toEqual(['10R1']);
	});

	it('entfernt die Zeile, wenn ihre letzte Klasse geht und kein Vermerk bleibt', async () => {
		const { container, getByTitle } = zeige([{ klassen: ['10R1'], vermerk: '' }]);
		await fireEvent.click(getByTitle('10R1 aus dem Plan nehmen'));
		expect(container.querySelectorAll('tbody tr')).toHaveLength(0);
	});
});
