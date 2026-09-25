import { apiFetch } from '../../lib/apiFetch.js';
import { isbnFormen, normalisiereIsbn } from '../../lib/utils/isbnFormen.js';
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
function trifftJahrgang(b, jahrgang) {
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

	// Je Suchbegriff EINMAL vorab: Ist er eine ISBN, und in welchen Schreibweisen kann
	// dieselbe ISBN im Bestand stehen? Ohne das findet ein gescannter Strichcode nur den
	// Bestand, der zeichengleich gespeichert ist — Bindestriche oder eine zehnstellige
	// Alt-ISBN reichten, damit die Suche leer blieb (18.09.2026). Der Aufwand je Buch
	// entsteht nur bei ISBN-Begriffen; getippter Text läuft wie bisher.
	const isbnJeTerm = terms.map((term) => isbnFormen(term));

	return (Array.isArray(buecherArray) ? buecherArray : []).filter((/** @type {any} */ b) =>
		terms.every((term, i) => {
			if (b.title && b.title.toLowerCase().includes(term)) return true;
			if (b.isbn && b.isbn.toLowerCase().includes(term)) return true;
			if (b.isbn && isbnJeTerm[i].length > 0 && isbnJeTerm[i].includes(normalisiereIsbn(b.isbn)))
				return true;
			if (b.author && b.author.toLowerCase().includes(term)) return true;
			if (b.subject && b.subject.toLowerCase().includes(term)) return true;
			if (b.istLernmittel && 'lernmittel'.includes(term)) return true;
			// Die Signatur ist die Regaladresse (Handbuch) — bis zum 02.09.2026 fand die
			// Suche sie nicht, obwohl der Payload sie längst trug.
			if (b.signatur && b.signatur.toLowerCase().includes(term)) return true;
			// Schlagworte und die Verweise darauf (docs/OFFEN.md 4.20). Welche Wörter einen
			// Titel finden, entscheidet der Server (repository.SuchwoerterDerTitel) — dieselbe
			// Regel, nach der die Titel-Verwaltung und das Portal am Server suchen.
			if (
				Array.isArray(b.suchwoerter) &&
				b.suchwoerter.some((/** @type {string} */ w) => w.toLowerCase().includes(term))
			)
				return true;
			const num = parseInt(term, 10);
			return !isNaN(num) && trifftJahrgang(b, num);
		})
	);
}

/**
 * Ein Buch in mehreren Auflagen ist EINE Kachel (docs/OFFEN.md 4.18, Stufe 6; entschieden am
 * 17.09.2026: „Die Suche zeigt einen Treffer mit der Gesamtzahl und darunter die
 * Aufschlüsselung je Auflage"). Oben steht die Auflage, die die Suche getroffen hat — bei einer
 * gescannten ISBN genau diese, sonst die neueste (werkRang 1, am Server nach
 * repository.SQLNeuesteAuflageZuerst). Gezählt werden ALLE Auflagen des Buchs aus der ganzen
 * Liste, nicht nur die getroffenen: Wer eine alte Auflage scannt, fragt, ob die Schule das Buch
 * hat. Die Kachel steht dort, wo die erste getroffene Auflage stand; ein Titel ohne Buch bleibt,
 * wie er ist. Zusammengefasst wird nur hier, in der Anzeige: Titel-Verwaltung und
 * Zuordnen-Dialog lesen dieselbe Liste und brauchen jede Auflage einzeln.
 * @param {any[]} alle die ganze Katalogliste
 * @param {any[]} treffer das, was buecherSuchen davon übrig lässt
 */
export function buecherJeBuch(alle, treffer) {
	/** @type {Map<string, any[]>} */
	const auflagen = new Map();
	for (const b of alle) {
		if (b.werkId) auflagen.set(b.werkId, [...(auflagen.get(b.werkId) ?? []), b]);
	}
	/** @type {Map<string, any>} je Buch die getroffene Auflage mit dem kleinsten Rang */
	const vertreter = new Map();
	for (const b of treffer) {
		const bisher = b.werkId && vertreter.get(b.werkId);
		if (b.werkId && (!bisher || b.werkRang < bisher.werkRang)) vertreter.set(b.werkId, b);
	}
	const gezeigt = new Set();
	/** @type {any[]} */
	const karten = [];
	for (const b of treffer) {
		if (!b.werkId) karten.push(b);
		else if (!gezeigt.has(b.werkId)) {
			gezeigt.add(b.werkId);
			const liste = [...(auflagen.get(b.werkId) ?? [b])].sort((x, y) => x.werkRang - y.werkRang);
			const v = vertreter.get(b.werkId);
			karten.push(liste.length < 2 ? v : { ...v, buch: summeDesBuchs(liste) });
		}
	}
	return karten;
}

/**
 * Die Zahlen eines Buchs über seine Auflagen und die Liste für die Aufschlüsselung — in der
 * Form, die auflagenAufschluesselung auch im Bestellbedarf bekommt (gesamt_bestand).
 * @param {any[]} liste die Auflagen, die neueste zuerst
 */
function summeDesBuchs(liste) {
	const summe = (/** @type {string} */ feld) => liste.reduce((s, a) => s + (a[feld] ?? 0), 0);
	return {
		gesamt: summe('gesamt'),
		verfuegbar: summe('verfuegbar'),
		imZulauf: summe('imZulauf'),
		auflagen: liste.map((a) => ({
			id: a.id,
			auflage: a.auflage,
			erscheinungsjahr: a.erscheinungsjahr,
			gesamt_bestand: a.gesamt ?? 0
		}))
	};
}
