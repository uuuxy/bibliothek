import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import AbgaengerTabelle from './AbgaengerTabelle.svelte';

// Drei Zustände, ein Bildschirmbereich. Der E2E-Test kann nur den Zustand sehen, den der
// Kalender gerade hergibt (Saison Mai bis Juli) — hier stehen alle drei nebeneinander,
// damit die Verdrahtung des Fensters nicht acht Monate im Jahr ungeprüft bleibt.
const saison = { offen: true, von: '01.05.', bis: '31.07.' };
const winter = { offen: false, von: '01.05.', bis: '31.07.' };
const zeile = {
	id: 's1',
	vorname: 'Anna',
	nachname: 'Test',
	klasse: '09H1',
	offene_buecher: 2,
	ueberfaellig: 1,
	ist_gesperrt: false
};

describe('AbgaengerTabelle', () => {
	it('außerhalb der Saison: Hinweis mit beiden Daten, nicht „alle entlastet"', () => {
		const { getByText, queryByText, queryByRole } = render(AbgaengerTabelle, {
			zeilen: [],
			leer: true,
			fenster: winter,
			onProfil: () => {}
		});
		expect(getByText('Abschlussklassen erscheinen hier ab Mai')).toBeTruthy();
		expect(getByText(/01\.05\./)).toBeTruthy();
		expect(getByText(/31\.07\./)).toBeTruthy();
		expect(queryByText('Alle Abgänger entlastet!')).toBeNull();
		expect(queryByRole('table')).toBeNull();
	});

	it('in der Saison ohne Posten: alle entlastet', () => {
		const { getByText, queryByText } = render(AbgaengerTabelle, {
			zeilen: [],
			leer: true,
			fenster: saison,
			onProfil: () => {}
		});
		expect(getByText('Alle Abgänger entlastet!')).toBeTruthy();
		expect(queryByText('Abschlussklassen erscheinen hier ab Mai')).toBeNull();
	});

	it('in der Saison mit Posten: Tabelle mit Klasse, Name und Überfälligkeit', () => {
		const { getByRole, getByText } = render(AbgaengerTabelle, {
			zeilen: [zeile],
			leer: false,
			fenster: saison,
			onProfil: () => {}
		});
		expect(getByRole('table')).toBeTruthy();
		expect(getByRole('button', { name: /Profil von Anna Test \(Klasse 09H1\)/ })).toBeTruthy();
		expect(getByText(/1 überfällig/)).toBeTruthy();
	});
});
