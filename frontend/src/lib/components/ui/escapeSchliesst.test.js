import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { escapeSchliesst } from './escapeSchliesst.js';
import {
	srcRoot,
	sammleQuelldateien,
	relPfad,
	vergleicheMitBestand
} from '../../hygiene-quellen.js';

/** @param {string} key */
function taste(key) {
	window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
}

describe('escapeSchliesst', () => {
	it('schließt bei Escape und lässt andere Tasten in Ruhe', () => {
		const zu = vi.fn();
		const a = escapeSchliesst(document.createElement('div'), zu);

		taste('Enter');
		expect(zu).not.toHaveBeenCalled();
		taste('Escape');
		expect(zu).toHaveBeenCalledTimes(1);

		a.destroy();
		taste('Escape');
		expect(zu, 'nach destroy hört der Dialog nicht mehr zu').toHaveBeenCalledTimes(1);
	});

	// Der Fall, an dem eine naive Fassung scheitert: „Gebühr wirklich stornieren?" liegt
	// über der Gebührenliste. Ein Escape ist EIN Tastendruck — es darf nicht zwei Dialoge
	// schließen und den Anwender vor einem Bildschirm zurücklassen, den er nicht verlassen
	// wollte.
	it('nur der oberste Dialog reagiert', () => {
		const unten = vi.fn();
		const oben = vi.fn();
		const u = escapeSchliesst(document.createElement('div'), unten);
		const o = escapeSchliesst(document.createElement('div'), oben);

		taste('Escape');
		expect(oben).toHaveBeenCalledTimes(1);
		expect(unten).not.toHaveBeenCalled();

		o.destroy();
		taste('Escape');
		expect(unten, 'nach dem Schließen des oberen ist der untere dran').toHaveBeenCalledTimes(1);
		u.destroy();
	});

	it('nimmt einen ausgetauschten Schließweg an', () => {
		const alt = vi.fn();
		const neu = vi.fn();
		const a = escapeSchliesst(document.createElement('div'), alt);
		a.update?.(neu);

		taste('Escape');
		expect(alt).not.toHaveBeenCalled();
		expect(neu).toHaveBeenCalledTimes(1);
		a.destroy();
	});
});

// Ratsche: Wer ein Overlay baut, gibt ihm einen Tastaturweg hinaus. Am 05.09.2026 hatte
// KEINES der elf selbstgebauten Overlays einen — nur `Modal.svelte` und, in eigener
// Handarbeit, `ClassAssignPicker`. Beide benutzen jetzt dieselbe Aktion.
const KEIN_DIALOG = [
	// Ladeschleier über der ganzen Anwendung; er hat keinen Ausgang, er wartet.
	'src/App.svelte',
	// Der Flur-Monitor IST der Bildschirm, kein Overlay über etwas anderem.
	'src/lib/Monitor.svelte',
	// Sperrbildschirm: Escape darf ihn gerade NICHT schließen — das ist sein Zweck.
	'src/lib/components/auth/Sperrbildschirm.svelte',
	// Nur der Abdunkler; der Dialog darin (StrichcodeScanner) bringt die Taste mit.
	'src/inventur/routes/admin/+page.svelte'
];

describe('Overlay-Hygiene', () => {
	it('jedes Overlay hat einen Weg mit der Tastatur hinaus', () => {
		const ohne = sammleQuelldateien(srcRoot)
			.filter((f) => f.endsWith('.svelte'))
			.filter((f) => {
				const q = readFileSync(f, 'utf8');
				return (
					q.includes('fixed inset-0') &&
					!q.includes('escapeSchliesst') &&
					!/from '.*Modal\.svelte'/.test(q)
				);
			})
			.map(relPfad)
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(ohne, KEIN_DIALOG);

		expect(
			neu,
			'Neues Overlay ohne Tastaturweg hinaus. Der Klick auf den Hintergrund ist keiner. ' +
				'`use:escapeSchliesst={schliessen}` oder `Modal.svelte` — und wenn es wirklich kein ' +
				'Dialog ist, gehört es mit Begründung in KEIN_DIALOG.\n' +
				neu.join('\n')
		).toEqual([]);

		expect(
			inzwischenSauber,
			'Die Ausnahmeliste führt etwas, das es nicht mehr gibt oder das inzwischen einen ' +
				'Tastaturweg hat — bitte austragen.'
		).toEqual([]);
	});
});
