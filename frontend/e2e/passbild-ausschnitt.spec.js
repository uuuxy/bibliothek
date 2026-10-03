import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, oeffneSchuelerProfil, uniqueSuffix } from './helpers.js';

// Der Sucher der Passbild-Aufnahme zeigt das ganze Kamerabild und lässt den Ausschnitt hell, der
// gespeichert wird. Den hellen Bereich legt das Stylesheet fest (Form 3:4 über die volle Höhe),
// den gespeicherten berechnet passbildAusschnitt.js: zwei Wege zur selben Fläche. Dieser Test
// hält sie zusammen, an drei Formen der Kamera.
//
// Das Kamerabild ist eine Zeichenfläche, deren Farbe den Ort nennt: Rot die Spalte, Grün die
// Zeile. Am hochgeladenen Bild steht damit, aus welchem Teil der Kamera es stammt.

/**
 * @param {import('@playwright/test').Page} page
 * @param {number} breite
 * @param {number} hoehe
 */
async function kameraMitOrt(page, breite, hoehe) {
	await page.addInitScript(
		([b, h]) => {
			const flaeche = document.createElement('canvas');
			flaeche.width = b;
			flaeche.height = h;
			const ctx = /** @type {CanvasRenderingContext2D} */ (flaeche.getContext('2d'));
			const bild = ctx.createImageData(b, h);
			for (let y = 0; y < h; y++) {
				for (let x = 0; x < b; x++) {
					const i = (y * b + x) * 4;
					bild.data[i] = Math.round((x / (b - 1)) * 255);
					bild.data[i + 1] = Math.round((y / (h - 1)) * 255);
					bild.data[i + 2] = 128;
					bild.data[i + 3] = 255;
				}
			}
			// Der Strom liefert nur bei einer Änderung ein neues Bild: ein Bildpunkt wechselt.
			let n = 0;
			setInterval(() => {
				ctx.putImageData(bild, 0, 0);
				ctx.fillStyle = n++ % 2 ? 'rgb(0,0,128)' : 'rgb(1,0,128)';
				ctx.fillRect(0, 0, 1, 1);
			}, 100);
			Object.defineProperty(navigator, 'mediaDevices', {
				configurable: true,
				value: { getUserMedia: async () => flaeche.captureStream(10) }
			});
		},
		[breite, hoehe]
	);
}

/**
 * Maße und Randfarben des hochgeladenen Bilds, gelesen auf einer leeren Seite.
 * @param {import('@playwright/test').Page} page
 * @param {string} daten
 */
async function leseBild(page, daten) {
	const leer = await page.context().newPage();
	try {
		return await leer.evaluate(async (url) => {
			const bild = new Image();
			bild.src = url;
			await bild.decode();
			const flaeche = document.createElement('canvas');
			flaeche.width = bild.naturalWidth;
			flaeche.height = bild.naturalHeight;
			const ctx = /** @type {CanvasRenderingContext2D} */ (flaeche.getContext('2d'));
			ctx.drawImage(bild, 0, 0);
			const punkt = (/** @type {number} */ x, /** @type {number} */ y) => [
				...ctx.getImageData(x, y, 1, 1).data
			];
			const mx = Math.floor(flaeche.width / 2);
			const my = Math.floor(flaeche.height / 2);
			return {
				breite: flaeche.width,
				hoehe: flaeche.height,
				links: punkt(0, my)[0],
				rechts: punkt(flaeche.width - 1, my)[0],
				oben: punkt(mx, 0)[1],
				unten: punkt(mx, flaeche.height - 1)[1]
			};
		}, daten);
	} finally {
		await leer.close();
	}
}

