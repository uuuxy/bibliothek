import { describe, it, expect } from 'vitest';
import { vorauswahlAusGruppe } from './klassensatzVorauswahl.js';

// Der Dialog speichert überschreibend (UpdateClassBooks löscht und schreibt neu). Wäre
// ein aus den Ausleihen abgeleiteter Titel vorgewählt, machte der erste Klick auf
// Speichern ihn dauerhaft — ohne dass jemand ihn der Klasse zugeordnet hätte.
describe('Vorauswahl im Dialog „Bücher verwalten"', () => {
	const gruppe = {
		className: '05A',
		books: [
			{ id: 1, quelle: 'hand' },
			{ id: 2, quelle: 'ausleihe' },
			{ id: 3, quelle: 'hand' },
			{ id: 4, quelle: 'ausleihe' }
		]
	};

	it('wählt nur handgepflegte Titel vor', () => {
		expect([...vorauswahlAusGruppe(gruppe)]).toEqual([1, 3]);
	});

	it('behandelt einen Titel ohne Quelle als handgepflegt (Altbestand vor 05.09.2026)', () => {
		expect([...vorauswahlAusGruppe({ books: [{ id: 7 }] })]).toEqual([7]);
	});

	it('kommt mit fehlender Gruppe und leerer Liste zurecht', () => {
		expect(vorauswahlAusGruppe(null).size).toBe(0);
		expect(vorauswahlAusGruppe({ books: [] }).size).toBe(0);
	});
});
