import { describe, it, expect } from 'vitest';
import { ordneScanEin } from './scanEinordnen.js';

// Je FORM ein Fall — das ist der Punkt dieser Datei.
//
// Anlass: Der zweite Stufe-1-Nachweis am Stack (16.09.2026) endete mit „Netzwerkfehler",
// weil die Warteschlange nur `B-` annahm. Kein Gate konnte das sehen: Jeder Fall in
// stores/omniboxOffline.test.js scannt `B-10234`. Geprueft war genau der eine Weg, der
// funktionierte — deshalb hier jede Form, die an der Theke wirklich ueber den Tisch geht.
describe('Scan ohne Netz einordnen', () => {
	// Die Barcode-Liste des Rechners, klein gehalten: zwei nackte Littera-Nummern.
	const liste = new Set(['58968', '124117']);
	/** @param {string} n */
	const istBuch = (n) => liste.has(n);
	/** @param {string} s */
	const ein = (s) => ordneScanEin(s, istBuch);

	it('erkennt die Ausweis-Vorsilben A-, S- und L-', () => {
		// A- wird seit dem 16.09.2026 vergeben, S-/L- gibt es von frueher.
		expect(ein('A-00042')).toEqual({ art: 'ausweis', nummer: 'A-00042' });
		expect(ein('S-00042')).toEqual({ art: 'ausweis', nummer: 'S-00042' });
		expect(ein('L-4711')).toEqual({ art: 'ausweis', nummer: 'L-4711' });
	});

	it('erkennt B- und LMF- als Buch — auch ohne Barcode-Liste', () => {
		// Entscheidung vom 13.09.2026: Die Vorsilbe ist eindeutig, eine fehlende Liste darf
		// ein klar erkennbares Buch nicht verwerfen.
		const ohneListe = () => false;
		expect(ordneScanEin('B-00123', ohneListe)).toEqual({ art: 'buch', nummer: 'B-00123' });
		expect(ordneScanEin('LMF-2025-0007', ohneListe)).toEqual({
			art: 'buch',
			nummer: 'LMF-2025-0007'
		});
	});

	it('nimmt eine nackte Nummer, die auf der Liste steht', () => {
		// Der Altbestand traegt seine Littera-Mediennummer nackt als Exemplar-Barcode.
		expect(ein('58968')).toEqual({ art: 'buch', nummer: '58968' });
	});

	it('rechnet ein Littera-Etikett auf die Nummer zurueck und bucht unter DIESER', () => {
		// Der Strichcode traegt die EAN-13, die Liste die Nummer darin (gemessener Fall
		// aus litteraEtikett.faelle.json: Aufdruck 58968).
		expect(ein('5896800039556')).toEqual({ art: 'buch', nummer: '58968' });
	});

	it('nennt ein Geraet beim Namen statt es still zu verwerfen', () => {
		// Geraete offline stehen nicht im Umfang (OFFEN.md 2.4) — sie haengen an einer
		// Checkliste, die es ohne Netz nicht gibt. Der Bediener soll das erfahren.
		expect(ein('G-17')).toEqual({ art: 'geraet', nummer: 'G-17' });
	});

	it('haelt eine unbekannte Nummer fuer unklar — und raet NICHT auf Ausweis', () => {
		// Ein Schuelerausweis des Altbestands liefert gemessen B97601826457, ein
		// Buchetikett eine 13-stellige EAN. Wer hier raet, schreibt die naechsten Buecher
		// einem fremden Kind zu.
		expect(ein('B97601826457')).toEqual({ art: 'unklar', nummer: 'B97601826457' });
		expect(ein('999999')).toEqual({ art: 'unklar', nummer: '999999' });
	});

	it('ein Etikett, dessen Nummer nicht auf der Liste steht, bleibt unklar', () => {
		// Sonst wuerde eine zufaellig gueltige EAN-13 zu einem Buch, das es nicht gibt.
		expect(ein('1234567039572')).toEqual({ art: 'unklar', nummer: '1234567039572' });
	});

	it('haelt eine getippte Eingabe fuer eine Suche, nicht fuer eine Nummer', () => {
		// „steht nicht in der Buchliste" waere ueber einen Namen schlicht falsch. Erkannt
		// am Fehlen von Ziffern oder an einem Leerzeichen — beides kommt aus keinem
		// Strichcode.
		expect(ein('Mueller')).toEqual({ art: 'suche', nummer: 'Mueller' });
		expect(ein('Anna Mueller')).toEqual({ art: 'suche', nummer: 'Anna Mueller' });
	});

	it('eine Nummer mit Ziffern bleibt unklar und wird NICHT zur Suche', () => {
		// Gegenprobe: Ein alter Ausweis liefert gemessen B97601826457 — Ziffern, kein
		// Leerzeichen. Der gehoert nicht in die Namenssuche.
		expect(ein('B97601826457').art).toBe('unklar');
	});

	// Von Hand getippt: Gespeichert sind die Nummern mit grosser Vorsilbe, und der Server
	// schlaegt exakt nach. Gemeldet am Stack am 16.09.2026: „s-10001 ... hat nicht
	// funktioniert" — es war klein geschrieben.
	it('versteht eine klein geschriebene Vorsilbe und bucht unter der grossen', () => {
		expect(ein('s-10001')).toEqual({ art: 'ausweis', nummer: 'S-10001' });
		expect(ein('a-00042')).toEqual({ art: 'ausweis', nummer: 'A-00042' });
		expect(ordneScanEin('b-00123', () => false)).toEqual({ art: 'buch', nummer: 'B-00123' });
		expect(ordneScanEin('lmf-2025-0007', () => false)).toEqual({
			art: 'buch',
			nummer: 'LMF-2025-0007'
		});
	});

	it('macht aus „s-bahn" KEINEN Ausweis', () => {
		// Die Gegenprobe zur Vereinheitlichung: Ohne die Ziffern-Bedingung waere jedes
		// Wort mit Bindestrich eine Kartennummer.
		expect(ein('s-bahn').art).not.toBe('ausweis');
		expect(ein('b-movie').art).not.toBe('buch');
	});

	it('leerer Scan ist unklar, nicht Buch', () => {
		expect(ein('   ')).toEqual({ art: 'unklar', nummer: '' });
	});
});

