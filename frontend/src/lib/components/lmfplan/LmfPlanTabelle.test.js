import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LmfPlanTabelle from './LmfPlanTabelle.svelte';

// Die Tabelle ist die Form, die das Kollegium aus der Excel-Liste kennt: je Art ein
// Block, darin Wochentag, Datum, Stunde, Klassen, Besonderheiten. Verwaltung und Portal
// zeigen DIESELBE Komponente — nur die Verwaltung bekommt die Aktionen.
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
		expect(getByRole('region', { name: 'Bücherrückgabe' })).toBeTruthy();
		expect(getByRole('region', { name: 'Bücherausgabe' })).toBeTruthy();
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

	it('bearbeitbar: Bearbeiten und Löschen rufen mit der Zeile zurück', async () => {
		const onBearbeiten = vi.fn();
		const onLoeschen = vi.fn();
		const { getAllByRole } = render(LmfPlanTabelle, {
			termine,
			bearbeitbar: true,
			onBearbeiten,
			onLoeschen
		});
		await fireEvent.click(getAllByRole('button', { name: /Bearbeiten/ })[1]);
		expect(onBearbeiten).toHaveBeenCalledWith(termine[1]);
		await fireEvent.click(getAllByRole('button', { name: /Löschen/ })[2]);
		expect(onLoeschen).toHaveBeenCalledWith(termine[2]);
	});

	it('lässt einen leeren Block weg', () => {
		const { queryByRole } = render(LmfPlanTabelle, {
			termine: termine.filter((t) => t.art === 'ausgabe')
		});
		expect(queryByRole('region', { name: 'Bücherrückgabe' })).toBeNull();
		expect(queryByRole('region', { name: 'Bücherausgabe' })).toBeTruthy();
	});
});
