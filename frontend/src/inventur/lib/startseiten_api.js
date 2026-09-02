import { apiFetch } from '../../lib/apiFetch.js';
/**
 * startseiten_api.js
 *
 * Enthält alle API-Aufrufe und Hilfsfunktionen für die Gast-Startseite.
 * Hierzu gehören: Bücher laden, Klassen laden,
 * sowie Filterung und Gruppierung der Bücher nach Klassen.
 */

/**
 * Lädt alle Bücher aus der API.
 * @returns {Promise<any[]>} Liste der Bücher
 */
export async function buecherLaden() {
	const antwort = await apiFetch('/api/books', {
		credentials: 'include'
	});
	if (!antwort.ok) {
		if (antwort.status === 401) {
			throw new Error('UNAUTHORIZED');
		}
		throw new Error('Fehler beim Laden der Bücher');
	}
	return (await antwort.json()).data ?? [];
}

// WZ-Synonyme für Suchbegriffe auf der Startseite.
const suchSynonyme = new Map([
	['powi', 'politik'],
	['mathe', 'mathematik'],
	['eng', 'englisch'],
	['deu', 'deutsch'],
	['franz', 'französisch'],
	['bio', 'biologie'],
	['che', 'chemie'],
	['phy', 'physik'],
	['geo', 'geographie'],
	['info', 'informatik'],
	['lat', 'latein'],
	['span', 'spanisch'],
	['rel', 'religion'],
	['reli', 'religion']
]);

/**
 * Trifft ein Buch den Jahrgang? Entweder über gradeLevel oder über die gepflegte
 * Spanne von–bis. Eine Regel für Suche UND Filter — zwei Definitionen wären nur
 * zufällig einig.
 * @param {any} b
 * @param {number} jahrgang
 */
export function trifftJahrgang(b, jahrgang) {
	if (b.gradeLevel && Number(b.gradeLevel) === jahrgang) return true;
	return (
		!!b.jahrgangVon && !!b.jahrgangBis && jahrgang >= b.jahrgangVon && jahrgang <= b.jahrgangBis
	);
}

/**
 * Die Buch-Suche der Startseite: jeder Begriff muss mindestens ein Feld treffen;
 * Zahlen zählen als Jahrgang (trifft gradeLevel ODER die Spanne von–bis), Füllwörter
 * wie „Klasse"/„Jg." fallen dann weg.
 * @param {any[]} buecherArray
 * @param {string} searchQuery
 */
export function buecherSuchen(buecherArray, searchQuery) {
	const q = searchQuery.toLowerCase().trim();
	if (q === '') return Array.isArray(buecherArray) ? buecherArray : [];

	let terms = q.split(/\s+/).map((term) => suchSynonyme.get(term) || term);
	const hasNumber = terms.some((t) => !isNaN(parseInt(t, 10)));
	if (hasNumber) {
		terms = terms.filter((t) => !['klasse', 'kl', 'kl.', 'jahrgang', 'jg', 'jg.'].includes(t));
	}

	return (Array.isArray(buecherArray) ? buecherArray : []).filter((/** @type {any} */ b) =>
		terms.every((term) => {
			if (b.title && b.title.toLowerCase().includes(term)) return true;
			if (b.isbn && b.isbn.toLowerCase().includes(term)) return true;
			if (b.author && b.author.toLowerCase().includes(term)) return true;
			if (b.subject && b.subject.toLowerCase().includes(term)) return true;
			if (b.track && b.track.toLowerCase().includes(term)) return true;
			// Die Signatur ist die Regaladresse (Handbuch) — bis zum 02.09.2026 fand die
			// Suche sie nicht, obwohl der Payload sie längst trug.
			if (b.signatur && b.signatur.toLowerCase().includes(term)) return true;
			const num = parseInt(term, 10);
			return !isNaN(num) && trifftJahrgang(b, num);
		})
	);
}

// Der Import-Default der Jahrgangsspanne (schema.sql: jahrgang_von DEFAULT 5,
// jahrgang_bis DEFAULT 10). Ein Titel mit genau dieser Spanne wurde aller
// Wahrscheinlichkeit nach nie gepflegt — er behauptet „gilt für 5–10", weil das
// beim Import jeder tat.
const STANDARD_SPANNE = { von: 5, bis: 10 };
export const STANDARD_GRUPPE = 'Ohne genaue Zuordnung';

