import { fehlertext } from '../../../../lib/utils/fehlertext.js';

/**
 * Die Strichcode-Arten, die an dieser Schule vorkommen: eine Liste für beide Kameras und für
 * das Gate, das sie gegen den Druck hält (e2e/barcode-lesbar.spec.js).
 *
 * `code_128` ist, was die Anwendung selbst druckt (Ausweise, Buchetiketten, Mahnbriefe).
 * `code_39` steht auf älteren eigenen Etiketten und Ausweisen, die weiter funktionieren
 * sollen. Die EAN/UPC-Arten sind die Littera-Altetiketten und die aufgedruckten
 * Verlags-Barcodes. `qr_code` steht drauf, weil Ausweis-Designer und Etikettendruck den QR
 * als Alternative anbieten.
 *
 * Fehlt eine Art, meldet die Kamera nichts: Es sieht dann aus, als läge kein Code vor der
 * Linse.
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
	const text = fehlertext(fehler);
	return (
		text.includes('NotFound') ||
		text.includes('able to detect') ||
		text.includes('No MultiFormat Readers')
	);
}

export async function erzeugeBarcodeErkenner() {
	if ('BarcodeDetector' in window) {
		const BarcodeErkenner = /** @type {any} */ (window).BarcodeDetector;
		const formats = await BarcodeErkenner.getSupportedFormats();
		const neededFormats = GELESENE_FORMATE;
		const supportedFormats = neededFormats.filter((/** @type {string} */ format) =>
			formats.includes(format)
		);
		if (supportedFormats.length > 0) {
			return {
				name: 'native',
				detector: new BarcodeErkenner({ formats: supportedFormats })
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
						const ergebnis = await scanner.scanFileV2(file, false);
						return [{ rawValue: ergebnis.decodedText }];
					} catch (err) {
						// „Kein Code in diesem Bild" ist der Normalfall, zehnmal in der Sekunde;
						// alles andere ist ein Grund, warum die Erkennung nicht arbeitet, und gehört
						// nach oben. Ein Browser, der `File` nicht kennt oder kein Bild liefert, sähe
						// sonst aus wie eine Kamera, die nur nichts vor der Linse hat.
						if (istNichtsGefunden(err)) return [];
						throw err instanceof Error ? err : new Error(fehlertext(err));
					}
				}
			}
		};
	} catch {
		return null;
	}
}
