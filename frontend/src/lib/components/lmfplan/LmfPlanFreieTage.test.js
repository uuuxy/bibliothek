import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LmfPlanFreieTage from './LmfPlanFreieTage.svelte';

// Freie Tage des Plans (Brückentag, pädagogischer Tag) trägt die Bibliothek ein; was der
// Server im Plan-Zeitraum übersprungen hat — Feiertage eingeschlossen —, steht mit
// Grund darunter. Geprüft wird das Bauteil: eintragen, Dublette ersetzt, entfernen,
// Ausfälle lesbar.
describe('LmfPlanFreieTage', () => {
	it('nimmt einen freien Tag auf, ersetzt ein Duplikat und entfernt ihn wieder', async () => {
		const { getByLabelText, getByRole, getByTestId, queryByTestId } = render(LmfPlanFreieTage, {
			tage: [],
			ausfaelle: []
		});
		await fireEvent.input(getByLabelText('Freier Tag'), { target: { value: '2026-06-05' } });
		await fireEvent.input(getByLabelText('Grund'), { target: { value: 'Brückentag' } });
		await fireEvent.click(getByRole('button', { name: 'Tag freihalten' }));
		expect(getByTestId('lmf-freie-tage').textContent).toContain('05.06.26 Brückentag');

		// Derselbe Tag noch einmal, anderer Grund: eine Zeile, der neue Grund.
		await fireEvent.input(getByLabelText('Freier Tag'), { target: { value: '2026-06-05' } });
		await fireEvent.input(getByLabelText('Grund'), { target: { value: 'Studientag' } });
		await fireEvent.click(getByRole('button', { name: 'Tag freihalten' }));
		expect(getByTestId('lmf-freie-tage').querySelectorAll('span').length).toBe(1);
		expect(getByTestId('lmf-freie-tage').textContent).toContain('Studientag');

		await fireEvent.click(getByLabelText('05.06.26 Studientag aus dem Plan nehmen'));
		expect(queryByTestId('lmf-freie-tage')).toBeNull();
	});

	it('nennt die übersprungenen Tage mit Wochentag und Grund', () => {
		const { getByTestId } = render(LmfPlanFreieTage, {
			tage: [],
			ausfaelle: [
				{ datum: '2026-06-04', grund: 'Fronleichnam' },
				{ datum: '2026-06-05', grund: 'Brückentag' }
			]
		});
		const text = getByTestId('lmf-ausfaelle').textContent?.replace(/\s+/g, ' ');
		expect(text).toContain('Donnerstag 04.06.26 (Fronleichnam)');
		expect(text).toContain('Freitag 05.06.26 (Brückentag)');
	});
});
