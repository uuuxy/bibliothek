import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import EtikettenListe from './EtikettenListe.svelte';

// Titel und Autor kürzen sich in der Liste; die Sprechblase am Titel nennt beide ganz
// (M3, Text truncation: „Truncated text can be replaced with an ellipsis if the text is
// available through a tooltip or link").
describe('Fehlende Etiketten: gekürzter Titel', () => {
	it('trägt den ganzen Titel und den Autor in der Sprechblase', () => {
		const titel = 'Wirtschaftsgeographie und Sozialwissenschaften für die gymnasiale Oberstufe';
		const k = render(EtikettenListe, {
			zeilen: [
				{
					barcode_id: '5896800039556',
					titel,
					autor: 'Schmidt-Rottluff, Karl-Heinz',
					zugang_am: '2026-10-01',
					etikett_gedruckt: false
				},
				{
					barcode_id: '5896800039563',
					titel: 'Ohne Autor',
					autor: '',
					zugang_am: '',
					etikett_gedruckt: false
				}
			],
			gewaehlt: [],
			status: 'offen',
			onumschalten: () => {},
			onalleUmschalten: () => {}
		});

		expect(k.getByText(titel).getAttribute('data-tip')).toBe(
			`${titel} · Schmidt-Rottluff, Karl-Heinz`
		);
		expect(k.getByText('Ohne Autor').getAttribute('data-tip')).toBe('Ohne Autor');
	});
});