/**
 * Gruppiert Bücher nach ihrer JAHRGANGSSPANNE + Zweig (z. B. "Klasse 5",
 * "Klasse 5–6 Förderstufe").
 *
 * Bis zum 24.08.2026 gruppierte diese Funktion nach `gradeLevel` — einem Feld, das
 * der Littera-Import nie setzt und das sonst fast niemand liest. Ergebnis: Der
 * Reiter „Jahrgänge" war für importierte Bestände LEER, während handgepflegte
 * Spannen (5–5, 6–6) unsichtbar in `jahrgangVon/Bis` standen — dem Feld, das der
 * Rest des Systems (Ausleihlimit-Scope, Suche) längst als maßgeblich behandelt.
 *
 * Titel mit dem unangetasteten Import-Default 5–10 sammeln sich in EINER Gruppe am
 * Ende (Betreiber-Entscheidung 24.08.2026): einsortiert in jeden Jahrgang würden
 * sie jede Filterung fluten, ausgeblendet wären sie unauffindbar.
 * @param {any[]} buecherArray - Alle verfügbaren Bücher
 * @returns {any[]} Sortierte Liste von Klasseobjekten ({ name, von, bis, standard, books })
 */
export function buecherNachKlassenGruppieren(buecherArray) {
	const klassenMap = new Map();
	for (const buch of buecherArray) {
		const von = buch.jahrgangVon || 0;
		const bis = buch.jahrgangBis || von;
		const standard = von === 0 || (von === STANDARD_SPANNE.von && bis === STANDARD_SPANNE.bis);
		const name = standard
			? STANDARD_GRUPPE
			: `Klasse ${von === bis ? von : `${von}–${bis}`}${buch.track ? ' ' + buch.track : ''}`;
		if (!klassenMap.has(name)) {
			klassenMap.set(name, { name, von, bis, standard, books: [] });
		}
		klassenMap.get(name).books.push(buch);
	}
	return Array.from(klassenMap.values()).sort(
		(a, b) =>
			Number(a.standard) - Number(b.standard) ||
			a.von - b.von ||
			a.bis - b.bis ||
			a.name.localeCompare(b.name)
	);
}

/**
 * Wendet Zweig- und Jahrgangsfilter auf die Gruppen an. „Jahrgang 6" heißt: Die
 * Spanne der Gruppe DECKT die 6 ab (5–6 zählt, 7–8 nicht). Die Standard-Gruppe
 * bleibt immer stehen — sie ist der sichtbare Rest, der noch Pflege braucht.
 * @param {any[]} classes
 * @param {string} zweig
 * @param {string} jahrgang
 */
export function klassenFiltern(classes, zweig, jahrgang) {
	const j = parseInt(jahrgang, 10);
	return (Array.isArray(classes) ? classes : []).filter((cls) => {
		if (cls.standard) return true;
		const zw = zweig === '' || cls.books.some((/** @type {any} */ b) => b.track === zweig);
		const jg = jahrgang === '' || (cls.von <= j && j <= cls.bis);
		return zw && jg;
	});
}

/**
 * Die Filter-Optionen entstehen aus den GELADENEN Büchern, nicht aus einer zweiten
 * handgepflegten Liste: Die alte Kopie im Filter kannte drei Zweige, während das
 * Bearbeiten-Formular sechs führt — die Förderstufe war so nie wählbar. Was hier
 * steht, existiert auch wirklich im Bestand; ein neuer Zweig erscheint von selbst.
 * @param {any[]} buecherArray
 */
export function zweigOptionenAus(buecherArray) {
	const zweige = [...new Set(buecherArray.map((b) => b.track).filter(Boolean))].sort();
	return [{ value: '', label: 'Alle Zweige' }, ...zweige.map((z) => ({ value: z, label: z }))];
}

/** Alle Jahrgänge, die eine gepflegte Spanne abdeckt. @param {any[]} classes */
export function jahrgangOptionenAus(classes) {
	const jahre = new Set();
	for (const cls of classes) {
		if (cls.standard) continue;
		for (let j = cls.von; j <= cls.bis; j++) jahre.add(j);
	}
	return [
		{ value: '', label: 'Alle Jahrgänge' },
		...[...jahre].sort((a, b) => a - b).map((j) => ({ value: String(j), label: `Klasse ${j}` }))
	];
}

/**
 * Bestimmt die CSS-Klasse für die Bestandsanzeige (Farbampel).
 * @param {number} bestand - Aktueller Buchbestand
 * @returns {string} Tailwind-CSS-Klassen für die Bestandsanzeige
 */
export function bestandsFarbe(bestand) {
	if (bestand === 0) return 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.5)]';
	if (bestand < 5) return 'bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.5)]';
	return 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]';
}
