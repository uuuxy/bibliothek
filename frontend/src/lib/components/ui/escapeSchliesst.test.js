import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { escapeSchliesst } from './escapeSchliesst.js';
import {
	ohneKommentare,
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

// Zweite Form derselben Regel (Rasterdurchgang 06.09.2026): Ein Bauteil, das Escape
// SELBST behandelt (`<svelte:window onkeydown>`), muss den Tastendruck auch als
// verarbeitet melden — sonst schließt es sich und der globale Kurzbefehl in Router.svelte
// springt zusätzlich an die Theke. Die Ratsche oben erkennt Overlays am `fixed inset-0`
// und sah diese Form nicht; in ihrem blinden Winkel stand ein lebender Defekt
// (designer/VorlagenGalerie.svelte). Regel 6 der Rot-Beweis-Battery: Selbstprobe über
// ALLE Formen, nicht über die eine, die man gerade im Kopf hat.
const ESCAPE_OHNE_ANSPRUCH = [
	// Der globale Kurzbefehl selbst — er IST der Empfänger, er meldet nichts weiter.
	'src/lib/Router.svelte',
	// Die Omnibox lebt ausschließlich im Kiosk (Router.svelte rendert sie unter
	// activeTab === 'kiosk'), und genau dort steigt der globale Kurzbefehl als Erstes aus
	// („Escape an der Theke heißt Eingabe verwerfen, nicht Ansicht verlassen"). Es gibt
	// also keinen zweiten Empfänger, dem sie etwas melden müsste.
	'src/lib/Omnibox.svelte'
];

describe('Escape-Hygiene: wer die Taste nimmt, meldet sie als verarbeitet', () => {
	it('jedes eigene Escape meldet preventDefault (oder steht begründet in der Liste)', () => {
		const ohne = sammleQuelldateien(srcRoot)
			.filter((f) => f.endsWith('.svelte'))
			.filter((f) => {
				const q = ohneKommentare(readFileSync(f, 'utf8'));
				if (q.includes('escapeSchliesst') || q.includes('escapeBelegen')) return false;
				// Nur das FENSTER zählt hier. Ein Lauscher am Element (eine Tabellenzelle, ein
				// Dialoginhalt) kann die Weitergabe stoppen, bevor sie das Fenster erreicht —
				// `stopPropagation` reicht dort. Am Fenster selbst nützt es nichts: Der Router
				// hört auf demselben Ziel, und `stopPropagation` hält Geschwister nicht auf.
				// Ohne diese Verengung meldete die Ratsche LmfPlanKlassenZelle.svelte, die
				// Escape korrekt an der Zelle abfängt (Selbstprobe 06.09.2026).
				const fenster = /<svelte:window\b[\s\S]*?\/>/g;
				if (
					[...q.matchAll(fenster)].some(
						(m) => m[0].includes("'Escape'") && !m[0].includes('preventDefault')
					)
				)
					return true;
				// Zweite Schreibweise desselben Anspruchs: ein Lauscher von Hand am Fenster.
				return (
					q.includes("window.addEventListener('keydown'") &&
					q.includes("'Escape'") &&
					!q.includes('preventDefault')
				);
			})
			.map(relPfad)
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(ohne, ESCAPE_OHNE_ANSPRUCH);

		expect(
			neu,
			'Dieses Bauteil nimmt Escape, meldet es aber nicht als verarbeitet — der globale ' +
				'Kurzbefehl feuert zusätzlich und die Ansicht springt an die Theke. ' +
				'`e.preventDefault()` nach dem Schließen (Vorbild: ui/CoverPeek.svelte).\n' +
				neu.join('\n')
		).toEqual([]);

		expect(
			inzwischenSauber,
			'Die Ausnahmeliste führt etwas, das es nicht mehr gibt oder das inzwischen meldet — austragen.'
		).toEqual([]);
	});
});

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
