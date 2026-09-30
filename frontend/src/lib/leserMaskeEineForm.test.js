import { describe, it, expect } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte';
import LeserEditFelder from './components/students/LeserEditFelder.svelte';
import { LESER_ARTEN } from './leserArt.js';

// EINE Maske für jeden Leser.
//
// Der erste Bau blendete die Felder je nach Art ein und aus: Ein Kollege bekam vier
// Felder weniger und einen anderen Abschnittstitel. Absprache vom 16.09.2026, beim Blick
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

	/** Die ids der verschlossenen Felder, sortiert. @param {string} art @returns {string[]} */
	function verschlossen(art) {
		const screen = render(LeserEditFelder, { formData: formular(art) });
		return [...screen.container.querySelectorAll('input, [role="combobox"]')]
			.filter((e) => /** @type {HTMLInputElement} */ (e).disabled)
			.map((e) => e.id)
			.sort();
	}

	/** @param {string} art @returns {string[]} */
	function felderVon(art) {
		const screen = render(LeserEditFelder, { formData: formular(art) });
		// Seit dem 30.09.2026 ist die Klasse ein Auswahlfeld (Knopf mit role="combobox").
		return [...screen.container.querySelectorAll('input, [role="combobox"]')]
			.map((e) => e.id || e.getAttribute('value') || '')
			.filter(Boolean);
	}

	it('zeigt einem Kollegen genau dieselben Felder wie einem Schüler', () => {
		for (const art of LESER_ARTEN) {
			expect(felderVon(art), `Felder bei art=${art}`).toEqual(felderVon('schueler'));
		}
	});

	it('nennt die Abschnitte für jeden gleich', () => {
		for (const art of LESER_ARTEN) {
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
		expect(verschlossen('lehrkraft')).toEqual(['abgangsjahr', 'eltern_email', 'klasse', 'lusd_id']);
	});

	// Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal" (30.09.2026): Ihnen ist
	// zusätzlich die Schul-Adresse verschlossen. Sekretariat und U-plus behalten sie wie eine
	// Lehrkraft.
	it('verschliesst Praktikum und Fachbereich auch die Schul-Adresse', () => {
		for (const art of ['praktikum', 'fachbereich']) {
			expect(verschlossen(art), `art=${art}`).toEqual([
				'abgangsjahr',
				'eltern_email',
				'klasse',
				'lusd_id',
				'schul_email'
			]);
		}
		for (const art of ['sekretariat', 'uplus']) {
			expect(verschlossen(art), `art=${art}`).toEqual(verschlossen('lehrkraft'));
		}
	});

	// Die Gegenrichtung: Dem Schüler ist genau EIN Feld verschlossen, die Schul-Adresse.
	// Sie ist kein Kontaktfeld, sondern das Konto — „Ein Schüler bekommt kein Konto und
	// keine E-Mail-Adresse" weist auch der Server ab (pruefeSchulEmail). Ein offenes Feld,
	// das beim Speichern mit 400 zurückkommt, wäre die schlechtere Auskunft.
	it('verschliesst dem Schüler nur die Schul-Adresse', () => {
		expect(verschlossen('schueler')).toEqual(['schul_email']);
	});

	// Die Art selbst: Die Grenze zum Schüler ist in BEIDE Richtungen zu, und zwar als
	// abgeschalteter Eintrag statt als fehlender. Wer die Liste öffnet, soll sehen, dass es
	// die anderen Arten gibt und dass sie hier nicht offen stehen. Seit dem 30.09.2026 eine
	// Auswahlliste mit sieben Einträgen statt drei Knöpfen.
	it('sperrt die Schüler-Wahl beim Kollegen und die Kollegen-Wahl beim Schüler', async () => {
		/** @param {string} art */
		async function eintraege(art) {
			// Die Liste hängt am Dokument, nicht am Container: vorher aufräumen, sonst fände die
			// Abfrage die Liste der ersten Maske ein zweites Mal.
			cleanup();
			const screen = render(LeserEditFelder, { formData: formular(art) });
			await fireEvent.click(screen.getByRole('combobox', { name: 'Art des Lesers' }));
			return [...screen.getByRole('listbox').querySelectorAll('[role="option"]')].map((o) => ({
				text: o.textContent?.trim(),
				zu: o.getAttribute('aria-disabled') === 'true'
			}));
		}

		const beimKollegen = await eintraege('lehrkraft');
		expect(beimKollegen, 'alle sieben Arten stehen da').toHaveLength(7);
		expect(beimKollegen.filter((e) => e.zu).map((e) => e.text)).toEqual(['Schüler']);

		const beimSchueler = await eintraege('schueler');
		expect(beimSchueler).toHaveLength(7);
		expect(beimSchueler.filter((e) => !e.zu).map((e) => e.text)).toEqual(['Schüler']);
	});
});
