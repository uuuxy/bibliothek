import { describe, it, expect } from 'vitest';
import { baueAusleiherDruckHtml } from './ausleiherDruck.js';

// Die Nutzlast steht in JEDEM Feld, das aus Stammdaten stammt. Ein Test, der nur
// den Nachnamen vergiftet, bliebe grün, sobald jemand später eine Spalte ergänzt
// und dort die Maskierung vergisst.
const NUTZLAST = `<img src="https://fremder-host/?daten=1"><script>alert(1)</script>`;

const ausleiherMitNutzlast = [
	{
		schueler_name: NUTZLAST,
		schueler_nachname: NUTZLAST,
		klasse: NUTZLAST,
		exemplar_barcode: NUTZLAST,
		ausgeliehen_am: NUTZLAST,
		rueckgabe_frist: NUTZLAST
	}
];

describe('baueAusleiherDruckHtml', () => {
	it('lässt aus keinem Stammdatenfeld ein Element ins Dokument', () => {
		const html = baueAusleiherDruckHtml(ausleiherMitNutzlast, { title: NUTZLAST }, NUTZLAST);

		// Der Kern: Nach dem Entfernen des bekannten, statischen Gerüsts darf kein
		// einziges "<" mehr aus den Daten stammen.
		expect(html).not.toContain('<img');
		expect(html).not.toContain('<script');
		expect(html).not.toContain('fremder-host/?daten=1"');
	});

	it('schreibt kein Skript ins Dokument — die CSP des Openers würde es blockieren', () => {
		// Der frühere Auto-Druck lag als <script> im geschriebenen Dokument und lief
		// deshalb nie (about:blank erbt script-src 'self'). Gedruckt wird jetzt vom
		// Opener aus; taucht hier je wieder ein Skript auf, ist der Fehler zurück.
		const html = baueAusleiherDruckHtml([], { title: 'Emil' }, 'Alle');
		expect(html.toLowerCase()).not.toContain('<script');
		expect(html.toLowerCase()).not.toContain('onload=');
	});

	it('zeigt die echten Werte weiterhin lesbar an', () => {
		const html = baueAusleiherDruckHtml(
			[
				{
					schueler_name: 'Lena',
					schueler_nachname: 'Groß',
					klasse: '7b',
					exemplar_barcode: 'EX-4711',
					ausgeliehen_am: '2026-01-02',
					rueckgabe_frist: '2026-02-01'
				}
			],
			{ title: 'Die Räuber' },
			'7b'
		);

		expect(html).toContain('Lena');
		expect(html).toContain('Groß');
		expect(html).toContain('7b');
		expect(html).toContain('EX-4711');
		expect(html).toContain('Die Räuber');
		expect(html).toContain('2.1.2026'); // de-DE formatiert ohne führende Nullen
	});

	it('markiert nur überfällige Rückgaben', () => {
		const jetzt = new Date('2026-03-01T12:00:00Z');
		const html = baueAusleiherDruckHtml(
			[
				{ schueler_name: 'A', schueler_nachname: 'A', rueckgabe_frist: '2026-02-01' },
				{ schueler_name: 'B', schueler_nachname: 'B', rueckgabe_frist: '2026-04-01' }
			],
			{ title: 'X' },
			'Alle',
			jetzt
		);
		expect(html.match(/class="overdue"/g) ?? []).toHaveLength(1);
	});

	it('maskiert Anführungszeichen auch im Titel-Element', () => {
		const html = baueAusleiherDruckHtml([], { title: '"><b>weg</b>' }, 'Alle');
		expect(html).not.toContain('<b>weg</b>');
	});
});
