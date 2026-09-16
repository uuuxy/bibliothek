import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import LeserEditFelder from './components/students/LeserEditFelder.svelte';

// EINE Maske für jeden Leser.
//
// Der erste Bau blendete die Felder je nach Art ein und aus: Ein Kollege bekam vier
// Felder weniger und einen anderen Abschnittstitel. Peter am 16.09.2026, beim Blick
// darauf: „warum eine andere maske als bei schülern? das ist doch schon wieder viel zu
// kompliziert." Zwingend ist nur, was die Datenbank verbietet oder was fachlich falsch
// wäre — nicht, was mir vorsichtig erschien.
//
// Dieser Test prüft die FORM, nicht die Werte: dieselben Felder, an derselben Stelle, in
// derselben Reihenfolge. Verschlossen (disabled) darf etwas sein, verschwinden nicht.
// Ein Render-Test ist hier richtig und kein Quelltext-Vergleich: Die Frage ist, was am
// Ende im Browser steht, und ein `{#if}` mehr fiele in einer Quelltext-Prüfung nicht auf.
describe('Leser-Maske hat für jeden dieselbe Form', () => {
	/** @param {string} art */
	const formular = (art) => ({
		vorname: 'Ahmet',
		nachname: 'Görgü',
		art,
		geburtsdatum: '',
		lusd_id: '',
		klasse: '',
		barcode_id: '',
		abgaenger_jahr: '',
		strasse: '',
		hausnummer: '',
		plz: '',
		ort: '',
		eltern_email: '',
		email: ''
	});

	/** @param {string} art @returns {string[]} */
	function felderVon(art) {
		const screen = render(LeserEditFelder, { formData: formular(art) });
		return [...screen.container.querySelectorAll('input')]
			.map((e) => e.id || e.getAttribute('value') || '')
			.filter(Boolean);
	}

	it('zeigt einem Kollegen genau dieselben Felder wie einem Schüler', () => {
		expect(felderVon('lehrkraft')).toEqual(felderVon('schueler'));
		expect(felderVon('liv')).toEqual(felderVon('schueler'));
	});

	it('nennt die Abschnitte für jeden gleich', () => {
		for (const art of ['schueler', 'lehrkraft', 'liv']) {
			const text = render(LeserEditFelder, { formData: formular(art) }).container.textContent ?? '';
			expect(text, `Abschnitte bei art=${art}`).toContain('Persönliche Daten');
			expect(text).toContain('Schuldaten');
			expect(text).toContain('Kontaktdaten');
		}
	});

	// Die Ausnahmen, und warum es genau diese sind:
	//   klasse/abgangsjahr — „eine klasse muss ja keinem lehrer/liv zugeordnet werden"
	//   lusd_id            — chk_leser_nur_schueler_werden_abgaenger verbietet sie
	//   eltern_email       — ein Kollege hat keine Eltern. Offen wäre dieses Feld die
	//                        Falle, in die seine SCHUL-Adresse wandert: Bis zum 16.09.2026
	//                        war es das einzige E-Mail-Feld der Maske, mit dem Platzhalter
	//                        „eltern@schule.de", auch in der Akte einer Lehrkraft.
	// Verschlossen heisst hier disabled: sichtbar an derselben Stelle, aber nicht zu füllen.
	it('verschliesst dem Kollegen Klasse, Abgangsjahr, LUSD-ID und die Eltern-Adresse — mehr nicht', () => {
		const screen = render(LeserEditFelder, { formData: formular('lehrkraft') });
		// Ohne die Radios: Die gesperrte Art prüft der Test darunter, und sie haben keine
		// id — sie kämen hier als leerer Name mit und machten die Zusage unlesbar.
		const zu = [...screen.container.querySelectorAll('input:not([type="radio"])')]
			.filter((e) => /** @type {HTMLInputElement} */ (e).disabled)
			.map((e) => e.id)
			.sort();
		expect(zu).toEqual(['abgangsjahr', 'eltern_email', 'klasse', 'lusd_id']);
	});

	// Die Gegenrichtung: Dem Schüler ist genau EIN Feld verschlossen, die Schul-Adresse.
	// Sie ist kein Kontaktfeld, sondern das Konto — „Ein Schüler bekommt kein Konto und
	// keine E-Mail-Adresse" weist auch der Server ab (pruefeSchulEmail). Ein offenes Feld,
	// das beim Speichern mit 400 zurückkommt, wäre die schlechtere Auskunft.
	it('verschliesst dem Schüler nur die Schul-Adresse', () => {
		const screen = render(LeserEditFelder, { formData: formular('schueler') });
		// Ohne die Radios: Die gesperrte Art prüft der Test darunter, und sie haben keine
		// id — sie kämen hier als leerer Name mit und machten die Zusage unlesbar.
		const zu = [...screen.container.querySelectorAll('input:not([type="radio"])')]
			.filter((e) => /** @type {HTMLInputElement} */ (e).disabled)
			.map((e) => e.id)
			.sort();
		expect(zu).toEqual(['schul_email']);
	});

	// Die Art selbst: Die Grenze zum Schüler ist in BEIDE Richtungen zu, und zwar als
	// abgeschalteter Knopf statt als fehlender. Wer die Maske ansieht, soll sehen, dass es
	// die dritte Möglichkeit gibt und dass sie hier nicht offen steht.
	it('sperrt die Schüler-Wahl beim Kollegen und die Kollegen-Wahl beim Schüler', () => {
		const beimKollegen = render(LeserEditFelder, { formData: formular('lehrkraft') });
		const radiosK = [...beimKollegen.container.querySelectorAll('input[type="radio"]')];
		expect(radiosK, 'alle drei Arten stehen da').toHaveLength(3);
		expect(radiosK.filter((e) => /** @type {HTMLInputElement} */ (e).disabled)).toHaveLength(1);

		const beimSchueler = render(LeserEditFelder, { formData: formular('schueler') });
		const radiosS = [...beimSchueler.container.querySelectorAll('input[type="radio"]')];
		expect(radiosS).toHaveLength(3);
		expect(radiosS.filter((e) => /** @type {HTMLInputElement} */ (e).disabled)).toHaveLength(2);
	});
});
