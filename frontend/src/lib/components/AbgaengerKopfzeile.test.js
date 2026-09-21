import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import AbgaengerKopfzeile from './AbgaengerKopfzeile.svelte';
import { authStore } from '../stores/authStore.svelte.js';

// Der Knopf „An Klassenleitungen mailen" schickt an POST /api/abgaenger/mail, und diese
// Route verlangt create_orders (api/routes_students.go). Die Seite selbst hängt an
// view_graduates — wer sie sehen darf, darf also nicht zwangsläufig mailen.
//
// Bis zum 12.09.2026 stand der Knopf trotzdem für jeden da (Bestands-Durchgang
// 10.09.2026). Wer ihn ohne das Recht drückte, bekam nach dem Versanddialog — samt
// Klassenauswahl — einen 403. Die Regel im Haus lautet: Sichtbarkeit einer Aktion =
// hatRecht(user, '<Recht der Route>'), damit der Schalter auf der Berechtigungsseite die
// eine Wahrheit bleibt (frontend-hygiene-rechte.test.js).
const PROPS = {
	suche: '',
	klasse: '',
	klassen: ['09H1'],
	gesamt: 3,
	gefiltert: 3,
	laedt: false,
	druckLaeuft: false,
	onDrucken: () => {},
	onMailen: () => {}
};

/** @param {string[]} permissions */
function alsBenutzerMit(permissions) {
	authStore.currentUser = { id: 1, rolle: 'helfer', permissions };
}

describe('AbgaengerKopfzeile', () => {
	it('zeigt den Mail-Knopf nur mit create_orders', () => {
		alsBenutzerMit(['view_graduates', 'create_orders']);
		const mit = render(AbgaengerKopfzeile, { ...PROPS });
		expect(mit.queryByRole('button', { name: /An Klassenleitungen mailen/ })).toBeTruthy();
		mit.unmount();

		alsBenutzerMit(['view_graduates']);
		const ohne = render(AbgaengerKopfzeile, { ...PROPS });
		expect(ohne.queryByRole('button', { name: /An Klassenleitungen mailen/ })).toBeNull();
		// Der Druck bleibt: /api/abgaenger/pdf hängt an view_graduates, also am Recht,
		// mit dem diese Seite überhaupt offen ist. Papier ist hier der Notweg.
		expect(ohne.queryByRole('button', { name: /Kontoauszüge/ })).toBeTruthy();
	});

	// Der Druck folgt der Suche (seit 21.09.2026): gedruckt wird, was die Liste zeigt. Zeigt
	// die Suche niemanden, gibt es nichts zu drucken — der Knopf ist dann gesperrt, statt
	// eine Anfrage zu schicken, die der Server abweist. M3 (Interaction states): „A disabled
	// state communicates when a component or element isn't interactive."
	//
	// Vorher hing die Sperre an `gesamt`: Bei fünf Abgängern und einer Suche ohne Treffer war
	// der Knopf bedienbar und druckte alle fünf.
	it('sperrt den Druck, wenn die Suche niemanden zeigt', () => {
		alsBenutzerMit(['view_graduates']);
		const leer = render(AbgaengerKopfzeile, { ...PROPS, suche: 'zzz', gesamt: 5, gefiltert: 0 });
		expect(leer.getByRole('button', { name: /Kontoauszüge/ }).hasAttribute('disabled')).toBe(true);
		leer.unmount();

		// Die Gegenprobe: mit Treffern ist er bedienbar.
		const treffer = render(AbgaengerKopfzeile, { ...PROPS, suche: 'mü', gesamt: 5, gefiltert: 1 });
		expect(treffer.getByRole('button', { name: /Kontoauszüge/ }).hasAttribute('disabled')).toBe(
			false
		);
	});
});