// Ein Etikett von VOR dem 17.09.2026 (OFFEN.md 9.2). Damals druckte die Anwendung Code 39
// MIT Prüfzeichen; das Lesegerät gibt es als Teil der Nummer zurück. Online löst der
// Server das auf — ohne Netz muss die Theke dasselbe können, sonst gälte ein Buch offline
// als unklar, das online gebucht wird.
describe('Aufdruck mit Code-39-Prüfzeichen (Etiketten von früher)', () => {
	// „58968" ist eine nackte Littera-Mediennummer aus dem Altbestand; ihr Prüfzeichen ist
	// ein Bindestrich (code39.faelle.json).
	const listeKennt58968 = (/** @type {string} */ n) => n === '58968';

	it('findet das Buch über die gekürzte Nummer', () => {
		expect(ordneScanEin('58968-', listeKennt58968)).toEqual({ art: 'buch', nummer: '58968' });
	});

	it('bucht unter der NUMMER, nicht unter dem Aufdruck', () => {
		// Der Server kennt nur „58968". Käme hier der volle Scan zurück, scheiterte das
		// Nachbuchen später — und zwar erst, wenn das Netz wieder da ist.
		expect(ordneScanEin('58968-', listeKennt58968).nummer).not.toBe('58968-');
	});

	it('rät nicht, wenn die gekürzte Nummer nicht in der Liste steht', () => {
		// Die Nachsicht darf nichts erfinden: Kennt die Liste die Nummer nicht, bleibt der
		// Scan unklar — und die Theke sperrt die Zuordnung, statt auf ein fremdes Buch zu
		// buchen.
		expect(ordneScanEin('58968-', () => false).art).toBe('unklar');
	});

	it('lässt einen Scan OHNE Prüfzeichen unverändert', () => {
		// Der Normalfall seit dem 17.09.2026: Code 128. Er wird roh gefunden, und die
		// Nachsicht fasst ihn nie an.
		expect(ordneScanEin('58968', listeKennt58968)).toEqual({ art: 'buch', nummer: '58968' });
	});

	it('greift erst NACH der Littera-Rückrechnung', () => {
		// Ein gültiges Littera-Etikett muss über seinen eigenen Weg aufgelöst werden. Die
		// Reihenfolge ist die Zusicherung: roh, Littera, dann erst das Prüfzeichen.
		expect(ordneScanEin('5896800039556', (n) => n === '58968')).toEqual({
			art: 'buch',
			nummer: '58968'
		});
	});
});
