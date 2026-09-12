import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('./apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ({}) })),
	apiClient: { post: vi.fn(async () => ({ ok: true, json: async () => ({}) })) }
}));

import StudentProfileAusleihen from './StudentProfileAusleihen.svelte';

// Der Reiter „Ausleihen & Historie" ist am 12.09.2026 aus StudentProfile herausgelöst
// worden (Größen-Ratsche). Reine Verschiebung — aber eine Verschiebung, bei der ein
// nicht durchgereichtes Prop still eine ganze Karte leer ließe. Deshalb der Rauchtest:
// alle vier Karten, mit ihren Daten.
const daten = {
	buecher: [{ id: 'b1', titel: 'Der Hobbit', barcode_id: 'B-1', rueckgabe_frist: '2026-10-01' }],
	vormerkungen: [{ id: 'v1', titel_name: 'Momo', status: 'wartend', erstellt_am: '2026-09-01' }],
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

describe('StudentProfileAusleihen', () => {
	it('zeigt Ausleihen, Vormerkungen, Gebühren und Bescheide', () => {
		const screen = render(StudentProfileAusleihen, daten);

		expect(screen.getByText(/Der Hobbit/)).toBeTruthy();
		expect(screen.getByText(/Momo/)).toBeTruthy();
		expect(screen.getByText(/Wasserschaden/)).toBeTruthy();
		expect(screen.getByText('4801-2026-1234-0001')).toBeTruthy();
	});

	// Ohne Bescheid bleibt die Überschrift weg — eine leere Karte „Schadensersatz-Bescheide"
	// in jeder Akte läse sich wie eine offene Forderung.
	it('schweigt über Bescheide, wenn es keine gibt', () => {
		const screen = render(StudentProfileAusleihen, { ...daten, bescheide: [] });

		expect(screen.queryByText(/Schadensersatz-Bescheide/)).toBeNull();
	});
});
