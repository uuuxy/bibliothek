import { apiFetch } from '../../lib/apiFetch.js';
import { isbnFormen, normalisiereIsbn } from '../../lib/utils/isbnFormen.js';
/**
 * Laden, Suchen und Zusammenfassen je Buch für den Medienkatalog (Suche & Filter).
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
 * Trifft ein Buch den Jahrgang? Über die Spanne von–bis; ohne Spanne (0) trifft sie keinen.
 * @param {any} b
 * @param {number} jahrgang
 */
function trifftJahrgang(b, jahrgang) {
	const von = Number(b.jahrgangVon);
	const bis = Number(b.jahrgangBis);
	if (!von || !bis) return false;
	return jahrgang >= von && jahrgang <= bis;
}

/**
 * Die Buch-Suche des Medienkatalogs: Jeder Begriff muss mindestens ein Feld treffen. Eine
 * Zahl zählt als Jahrgang (die Spanne von–bis), Füllwörter wie
 * „Klasse"/„Jg." fallen dann weg.
 * @param {any[]} buecherArray
 * @param {string} searchQuery
 */
export function buecherSuchen(buecherArray, searchQuery) {
	const q = searchQuery.toLowerCase().trim();
	if (q === '') return Array.isArray(buecherArray) ? buecherArray : [];

	let terms = q.split(/\s+/).map((term) => suchSynonyme.get(term) || term);
	const hasNumber = terms.some((t) => !Number.isNaN(Number.parseInt(t, 10)));
	if (hasNumber) {
		terms = terms.filter((t) => !['klasse', 'kl', 'kl.', 'jahrgang', 'jg', 'jg.'].includes(t));
	}

	// Je Begriff einmal vorab die Schreibweisen derselben ISBN: Ein gescannter Strichcode
	// findet so auch den Bestand mit Bindestrichen oder zehnstelliger Alt-ISBN.
	const isbnJeTerm = terms.map((term) => isbnFormen(term));
	// Ein oder zwei Ziffern sind ein Jahrgang und kein Stück einer ISBN: „7" steht in jeder
	// ISBN-13 (978…) und träfe sonst fast jedes Buch.
	const istJahrgang = terms.map((term) => /^\d{1,2}\.?$/.test(term));

	return (Array.isArray(buecherArray) ? buecherArray : []).filter((/** @type {any} */ b) =>
		terms.every((term, i) => trifftBegriff(b, term, isbnJeTerm[i], istJahrgang[i]))
	);
}

/**
 * Trifft der Begriff das Buch in mindestens einem Feld?
 * @param {any} b
 * @param {string} term
 * @param {string[]} isbnFormenDesBegriffs die Schreibweisen des Begriffs als ISBN; leer, wenn er keine ist
 * @param {boolean} istJahrgang der Begriff ist ein Jahrgang und kein Stück einer ISBN
 */
function trifftBegriff(b, term, isbnFormenDesBegriffs, istJahrgang) {
	if (b.title?.toLowerCase().includes(term)) return true;
	if (!istJahrgang && b.isbn?.toLowerCase().includes(term)) return true;
	if (
		b.isbn &&
		isbnFormenDesBegriffs.length > 0 &&
		isbnFormenDesBegriffs.includes(normalisiereIsbn(b.isbn))
	)
		return true;
	if (b.author?.toLowerCase().includes(term)) return true;
	if (b.subject?.toLowerCase().includes(term)) return true;
	if (b.istLernmittel && 'lernmittel'.includes(term)) return true;
	// Die Signatur ist die Regaladresse (Handbuch).
	if (b.signatur?.toLowerCase().includes(term)) return true;
	// Schlagworte und die Verweise darauf (docs/OFFEN.md 4.20). Welche Wörter einen
	// Titel finden, entscheidet der Server (repository.SuchwoerterDerTitel) — dieselbe
	// Regel, nach der die Titel-Verwaltung und das Portal am Server suchen.
	if (
		Array.isArray(b.suchwoerter) &&
		b.suchwoerter.some((/** @type {string} */ w) => w.toLowerCase().includes(term))
	)
		return true;
	const num = Number.parseInt(term, 10);
	return !Number.isNaN(num) && trifftJahrgang(b, num);
}

/**
 * Ein Buch in mehreren Auflagen ist eine Kachel: oben die neueste der getroffenen Auflagen
 * (kleinster werkRang), gezählt über alle Auflagen der ganzen Liste — wer eine alte Auflage
 * scannt, fragt, ob die Schule das Buch hat. Zusammengefasst wird nur in der Anzeige;
 * Titel-Verwaltung und Zuordnen-Dialog lesen dieselbe Liste und brauchen jede Auflage einzeln.
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
