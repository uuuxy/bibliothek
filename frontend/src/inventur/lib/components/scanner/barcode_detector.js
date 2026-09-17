/**
 * Die Strichcode-Arten, die an dieser Schule vorkommen — EINE Liste für beide Kameras und
 * für das Gate, das sie gegen den Druck hält (e2e/barcode-lesbar.spec.js).
 *
 * `code_128` ist, was die Anwendung selbst druckt (Ausweise, Buchetiketten, Mahnbriefe).
 * `code_39` stand bis zum 17.09.2026 auf unseren eigenen Etiketten und bleibt lesbar —
 * gedruckte Karten aus dieser Zeit sollen weiter funktionieren. Die EAN/UPC-Arten sind
 * die Littera-Altetiketten und die aufgedruckten Verlags-Barcodes. `qr_code` steht drauf,
 * weil Ausweis-Designer und Etikettendruck den QR als Alternative anbieten — die Theke
 * konnte ihn immer lesen, und das soll so bleiben.
 *
 * Die Liste kannte bis zum 17.09.2026 `code_39` NICHT — also genau die Art, die wir selbst
 * druckten. Die Kamera im Inventur-Bereich konnte einen Schülerausweis deshalb nie lesen,
 * und weil ein nicht erkannter Code keine Meldung erzeugt, sah es aus wie „nichts passiert".
 */
export const GELESENE_FORMATE = [
	'code_128',
	'code_39',
	'qr_code',
	'ean_13',
	'ean_8',
	'upc_a',
	'upc_e'
];

/**
 * Unterscheidet „in diesem Bild ist kein Code" von „die Erkennung ist kaputt".
 *
 * Die Rückfall-Bibliothek meldet beides als Fehler; nur der Wortlaut trennt sie. Der
 * Normalfall darf nicht gemeldet werden (er tritt bei jedem Bild ohne Code ein), der
 * echte Fehler muss es.
 *
 * @param {unknown} fehler
 * @returns {boolean}
 */
export function istNichtsGefunden(fehler) {
	const text = String(
		fehler instanceof Error ? fehler.message : typeof fehler === 'string' ? fehler : (fehler ?? '')
	);
	return (
		text.includes('NotFound') ||
		text.includes('able to detect') ||
		text.includes('No MultiFormat Readers')
	);
}

export async function createBarcodeDetector() {
	if ('BarcodeDetector' in window) {
		const BarcodeDetector = /** @type {any} */ (window).BarcodeDetector;
		const formats = await BarcodeDetector.getSupportedFormats();
		const neededFormats = GELESENE_FORMATE;
		const supportedFormats = neededFormats.filter((/** @type {string} */ format) =>
			formats.includes(format)
		);
		if (supportedFormats.length > 0) {
			return {
				name: 'native',
				detector: new BarcodeDetector({ formats: supportedFormats })
			};
		}
	}

	try {
		const { Html5Qrcode, Html5QrcodeSupportedFormats } = await import('html5-qrcode');

		// Initialize fallback scanner once to prevent performance issues
		let fallbackNode = document.getElementById('zxing-fallback-node');
		if (!fallbackNode) {
			fallbackNode = document.createElement('div');
			fallbackNode.id = 'zxing-fallback-node';
			fallbackNode.style.cssText = 'position:absolute;top:-9999px;left:-9999px;';
			document.body.appendChild(fallbackNode);
		}

		const scanner = new Html5Qrcode('zxing-fallback-node', {
			formatsToSupport: [
				Html5QrcodeSupportedFormats.CODE_128,
				Html5QrcodeSupportedFormats.CODE_39,
				Html5QrcodeSupportedFormats.QR_CODE,
				Html5QrcodeSupportedFormats.EAN_13,
				Html5QrcodeSupportedFormats.EAN_8,
				Html5QrcodeSupportedFormats.UPC_A,
				Html5QrcodeSupportedFormats.UPC_E
			],
			verbose: false
		});

		return {
			name: 'zxing-fallback',
			detector: {
				/** @param {HTMLVideoElement | HTMLImageElement} source */
				async detect(source) {
					const canvas = document.createElement('canvas');
					canvas.width = /** @type {any} */ (source).videoWidth || source.width;
					canvas.height = /** @type {any} */ (source).videoHeight || source.height;
					const context = canvas.getContext('2d');
					if (context) context.drawImage(source, 0, 0, canvas.width, canvas.height);

					const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.85));
					if (!blob) throw new Error('Der Browser liefert kein Bild aus der Kamera (toBlob).');
					const file = new File([blob], 'frame.jpg', { type: 'image/jpeg' });

					try {
						// showImage=false: kein Bild in den versteckten Knoten malen.
						const result = await scanner.scanFileV2(file, false);
						return [{ rawValue: result.decodedText }];
					} catch (fehler) {
						// „Kein Code in diesem Bild" ist der Normalfall, zehnmal in der Sekunde —
						// alles andere ist ein Grund, warum die Erkennung NICHT arbeitet, und der
						// gehoert nach oben. Bis zum 17.09.2026 fiel beides in dasselbe leere
						// catch: Ein Browser, der `File` nicht kennt oder kein Bild liefert, sah
						// danach genauso aus wie eine Kamera, die nur nichts vor der Linse hat.
						if (istNichtsGefunden(fehler)) return [];
						throw fehler instanceof Error ? fehler : new Error(String(fehler));
					}
				}
			}
		};
	} catch {
		return null;
	}
}
