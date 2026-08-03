// Gate gegen die abgeschnittene Bestellung.
//
// Aus dem Betrieb gemeldet: „Bestellung abgeschnitten." Die rechte Spalte (Hinweis auf
// offene Etiketten + Warenkorb) klebt beim Scrollen fest. Füllt sich der Korb, wächst sie
// über den Fensterrand hinaus — bei 1366×700 gemessen 1288 px hoch, 826 px davon unter
// dem sichtbaren Bereich.
//
// Sie war nicht unerreichbar: Wer die Bestellbedarfs-Liste daneben (105 Titel) weit genug
// nach unten scrollt, schiebt den klebenden Rahmen irgendwann hoch. Genau daran ist die
// erste Fassung dieses Tests gescheitert — scrollIntoViewIfNeeded fand den Knopf und war
// auch OHNE die Korrektur grün.
//
// Geprüft wird deshalb das, was im Alltag zählt: Der Absenden-Knopf muss erreichbar sein,
// OHNE die Seite zu verlassen, an der man gerade arbeitet. Der Warenkorb bringt dafür
// seinen eigenen Scrollbalken mit; die Seite darunter bleibt stehen.
import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

/** Schul-PCs sind kurz. Die erste Größe ist der gemeldete Fall. */
const FENSTER = [
	[1366, 700],
	[1280, 800],
	[1920, 1080]
];

for (const [breite, hoehe] of FENSTER) {
	test(`Warenkorb bleibt bedienbar bei ${breite}×${hoehe}`, async ({ page }) => {
		test.setTimeout(120_000);
		await page.setViewportSize({ width: breite, height: hoehe });
		await uiLogin(page);
		await page.goto('/bestellungen');

		// Genug Positionen, damit die Spalte über den Fensterrand wächst.
		//
		// Bewusst OHNE if-visible-Wächter: Die erste Fassung übersprang das Befüllen still,
		// wenn die Trefferliste noch nicht stand — der Test lief dann mit LEEREM Warenkorb
		// grün und maß nichts. Hier wird auf jeden Schritt gewartet und am Ende belegt,
		// dass wirklich Positionen im Korb liegen.
		// Das Plus in der Bedarfsliste legt DIREKT in den Korb (OrderRecommendations →
		// addToCart). Der Zwischendialog „In den Warenkorb" gehört zur Titelsuche und
		// erscheint hier nicht — die erste Fassung wartete darauf und lief in den Timeout.
		const ZIEL_POSITIONEN = 6;
		const knopf = page.getByRole('button', { name: /Bestellung auslösen/ });
		for (let i = 0; i < ZIEL_POSITIONEN; i++) {
			const plus = page.getByRole('button', { name: /zur Bestellung hinzufügen/ }).nth(i);
			await plus.waitFor({ timeout: 15_000 });
			await plus.click();
			// Auf die Wirkung warten statt auf eine feste Zeit: Der Korb muss die Position
			// übernommen haben, bevor die nächste dazukommt.
			await expect(knopf).toHaveText(new RegExp(`${i + 1}\\s*Expl`), { timeout: 10_000 });
		}

		await expect(knopf).toBeVisible();
		// Beweist, dass der Korb gefüllt ist — sonst wäre die Spalte gar nicht zu hoch,
		// und der Test prüfte den Fall nicht, für den er geschrieben wurde.
		await expect(
			knopf,
			'Der Warenkorb muss die erwarteten Positionen tragen — sonst misst der Test den falschen Fall'
		).toHaveText(new RegExp(`${ZIEL_POSITIONEN}\\s*Expl`), { timeout: 10_000 });

		// Alle Scroll-Container auf Anfang — die Arbeitsfläche steht oben, wie nach dem
		// Öffnen der Ansicht.
		await page.evaluate(() => {
			window.scrollTo(0, 0);
			for (const el of document.querySelectorAll('*')) {
				if (el.scrollTop > 0) el.scrollTop = 0;
			}
		});
		// Die Spalte rechnet ihre Höhe aus dem Abstand zum Fensterrand. Nach dem
		// Zurücksetzen der Scrollposition braucht dieser Handler einen Tick — deshalb
		// wartend prüfen statt einmalig messen.
		await expect
			.poll(
				async () =>
					page.evaluate(() => {
						const r = document.querySelector('[class*="lg:sticky"]').getBoundingClientRect();
						return Math.round(r.bottom);
					}),
				{
					timeout: 5000,
					intervals: [50, 100, 200, 400],
					message:
						`Die Bestellspalte reicht unter den ${hoehe}-px-Fensterrand und ist damit ` +
						`„abgeschnitten". Sie braucht eine Höhengrenze mit eigenem Scrollbalken, die den ` +
						`tatsächlichen Abstand zum oberen Rand berücksichtigt (siehe railMaxHeight in ` +
						`BestellWorkspace.svelte).`
				}
			)
			.toBeLessThanOrEqual(hoehe)

		// Gemessen wird die Geometrie der RAIL-Spalte selbst, nicht ein Scroll-Weg.
		//
		// Die erste Fassung verglich Scroll-Positionen vor und nach scrollIntoViewIfNeeded.
		// Das war doppelt untauglich: Der Messpunkt griff per '.overflow-y-auto' das
		// falsche Element ab (es gibt mehrere), und scrollIntoViewIfNeeded findet den Knopf
		// ohnehin — notfalls, indem es die ganze Bedarfsliste wegscrollt. Der Test war
		// deshalb auch ohne die Korrektur grün, während der Knopf nachweislich bei y=1279
		// in einem 700-px-Fenster lag.
		const mass = await page.evaluate(() => {
			const rail = document.querySelector('[class*="lg:sticky"]');
			const r = rail.getBoundingClientRect();
			const b = [...document.querySelectorAll('button')].find((x) =>
				/Bestellung auslösen/.test(x.textContent || '')
			);
			return {
				railOben: Math.round(r.top),
				railUnten: Math.round(r.bottom),
				railInhalt: Math.round(rail.scrollHeight),
				railScrollbar: rail.scrollHeight > rail.clientHeight + 1,
				knopfY: Math.round(b.getBoundingClientRect().y),
				fenster: window.innerHeight
			};
		});

		// Die Bestellspalte muss ins Fenster passen. Tut sie das nicht, ist sie „abgeschnitten"
		// — genau die Meldung aus dem Betrieb.
		expect(
			mass.railUnten,
			`Die Bestellspalte reicht bis y=${mass.railUnten} und damit ${mass.railUnten - hoehe} px ` +
				`unter den ${hoehe}-px-Fensterrand (Inhalt ${mass.railInhalt} px). Sie braucht eine ` +
				`Höhengrenze mit eigenem Scrollbalken (max-h + overflow-y-auto).`
		).toBeLessThanOrEqual(hoehe);

		// Und der Absenden-Knopf muss sich innerhalb der Spalte erreichen lassen, ohne die
		// Arbeitsfläche daneben zu verlassen.
		await knopf.scrollIntoViewIfNeeded({ timeout: 5000 });
		const box = await knopf.boundingBox();
		expect(box, 'Absenden-Knopf muss eine Position haben').not.toBeNull();
		expect(
			box.y >= 0 && box.y + box.height <= hoehe,
			`Absenden-Knopf liegt bei y=${Math.round(box?.y ?? -1)} ausserhalb des ${hoehe}-px-Fensters`
		).toBe(true);
	});
}
