// Das Cover-Bild ist ein BILD, kein Knopf — und „Cover ändern" bleibt erreichbar.
//
// Bis zum 11.08.2026 lag über dem ganzen Cover ein <button> mit `absolute inset-0` und
// `opacity-0`. Unsichtbar heisst nicht wirkungslos: Er war per Tab erreichbar (der Fokus
// landete auf etwas, das man nicht sieht — WCAG 2.4.7), und auf einem Tablet öffnete ein
// Tipp auf das Bild ohne jede Ankündigung den Dateidialog.
//
// Gebraucht wurde er nie: Direkt unter dem Cover steht seit jeher „Cover ändern",
// sichtbar und beschriftet, mit derselben Funktion. Der Overlay war ein Duplikat, das nur
// die zwei Fallen beisteuerte.
//
// Dieser Bildschirm hatte bis dahin NULL Abdeckung — deshalb prüft der Test beide Seiten:
// dass die Falle weg ist UND dass die Fähigkeit geblieben ist. Nur das erste zu prüfen
// hiesse, den Rückbau gegen sich selbst zu testen.
import { test, expect } from '@playwright/test';
import { writeFileSync } from 'node:fs';
import { uiLogin, gehZu, seedSQL, uniqueSuffix, raeumtNachJedemFallAuf } from './helpers.js';

const angelegt = raeumtNachJedemFallAuf(test);

test('Cover: das Bild ist kein verstecktes Bedienelement, „Cover ändern" bleibt', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');

	// Derselbe Weg, den eine Bibliothekskraft nimmt: Titel auf der Buchkarte → Buchakte →
	// „Titel bearbeiten". Zwei Klicks, kein Abkürzen über den Store — sonst prüfte der
	// Test einen Zustand, den man von Hand gar nicht erreicht.
	//
	// Bis zum 09.09.2026 stand hier getByRole('button', { name: 'Buch schnell bearbeiten' })
	// .first() — und traf NICHT den Stift, sondern die Kachel: Sie trug role="button", ihr
	// berechneter Name enthielt den Stift-Text als Teil, und sie kam im DOM zuerst. Der
	// Test öffnete die Akte also über die Fläche und hielt das für den Stift (der die
	// Titel-Verwaltung öffnet, nicht die Akte). Seit die Kachel kein Knopf mehr ist
	// (nested-interactive), ist der Titel der Knopf — und der Weg ist der, den der
	// Kommentar oben schon immer meinte.
	await page.getByRole('heading', { level: 2 }).first().getByRole('button').click();
	await page.getByRole('button', { name: 'Titel bearbeiten' }).click();

	const aendern = page.getByRole('button', { name: 'Cover ändern' });
	await expect(aendern, 'die sichtbare Aktion muss da sein').toBeVisible();

	// Seit dem 22.09.2026 steht daneben „Cover neu holen" (OFFEN.md 4.16): die automatische
	// Suche für EINEN Titel statt des Sammellaufs über den ganzen Bestand.
	await expect(page.getByRole('button', { name: 'Cover neu holen' })).toBeVisible();

	// Das versteckte Dateifeld gehört dazu — ohne es klickt „Cover ändern" ins Leere.
	await expect(page.locator('#cover-upload-drawer')).toHaveCount(1);

	// Der Kern: Was liegt tatsächlich auf dem Cover?
	//
	// elementFromPoint statt eines Selektors: Es beantwortet genau die Frage, die zählt —
	// worauf trifft ein Finger, der auf das Bild tippt. Ein Selektor fände den Overlay
	// nicht mehr, sagte aber nichts darüber, ob an seiner Stelle etwas anderes liegt.
	//
	// closest() ist dabei nicht optional. Die erste Fassung prüfte nur das getroffene
	// Element selbst — und blieb beim Rückbau GRÜN: Getroffen wird das <svg> IM Knopf,
	// nicht der Knopf. Ein Detektor, der nur die oberste Ebene ansieht, übersieht jedes
	// Bedienelement mit Inhalt.
	const daraufGetippt = await page.evaluate(() => {
		const huelle = document.querySelector('#cover-upload-drawer')?.closest('.flex-col');
		const box = huelle?.querySelector('.group');
		const r = box?.getBoundingClientRect();
		if (!r || !box) return { gefunden: false, imCover: false, bedienbar: '' };
		const el = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
		const knopf = el?.closest('button, a[href], [role="button"], input, label');
		return {
			gefunden: !!el,
			imCover: box.contains(el),
			bedienbar: knopf ? `<${knopf.tagName.toLowerCase()}>` : ''
		};
	});

	// Zwei Gegenproben, damit eine Fehlmessung nicht als Erfolg durchgeht: Es muss
	// überhaupt etwas getroffen worden sein, und der Treffer muss IM Cover liegen. Läge
	// die Fläche ausserhalb des Fensters, lieferte elementFromPoint null — und „kein
	// Knopf getroffen" wäre die richtige Antwort auf die falsche Frage.
	expect(daraufGetippt.gefunden, 'auf der Cover-Fläche wurde nichts getroffen').toBe(true);
	expect(daraufGetippt.imCover, 'gemessen wurde ausserhalb des Covers').toBe(true);

	expect(
		daraufGetippt.bedienbar,
		`Auf dem Cover liegt ein Bedienelement (${daraufGetippt.bedienbar}). Ein Tipp auf das ` +
			`Bild darf keinen Dateidialog öffnen — dafür gibt es „Cover ändern" darunter.`
	).toBe('');
});

