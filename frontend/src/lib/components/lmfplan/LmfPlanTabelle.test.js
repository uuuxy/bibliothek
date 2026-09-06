import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import LmfPlanTabelle from './LmfPlanTabelle.svelte';

// Die Tabelle ist die Form, die das Kollegium aus der Excel-Liste kennt: je Art ein
// Block, darin Wochentag, Datum, Stunde, Klassen, Besonderheiten. Lesend — bearbeitet
// wird im Planer (LmfPlanReihenfolge), seit 05.09.2026 abends.
/** @type {import('../../lmfplanDienst.js').LmfTermin[]} */
const termine = [
	{ id: 'a', datum: '2027-06-28', stunde: 3, art: 'rueckgabe', klassen: ['09H1'], vermerk: '' },
	{
		id: 'b',
		datum: '2027-07-01',
		stunde: 6,
		art: 'rueckgabe',
		klassen: ['10R2', '10R3'],
		vermerk: 'erst zur 2. Hälfte'
	},
	{ id: 'c', datum: '2027-08-10', stunde: 2, art: 'ausgabe', klassen: ['07G1'], vermerk: 'neu' }
];

describe('LmfPlanTabelle', () => {
	it('gruppiert nach Art und zeigt Wochentag, Datum, Stunde, Klassen und Vermerk', () => {
		const { getByRole, getAllByRole, getByText } = render(LmfPlanTabelle, { termine });
		expect(getByRole('region', { name: 'Büchertausch vor den Sommerferien' })).toBeTruthy();
		expect(getByRole('region', { name: 'Bücherausgabe nach den Sommerferien' })).toBeTruthy();
		expect(getAllByRole('table')).toHaveLength(2);
		expect(getByText('Montag')).toBeTruthy(); // 28.06.2027
		expect(getByText('28.06.27')).toBeTruthy();
		expect(getByText('3. Std.')).toBeTruthy();
		expect(getByText('10R2 / 10R3')).toBeTruthy();
		expect(getByText('erst zur 2. Hälfte')).toBeTruthy();
	});

	it('lesend: keine Aktionsspalte', () => {
		const { queryByRole } = render(LmfPlanTabelle, { termine });
		expect(queryByRole('button', { name: /Bearbeiten/ })).toBeNull();
	});

	it('lässt einen leeren Block weg', () => {
		const { queryByRole } = render(LmfPlanTabelle, {
			termine: termine.filter((t) => t.art === 'ausgabe')
		});
		expect(queryByRole('region', { name: 'Büchertausch vor den Sommerferien' })).toBeNull();
		expect(queryByRole('region', { name: 'Bücherausgabe nach den Sommerferien' })).toBeTruthy();
	});

	// „Nur Rückgabe" kommt vom Server (Abschlussklasse, neu gebildete Klasse) und steht als
	// Chip vor dem Vermerk; der Satz unter der Überschrift nennt die Eingangsjahrgänge.
	it('markiert Zeilen, die nur zurückgeben, und erklärt die Blöcke', () => {
		const { getByText, container } = render(LmfPlanTabelle, {
			termine: [
				{
					id: 'a',
					datum: '2027-06-28',
					stunde: 1,
					art: 'rueckgabe',
					klassen: ['09H1'],
					vermerk: '',
					nur_rueckgabe: true
				},
				{
					id: 'b',
					datum: '2027-08-10',
					stunde: 2,
					art: 'ausgabe',
					klassen: ['07G1'],
					vermerk: 'neu',
					nur_rueckgabe: false
				}
			],
			eingangsjahrgaenge: [5, 7]
		});
		expect(getByText('nur Rückgabe')).toBeTruthy();
		const zeilen = [...container.querySelectorAll('tbody tr')].map((tr) => tr.textContent ?? '');
		expect(zeilen.find((z) => z.includes('09H1'))).toContain('nur Rückgabe');
		expect(zeilen.find((z) => z.includes('07G1'))).not.toContain('nur Rückgabe');
		expect(getByText(/Jahrgang 5 und 7/)).toBeTruthy();
	});
});
