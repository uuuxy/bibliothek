import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import StudentBescheideCard from './StudentBescheideCard.svelte';

// Die Akte zeigt, wonach Eltern in der Bibliothek fragen: Referenznummer, Frist, Betrag
// und ob der Fall noch bei der Schule liegt. Bis zum 12.09.2026 stand davon nichts in der
// Akte — der Bescheid lebte nur in der Arbeitsliste des Mahnwesens.
const bescheid = {
	id: 'b-1',
	referenznummer: '4801-2026-1234-0001',
	brief_datum: '2026-09-01T00:00:00Z',
	frist_bis: '2026-09-29T00:00:00Z',
	gesamtbetrag: 24.5,
	anzahl_positionen: 2,
	status: 'offen',
	frist_abgelaufen: false
};

describe('StudentBescheideCard', () => {
	it('nennt Referenznummer, Positionen, Betrag und Frist', () => {
		const screen = render(StudentBescheideCard, { bescheide: [bescheid] });

		expect(screen.getByText('4801-2026-1234-0001')).toBeTruthy();
		expect(screen.getByText(/2 Positionen/)).toBeTruthy();
		expect(screen.getByText(/24,50 €/)).toBeTruthy();
		// Datumsform wie in der Arbeitsliste des Mahnwesens (toLocaleDateString('de-DE')):
		// ohne führende Null. Zwei Formen für dasselbe Datum wären zwei Wahrheiten.
		expect(screen.getByText(/Frist: 29\.9\.2026/)).toBeTruthy();
		expect(screen.getByText('offen')).toBeTruthy();
	});

	// Der Zustand kommt aus bescheidStatus — hier geprüft an dem Fall, der eine Handlung
	// verlangt: Er muss in der Akte genauso stehen wie in der Arbeitsliste.
	it('zeigt die Rückgabe nach der Übergabe', () => {
		const screen = render(StudentBescheideCard, {
			bescheide: [{ ...bescheid, status: 'uebergeben', rueckgabe_nach_uebergabe: true }]
		});
		expect(screen.getByText('Rückgabe nach Übergabe')).toBeTruthy();
	});

	it('öffnet den Nachdruck unter derselben Nummer', async () => {
		const oeffne = vi.fn();
		vi.stubGlobal('open', oeffne);
		const screen = render(StudentBescheideCard, { bescheide: [bescheid] });

		screen.getByRole('button', { name: /Nachdruck/ }).click();

		expect(oeffne).toHaveBeenCalledWith('/api/bescheide/b-1/pdf', '_blank');
		vi.unstubAllGlobals();
	});
});
