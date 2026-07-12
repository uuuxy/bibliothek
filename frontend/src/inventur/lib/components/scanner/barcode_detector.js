export async function createBarcodeDetector() {
	if ('BarcodeDetector' in window) {
		const BarcodeDetector = /** @type {any} */ (window).BarcodeDetector;
		const formats = await BarcodeDetector.getSupportedFormats();
		const neededFormats = ['ean_13', 'ean_8', 'upc_a', 'upc_e', 'code_128'];
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
				Html5QrcodeSupportedFormats.EAN_13,
				Html5QrcodeSupportedFormats.EAN_8,
				Html5QrcodeSupportedFormats.UPC_A,
				Html5QrcodeSupportedFormats.UPC_E,
				Html5QrcodeSupportedFormats.CODE_128
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
					const file = new File([blob], 'frame.jpg', { type: 'image/jpeg' });

					try {
						// Set showImage to false to avoid manipulating DOM unnecessarily
						const result = await scanner.scanFileV2(file, false);
						return [{ rawValue: result.decodedText }];
					} catch {
						return [];
					}
				}
			}
		};
	} catch {
		return null;
	}
}
