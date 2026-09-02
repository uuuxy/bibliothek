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
			if (b.istLernmittel && 'lernmittel'.includes(term)) return true;
			// Die Signatur ist die Regaladresse (Handbuch) — bis zum 02.09.2026 fand die
			// Suche sie nicht, obwohl der Payload sie längst trug.
			if (b.signatur && b.signatur.toLowerCase().includes(term)) return true;
			const num = parseInt(term, 10);
			return !isNaN(num) && trifftJahrgang(b, num);
		})
	);
}