// Nach einem gescheiterten Hochladen ging dieselbe Datei kein zweites Mal hinaus: Das Dateifeld
// hielt sie noch, und der Browser meldet kein change, wenn die Wahl gleich bleibt (Chromium und
// WebKit; Firefox schickte sie). Gewählt wird über den Dateidialog wie von Hand (OFFEN.md 5.106).
test('Cover: nach einem gescheiterten Hochladen geht dieselbe Datei noch einmal hinaus', async ({
	page
}, info) => {
	const marke = uniqueSuffix();
	const titel = `Cover zweiter Versuch ${marke}`;
	angelegt.titel.push(titel);
	// Mit einem Exemplar: Die Titelliste zeigt zuerst die Titel mit Bestand.
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel, autor, signatur) VALUES ('${titel}', 'Probe', 'CZ 1') RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT id, 'CZ-${marke.slice(-8)}', true FROM t;`);
	const datei = info.outputPath('cover.png');
	// Ein PNG von 1 × 1 Bildpunkt.
	const png =
		'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==';
	writeFileSync(datei, Buffer.from(png, 'base64'));
	let versuche = 0;
	// Beide Antworten kommen von hier: Ein echtes Hochladen legte bei jedem Lauf eine Datei unter
	// uploads ab.
	await page.route(/\/api\/books\/[^/]+\/cover-upload$/, (route) => {
		versuche += 1;
		const rumpf =
			versuche === 1
				? { error: 'Datenbank nicht erreichbar' }
				: { coverUrl: '/uploads/covers/e2e.webp' };
		return route.fulfill({
			status: versuche === 1 ? 503 : 200,
			contentType: 'application/json',
			body: JSON.stringify(rumpf)
		});
	});
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(titel);
	await page.getByText(titel, { exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();

	const waehle = async () => {
		const dialog = page.waitForEvent('filechooser');
		await page.getByRole('button', { name: 'Cover ändern' }).click();
		await (await dialog).setFiles(datei);
	};
	await waehle();
	await expect(page.getByText('Datenbank nicht erreichbar')).toBeVisible();
	await waehle();

	await expect
		.poll(() => versuche, { message: 'dieselbe Datei ging kein zweites Mal hinaus', timeout: 5000 })
		.toBe(2);
	await expect(page.getByText('Cover erfolgreich hochgeladen')).toBeVisible();
});
