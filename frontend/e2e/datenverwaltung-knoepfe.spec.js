import { test, expect } from '@playwright/test';
import { uiLogin, einstellungsKategorie } from './helpers.js';

// M3, Buttons, Guidelines: „the filled style should be used sparingly, ideally for only one
// action on a page". Die Datenverwaltung trägt sechs Werkzeuge untereinander. Gefüllt ist dort
// nur der Import einer gewählten Datei, weil er einen Ablauf abschließt; Cover-Abgleich, Export
// und die Offline-Sicherungen sind eigene Werkzeuge und umrandet.
//
// Gemessen wird die Farbe im Browser, nicht die Klasse: Die Variante eines Knopfs kann am
// Element stehen und gegen eine zweite Farbklasse verlieren.

/**
 * Die Beschriftungen der bedienbaren Knöpfe im Inhalt, deren Fläche die Farbe primary trägt.
 * @param {import('@playwright/test').Page} page
 */
function gefuellteKnoepfe(page) {
	return page.evaluate(() => {
		const probe = document.createElement('div');
		probe.style.backgroundColor = 'var(--color-primary)';
		document.body.append(probe);
		const primary = getComputedStyle(probe).backgroundColor;
		probe.remove();
		return [...document.querySelectorAll('button')]
			.filter((k) => !k.disabled && !k.closest('aside') && !k.closest('nav'))
			.filter((k) => getComputedStyle(k).backgroundColor === primary)
			.map((k) => (k.textContent || '').replace(/\s+/g, ' ').trim());
	});
}

test('Datenverwaltung: gefüllt ist nur der Import einer gewählten Datei', async ({ page }) => {
	await uiLogin(page);
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Datenverwaltung').click();

	const liste = page.getByRole('button', { name: 'Liste importieren' });
	await expect(liste).toBeDisabled();
	await expect(page.getByRole('button', { name: 'Bestand importieren' })).toBeDisabled();
	await expect(page.getByRole('button', { name: 'Import starten' })).toBeDisabled();
	for (const werkzeug of [
		'Fehlende Cover im Hintergrund laden',
		'Katalog als CSV herunterladen',
		'Sicherungsdateien auswählen'
	])
		await expect(page.getByRole('button', { name: werkzeug })).toBeEnabled();

	expect(await gefuellteKnoepfe(page), 'ohne gewählte Datei ist kein Knopf gefüllt').toEqual([]);

	// Gegenprobe am Messweg: Mit einer gewählten Datei findet dieselbe Messung genau den einen.
	await page.getByTestId('listenimport-datei').setInputFiles({
		name: 'liste.csv',
		mimeType: 'text/csv',
		buffer: Buffer.from('isbn,titel,bestand\n')
	});
	await expect(liste).toBeEnabled();
	// Die Farbe wechselt mit einem Übergang; erst am Ende steht primary.
	await expect
		.poll(() => gefuellteKnoepfe(page), { message: 'gefüllt ist der Import der gewählten Datei' })
		.toEqual(['Liste importieren']);
});
