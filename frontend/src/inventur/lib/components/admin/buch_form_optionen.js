export const klassenStufen = [0, 5, 6, 7, 8, 9, 10, 11, 12, 13];

/**
 * Der Hinweis unter dem Schalter „Mehrjahresband" (docs/OFFEN.md 9.6, 22.09.2026): Die
 * Zahl, bis zu der das Buch beim Kind bleibt, ist „bis" aus der Spanne — eine zweite gibt
 * es nicht. Dieselbe Regel wie am Server (inventur/mehrjahresband.go): nur mit einer Spanne
 * über mehr als einen Jahrgang.
 * @param {boolean} an
 * @param {number|string} von
 * @param {number|string} bis
 * @returns {string}
 */
export function mehrjahresbandHinweis(an, von, bis) {
	const v = Number(von);
	const b = Number(bis);
	const spanneOk = Number.isInteger(v) && Number.isInteger(b) && v >= 1 && b <= 13 && b > v;
	if (!an) {
		return 'Aus: Das Buch kommt wie jedes Schulbuch am Rückgabetermin der Klasse zurück. An: Es bleibt über die Spanne beim Kind.';
	}
	if (!spanneOk) {
		return 'Braucht eine Spanne über mehr als einen Jahrgang: „bis" muss über „von" liegen, sonst lässt sich der Titel nicht speichern.';
	}
	return `Bleibt beim Kind bis zum Ende von Jahrgang ${b}. Ein Kind der ${v} gibt es nach ${b - v + 1} Schuljahren zurück; die Frist ist der Stichtag dieses Schuljahres.`;
}

/**
 * Der Hinweis unter „Aktueller Bestand". Neuer Titel: Ohne Exemplar zeigt ihn kein Katalog
 * (repository.SQLTitelHatExemplar), nur die Sicht „Ohne Exemplare" der Titelliste.
 * Vorhandener Titel: Ein leeres Feld ändert nichts (bestandsAngabe in buch_speichern.js).
 * @param {string|null|undefined} id
 * @param {number|string|null|undefined} bestand
 * @returns {string}
 */
export function bestandHinweis(id, bestand) {
	const leer = bestand === null || bestand === undefined || bestand === '';
	if (id) return leer ? 'Ohne Zahl bleibt der Bestand, wie er ist.' : '';
	if (Number(bestand) > 0) return '';
	return 'Ohne Exemplar steht der Titel in keinem Katalog, nur in der Titelliste unter „Ohne Exemplare“.';
}

/**
 * leeresBuchFormular: die EINE Vorlage für ein neues Buch. Sie stand bis zum 03.09.2026
 * zweimal wörtlich in routes/admin/+page.svelte (Anfangszustand und „Neues Buch"); beim
 * Nachtragen des Schulzweigs fiel auf, dass ein neues Feld an beiden Stellen gepflegt
 * werden muss — vergisst man eine, schickt genau einer der beiden Wege das Feld nie mit.
 * @returns {{ id: null, isbn: string, title: string, author: string, subject: string, gradeLevel: number, istLernmittel: boolean, track: string, mehrjahresband: boolean, stock: number, coverUrl: string, lastCounted: string, medientyp: string, auflage: string, schlagworte: string[], listenpreis: number|null }}
 */
export function leeresBuchFormular() {
	return {
		id: null,
		isbn: '',
		title: '',
		author: '',
		subject: '',
		gradeLevel: 5,
		istLernmittel: false,
		track: '',
		mehrjahresband: false,
		// Wie bei Littera kommt mit dem Titel das erste Exemplar: Ohne Exemplar stünde das
		// neue Buch in keinem Katalog. Wer nur den Titel will, trägt 0 ein.
		stock: 1,
		coverUrl: '',
		lastCounted: '',
		medientyp: 'Buch',
		auflage: '',
		// Schlagworte (Migration 138): Ein neuer Titel beginnt ohne. Eine leere Liste ist
		// hier richtig — beim Anlegen gibt es nichts, was sie überschreiben könnte.
		schlagworte: [],
		// null, NICHT 0: Ein leeres Feld heißt „nicht erfasst" — dann rechnet der
		// Schadensersatz mit dem Einkaufspreis und sagt das. Eine 0 hieße „kostet heute
		// nichts" und ergäbe einen Ersatzbetrag von 0,00 € (Migration 127).
		listenpreis: null
	};
}
