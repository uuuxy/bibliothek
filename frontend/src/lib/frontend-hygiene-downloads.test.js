import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { sammleQuelldateien, srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ein Dokument mit Personendaten geht im Tab auf und landet nicht als Datei im Download-Ordner
// des Arbeitsplatzes, wo es für den Nächsten am Rechner liegen bliebe (ARCHITEKTUR.md 8.9). Als
// Datei kommt nur, was keine Personendaten trägt oder als Sicherung aufgehoben werden soll.
// Ein PDF kommt nur über zeigePdf (utils/pdfImFenster.js) in einen Tab: Sie öffnet ihn beim Klick,
// vor der Anfrage, und gibt die Adresse erst frei, wenn er zu ist.
//
// Sieht nicht: einen Link auf eine Adresse, die der Server als Anhang schickt (das hält
// api/anhang_ratsche_test.go), und einen Download über eine andere Schreibweise
// als `.download =` oder das Attribut download an einem Link.
const ALS_DATEI = {
	'src/inventur/lib/admin_api.js': 'der Katalog als CSV',
	'src/lib/stores/offlineSync.svelte.js': 'die Sicherung der Offline-Scans, zum Aufheben',
	'src/lib/components/lmfplan/lmfplanDienst.js': 'der LMF-Plan, Termine je Klasse',
	'src/lib/components/bestellungen/Bestellbedarf.svelte': 'die Bestellliste',
	'src/lib/components/portal/FachKarte.svelte': 'die Bücher eines Fachs im Portal'
};
const DOWNLOAD = /\.download\s*=|<a\b[^>]*\sdownload\b/;
const ZEIGE_PDF = 'src/lib/utils/pdfImFenster.js';
const BLOB = /createObjectURL\(/;
const IN_EIN_FENSTER = /window\.open\(|\.location\.replace\(/;

describe('Dokumente gehen im Tab auf, nicht in den Download-Ordner', () => {
	const quellen = sammleQuelldateien(srcRoot).map((f) => ({
		pfad: relPfad(f),
		text: ohneKommentare(readFileSync(f, 'utf8'))
	}));

	it('lädt nur herunter, was keine Personendaten trägt', () => {
		const downloads = quellen.filter((q) => DOWNLOAD.test(q.text)).map((q) => q.pfad);
		expect(
			downloads.filter((p) => !(p in ALS_DATEI)),
			'Download einer Datei — ein PDF mit Personendaten über zeigePdf (utils/pdfImFenster.js) zeigen'
		).toEqual([]);
		// Jeder Eintrag muss noch einen Download finden, sonst misst die Liste nichts mehr.
		expect(
			Object.keys(ALS_DATEI).filter((p) => !downloads.includes(p)),
			'Eintrag in ALS_DATEI ohne Download — austragen'
		).toEqual([]);
	});

	it('öffnet ein geholtes PDF nur über zeigePdf in einem Tab', () => {
		const eigene = quellen
			.filter((q) => BLOB.test(q.text) && IN_EIN_FENSTER.test(q.text))
			.map((q) => q.pfad);
		expect(eigene, 'der Detektor findet zeigePdf nicht').toContain(ZEIGE_PDF);
		expect(
			eigene.filter((p) => p !== ZEIGE_PDF),
			'PDF von Hand in ein Fenster geladen — zeigePdf aus utils/pdfImFenster.js nehmen'
		).toEqual([]);
	});

	// Die Helfer sagen, ob der Browser den Tab verweigert hat, und der Aufrufer meldet es. Die Briefe
	// der Bescheide öffneten ihren Tab von Hand, und ein blockierter Tab blieb ohne Wort.
	it('öffnet einen Tab nur über die Helfer, die einen blockierten melden lassen', () => {
		const helfer = [ZEIGE_PDF, 'src/lib/utils/listenDruck.js'];
		const mitGrund = {
			'src/lib/components/bestellungen/BestaetigungEtiketten.svelte':
				'die Seite des Lieferanten: mit noopener liefert window.open immer null'
		};
		const vonHand = quellen.filter((q) => /window\.open\(/.test(q.text)).map((q) => q.pfad);
		expect(vonHand, 'der Detektor findet die Helfer nicht').toEqual(expect.arrayContaining(helfer));
		expect(
			vonHand.filter((p) => !helfer.includes(p) && !(p in mitGrund)),
			'Tab von Hand geöffnet — zeigePdf, zeigePdfAdresse oder druckeDokument nehmen und einen blockierten Tab melden'
		).toEqual([]);
		expect(
			Object.keys(mitGrund).filter((p) => !vonHand.includes(p)),
			'Eintrag ohne Fund — austragen'
		).toEqual([]);
	});
});