for (const [name, breite, hoehe] of /** @type {[string, number, number][]} */ ([
	['Breitbild 16:9', 1280, 720],
	['4:3', 640, 480],
	['hochkant 9:16', 720, 1280]
])) {
	test(`Passbild: gespeichert wird der helle Ausschnitt des Suchers (Kamera ${name})`, async ({
		page
	}) => {
		const s = uniqueSuffix();
		const vorname = `Sucher${s}`;
		seedSQL(`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		         VALUES ('E2E-SUC-${s}', '${vorname}', 'Probe', '07A', 2030);`);
		try {
			let hochgeladen = '';
			await page.route('**/api/schueler/*/photo', async (route) => {
				if (route.request().method() !== 'POST') return route.continue();
				hochgeladen = route.request().postDataJSON().photo_data;
				await route.fulfill({ status: 200, contentType: 'application/json', body: '{"url":""}' });
			});
			await kameraMitOrt(page, breite, hoehe);
			await uiLogin(page);
			await oeffneSchuelerProfil(page, vorname);
			await page.getByRole('button', { name: 'Passbild mit Webcam aufnehmen' }).click();
			const dialog = page.getByRole('dialog', { name: 'Passbild aufnehmen' });
			const ausschnitt = dialog.getByTestId('passbild-ausschnitt');
			await expect(ausschnitt).toBeVisible();

			// Der helle Bereich in Bildpunkten der Kamera. Der Sucher hat die Form des Kamerabilds,
			// ein Maßstab gilt deshalb für beide Richtungen.
			const hell = async () =>
				dialog.locator('video').evaluate((video) => {
					const v = /** @type {HTMLVideoElement} */ (video);
					const flaeche = v.getBoundingClientRect();
					const a = /** @type {Element} */ (
						document.querySelector('[data-testid="passbild-ausschnitt"]')
					).getBoundingClientRect();
					const f = v.videoWidth / flaeche.width;
					return {
						kamera: `${v.videoWidth}x${v.videoHeight}`,
						sucherForm: flaeche.width / flaeche.height,
						x: (a.left - flaeche.left) * f,
						y: (a.top - flaeche.top) * f,
						breite: a.width * f,
						hoehe: a.height * f
					};
				});
			await expect
				.poll(async () => (await hell()).kamera, { message: 'die Kamera liefert ihr Bild' })
				.toBe(`${breite}x${hoehe}`);
			await expect
				.poll(async () => (await hell()).sucherForm, {
					message: 'der Sucher hat die Form des Kamerabilds'
				})
				.toBeCloseTo(breite / hoehe, 1);
			const sucher = await hell();
			expect(sucher.breite / sucher.hoehe, 'der helle Bereich hat die Form 3:4').toBeCloseTo(
				0.75,
				2
			);

			await dialog.getByRole('button', { name: 'Foto aufnehmen' }).click();
			await expect.poll(() => hochgeladen.length, 'der Upload wird gesehen').toBeGreaterThan(0);
			const bild = await leseBild(page, hochgeladen);

			// Größe: genau der helle Bereich, auf einen Bildpunkt.
			expect(Math.abs(bild.breite - sucher.breite), 'Breite wie der helle Bereich').toBeLessThan(2);
			expect(Math.abs(bild.hoehe - sucher.hoehe), 'Höhe wie der helle Bereich').toBeLessThan(2);

			// Lage: Die Randfarben nennen Spalte und Zeile der Kamera. Eine Farbstufe sind
			// breite / 255 Bildpunkte; drei Stufen fängt die Rundung der Bildkompression ab.
			const spalte = (/** @type {number} */ rot) => (rot / 255) * (breite - 1);
			const zeile = (/** @type {number} */ gruen) => (gruen / 255) * (hoehe - 1);
			const tolX = (3 * breite) / 255 + 2;
			const tolY = (3 * hoehe) / 255 + 2;
			expect(Math.abs(spalte(bild.links) - sucher.x), 'linker Rand').toBeLessThan(tolX);
			expect(
				Math.abs(spalte(bild.rechts) - (sucher.x + sucher.breite)),
				'rechter Rand'
			).toBeLessThan(tolX);
			expect(Math.abs(zeile(bild.oben) - sucher.y), 'oberer Rand').toBeLessThan(tolY);
			expect(Math.abs(zeile(bild.unten) - (sucher.y + sucher.hoehe)), 'unterer Rand').toBeLessThan(
				tolY
			);
		} finally {
			seedSQL(`DELETE FROM schueler WHERE barcode_id = 'E2E-SUC-${s}';`);
		}
	});
}
