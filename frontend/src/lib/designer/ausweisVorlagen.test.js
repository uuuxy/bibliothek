import { describe, it, expect, beforeEach } from 'vitest';
import { AUSWEIS_VORLAGEN, vorlage, wendeVorlageAn } from './ausweisVorlagen.js';
import {
	idStore,
	resetDesign,
	PLATZHALTER_SCHULNAME,
	PLATZHALTER_ADRESSE
} from './idDesignerStore.svelte.js';

// ISO 7810 ID-1 — dieselben Maße, die der Designer und der Kartendruck verwenden.
const KARTE_BREITE = 85.6;
const KARTE_HOEHE = 53.98;

const alleKennungen = AUSWEIS_VORLAGEN.map((v) => v.value);

describe('Design-Vorlagen: Inhalt', () => {
	it.each(alleKennungen)('%s: jedes Element liegt vollständig auf der Karte', (kennung) => {
		const v = vorlage(kennung);
		for (const seite of [v?.front, v?.back]) {
			for (const el of seite?.elements ?? []) {
				expect(el.x, `${kennung}/${el.id}: x`).toBeGreaterThanOrEqual(0);
				expect(el.y, `${kennung}/${el.id}: y`).toBeGreaterThanOrEqual(0);
				expect(el.x + el.width, `${kennung}/${el.id}: rechter Rand`).toBeLessThanOrEqual(
					KARTE_BREITE + 0.01
				);
				expect(el.y + el.height, `${kennung}/${el.id}: unterer Rand`).toBeLessThanOrEqual(
					KARTE_HOEHE + 0.01
				);
			}
		}
	});

	it.each(alleKennungen)('%s: IDs sind je Seite eindeutig', (kennung) => {
		const v = vorlage(kennung);
		for (const seite of [v?.front, v?.back]) {
			const ids = (seite?.elements ?? []).map((e) => e.id);
			expect(new Set(ids).size).toBe(ids.length);
		}
	});

	it.each(alleKennungen)(
		'%s: Vorderseite trägt Foto, Name, Gültigkeit, Barcode und ein LEERES Logofeld',
		(kennung) => {
			const front = vorlage(kennung)?.front.elements ?? [];
			for (const typ of ['photo', 'name', 'validity', 'barcode', 'logo']) {
				expect(front.filter((e) => e.type === typ).length, typ).toBe(1);
			}
			// Leer, damit nie ein falsches (nachgezeichnetes) Logo auf echten Karten landet;
			// die Schule lädt ihre echte Logodatei selbst hoch.
			expect(front.find((e) => e.type === 'logo')?.content).toBe('');
		}
	);

	it.each(alleKennungen)(
		'%s: Schulname/Adresse nutzen die Platzhalter-IDs, damit die Stammdaten-Heilung greift',
		(kennung) => {
			const front = vorlage(kennung)?.front.elements ?? [];
			expect(front.find((e) => e.id === 'header')?.content).toBe(PLATZHALTER_SCHULNAME);
			expect(front.find((e) => e.id === 'address')?.content).toBe(PLATZHALTER_ADRESSE);
		}
	);

	it.each(alleKennungen)(
		'%s: keine Klassen-/Schuljahreszeile (Karte gilt die ganze Schulzeit)',
		(kennung) => {
			const v = vorlage(kennung);
			for (const seite of [v?.front, v?.back]) {
				expect((seite?.elements ?? []).some((e) => e.type === 'details')).toBe(false);
			}
		}
	);

	it('Waldgrün: der Barcode liegt auf dem hellen Fußband (Nummer wird dunkel gerendert)', () => {
		const front = vorlage('waldgruen')?.front.elements ?? [];
		const band = front.find((e) => e.id === 'fussband');
		const barcode = front.find((e) => e.type === 'barcode');
		expect(barcode?.y).toBeGreaterThanOrEqual(band?.y ?? Infinity);
	});

	it('liefert bei jedem Aufruf frische Elemente (keine geteilten Objekte)', () => {
		const a = vorlage('schwarz-gruen');
		const b = vorlage('schwarz-gruen');
		expect(a?.front.elements[0]).not.toBe(b?.front.elements[0]);
		if (a) a.front.elements[0].x = 99;
		expect(b?.front.elements[0].x).not.toBe(99);
	});
});

describe('wendeVorlageAn', () => {
	beforeEach(() => {
		resetDesign();
	});

	it('ersetzt Elemente UND Theme beider Seiten', () => {
		expect(wendeVorlageAn('waldgruen')).toBe(true);
		expect(idStore.front.theme).toContain('from-[#16330a]');
		expect(idStore.back.theme).toContain('from-[#16330a]');
		expect(idStore.front.elements.some((e) => e.id === 'fussband')).toBe(true);
		expect(idStore.back.elements.some((e) => e.id === 'back-panel')).toBe(true);
	});

	it('übernimmt ein bereits hochgeladenes Schullogo in die Vorlage (leer heißt „noch keins", nicht „entfernen")', () => {
		const logo = idStore.front.elements.find((e) => e.type === 'logo');
		if (logo) logo.content = 'data:image/png;base64,ECHTES-SCHULLOGO';

		expect(wendeVorlageAn('schwarz-gruen')).toBe(true);

		expect(idStore.front.elements.find((e) => e.type === 'logo')?.content).toBe(
			'data:image/png;base64,ECHTES-SCHULLOGO'
		);
	});

	it('lässt das Logofeld leer, wenn vorher keines hochgeladen war', () => {
		expect(wendeVorlageAn('reis-welle')).toBe(true);
		expect(idStore.front.elements.find((e) => e.type === 'logo')?.content).toBe('');
	});

	it('lässt den Store bei unbekannter Kennung unangetastet', () => {
		const vorher = JSON.stringify(idStore.front.elements.map((e) => e.id));
		expect(wendeVorlageAn('gibt-es-nicht')).toBe(false);
		expect(JSON.stringify(idStore.front.elements.map((e) => e.id))).toBe(vorher);
	});

	it('jede angebotene Vorlage existiert wirklich (Dropdown und Daten laufen nicht auseinander)', () => {
		for (const kennung of alleKennungen) {
			expect(vorlage(kennung), kennung).not.toBeNull();
		}
	});
});
