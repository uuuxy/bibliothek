// Ein Bild für die Ausweiskarte einlesen. Das Design liegt mit allen Bildern als ein Text am
// Server und geht nach jeder Änderung ganz dorthin; ein Foto vom Handy machte daraus mehrere
// Megabyte je Speichern. Deshalb wird ein großes Bild vor dem Einbetten verkleinert.

import { verkleinereBild } from '../utils/bildVerkleinern.js';

// Bis hierher bleibt die Datei, wie sie ist: Ein kleines Logo verliert so nichts.
const UNVERAENDERT_BIS_BYTES = 300_000;
// Die Karte ist 85,6 mm breit. 1200 Pixel über die ganze Breite sind rund 350 dpi; üblich sind
// bei Kartendruckern 300.
const KANTE_PIXEL = 1200;
// Was auch verkleinert darüber liegt, nimmt der Designer nicht an. Der Server lässt für das
// ganze Design 5 MiB zu (api/druck, MaxAusweisLayoutBytes).
const HOECHSTENS_BYTES = 1_000_000;

/** @param {Blob} blob @returns {Promise<string>} */
function alsDatenAdresse(blob) {
	return new Promise((resolve, reject) => {
		const leser = new FileReader();
		leser.onload = () => resolve(/** @type {string} */ (leser.result));
		leser.onerror = () => reject(new Error('Die Datei ließ sich nicht lesen.'));
		leser.readAsDataURL(blob);
	});
}

/** @param {number} bytes */
const megabyte = (bytes) =>
	(bytes / 1_000_000).toLocaleString('de-DE', { maximumFractionDigits: 1 });

/**
 * Liest eine Bilddatei als Data-URL für ein Element der Karte. Der Fehler nennt die Datei,
 * damit bei mehreren Bildern klar ist, welches nicht ankam.
 * @param {File} datei
 * @returns {Promise<string>}
 */
export async function leseKartenBild(datei) {
	if (datei.size <= UNVERAENDERT_BIS_BYTES) return alsDatenAdresse(datei);
	/** @type {Blob} */
	let bild = datei;
	// Eine Vektorgrafik bleibt Vektor; sie durch ein Pixelbild zu ersetzen, machte sie unscharf.
	if (datei.type !== 'image/svg+xml') {
		try {
			bild = await verkleinereBild(datei, {
				maxBreite: KANTE_PIXEL,
				maxHoehe: KANTE_PIXEL,
				guete: 0.9
			});
		} catch {
			throw new Error(`„${datei.name}“ ist kein Bild, das der Browser lesen kann.`);
		}
	}
	if (bild.size > HOECHSTENS_BYTES) {
		throw new Error(
			`„${datei.name}“ ist auch verkleinert zu groß (${megabyte(bild.size)} MB). Bitte kleiner zuschneiden oder als JPEG speichern.`
		);
	}
	return alsDatenAdresse(bild);
}
