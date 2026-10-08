import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	designFuerDruck,
	ladeAusweisDesign,
	meldeDesignNichtGeladen
} from './ausweisDesignLaden.js';
import { idStore, vergissDesignLadezustand } from './idDesignerStore.svelte.js';
import { apiFetch } from '../apiFetch.js';
import { toastStore } from '../stores/toastStore.svelte.js';
import { ohneKommentare, relPfad, sammleQuelldateien, srcRoot } from '../hygiene-quellen.js';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

/** @param {any} daten */
const antwort = (daten) => /** @type {any} */ ({ ok: true, json: async () => daten });

beforeEach(() => {
	vi.clearAllMocks();
	vergissDesignLadezustand();
	idStore.printMode = 'card';
});

describe('ladeAusweisDesign', () => {
	it('zwei Bauteile derselben Seite teilen sich einen Abruf', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({ printMode: 'etikett' }));

		const ergebnisse = await Promise.all([ladeAusweisDesign(), ladeAusweisDesign()]);

		expect(ergebnisse).toEqual([true, true]);
		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(apiFetch).toHaveBeenCalledWith('/api/ausweis-layout');
		expect(idStore.printMode).toBe('etikett');
	});

	// Die Akte lädt auch an der Theke: Eine Meldung ginge dort bei jedem Leser auf.
	it('bleibt bei einem gescheiterten Abruf still und nennt das Ergebnis', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Netz weg'));

		expect(await ladeAusweisDesign()).toBe(false);
		expect(toastStore.addToast).not.toHaveBeenCalled();
		expect(idStore.printMode).toBe('card');
	});

	it('wertet eine Antwort mit Fehlerstatus wie einen gescheiterten Abruf', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 500 }));

		expect(await ladeAusweisDesign()).toBe(false);
	});

	it('versucht es nach einem gescheiterten Abruf beim nächsten Aufruf erneut', async () => {
		vi.mocked(apiFetch).mockRejectedValueOnce(new Error('Netz weg'));
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort({ printMode: 'etikett' }));

		expect(await ladeAusweisDesign()).toBe(false);
		expect(await ladeAusweisDesign()).toBe(true);
		expect(apiFetch).toHaveBeenCalledTimes(2);
		expect(idStore.printMode).toBe('etikett');
	});

	it('lädt in derselben Sitzung kein zweites Mal', async () => {
		// Ein zweiter Abruf überschriebe Änderungen aus dem Designer, die noch nicht
		// gespeichert sind.
		vi.mocked(apiFetch).mockResolvedValue(antwort({ printMode: 'etikett' }));
		await ladeAusweisDesign();
		idStore.printMode = 'card';

		expect(await ladeAusweisDesign()).toBe(true);
		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(idStore.printMode).toBe('card');
	});
});

describe('designFuerDruck', () => {
	afterEach(() => document.body.replaceChildren());

	/** Eine Karte der Druckfläche mit einem Bild, dessen Laden der Test in der Hand hat. */
	function karteMitBild() {
		const karte = document.createElement('div');
		karte.className = 'print-card-box';
		const bild = document.createElement('img');
		/** @type {() => void} */
		let fertig = () => {};
		bild.decode = vi.fn(() => new Promise((ok) => (fertig = () => ok(undefined))));
		karte.append(bild);
		document.body.append(karte);
		return { bild, bildIstDa: () => fertig() };
	}

	it('gibt den Druck frei, wenn das Design schon geladen ist, ohne Abruf', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({}));
		await ladeAusweisDesign();
		const { bild } = karteMitBild();

		expect(await designFuerDruck()).toBe(true);
		expect(apiFetch).toHaveBeenCalledTimes(1);
		// Die Karten stehen seit dem Laden: Auf ihre Bilder wartet der Druck nicht noch einmal.
		expect(bild.decode).not.toHaveBeenCalled();
	});

	it('druckt nicht mit den Standardwerten: Scheitert der Abruf, steht die Meldung da', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Netz weg'));

		expect(await designFuerDruck()).toBe(false);
		expect(toastStore.addToast).toHaveBeenCalledTimes(1);
		expect(toastStore.addToast).toHaveBeenCalledWith(
			'Nicht gedruckt: Das Ausweis-Design ist nicht geladen.',
			'error'
		);
	});

	it('lädt ein fehlendes Design vor dem Druck nach und wartet auf die Bilder der Karten', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({ printMode: 'card' }));
		const { bild, bildIstDa } = karteMitBild();
		let frei = false;

		const druck = designFuerDruck().then((ergebnis) => {
			frei = true;
			return ergebnis;
		});
		await vi.waitFor(() => expect(bild.decode).toHaveBeenCalledTimes(1));
		expect(frei).toBe(false);

		bildIstDa();
		expect(await druck).toBe(true);
		expect(toastStore.addToast).not.toHaveBeenCalled();
	});

	it('wartet nicht ewig auf ein Bild, das sich nicht laden lässt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({ printMode: 'card' }));
		const karte = document.createElement('div');
		karte.className = 'print-card-box';
		const bild = document.createElement('img');
		bild.decode = vi.fn(() => Promise.reject(new Error('kein Bild')));
		karte.append(bild);
		document.body.append(karte);

		expect(await designFuerDruck()).toBe(true);
	});
});

describe('meldeDesignNichtGeladen', () => {
	it('sagt, dass ohne das Design nicht gedruckt wird', () => {
		meldeDesignNichtGeladen();

		expect(toastStore.addToast).toHaveBeenCalledWith(
			'Ausweis-Design nicht geladen: Ausweise lassen sich gerade nicht drucken.',
			'error'
		);
	});
});

// Ein Druckweg, der das Design selbst abruft, kennt weder den geteilten Abruf noch die
// Sperre des Drucks. Die Ablage des Designers lädt eigens: Sie zeigt den Fehler an der
// Leinwand und sperrt das Speichern.
const ABRUFER = [
	'src/lib/designer/ausweisDesignLaden.js',
	'src/lib/designer/idDesignPersistenz.svelte.js'
];
const MUSTER = /['"`]\/api\/ausweis-layout['"`]/;

describe('Abruf des Ausweis-Designs', () => {
	it('steht nur im Lader und in der Ablage des Designers', () => {
		const treffer = sammleQuelldateien(srcRoot)
			.filter((f) => MUSTER.test(ohneKommentare(readFileSync(f, 'utf8'))))
			.map(relPfad)
			.sort();
		expect(treffer).toEqual(ABRUFER);
	});

	it('Selbstprobe: der Detektor fasst die Aufrufformen und keine Erwähnung im Kommentar', () => {
		for (const form of [
			"apiFetch('/api/ausweis-layout')",
			'apiFetch("/api/ausweis-layout", { method: \'PUT\' })',
			'fetch(`/api/ausweis-layout`)'
		]) {
			expect(MUSTER.test(ohneKommentare(form)), form).toBe(true);
		}
		expect(MUSTER.test(ohneKommentare("// lädt über apiFetch('/api/ausweis-layout')"))).toBe(false);
	});
});
