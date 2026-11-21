import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import RechteMatrix from './RechteMatrix.svelte';

// Ohne Administratorrolle sind alle Schalter gesperrt. Gesperrte Schalter bekommen keinen Fokus,
// eine Sprechblase je Zeile sah also nur, wer mit der Maus darüberfuhr; der Grund steht deshalb
// sichtbar über der Matrix (M3 Tooltips: „Don't hide critical information within tooltips“).
const beschreibungen = [
	{
		category: 'Ausleihe',
		icon: 'barcode_scanner',
		items: [{ key: 'checkout', label: 'Ausleihen', desc: '' }]
	}
];

describe('RechteMatrix', () => {
	it('nennt ohne Administratorrolle sichtbar, wer die Rechte ändern kann', () => {
		const s = render(RechteMatrix, {
			schreibgeschuetzt: true,
			beschreibungen,
			rechteJeRolle: {},
			updatingKeys: {},
			onToggle: () => {}
		});
		expect(s.getByText('Nur ein Administrator kann die Rechte ändern')).toBeTruthy();
	});

	it('schweigt davon, wenn ein Administrator die Matrix bedient', () => {
		const s = render(RechteMatrix, {
			schreibgeschuetzt: false,
			beschreibungen,
			rechteJeRolle: {},
			updatingKeys: {},
			onToggle: () => {}
		});
		expect(s.queryByText('Nur ein Administrator kann die Rechte ändern')).toBeNull();
	});
});
