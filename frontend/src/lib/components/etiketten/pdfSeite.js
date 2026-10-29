// Die erste Seite eines PDFs als Bild, gezeichnet von pdf.js (Mozilla, die PDF-Anzeige von
// Firefox). So zeigt die Vorschau des Druck-Centers das Druck-PDF selbst statt einer zweiten
// Zeichnung im Browser. Das Paket wird erst geladen, wenn eine Vorschau gebraucht wird.

import workerAdresse from 'pdfjs-dist/build/pdf.worker.min.mjs?url';

/**
 * Zeichnet die erste Seite in der gewünschten Breite, scharf auch bei doppelter Pixeldichte.
 * @param {ArrayBuffer} daten das PDF
 * @param {number} breite Breite des Bilds in CSS-Pixeln
 * @returns {Promise<Blob>} die Seite als PNG
 */
export async function ersteSeiteAlsBild(daten, breite) {
	const pdfjs = await import('pdfjs-dist');
	pdfjs.GlobalWorkerOptions.workerSrc = workerAdresse;
	// Die Seite kommt vom eigenen Server. pdf.js 6 übersetzt nichts mehr mit eval, den Weg der Lücke
	// CVE-2024-4367; die Sicherheitsregeln des Programms verböten es ohnehin.
	const aufgabe = pdfjs.getDocument({ data: daten });
	try {
		const seite = await (await aufgabe.promise).getPage(1);
		const massstab =
			(breite / seite.getViewport({ scale: 1 }).width) * (window.devicePixelRatio || 1);
		const ansicht = seite.getViewport({ scale: massstab });
		const leinwand = document.createElement('canvas');
		leinwand.width = Math.round(ansicht.width);
		leinwand.height = Math.round(ansicht.height);
		await seite.render({ canvas: leinwand, viewport: ansicht }).promise;
		return await new Promise((fertig, gescheitert) =>
			leinwand.toBlob((bild) =>
				bild ? fertig(bild) : gescheitert(new Error('Vorschau nicht gezeichnet'))
			)
		);
	} finally {
		await aufgabe.destroy();
	}
}
