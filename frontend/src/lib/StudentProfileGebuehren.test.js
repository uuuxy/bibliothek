import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('./apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ({}) })),
	apiClient: { post: vi.fn(async () => ({ ok: true, json: async () => ({}) })) }
}));

import StudentProfileGebuehren from './StudentProfileGebuehren.svelte';

// Rauchtest des Reiters „Gebühren & Schäden": beide Karten mit ihren Daten.
const daten = {
	gebuehren: [{ id: 'g1', beschreibung: 'Wasserschaden', betrag: 7.5, ist_bezahlt: false }],
	bescheide: [
		{
			id: 'be1',
			referenznummer: '4801-2026-1234-0001',
			brief_datum: '2026-09-01',
			frist_bis: '2026-09-29',
			gesamtbetrag: 24.5,
			anzahl_positionen: 1,
			status: 'offen'
		}
	],
	canEdit: true,
	onChanged: () => {}
};

describe('StudentProfileGebuehren', () => {
	it('zeigt Gebühren und Bescheide', () => {
		const screen = render(StudentProfileGebuehren, daten);

		expect(screen.getByText(/Wasserschaden/)).toBeTruthy();
		expect(screen.getByText('4801-2026-1234-0001')).toBeTruthy();
	});

	// Eine leere Karte sähe aus wie „nichts offen": Der gescheiterte Abruf steht über den Karten.
	it('nennt die Listen, deren Abruf gescheitert ist, statt „Keine …" zu sagen', () => {
		const screen = render(StudentProfileGebuehren, {
			...daten,
			gebuehren: [],
			bescheide: [],
			fehlendeListen: ['Gebühren', 'Bescheide']
		});
		expect(screen.getByRole('alert').textContent).toMatch(/Nicht geladen: Gebühren, Bescheide/);
		expect(screen.queryByText(/Keine Gebühren/)).toBeNull();
	});

	it('sagt, wenn es nichts gibt', () => {
		const screen = render(StudentProfileGebuehren, { ...daten, gebuehren: [], bescheide: [] });
		expect(screen.getByText('Keine Gebühren, Schäden oder Bescheide.')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	// Ohne Bescheid bleibt dessen Überschrift weg; sie läse sich wie eine offene Forderung.
	it('schweigt über Bescheide, wenn es keine gibt', () => {
		const screen = render(StudentProfileGebuehren, { ...daten, bescheide: [] });
		expect(screen.queryByText(/Schadensersatz-Bescheide/)).toBeNull();
	});
});
