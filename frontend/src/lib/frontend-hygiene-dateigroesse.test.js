import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, relPfad, sammleQuelldateien } from './hygiene-quellen.js';

// Ratsche auf die Regel „höchstens 200 Zeilen je Frontend-Datei" (CLAUDE.md,
// docs/ARCHITEKTUR.md 5.3): Komponenten und JS-Module, Tests nicht. Googles Maßstab für
// Dateien: „files should be focused enough that a maintainer can tell which file contains
// something" (Best Practices, „Package size").
//
//   - Eine neue Datei über 200 Zeilen ist rot.
//   - Eine Datei aus dem Bestand, die weiter wächst, ist rot; die Ausnahme ist kein Freibrief.
//   - Wer eine Datei unter 200 bringt, trägt sie aus: Der Bestand ist eine Arbeitsliste.
const BESTAND = {
	'src/inventur/lib/components/admin/KlassensatzDialog.svelte': 206,
	'src/inventur/lib/components/admin/KlassenUebersicht.svelte': 209,
	'src/lib/components/bestellungen/Bestellwesen.svelte': 216,
	'src/lib/components/omnibox/Omnibox.svelte': 202,
	'src/lib/components/statistik/Statistik.svelte': 341,
	'src/lib/components/leserakte/Leserakte.svelte': 211,
	'src/lib/components/benutzer/Konten.svelte': 208,
	'src/lib/components/etiketten/EtikettenNachdruck.svelte': 359,
	'src/lib/components/statistik/StatistikVerlauf.svelte': 236,
	'src/lib/components/leser/LusdImport.svelte': 310,
	'src/lib/apiFetch.js': 291,
	'src/lib/components/leser/schuelerSuche.svelte.js': 206,
	'src/lib/designer/ausweisVorlagen.js': 451,
	'src/lib/designer/idDesignerStore.svelte.js': 541,
	'src/lib/components/lmfplan/lmfplanDienst.js': 380,
	'src/lib/components/lmfplan/lmfplanPlaner.svelte.js': 232,
	'src/lib/menue.js': 272,
	'src/lib/stores/authStore.svelte.js': 351,
	'src/lib/stores/bestellStore.svelte.js': 531,
	'src/lib/stores/etiketten.svelte.js': 348,
	'src/lib/stores/leerlaufSperre.svelte.js': 382,
	'src/lib/stores/mahnwesen.svelte.js': 231,
	'src/lib/stores/offlineSync.svelte.js': 420,
	'src/lib/stores/omnibox.svelte.js': 412,
	'src/lib/components/inventur/useInventur.svelte.js': 233,
	'src/lib/components/leserakte/useLeserakte.svelte.js': 298
};

describe('Dateigröße: höchstens 200 Zeilen je Komponente und JS-Modul', () => {
	const dateien = sammleQuelldateien(srcRoot);
	// Wie `wc -l` zaehlen, also Zeilenumbrueche: Ein abschliessendes \n erzeugt beim
	// Split ein leeres letztes Element, und das ist keine Zeile. Ohne diese Zeile meldete
	// das Gate JEDE Datei um eins zu gross und damit den ganzen Bestand als gewachsen.
	const zeilenVon = (absolut) => {
		const teile = readFileSync(absolut, 'utf8').split('\n');
		if (teile.at(-1) === '') teile.pop();
		return teile.length;
	};
	const aktuell = new Map(dateien.map((a) => [relPfad(a), zeilenVon(a)]));

	it('durchsucht Komponenten und JS-Module — sonst waere das Gate wertlos gruen', () => {
		expect(dateien.filter((p) => p.endsWith('.svelte')).length).toBeGreaterThan(150);
		expect(dateien.filter((p) => p.endsWith('.js')).length).toBeGreaterThan(100);
	});

	it('kennt keine NEUE Datei ueber 200 Zeilen', () => {
		const neu = [];
		for (const [pfad, zeilen] of aktuell) {
			if (zeilen > 200 && !(pfad in BESTAND)) neu.push(`${pfad} (${zeilen})`);
		}
		expect(
			neu,
			'Neue Datei ueber 200 Zeilen. CLAUDE.md und docs/ARCHITEKTUR.md (5.3) verlangen hoechstens ' +
				'200 — aufteilen (Teilkomponente, {#snippet}, ein Modul je Sache) oder die Regel dort aendern.'
		).toEqual([]);
	});

	it('laesst keine Datei aus dem Bestand weiter wachsen', () => {
		const gewachsen = [];
		for (const [pfad, erlaubt] of Object.entries(BESTAND)) {
			const zeilen = aktuell.get(pfad);
			if (zeilen !== undefined && zeilen > erlaubt)
				gewachsen.push(`${pfad}: ${erlaubt} → ${zeilen}`);
		}
		expect(
			gewachsen,
			'Eine geduldete Datei ist weiter gewachsen. Die Ausnahme ist kein Freibrief.'
		).toEqual([]);
	});

	it('fuehrt nichts, was es nicht mehr gibt oder inzwischen sauber ist', () => {
		const veraltet = [];
		for (const [pfad, erlaubt] of Object.entries(BESTAND)) {
			const zeilen = aktuell.get(pfad);
			if (zeilen === undefined) veraltet.push(`${pfad} (gibt es nicht mehr)`);
			else if (zeilen <= 200) veraltet.push(`${pfad} (nur noch ${zeilen} Zeilen)`);
			else if (zeilen < erlaubt)
				veraltet.push(`${pfad} (geschrumpft auf ${zeilen}, Eintrag sagt ${erlaubt})`);
		}
		expect(
			veraltet,
			'Der Bestand ist ueberholt — austragen bzw. die Zahl nachziehen. Eine Liste, die ' +
				'Erledigtes weiterfuehrt, verliert ihre Aussage.'
		).toEqual([]);
	});
});
