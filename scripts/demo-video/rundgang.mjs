// Kompletter Produktrundgang Bibliosys – Aufnahme als Video.
import { start, BASE } from './harness.mjs';
import { mkdirSync, writeFileSync } from 'node:fs';
import { demoDaten } from './db.mjs';
const D = demoDaten();
console.log('Demo-Daten', JSON.stringify(D));

const OUT = process.argv[2] || './out';
mkdirSync(OUT, { recursive: true });
const ONLY = (process.env.DEMO_ONLY || '').split(',').filter(Boolean);

const { browser, page, d } = await start(OUT);
// Sicherheitsnetz: nichts verlässt den Demo-Stack, selbst wenn ein Klick daneben geht.
for (const p of ['**/api/mail/send-bulk-overdue', '**/api/abgaenger/mail', '**/api/bestellungen'])
  await page.route(p, r => (r.request().method() === 'GET' ? r.continue() : r.abort()));

const fails = [];
async function kapitel(name, fn) {
  if (ONLY.length && !ONLY.includes(name)) return;
  try { await fn(); }
  catch (e) {
    fails.push(name + ': ' + String(e).split('\n').slice(0,4).join(' | '));
    await page.screenshot({ path: `${OUT}/FAIL-${name}.png` }).catch(() => {});
    console.error('✗', name, String(e).split('\n')[0]);
    await d.captionOff().catch(() => {});
    await page.keyboard.press('Escape').catch(() => {});
  }
}
const tab = (name) => page.getByRole('tab', { name });
const btn = (name, exact) => page.getByRole('button', { name, exact });
const nav = (title) => d.click(page.getByTitle(title, { exact: true }), { settle: 1400 });
const scan = (code, settle) => d.scan('#omnibox-input', code, { settle });
const system = async () => {
  const b = page.getByRole('button', { name: 'System', exact: true });
  if ((await b.getAttribute('aria-expanded')) !== 'true') await d.click(b, { settle: 700 });
};

// ───────────────────────── Intro & Login
await kapitel('intro', async () => {
  await page.goto(BASE + '/');
  await d.title('Bibliosys', 'Die Schulbibliothek, die mitdenkt.', 'Produktrundgang · alle Bereiche', 3800);
  await d.titleOff();
});

await kapitel('opac', async () => {
  await page.goto(BASE + '/katalog');
  await d.caption('Für Schüler', 'Der öffentliche Katalog.', 'Ohne Anmeldung – vom Klassenraum, vom Handy, von zu Hause.', 2000);
  await d.type('#opac-suchfeld', 'Harry Potter', { delay: 90, settle: 2000 });
  await d.zoom(page.getByText('Joanne K. Rowling').first(), 1.4, 2000);
  await d.zoomOff();
  await page.locator('#opac-suchfeld').fill(''); await d.pause(500);
  await d.type('#opac-suchfeld', 'Dürrenmatt', { delay: 90, settle: 2200 });
  await d.captionOff();
  await page.goto(BASE + '/monitor');
  await d.caption('Für den Flur', 'Der Bibliotheks-Monitor.', 'Buch des Monats, Neuzugänge, Beliebt diese Woche – läuft auf jedem Bildschirm ohne Anmeldung.', 0);
  // Der Monitor wechselt von selbst alle 15 s — zu langsam fürs Video. Die Folienpunkte
  // oben zeigen alle drei Folien in ~12 s (Buch des Monats steht beim Laden schon).
  await d.moveTo(1700, 900, 800);
  await d.pause(3500);
  await d.click(btn('Neu eingetroffen anzeigen'), { settle: 4200 });
  await d.click(btn('Beliebt diese Woche anzeigen'), { settle: 4200 });
  await d.captionOff();
  await page.goto(BASE + '/');
});

await kapitel('login', async () => {
  if (!page.url().endsWith('/')) await page.goto(BASE + '/');
  await d.caption('Anmeldung', 'Ein Konto, ein Passwort.', 'Anmeldung über das Schul-Postfach – keine zweite Benutzerverwaltung.', 0);
  await d.type('#login-email', 'bibliothek@musterschule.de');
  await d.type('#login-password', '••••••••••', { delay: 40 });
  await page.locator('#login-password').fill('demo');
  await d.click(btn('Anmelden'), { settle: 1800 });
  await page.getByRole('button', { name: 'Abmelden' }).waitFor();
  await d.captionOff();
});

