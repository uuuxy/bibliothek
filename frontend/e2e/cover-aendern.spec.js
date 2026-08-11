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
import { uiLogin, gehZu } from './helpers.js';

test('Cover: das Bild ist kein verstecktes Bedienelement, „Cover ändern" bleibt', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');

	// Derselbe Weg, den eine Bibliothekskraft nimmt: Stift auf der Buchkarte → Buchakte →
	// „Titel bearbeiten". Zwei Klicks, kein Abkürzen über den Store — sonst prüfte der
	// Test einen Zustand, den man von Hand gar nicht erreicht.
	await page.getByRole('button', { name: 'Buch schnell bearbeiten' }).first().click();
	await page.getByRole('button', { name: 'Titel bearbeiten' }).click();

	const aendern = page.getByRole('button', { name: 'Cover ändern' });
	await expect(aendern, 'die sichtbare Aktion muss da sein').toBeVisible();

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
