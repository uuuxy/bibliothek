import { NUMMERNZEILE_MM } from '../utils/strichcodeBild.js';

/**
 * Der sichere Rand des Ausweises. Die Kartendrucker der Schule (Datacard Sigma DS1 und DS2)
 * drucken bis an den Rand, lassen dort aber einen schmalen weißen Streifen frei, weil der
 * Druckkopf am Rand angehoben und abgesenkt wird (Datacard, Installation and Administrator's
 * Guide, „Margins at the Edge of a Card"). Was darin steht, fehlt auf der Karte.
 */

/** Abstand zum Kartenrand in mm für Text, Bilder und Strichcode; der Designer zeigt ihn gestrichelt. */
export const SICHERHEITSRAND_MM = 3;

/** Abstand der Striche eines Strichcodes zum Kartenrand in mm, nach Datacard („Bar Code Placement"). */
export const STRICHCODE_RAND_MM = 6.3;

/** Die Karte nach ISO 7810 ID-1 in mm, wie Designer und Kartendruck sie zeichnen. */
export const KARTE_BREITE_MM = 85.6;
export const KARTE_HOEHE_MM = 53.98;

/**
 * So nah muss eine Kante beim Ziehen ihrer Linie kommen, damit sie einrastet, in mm. Fest in
 * Millimetern statt in Pixeln: So bleibt bei jedem Zoom der Streifen bis zum Kartenrand erreichbar.
 */
export const FANGWEITE_MM = 1;

/**
 * Ob ein Element mit Absicht bis an den Kartenrand reicht: Farbflächen und Motive über die ganze
 * Breite. Für sie gilt kein sicherer Rand.
 * @param {{ type: string, width: number }} el
 */
export function reichtBisAnDenRand(el) {
	return el.type === 'box' || (el.type === 'image' && el.width >= KARTE_BREITE_MM);
}

/**
 * Wie weit jede Kante des Elements vom Kartenrand bleiben muss, in mm; null, wenn es bis an den
 * Rand reichen darf. Beim Strichcode zählen die Striche: Unter ihnen steht die Nummernzeile, sie
 * enden um deren Höhe über der Unterkante des Elements.
 * @param {{ type: string, width: number }} el
 * @returns {{ links: number, oben: number, rechts: number, unten: number } | null}
 */
export function sichererAbstand(el) {
	if (reichtBisAnDenRand(el)) return null;
	if (el.type !== 'barcode') {
		const r = SICHERHEITSRAND_MM;
		return { links: r, oben: r, rechts: r, unten: r };
	}
	const s = STRICHCODE_RAND_MM;
	return { links: s, oben: s, rechts: s, unten: Math.max(SICHERHEITSRAND_MM, s - NUMMERNZEILE_MM) };
}

// Auf hundertstel Millimeter: 85,6 − 3 − 55,6 ergibt in Gleitkomma 2,999…, nicht 3.
const hundertstel = (/** @type {number} */ wert) => Math.round(wert * 100) / 100;

/**
 * Ob das Element dem Kartenrand näher steht als sein sicherer Abstand; dann fehlt am Drucker
 * womöglich ein Teil davon.
 * @param {{ type: string, x: number, y: number, width: number, height: number }} el
 */
export function stehtImRand(el) {
	const soll = sichererAbstand(el);
	if (!soll) return false;
	return (
		hundertstel(el.x) < soll.links ||
		hundertstel(el.y) < soll.oben ||
		hundertstel(KARTE_BREITE_MM - el.x - el.width) < soll.rechts ||
		hundertstel(KARTE_HOEHE_MM - el.y - el.height) < soll.unten
	);
}

/**
 * Eine Achse: Kommt die vordere oder die hintere Kante ihrer Linie näher als die Fangweite, steht
 * sie genau auf ihr; sonst bleibt die Lage auf ein Zehntel Millimeter gerundet, wie beim freien Ziehen.
 * @param {number} lage
 * @param {number} groesse
 * @param {number} karte
 * @param {number} vorn  Abstand der Linie an der vorderen Kante (links oder oben)
 * @param {number} hinten  Abstand der Linie an der hinteren Kante (rechts oder unten)
 */
function rasteAchse(lage, groesse, karte, vorn, hinten) {
	if (Math.abs(lage - vorn) <= FANGWEITE_MM) return { lage: vorn, eingerastet: true };
	const amEnde = hundertstel(karte - hinten - groesse);
	if (Math.abs(lage - amEnde) <= FANGWEITE_MM) return { lage: amEnde, eingerastet: true };
	return { lage: Math.round(lage * 10) / 10, eingerastet: false };
}

/**
 * Die Lage eines gezogenen Elements mit Einrasten: Kommt eine Kante der Linie ihres sicheren
 * Abstands näher als FANGWEITE_MM, steht sie genau auf ihr; weiter gezogen, läuft das Element frei
 * über die Linie hinaus. Was bis an den Rand reichen darf, rastet nicht ein.
 * @param {{ type: string, x: number, y: number, width: number, height: number }} el  mit der Lage aus der Zeigerbewegung
 * @returns {{ x: number, y: number, eingerastet: boolean }}
 */
export function rasteEin(el) {
	const soll = sichererAbstand(el);
	if (!soll) {
		return { x: Math.round(el.x * 10) / 10, y: Math.round(el.y * 10) / 10, eingerastet: false };
	}
	const x = rasteAchse(el.x, el.width, KARTE_BREITE_MM, soll.links, soll.rechts);
	const y = rasteAchse(el.y, el.height, KARTE_HOEHE_MM, soll.oben, soll.unten);
	return { x: x.lage, y: y.lage, eingerastet: x.eingerastet || y.eingerastet };
}