// ───────────────────────── 01 Ausleihe
await kapitel('ausleihe', async () => {
  await d.caption('01 · Ausleihe', 'Scannen. Fertig.', 'Schülerausweis scannen, Buch scannen – die Theke erledigt den Rest.', 2200);
  await d.click('#omnibox-input', { settle: 400 });
  await scan(D.S_OK, 1800);
  await scan(D.B_FREE[0], 2200);
  await scan(D.B_FREE[1], 2200);
  await d.caption('01 · Ausleihe', 'Alles auf einen Blick.', 'Fristen, Status, Verlängern, Rückgabe, Schaden melden – direkt an der Zeile.', 0);
  await d.hover(btn('Ausleihe verlängern').first()); await d.pause(1200);
  await d.zoom(page.locator('tbody tr').first(), 1.45, 2000);
  await d.zoomOff();
  await d.caption('01 · Ausleihe', 'Schaden? Sofort erfasst.', 'Grund, Ersatzbetrag – die Forderung landet in der Akte, der Elternbrief kommt als PDF.', 0);
  await d.click(btn('Verlust oder Schaden melden').first(), { settle: 1200 });
  await d.type('#damage-reason', 'Wasserschaden, Seiten gewellt', { delay: 60, settle: 500 });
  await d.type('#damage-amount', '12,90', { delay: 90, settle: 1400 });
  await d.clickSure(btn('Abbrechen').first(), { settle: 900 });
  await d.caption('01 · Ausleihe', 'Vorgemerkt? Die Theke warnt.', 'Kommt ein reservierter Titel zurück, geht er nicht ins Regal – sondern zur nächsten Leserin.', 0);
  await d.click('#omnibox-input', { settle: 300 });
  await scan(D.B_RESERVED, 1800);
  await d.click(page.locator('tr', { hasText: D.B_RESERVED_TITEL }).getByRole('button', { name: 'Buch zurückgeben' }), { settle: 1800 });
  await d.pause(2200);
  await d.clickSure(btn('Verstanden'), { settle: 1200 });
  await d.click(page.getByTitle('Schüler schließen (ESC)'), { settle: 1200 });
  await d.caption('01 · Ausleihe', 'Rückgabe ohne Umweg.', 'Buch ohne geöffnete Theke scannen – es wird sofort zurückgebucht.', 0);
  await d.click('#omnibox-input', { settle: 300 });
  await scan(D.B_FREE[0], 2400);
  await d.click(page.getByTitle('Schüler schließen (ESC)'), { settle: 800 }).catch(() => {});
  await d.caption('01 · Ausleihe', 'Oder einfach den Namen tippen.', 'Kein Ausweis dabei? Namenssuche mit Sofortvorschlägen.', 0);
  await d.type('#omnibox-input', D.S_OVERDUE_NAME, { delay: 90, settle: 1400 });
  const treffer = page.locator('[role="listbox"] [role="option"], [role="listbox"] button, [role="listbox"] li').first();
  if (await treffer.isVisible({ timeout: 4000 }).catch(() => false)) await d.click(treffer, { settle: 2200 });
  else await d.key('Enter', 2200);
  await page.getByTitle('Schüler schließen (ESC)').waitFor({ timeout: 8000 });
  await d.caption('01 · Ausleihe', 'Überfällig? Sieht man sofort.', 'Mahnstufe und Fristen stehen an der Ausleihe, nicht in einer Liste irgendwo.', 1200);
  await d.zoom(page.locator('tbody tr').first(), 1.45, 2400);
  await d.zoomOff();
  await d.click(page.getByTitle('Schüler schließen (ESC)'), { settle: 1000 });
  await d.caption('01 · Ausleihe', 'Sperren greifen an der Theke.', 'Ein gesperrter Ausweis wird angehalten – mit Grund und bewusstem Override.', 0);
  await d.click('#omnibox-input', { settle: 300 });
  await scan(D.S_LOCKED, 2000);
  await scan(D.B_FREE[2], 2200);
  await d.pause(1400);
  await d.hover(btn('Sperre aufheben').first()); await d.pause(1200);
  await d.click(page.getByTitle('Schüler schließen (ESC)'), { settle: 800 });
  await d.captionOff();
});

