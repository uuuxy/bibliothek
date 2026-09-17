/**
 * Die eine Adresse, unter der ein Strichcode-Bild geholt wird.
 *
 * Der Server liefert `/api/barcode` mit `Cache-Control: max-age=31536000` — ein Jahr.
 * Das ist richtig, solange dieselbe Adresse dasselbe Bild bedeutet: Ein Ausweisbogen mit
 * 30 Karten holt sonst 30-mal dasselbe PNG.
 *
 * Genau diese Annahme brach am 17.09.2026. Der Drucker stellte von Code 39 (mit
 * Prüfzeichen) auf Code 128 um, die Adresse blieb gleich — und jedes Gerät, das die Karte
 * vorher einmal angezeigt hatte, zeigte weiter den ALTEN Strichcode. Auf dem Handy stand
 * unter der Karte „S-10001", gescannt wurde „S-10001N", und der Server suchte eine Nummer,
 * die es nicht gibt. Der Code war da schon repariert; im Browser lag noch das Bild von
 * gestern.
 *
 * Deshalb trägt die Adresse die Fassung der Kodierung. Wer am Erzeuger etwas ändert
 * (api/barcode_generate.go), zählt FASSUNG hoch — dann ist es für jeden Browser ein neues
 * Bild, ohne dass jemand einen Cache leeren muss.
 */
export const FASSUNG = 'c128';

/**
 * @param {string | undefined | null} inhalt die Nummer, die im Strichcode stehen soll
 * @param {{ qr?: boolean, width?: number, height?: number }} [optionen]
 * @returns {string}
 */
export function strichcodeBildUrl(inhalt, optionen = {}) {
	const { qr = false, width, height } = optionen;
	const teile = [`content=${encodeURIComponent(inhalt ?? '')}`, `qr=${qr ? 'true' : 'false'}`];
	if (width) teile.push(`width=${width}`);
	if (height) teile.push(`height=${height}`);
	teile.push(`v=${FASSUNG}`);
	return `/api/barcode?${teile.join('&')}`;
}
