import { describe, it, expect, vi, beforeEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, ohneKommentare } from '../hygiene-quellen.js';
import { erzeugeDesignAblage } from './idDesignPersistenz.svelte.js';
import { apiFetch } from '../apiFetch.js';
import { applyDesign, wendeSchulstammdatenAn } from './idDesignerStore.svelte.js';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));
// Teilweise gemockt: ausweisVorlagen.js liest echte Konstanten aus diesem Store.
vi.mock('./idDesignerStore.svelte.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	applyDesign: vi.fn(),
	resetDesign: vi.fn(),
	wendeSchulstammdatenAn: vi.fn()
}));

/**
 * Der Ausweis-Designer speichert automatisch: Jede Änderung an der Leinwand geht 800 ms
 * später zentral an ALLE Arbeitsplätze. Scharf wird diese Automatik durch `geladen`.
 *
 * Bis zum Sweep am 06.09.2026 setzte `laden()` das Flag im `finally` — also auch, wenn
 * der GET scheiterte. Dann zeigte die Leinwand die VORGABEWERTE, die Automatik war
 * scharf, und die direkt danach laufende Stammdaten-Heilung fasst den Store an. Das
 * allein genügte: Das Design der Schule wurde durch das Vorgabe-Design ersetzt, ohne
 * dass jemand etwas anklickte — nur weil der Bildschirm geöffnet wurde, während der
 * Abruf scheiterte. (Dieselbe Bugklasse wie das Einstellungs-Formular am 31.08.2026.)
 */
describe('Ausweis-Design laden', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(applyDesign).mockReset();
	});

	it('bewaffnet die Auto-Speicherung nicht, wenn das Laden scheitert', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: false, status: 500, json: async () => ({}) })
		);
		const ablage = erzeugeDesignAblage();
		await ablage.laden();

		expect(ablage.geladen, 'die Auto-Speicherung würde die Vorgabewerte zentral schreiben').toBe(
			false
		);
		expect(ablage.ladefehler).toMatch(/nicht geladen werden/);
		expect(vi.mocked(applyDesign)).not.toHaveBeenCalled();
		// Nur EIN Abruf: Die Stammdaten-Heilung fasst den Store an und darf auf einer
		// Leinwand voller Vorgabewerte gar nicht erst laufen.
		expect(vi.mocked(apiFetch)).toHaveBeenCalledTimes(1);
	});

	it('bewaffnet sie auch bei einem Netzfehler nicht', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('offline'));
		const ablage = erzeugeDesignAblage();
		await ablage.laden();
		expect(ablage.geladen).toBe(false);
		expect(ablage.ladefehler).toMatch(/Netzwerkfehler/);
	});

	it('lädt beim Erststart ein leeres Design und speichert danach normal weiter', async () => {
		// {} mit HTTP 200 ist die Antwort vor dem ersten Speichern — eine ECHTE Antwort.
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, status: 200, json: async () => ({}) })
		);
		const ablage = erzeugeDesignAblage();
		await ablage.laden();
		expect(ablage.geladen).toBe(true);
		expect(ablage.ladefehler).toBe('');
		expect(vi.mocked(applyDesign)).toHaveBeenCalledWith({});
	});

	// Die andere Hälfte der Kette: Das Flag nützt nur, solange der Auto-Save-Effekt es
	// auch fragt. Am Quelltext OHNE Kommentare geprüft — sonst genügte der erklärende
	// Satz daneben (Bugklasse „Lügende Ratsche durch Kommentar").
	it('die Auto-Speicherung des Bildschirms fragt dieses Flag', () => {
		const quelle = ohneKommentare(
			readFileSync(join(srcRoot, 'lib', 'StudentIdDesigner.svelte'), 'utf8')
		);
		expect(quelle).toMatch(/if\s*\(!ablage\.geladen\)\s*return;/);
	});
});

// Nach dem Laden setzt die Ablage Schulname und Adresse aus den Einstellungen an die Stelle
// der Platzhalter. Die Einstellungen darf nicht jeder lesen, der Ausweise druckt; dann bleibt
// der Platzhalter stehen, ohne Meldung.
describe('Ausweis-Design laden: Schulname und Adresse', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(wendeSchulstammdatenAn).mockReset();
	});

	/** @param {any} einstellungen was die Tür der Einstellungen antwortet */
	async function ladeMit(einstellungen) {
		vi.mocked(apiFetch).mockImplementation(async (pfad) =>
			pfad === '/api/einstellungen'
				? einstellungen
				: /** @type {any} */ ({ ok: true, status: 200, json: async () => ({}) })
		);
		const ablage = erzeugeDesignAblage();
		await ablage.laden();
		return ablage;
	}

	/** @param {Record<string, string>} felder */
	const lesbar = (felder) => ({ ok: true, status: 200, json: async () => felder });

	it('setzt die Adresse aus Straße, Postleitzahl und Ort zusammen', async () => {
		await ladeMit(
			lesbar({
				schule_name: 'Musterschule',
				schule_strasse: 'Schulweg 1',
				schule_plz: '12345',
				schule_ort: 'Musterstadt'
			})
		);

		expect(wendeSchulstammdatenAn).toHaveBeenCalledTimes(1);
		expect(wendeSchulstammdatenAn).toHaveBeenCalledWith(
			'Musterschule',
			'Schulweg 1, 12345 Musterstadt'
		);
	});

	it.each([
		[{ schule_name: 'Musterschule', schule_ort: 'Musterstadt' }, ['Musterschule', 'Musterstadt']],
		[{ schule_strasse: 'Schulweg 1', schule_plz: '12345' }, ['', 'Schulweg 1, 12345']],
		[{}, ['', '']]
	])('lässt weg, was in den Einstellungen fehlt: %j', async (felder, erwartet) => {
		await ladeMit(lesbar(felder));

		expect(wendeSchulstammdatenAn).toHaveBeenCalledWith(...erwartet);
	});

	it('lässt die Leinwand in Ruhe, wenn die Einstellungen nicht lesbar sind', async () => {
		const ablage = await ladeMit({ ok: false, status: 403, json: async () => ({ error: 'x' }) });

		expect(wendeSchulstammdatenAn).not.toHaveBeenCalled();
		expect(ablage.geladen).toBe(true);
		expect(ablage.ladefehler).toBe('');
	});

	it('übersteht einen Netzfehler beim Lesen der Einstellungen', async () => {
		vi.mocked(apiFetch).mockImplementation(async (pfad) => {
			if (pfad === '/api/einstellungen') throw new Error('offline');
			return /** @type {any} */ ({ ok: true, status: 200, json: async () => ({}) });
		});
		const ablage = erzeugeDesignAblage();

		await ablage.laden();

		expect(wendeSchulstammdatenAn).not.toHaveBeenCalled();
		expect(ablage.geladen).toBe(true);
	});
});
