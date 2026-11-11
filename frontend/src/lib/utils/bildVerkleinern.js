// Bilder im Browser verkleinern, bevor sie hochgeladen oder in ein Design eingebettet werden:
// das Cover am Buch und die Bilder des Ausweis-Designs. Ein Foto vom Handy hat zwölf Megapixel
// und mehrere Megabyte; gezeigt und gedruckt wird ein Bruchteil davon.

/**
 * Die Maße, mit denen ein Bild in den Rahmen passt, ohne sein Seitenverhältnis zu ändern. Ein
 * Bild, das schon passt, bleibt, wie es ist: Vergrößern macht es nur unscharf.
 * @param {number} breite @param {number} hoehe @param {number} maxBreite @param {number} maxHoehe
 * @returns {{ breite: number, hoehe: number }}
 */
export function passendeMasse(breite, hoehe, maxBreite, maxHoehe) {
	if (breite <= maxBreite && hoehe <= maxHoehe) return { breite, hoehe };
	const faktor = Math.min(maxBreite / breite, maxHoehe / hoehe);
	return { breite: Math.round(breite * faktor), hoehe: Math.round(hoehe * faktor) };
}

/**
 * Zeichnet die Bilddatei in den Rahmen und liefert sie als WebP. Ein Browser ohne
 * WebP-Kodierer liefert nach der Regel von toBlob PNG; verkleinert ist das Bild auch dann.
 * @param {Blob} datei
 * @param {{ maxBreite: number, maxHoehe: number, guete: number }} rahmen
 * @returns {Promise<Blob>}
 */
export function verkleinereBild(datei, { maxBreite, maxHoehe, guete }) {
	return new Promise((resolve, reject) => {
		const bild = new Image();
		bild.onload = () => {
			URL.revokeObjectURL(bild.src);
			const { breite, hoehe } = passendeMasse(bild.width, bild.height, maxBreite, maxHoehe);
			const flaeche = document.createElement('canvas');
			flaeche.width = breite;
			flaeche.height = hoehe;
			flaeche.getContext('2d')?.drawImage(bild, 0, 0, breite, hoehe);
			flaeche.toBlob(
				(blob) => (blob ? resolve(blob) : reject(new Error('Compression failed'))),
				'image/webp',
				guete
			);
		};
		bild.onerror = () => {
			URL.revokeObjectURL(bild.src);
			reject(new Error('Invalid image'));
		};
		bild.src = URL.createObjectURL(datei);
	});
}