// ───────────────────────── 02 Medienkatalog
await kapitel('katalog', async () => {
  await nav('Medienkatalog');
  await d.caption('02 · Medienkatalog', 'Der ganze Bestand, sofort durchsuchbar.', 'Titel, Autor, Fach oder Klasse – ein Feld für alles, Cover inklusive.', 2000);
  await d.type('#katalog-suchfeld', 'Dürrenmatt', { delay: 80, settle: 1800 });
  await d.pause(800);
  await page.locator('#katalog-suchfeld').fill(''); await d.pause(1200);
  await d.caption('02 · Medienkatalog', 'Nach Jahrgang sortiert.', 'Lernmittel je Jahrgang und Schulzweig – mit Stückzahlen.', 0);
  await d.click(tab('Jahrgänge'), { settle: 1600 });
  await d.scroll(500); await d.pause(800);
  await d.click(tab('Buch-Suche'), { settle: 1000 });
  await d.caption('02 · Medienkatalog', 'Die Buchakte.', 'Exemplare, Ausleiher, Vormerkungen, Historie – ein Klick auf die Karte.', 0);
  await d.type('#katalog-suchfeld', 'Tschick', { delay: 80, settle: 1500 });
  await d.click(page.getByText('Tschick', { exact: true }).first(), { settle: 1800 });
  for (const t of [/^Exemplare \(/, /^Ausleiher \(/, /^Vormerkungen \(/, 'Historie']) {
    const l = page.getByRole('tab', { name: t });
    if (await l.count()) { await d.click(l, { settle: 1500 }); }
  }
  await d.caption('02 · Medienkatalog', 'Titel pflegen, Geräte verwalten.', 'Auch iPads, Taschenrechner und Beamer laufen über dieselbe Ausleihe.', 0);
  await nav('Medienkatalog');
  await d.click(tab('Titel-Verwaltung'), { settle: 1600 });
  await d.scroll(300);
  await d.click(tab('Geräte'), { settle: 1800 });
  await d.captionOff();
});

// ───────────────────────── 03 Signaturen
await kapitel('signaturen', async () => {
  await nav('Signaturen');
  await d.caption('03 · Signaturen', 'Systematik statt Zettelwirtschaft.', 'Sachgruppen und Regale – Präfixsuche über den ganzen Bestand.', 2000);
  await d.type(page.getByLabel('Signatur suchen'), 'Jug', { delay: 110, settle: 1600 });
  const regal = page.getByRole('button', { name: /^Jug /i }).first();
  if (await regal.count()) await d.click(regal, { settle: 1800 });
  await d.scroll(300);
  await d.captionOff();
});

// ───────────────────────── 04 Druck-Center
await kapitel('druck', async () => {
  await nav('Druck-Center');
  await d.caption('04 · Druck-Center', 'Etiketten und Ausweise – aus einer Hand.', 'Buch-Etiketten, fehlende Etiketten als Warteschlange, Schülerausweise als Designer.', 2200);
  await d.scroll(300);
  await d.click(tab(/Fehlende Etiketten/), { settle: 1800 });
  await d.click(tab('Schülerausweise'), { settle: 2000 });
  await d.scroll(400); await d.pause(800);
  await d.click(tab('Klassenweise drucken'), { settle: 1600 });
  const kl = page.getByRole('combobox', { name: 'Klasse' });
  if (await kl.count()) { await d.click(kl, { settle: 600 }); await kl.selectOption({ index: 3 }).catch(() => {}); await d.pause(1200); }
  await d.captionOff();
});

// ───────────────────────── 05 Klassensätze
await kapitel('klassensaetze', async () => {
  await nav('Klassensätze');
  await d.caption('05 · Klassensätze', 'Welche Klasse hat welche Lektüre?', 'Klassensätze je Klasse – und die Warteschlange der Reservierungen aus dem Kollegium.', 2200);
  await d.type(page.getByLabel('Klasse suchen'), '07B', { delay: 110, settle: 1600 });
  await page.getByLabel('Klasse suchen').fill(''); await d.pause(800);
  await d.click(btn('Klasse hinzufügen'), { settle: 1600 });
  await d.type('#book-search-field', 'Krabat', { delay: 90, settle: 1500 });
  await d.click(page.locator('[aria-pressed]').first(), { settle: 1000 });
  await d.type('#class-input', '06A', { delay: 110, settle: 800 });
  await d.key('Enter', 1000);
  await d.pause(800);
  await d.clickSure(btn('Abbrechen').first(), { settle: 800 });
  await d.captionOff();
});

// ───────────────────────── 06 Schülerdatei
await kapitel('schueler', async () => {
  await nav('Schülerdatei');
  await d.caption('06 · Schülerdatei', 'Jeder Schüler, eine Akte.', 'Stammdaten, Ausleihen, Gebühren, Ausweis – Suche läuft serverseitig über den ganzen Bestand.', 2200);
  await d.type(page.getByLabel('Schüler suchen'), D.S_FEE_NAME, { delay: 100, settle: 1800 });
  await d.click(page.locator('tbody tr').first(), { settle: 1800 });
  await d.caption('06 · Schülerdatei', 'Gebühren & Schäden im Blick.', 'Offene Forderungen, Ersatzforderung als PDF, Bezahlt oder Storno mit Begründung.', 0);
  await d.scroll(400); await d.pause(1800);
  await d.click(btn('Stammdaten & Adresse'), { settle: 1800 });
  await d.caption('06 · Schülerdatei', 'Sperren – mit Begründung.', 'Kein Sperren ohne Grund: Der Text steht danach an der Theke und in der Akte.', 0);
  await d.click(btn('Schüler sperren'), { settle: 1200 });
  await d.type(page.getByLabel(/Grund der Sperre/), 'Ersatzforderung offen – Rücksprache Eltern', { delay: 55, settle: 1200 });
  await d.clickSure(btn('Abbrechen').first(), { settle: 900 });
  await d.click(page.getByTitle('Schüler schließen (ESC)'), { settle: 1000 });
  await page.getByLabel('Schüler suchen').fill(''); await d.pause(600);
  await d.caption('06 · Schülerdatei', 'Neu anlegen oder importieren.', 'Einzeln per Formular – oder klassenweise aus dem LUSD-Export.', 0);
  await d.click(btn('Neuen Schüler anlegen'), { settle: 1500 });
  await d.pause(1200);
  await d.clickSure(btn('Abbrechen').first(), { settle: 800 });
  await d.click(tab('Abgänger / Archiv'), { settle: 1600 });
  await d.click(tab('Aktive Schüler'), { settle: 800 });
  await d.captionOff();
});

// ───────────────────────── 07 Mahnwesen
await kapitel('mahnwesen', async () => {
  await nav('Mahnwesen');
  await d.caption('07 · Mahnwesen', 'Wer schuldet was – nach Dringlichkeit.', 'Akut fällig oder eskaliert, gefiltert nach Klasse. Mahnbriefe und Sammel-Mahnlauf per Mail.', 2400);
  await d.click(tab(/^Akut fällig/), { settle: 1600 });
  await d.click(tab(/^Eskaliert/), { settle: 1600 });
  await d.click(tab(/^Alle/), { settle: 1000 });
  const kl = page.getByRole('combobox', { name: 'Nach Klasse filtern' });
  if (await kl.count()) {
    await d.click(kl, { settle: 700 });
    const opt = page.getByRole('option').nth(2);
    if (await opt.isVisible().catch(() => false)) { await d.click(opt, { settle: 1800 }); await d.click(kl, { settle: 600 }); await d.clickIf(page.getByRole('option').first(), { settle: 800 }); }
    else await d.key('Escape', 500);
  }
  await d.click(btn('Weitere Druck- und Export-Optionen'), { settle: 1400 });
  await d.key('Escape', 600);
  await d.caption('07 · Mahnwesen', 'Der Mahnlauf – mit Sicherheitsnetz.', 'Klassen wählen, Empfänger prüfen, dann erst senden. Lehrkräfte werden nie angemahnt.', 0);
  await d.click(btn(/Alle anmahnen – Mahnlauf konfigurieren/).first(), { settle: 1800 });
  await d.pause(1800);
  await d.click(page.getByRole('dialog').getByRole('button', { name: 'Abbrechen' }), { settle: 900 });
  await d.captionOff();
});

// ───────────────────────── 08 Abgänger
await kapitel('abgaenger', async () => {
  await nav('Abgänger');
  await d.caption('08 · Abgänger', 'Nichts geht verloren.', 'Wer die Schule verlässt und noch Bücher hat, steht hier – Kontoauszüge an die Klassenleitung, per Klick.', 2400);
  await d.scroll(300);
  await d.click(btn(/An Klassenleitungen mailen/), { settle: 1800 });
  await d.pause(1500);
  await d.click(page.getByRole('dialog').getByRole('button', { name: 'Abbrechen' }), { settle: 900 });
  await d.captionOff();
});

// ───────────────────────── 09 Bestellungen
await kapitel('bestellungen', async () => {
  await nav('Bestellungen');
  await d.caption('09 · Bestellwesen', 'Bedarf erkennen, bevor er fehlt.', 'Lernmittel unter Meldebestand landen automatisch hier – Bestellung mit einem Klick.', 2400);
  await d.zoom(btn(/zur Bestellung hinzufügen/).first(), 1.4, 2000); await d.zoomOff();
  const add = btn(/zur Bestellung hinzufügen/).first();
  if (await add.count()) { await d.click(add, { settle: 1400 }); await d.clickIf(btn(/zur Bestellung hinzufügen/).first(), { settle: 1400 }); }
  await d.pause(800);
  await d.caption('09 · Bestellwesen', 'Vom Wareneingang bis zum Etikett.', 'Einbuchen, Historie mit Händler-Bestätigung, Berichte fürs Sekretariat.', 0);
  await d.click(tab(/Wareneingang/), { settle: 1600 });
  await d.click(tab('Bestellhistorie', true), { settle: 1600 });
  const open = btn(/^Bestellung .* öffnen|bei .* öffnen/).first();
  if (await open.count()) { await d.click(open, { settle: 1800 }); await d.clickIf(btn('Zurück zur Bestellhistorie'), { settle: 800 }); }
  await d.click(tab('Berichte'), { settle: 1500 });
  await d.caption('09 · Bestellwesen', 'Das Kollegium bestellt mit.', 'Klassensatz-Reservierungen als Warteschlange, Buchwünsche und Meldungen aus dem Portal.', 0);
  await d.click(tab(/Klassensatz-Reservierungen/i), { settle: 1800 });
  await d.pause(800);
  await d.click(tab('Wünsche & Meldungen'), { settle: 1800 });
  await d.pause(800);
  await d.captionOff();
});

// ───────────────────────── 10 Inventur
await kapitel('inventur', async () => {
  await nav('Inventur');
  await d.caption('10 · Inventur', 'Bestandsprüfung mit dem Scanner.', 'Nach Signatur, Fach oder Klasse eingrenzen – scannen, Fehlbestand erhalten.', 2200);
  await d.click(btn('Neue Bestandsprüfung starten'), { settle: 1500 });
  await d.click(page.getByText('Nur bestimmte Signatur'), { settle: 800 });
  await d.type(page.getByLabel('Signatur auswählen'), 'Jug', { delay: 110, settle: 900 });
  await d.click(btn('Inventur Starten'), { settle: 1800 });
  const sc = page.getByPlaceholder('Barcode scannen...');
  await d.click(sc, { settle: 300 });
  for (const b of D.INV) { await d.scan(sc, b, { settle: 1100 }); }
  await d.pause(1200);
  await d.hover(btn('Inventur abschließen')); await d.pause(1200);
  await d.captionOff();
});

// ───────────────────────── 11 Statistiken
await kapitel('statistik', async () => {
  await system();
  await nav('Statistiken');
  await d.caption('11 · Statistiken', 'Zahlen, die Entscheidungen tragen.', 'Zirkulation, Wiederbeschaffungswert, Renner und Ladenhüter – ohne Schülerklarnamen.', 1500);
  await d.zoom(page.getByText('Zirkulationsquote').first(), 1.35, 2400); await d.zoomOff();
  await d.scroll(400); await d.pause(600);
  await d.click(btn('LMF', true), { settle: 1500 });
  await d.click(btn('Gesamt', true), { settle: 1200 });
  await d.clickIf(btn('Renner', true), { settle: 1200 });
  await d.click(btn(/Detailansicht öffnen/), { settle: 1800 });
  await d.scroll(300);
  await d.clickIf(btn('Statistik', true), { settle: 900 });
  await d.captionOff();
});

// ───────────────────────── 12 System-Logs
await kapitel('logs', async () => {
  await system();
  await nav('System-Logs');
  await d.caption('12 · System-Logs', 'Jede Änderung nachvollziehbar.', 'Logbuch und Admin-Audit – wer hat wann was geändert.', 2200);
  await d.scroll(300);
  await d.click(btn('Admin-Audit-Log'), { settle: 1600 });
  await d.click(btn('Allgemeines Logbuch'), { settle: 800 });
  await d.captionOff();
});

// ───────────────────────── 13 Benutzer & Rechte
await kapitel('rechte', async () => {
  await system();
  await nav('Benutzer & Rechte');
  await d.caption('13 · Benutzer & Rechte', 'Rollen statt Passwörter.', 'Admin, Mitarbeit, Helfer, Kollegium – Rechte pro Rolle, Zugangsanfragen aus dem Kollegium.', 2200);
  await d.scroll(200);
  await d.click(tab('Rollen & Rechte'), { settle: 1800 });
  await d.scroll(400); await d.pause(800);
  await d.click(tab('Benutzer'), { settle: 800 });
  await d.captionOff();
});

// ───────────────────────── 14 Einstellungen
await kapitel('einstellungen', async () => {
  await system();
  await nav('Einstellungen');
  const kat = (t) => page.getByRole('navigation', { name: 'Einstellungs-Kategorien' }).getByRole('button', { name: new RegExp(`^${t} `) });
  await d.caption('14 · Einstellungen', 'Alles an einem Ort.', 'Fristen, Mahnwesen, Lieferanten, Mail, Datenschutz, Schuljahreswechsel – je Kategorie gespeichert.', 2200);
  for (const k of ['Ausleihe & Fristen','Mahnwesen','Mahnwesen-Routing','Bestellwesen','Lieferanten','Datenschutz & Sitzung','Erreichbarkeit & Alarme','Mail']) await d.click(kat(k), { settle: 1400 });
  await d.caption('14 · Einstellungen', 'Massenaktionen mit Netz und doppeltem Boden.', 'Lernmittel klassenweise verlängern, Littera- und LUSD-Import, Versetzung zum Schuljahreswechsel, Export & Offline-Sicherung.', 0);
  await d.click(kat('LMF-Aktionen'), { settle: 1400 });
  if (await page.getByLabel(/Klasse/i).first().isVisible().catch(() => false)) await d.type(page.getByLabel(/Klasse/i).first(), '07B', { delay: 110, settle: 900 });
  await d.click(kat('Datenverwaltung'), { settle: 1600 });
  await d.scroll(300); await d.pause(600);
  await d.click(kat('Schuljahreswechsel'), { settle: 1600 });
  await d.caption('14 · Einstellungen', 'Betriebsbereitschaft.', 'Backup, Mail, Erreichbarkeit – das System prüft sich selbst und sagt, was noch fehlt.', 0);
  await d.click(kat('Betriebsbereitschaft'), { settle: 2200 });
  await d.scroll(400); await d.pause(1000);
  await d.captionOff();
});

// ───────────────────────── 15 Kollegium-Portal
await kapitel('portal', async () => {
  await nav('Mein Portal');
  await d.caption('15 · Kollegium-Portal', 'Lehrkräfte helfen sich selbst.', 'Bestand suchen, Klassensatz reservieren, Buchwunsch abgeben – ohne die Bibliothek anzurufen.', 2400);
  await d.type('#portal-suchfeld', 'Die Welle', { delay: 90, settle: 1600 });
  await d.click(btn('Klassensatz reservieren').first(), { settle: 1400 });
  await d.type(page.getByLabel('Klasse *'), '09B', { delay: 110, settle: 500 });
  await d.type(page.getByLabel('Anzahl'), '27', { delay: 110, settle: 500 });
  await d.click(btn(/Anfrage senden/), { settle: 2000 });
  await d.click(tab('Klassensätze'), { settle: 1500 });
  await d.click(tab('Bestand nach Jahrgang'), { settle: 1800 });
  await d.captionOff();
});

// ───────────────────────── 16 Lehrkraft-Sicht
await kapitel('lehrkraft', async () => {
  await d.caption('16 · Rollen', 'Jede Rolle sieht nur ihren Teil.', 'Eine Lehrkraft meldet sich an – und bekommt genau das Portal, sonst nichts.', 0);
  await d.click(btn('Abmelden'), { settle: 1500 });
  await d.type('#login-email', 'a.berger@musterschule.de');
  await page.locator('#login-password').fill('demo');
  await d.click(btn('Anmelden'), { settle: 2200 });
  await page.getByRole('button', { name: 'Abmelden' }).waitFor();
  await d.moveTo(120, 300, 700);
  await d.pause(2200);
  await d.click(tab(/Meine Anliegen/), { settle: 1500 });
  await d.click(page.getByRole('radio', { name: /Etwas stimmt nicht/ }).first(), { settle: 900 });
  await d.type(page.getByLabel(/Worum geht es|Welches Buch/).first(), 'Lambacher Schweizer 7 – Lösungsheft fehlt', { delay: 50, settle: 800 });
  await d.type(page.getByLabel('Klasse / Kurs'), '07B', { delay: 100, settle: 600 });
  await d.click(btn('Absenden'), { settle: 2400 });
  await d.captionOff();
});

// ───────────────────────── Outro
await kapitel('outro', async () => {
  await d.liste('Auf einen Blick', 'Alles, was Bibliosys mitbringt', [
    'Ausleihe & Rückgabe per Scanner', 'Öffentlicher Katalog & Flur-Monitor',
    'Medienkatalog mit Buchakte', 'Signaturen & Systematik',
    'Druck-Center: Etiketten & Ausweise', 'Klassensätze & Reservierungen',
    'Schülerdatei mit Gebühren & Sperren', 'Mahnwesen mit Sammel-Mahnlauf',
    'Abgänger-Verfolgung', 'Bestellwesen bis zum Wareneingang',
    'Inventur per Scanner', 'Statistiken ohne Klarnamen',
    'System-Logs & Audit', 'Rollen & Rechte', 'Kollegium-Portal', 'Einstellungen in 13 Kategorien'
  ], 9000);
  await d.liste('Und außerdem …', 'Was in diesem Rundgang nicht vorkam', [
    'Kamera-Scanner & Passbild per Webcam', 'Ausweise, Etiketten & Kontoauszüge als PDF',
    'Gebühren: Bezahlt, Storno mit Begründung', 'Zubehör-Checkliste beim Geräteverleih',
    'Wareneingang einbuchen → Etiketten', 'Bestell-Bestätigung durch den Händler per Link',
    'Littera- & LUSD-Import, Versetzung', 'Ferien-Leseclub & Schließzeiten',
    'Selbstanmeldung fürs Kollegium', 'Live-Abgleich mehrerer Arbeitsplätze',
    'Sperrbildschirm bei Inaktivität', 'Verschlüsselte Backups, Alarm-Mails, Audit-Log',
    'DSGVO: Anonymisierung, Löschfristen, Auskunft', 'Ladenhüter-Analyse & Bestandsberichte'
  ], 9000);
  await d.title('Bibliosys', 'Die Schulbibliothek, die mitdenkt.', 'Open Source · DSGVO-konform · läuft auf dem Schulserver', 3500);
});

console.log(fails.length ? 'FEHLER:\n' + fails.join('\n') : 'Rundgang ohne Fehler.');
writeFileSync(`${OUT}/cues.json`, JSON.stringify(d.cues, null, 1));
const video = page.video();
await page.close();
const vp = await video.path();
await browser.close();
console.log('VIDEO', vp);
