# Fachkonzept & Feature-Spezifikation

Beschreibt die funktionale Business-Logik der Bibliothekssoftware auf Basis der Go-Implementierung. Zielgruppe: Administratoren, Betreiber und Entwickler.

---

## 1. Zentrale Scanner-Omnibox (Theke)

Die Software nutzt an der Theke ein eingabefokussiertes Design für Ausleihe und Rückgabe. Ein
einziges Eingabefeld (Omnibox) verarbeitet alle Scans.

**Der Normalfall ist ohne Präfix.** Die Ausweise und Buchetiketten aus dem Littera-
Altbestand tragen nackte Nummern und dürfen nicht neu gedruckt werden; ein Scan wird
deshalb der Reihe nach aufgelöst:

```
Buch-Exemplar → Ausweis (Schüler wie Kollegium) → Volltextsuche im Katalog
```

Alle Leser stehen in einer Tabelle (Migration 125); eine Stufe sucht den Leser zum Ausweis,
gleich welcher Art (`verarbeiteOhneVorsilbe`). Das geht auf, weil die Formen verschieden sind: Buchetiketten liefern eine 13-stellige
EAN-13, Schülerausweise die Nummer ihres Kartenherstellers. Die Vorgängersoftware Littera
kennt überhaupt keine Präfixe — sie hat getrennte Suchfelder, wir haben eines für alles.

**Präfixe sind eine Abkürzung**, kein Muss. Sie überspringen die Auflösung und sprechen
den Bereich direkt an:

- **`A-[Barcode]` (Leser):** Lädt das Konto eines Lesers (inkl. offener Ausleihen, Mahnungen und Sperren).
- **`S-[Barcode]`, `L-[Barcode]`:** dasselbe für die älteren Vorsilben; neu vergeben werden sie nicht.
- **`B-[Barcode]` (Buch-Exemplar):** Führt eine Aktion mit einem Buch aus.
- **`LMF-[Barcode]`:** dasselbe wie `B-`. Manche Lernmittel tragen ein Etikett, das so beginnt ([littera_schema_befund.md](littera_schema_befund.md), „Das Etikett ist entschlüsselt").
- **`G-[Barcode]` (Gerät):** Führt eine Aktion mit Hardware (z. B. Laptops, iPads) aus.

**Eine Vorsilbe für alle Ausweise: `A-`.** Eine Vorsilbe je Herkunft (`S-` aus Handanlage
und LUSD-Import, `L-` aus dem Littera-Personenlauf) behauptete etwas über die Person; wer
jemand ist, steht in den Stammdaten und nicht auf seinem Ausweis. Gelesen werden `S-` und
`L-` weiter: Es gibt Karten mit ihnen, und Nummern werden nicht recycelt.

Die Vorsilbe selbst BLEIBT, und der Grund ist nicht offensichtlich: **Ohne Netz ist sie
die einzige Information, an der die Theke einen Buchscan von einem Ausweisscan
unterscheiden kann.** Offline gibt es niemanden zu fragen. Littera kommt ohne aus, weil
dort Nummer und Scanwert zwei verschiedene Felder sind und der Scanwert vom
Kartenhersteller stammt; dieses zweite Feld haben wir nicht.

**Der Bindestrich als ß:** Ein Handscanner mit amerikanischer Tastatur tippt an einem
Rechner mit deutscher jeden Bindestrich als ß; aus `S-10001` wird `Sß10001`. Jedes Feld, in das
gescannt wird, liest ein ß hinter einer Vorsilbe als Bindestrich, auch das zweite einer
LMF-Nummer: Theke, Inventur, Suchen und die Masken, die eine Nummer speichern (`normalisiereScan`
in `frontend/src/lib/utils/scanEinordnen.js`; beim Bearbeiten erst nach dem Vergleich mit dem
Stand vom Öffnen, `mitVereinheitlichterNummer`). In einer Nummer steht nie ein ß.

**Ablauflogik:**

1. Wird ein Buch gescannt, _ohne_ dass ein Schüler/Lehrer aufgerufen ist, wird das Buch **zurückgegeben**.
2. Wird ein Schüler/Lehrer gescannt und danach Bücher gescannt, werden diese an die Person **ausgeliehen**.

---

## 2. Ausleih-Regelwerk und Fristen

Das System unterscheidet zwischen verschiedenen Medien und Leihertypen:

### 2.1. Fristenberechnung

- **Lernmittelfreiheit (LMF) - "Schulbücher":** Haben ein fixes Rückgabedatum, den Stichtag des laufenden (oder bei Sommer-Ausleihe des kommenden) Schuljahres: bis Jahrgang 10 den **Donnerstag vor den Sommerferien**, den letzten Tag des Büchertauschs, gerechnet aus den Sommerferien (§2.3); in der Oberstufe, die Rückgabe und Ausgabe an dieser Schule selbst organisiert, ohne Klasse mit Jahrgang und ohne bekannte Sommerferien den **31. Juli** (`internal/repository/lmf_stichtag.go`) — **es sei denn, der veröffentlichte LMF-Plan (§2.3) bestimmt sie:** Hat der Büchertausch des Schuljahres begonnen (erster Tag des Plans, gleich welche Klasse), gilt der Stichtag des FOLGENDEN Schuljahres — wer im Tausch oder danach ein Schulbuch bekommt, gibt es erst im nächsten Schuljahr zurück. Davor ist der nächste Rückgabe-Termin der Klasse NACH dem Ausleihtag die Frist (`LMFAbfragen.LadeRueckgabeTerminLage`, nur einjährige Ausleihe).
- **Freihand-Bestand:** Was kein Lernmittel ist, hat eine Frist in Tagen ab der Ausleihe, keine starre Jahresfrist: Bücher 21 Tage (`frist_buch_tage`), Medien, deren Medientyp „CD", „DVD" oder „Audio" enthält, 7 Tage (`frist_medien_tage`). Beides sind Vorgaben, einstellbar unter Ausleihe & Fristen (§17; `berechneFrist` in `internal/service/frist.go`).
- **Ferien:** Eine Frist, die in Tagen zählt — Buch, Medium, Gerät, Verlängerung und die Abholfrist einer Vormerkung (§4.1) —, rückt auf den nächsten Schultag, wenn sie auf ein Wochenende, einen gesetzlichen Feiertag Hessens oder in die Ferien fällt (`lmfplan.Ferientabelle.Tagesfrist`, eine Rechnung für alle fünf). Die Ferien stehen als Tabelle im Programm: die Sommerferien nach dem KMK-Beschluss samt der Jahre, die die Schule selbst einträgt (`internal/pkg/lmfplan/sommerferien.go`), Herbst-, Weihnachts- und Osterferien nach den Terminen des Hessischen Kultusministeriums bis 2029/30 (`internal/pkg/lmfplan/uebrige_ferien.go`; zwei Jahre vor dem Ende warnt die Betriebsbereitschaft, „Ferien für Leihfristen"). Die beweglichen Ferientage der Schule stehen nicht darin. Nicht verschoben werden Stichtage (Lernmittel, LMF-Plan, Ferien-Leseclub), von Hand gesetzte Fristen und die Jahresfrist der Dauerleihe; der Stichtag der Oberstufe, der 31.07., liegt in jedem Jahr in den Sommerferien. Eine Ferien-Datei zum Hochladen gibt es nicht — die Termine stehen Jahre im Voraus fest. Das Werkzeug für „Bücher über die Sommerferien mitnehmen" ist der **Ferien-Leseclub** (Kategorie Ausleihe & Fristen, §17): aktiv + Zieldatum → jede Ausleihe eines Schülers außer Lernmitteln bekommt dieses feste Rückgabedatum, solange es nicht vorbei ist. Eine Ferien-Pause des Mahnwesens gibt es ebenfalls nicht: Das Mahnwesen wird nur von Hand bedient (entschieden am 06.09.2026).
- **Abgänger** (Abschlussklassen, `repository.IstAbschlussklasse`): Was kein Lernmittel ist, Buch,
  Medium oder Gerät, ist spätestens am letzten Schultag vor den Sommerferien fällig, mit denen sie
  gehen; wer danach noch ausleiht, bekommt das Ende des Tages. Der Ferien-Leseclub gilt für sie
  nicht, und eine Verlängerung geht nicht über diesen Tag hinaus (`service.AbgaengerFrist`).
  Schulbücher behalten ihre Regel.
  Der Rückgabe-Termin der Klasse im LMF-Plan bestimmt diese Frist nicht: Ein zweiter Weg zur selben
  Frist (Plan nachschlagen, Ersatz ohne Termin, Nachziehen beim Veröffentlichen) wöge mehr als ein
  paar Tage früher; zum Büchertausch zeigt die Abgängerliste, wer noch etwas hat.
- **Lehrer (Handapparat):** Erhalten pauschal eine Frist von einem Jahr — `AddDate(1, 0, 0)`, also ein **Kalenderjahr**, nicht 365 Tage (im Schaltjahr sind es 366). Wie jede andere Frist läuft sie durch die eine Definition des Tagesendes, `schulzeit.TagesEnde`; eine zweite, rohe Berechnung gibt es bewusst nicht.
- **Verlängerungen:** Ausleihen können verlängert werden, es sei denn, eine Sperre am Leser hält sie an — nach der Regel der Theke (§2.2), beim Schulbuch also nur die Sperre von Hand. Forderung, Überfällig-Automatik und Ausleihlimit halten eine Verlängerung nicht an. Die neue Frist ist die Leihfrist für Bücher (`frist_buch_tage`), bei einem Gerät die Leihfrist der Geräte (`frist_geraete_tage`) wie bei seiner Ausleihe, gezählt ab dem späteren von alter Frist und heute, und rückt wie jede Tagesfrist auf den nächsten Schultag. Die Leserakte schickt die Frist mit, die sie zeigt (`frist_bisher`); steht am Server schon eine andere, verlängert die Tür nicht noch einmal, antwortet mit 409 und nennt die jetzige. So verlängert ein zweiter Klick nach einer verlorenen Antwort nicht doppelt, ebenso ein zweiter Platz mit derselben alten Frist; ohne Angabe verlängert die Tür wie bisher. War die Frist schon vorbei, beginnen Mahnfolge und Überfällig-Sperre neu; die Verlängerung steht dann wie die Frist von Hand im Verwaltungsprotokoll („Überfällige Ausleihe verlängert“, mit bisheriger und neuer Frist).

### 2.2. Blockaden und Limits

- **Ausleihlimit:** Es gibt ein konfigurierbares Maximum an gleichzeitigen Ausleihen pro Schüler (LMF-Bücher ausgenommen). Geräte zählen nicht mit: Ein Gerät kann immer ausgeliehen werden; eine Sperre am Leser hält es wie beim Buch auf.
- **Sperren: ein Prüfweg für Buch, Gerät und Nachbuchen** (entschieden am 24.09.2026, `pruefeAusleihSperren` in `internal/service/ausleih_sperren.go`). Die Regel richtet sich danach, wer gesperrt hat:
  - **Sperre am Leser** — von Hand (Akte → „Schüler sperren", Begründung Pflicht) oder vom Programm für Ehemalige (Versetzung, LUSD-Import). An der Theke geht dann nur die Rückgabe. Aufgehoben wird sie in der Akte oder im Sperr-Dialog der Theke („Sperre aufheben"), beides mit dem Recht `edit_students` über dieselbe Tür (`PATCH /api/admin/students/{id}/lock`); Aufheben nimmt beide Sperren weg.
  - **Hinweise** — eine unbezahlte Forderung, gleich aus welchem Topf (ein Konto je Leser), und die Überfällig-Automatik: ab `UeberfaelligSperreAbMedien` Medien, die länger als `UeberfaelligKarenzTage` überfällig sind. Wer das Recht „Leser bearbeiten“ hat, übergeht sie einmalig („Einmalig ignorieren", `override_block`).
  - **Lernmittel** (Entscheidung der Schule vom 22.09.2026, Grundlage: Lernmittelfreiheit in Hessen): Beim Schulbuch hält nur die Sperre von Hand auf. Keine automatische Sperre, auch keine übergehbare — auch nicht die der Ehemaligen, und auch nicht bei der Verlängerung (entschieden am 28.09.2026). Dieselbe Regel fragen Einzelverlängerung und Frist von Hand (`PruefeSperreAmLeser`), Klassenverlängerung und LMF-Plan (`repository.sqlSchulbuchFristGehtMit`). Das gilt auch zum Schuljahresbeginn, wenn die Oberstufe die Schulbücher des Vorjahres erst nach ihrer Frist, dem 31. Juli, zurückgibt: Neue Schulbücher bekommt sie ohne Sperre und ohne Hinweis auf die überfälligen; hat sie dasselbe Buch noch, nennt die Theke es wie jedes Doppel (§18.6), ohne anzuhalten. Ein überfälliges Schulbuch zählt aber in der Überfällig-Automatik mit, die ein Bibliotheksbuch und ein Gerät anhält (übergehbar, siehe Hinweise), und es steht in der Mahnliste; gemahnt wird nur von Hand (§3).
  - **Kollegen** werden nie gesperrt (entschieden am 16.09.2026, bestätigt am 24.09.2026): keine Frist, keine Forderung, und der Knopf in ihrer Akte ist verschlossen. Eine ältere Sperre an ihrem Konto zählt nicht.
  - **Anonymisierte Datensätze** bekommen nichts, auch kein Schulbuch; ihre Sperre lässt sich nicht aufheben.
- **Protokoll:** Sperren und Aufheben stehen im Protokoll (`LESER_GESPERRT`, `LESER_ENTSPERRT`, mit dem Grund, der galt), das Übergehen eines Hinweises auch (`OVERRIDE_BLOCK`) — erst wenn die Ausleihe steht; weist danach noch das Ausleihlimit ab, steht dort nichts.
- **Den Sperrgrund** sieht an der Theke nur, wer `view_students` hat; sonst steht dort „Bitte an die Bibliotheksleitung wenden." (`ohneSperrgrund` in `internal/api/theke/eingabe.go`). Dasselbe gilt für die Antwort des Nachbuchens (`internal/api/theke/nachbuchen.go`); unter dem Schlüssel des Eintrags liegt das Ergebnis immer ohne den Freitext, weil eine wiederholte Portion unter einer anderen Anmeldung kommen kann. Die Nachbuch-Meldung trägt den vollen Grund und liegt hinter `view_students`.
- **Die Anzeige „gesperrt"** in Leserdatei und Akte folgt demselben Paar wie die Theke: Sperre von Hand oder Sperre des Programms, beim Kollegium nie (`ausleiheGesperrt` in `frontend/src/lib/components/leser/sperrStatus.js`). Gate: `frontend/e2e/sperrstatus-eine-wahrheit.spec.js`.

---

### 2.3. LMF-Plan: Büchertausch vor und Bücherausgabe nach den Sommerferien

Zwei Pläne, zwei Zeitpunkte (entschieden am 06.09.2026 — „Rückgabe" und „Ausgabe" allein waren
unklar): **Büchertausch vor den Sommerferien** (`art = rueckgabe`): alle Klassen geben ab
und bekommen direkt die neuen Bücher; Abschlussklassen (`AbschlussklasseSQL`) und Klassen,
deren nächster Jahrgang ein Eingangsjahrgang ist (die 6er → 7H/7R/7G), geben **nur
zurück** (`nurRueckgabeSQL`). Der Vorschlag belegt den Vermerk dieser Zeilen mit „nur
Rückgabe" vor (`service.BelegeVermerkNurRueckgabeVor`); danach gehört der Text der Bibliothek, und Planer,
Portal und PDF zeigen den Vermerk.
**Bücherausgabe nach den Sommerferien** (`art = ausgabe`): nur die neu gebildeten Klassen
der **Eingangsjahrgänge** (Einstellung `lmf_eingangsjahrgaenge`, Vorgabe „5, 7",
`EingangsjahrgaengeAus`); der Vorschlag enthält nur sie. **Entwurf und Veröffentlichung**
(`lmf_plaene.veroeffentlicht_am`): Speichern legt einen Entwurf an — zentral,
aber unsichtbar für `GET /api/lmf-termine`, das Portal-PDF und `LMFAbfragen.LadeRueckgabeTerminLage`
(ein Entwurf setzt auch beim Ausleihen keine Frist);
`POST /api/lmf-plan/{art}/veroeffentlichen` stempelt ihn und koppelt die Fristen; danach
gilt jede Speicherung sofort. Das Entwurfs-PDF für die Schulleitung liefert
`GET /api/lmf-termine/entwurf/pdf` (edit_books). Klassen werden weder angelegt noch
gelöscht: Der Name im Plan registriert sich per Trigger im Vokabular, und
eine Klasse ohne aktive Schüler verschwindet von selbst aus jeder Liste — der Planer
markiert sie „ohne Schüler" (`klassen` = Klassen mit Schülern in der GET-Antwort). **Wer
noch keine Zeile hat, steht nur im Planer** über der Tabelle („Noch nicht im Plan",
`LmfPlanVorrat.svelte`); das Portal zeigt den Plan, nicht die Lücken darin.

**Der Plan ist eine Reihenfolge, die der Server auf Schultage × Stunden verteilt,** keine Liste
einzeln angelegter Zeilen. Beim Büchertausch steht das Ende fest (Donnerstag vor den Sommerferien,
4. Stunde: „es endet immer am gleichen Tag“), und die Reihenfolge fließt rückwärts davor; bei der
Bücherausgabe steht der Beginn fest. Eine Zeile hat null bis viele Klassen; „Bücher setzen“ oder
„Nachzügler“ sind Zeilen ohne Klasse. Die Verteilung rechnet eine Stelle (`lmfplan.VerteileMit`,
`VerteileRueckwaerts`: Mo–Fr, Feiertage Hessen, freie Tage des Plans ausgespart, feste Plätze
umflossen); die Vorschau im Planer ist derselbe Aufruf, kein Zwilling in JavaScript. Die
Sommerferien kommen aus der Tabelle in `internal/pkg/lmfplan/sommerferien.go` (KMK bis 2030, ein
Test erinnert vor dem Ende).

Der **Vorschlag** für den nächsten Plan ist die Reihenfolge des Vorjahres plus neue Klassen
(Klassennamen bleiben, die Versetzung verschiebt Schüler); ohne Vorjahr Abschlussklassen zuerst,
dann Jahrgang absteigend. Jahrgang ab 11 oder ohne Ziffer gilt als Oberstufe und bleibt draußen;
was ein Plan bewusst auslässt, lässt auch der nächste aus. Speichern im selben Schuljahr ersetzt
den Plan, in einem anderen legt es einen neuen an. Portal und PDF zeigen nur den nächsten
veröffentlichten Plan, einen laufenden bis einschließlich zu seinem letzten Tag
(`LMFAbfragen.LadeNaechstenPlan`). Einzelne Termine haben keine eigene Route: Eine zweite Tür je
Zeile gäbe zwei Wahrheiten. Fundstelle: `internal/api/lernmittel/lmf_plan.go` (alle `edit_books`),
Seite _System → Schuljahreswechsel_; Tests `internal/pkg/lmfplan/verteilung_test.go`,
`internal/repository/lmf_termine_pg_test.go`, `internal/api/lernmittel/lmf_termine_frist_pg_test.go`,
`frontend/e2e/lmf-plan.spec.js`.

**Kopplung an die Fristen** (`internal/service/lmf_frist_kopplung.go`): Der Rückgabe-Termin einer Klasse ist die Frist ihrer Lernmittel. Beim Ausleihen
liest `bestimmeFrist` die Lage (`LMFAbfragen.LadeRueckgabeTerminLage`): Hat der Büchertausch
des laufenden Schuljahres heute oder davor begonnen — erster Tag des veröffentlichten
Rückgabe-Plans, gleich welche Klasse oder Zeile —, ist die Frist der Stichtag des folgenden
Schuljahres (Entscheidung 12.11.2026): Im Zeitraum des Tauschs holt niemand mehr ein Schulbuch
für das laufende Schuljahr. Das gilt für Klassen außerhalb des Plans, für Nachzügler, für ein
getauschtes oder nachgereichtes Buch und für ein Kind, dessen Termin noch bevorsteht. Davor ist
ein bevorstehender Rückgabe-Termin der Klasse die Frist (vor dem Stichtag; ein Mehrjahresband
bleibt beim Stichtag plus die verbleibenden Jahre, siehe unten). Ohne veröffentlichten Plan
gilt der Stichtag des laufenden Schuljahres. Beim Speichern eines Plans folgt
der Bestand (`service.KoppleLMFPlanFristen`): Zuerst bekommen die Schulbücher, die am oder nach
dem ersten Tag des Plans ausgegeben wurden, den Stichtag des folgenden Schuljahres
(`SetzeLernmittelFristAbTauschbeginn`; für einen Plan, der erst während des Tauschs
veröffentlicht wird oder dessen Beginn nach vorn rückt). Dann kehren Klassen, die aus dem Plan
fallen, zum Stichtag zurück — genau die Fristen, die auf ihrem alten Termin-Tag lagen —, jede Klasse des neuen
Plans bekommt ihren Termin (`SetzeLernmittelFristFuerKlassen`: Schüler außerhalb des
Papierkorbs, nicht anonymisiert, ohne Sperre von Hand — die Sperre der Ehemaligen zählt beim
Schulbuch nicht, §2.2 —; nur Lernmittel, nur Fristen im Schuljahr des Termins, Mahnstufe zurück).
Ausgabe-Pläne setzen keine Frist. PUT/DELETE melden `fristen_angepasst`.

## 3. Mahnwesen

Das Mahnwesen kennt eine Frage: Welche offenen Bücher von Schülern sind über die Frist
(`rueckgabe_frist`)? Kollegen werden nicht gemahnt, eine Dauerleihe wird nicht überfällig.

- **Liste:** `GET /api/mahnwesen` nennt je Klasse und Schüler die offenen Bücher mit abgelaufener Frist. Die Reiter „Akut fällig" (bis 14 Tage) und „Eskaliert" (darüber) teilen sie nach der längsten Überschreitung je Schüler; gerechnet wird in der Oberfläche (`berechneMahnstufe`). Ehemalige stehen als eigene Gruppe am Ende (`ehemalige` in der Antwort), gleich welche Klasse sie zuletzt trugen: Der Klassenname gehört nach der Versetzung einem anderen Jahrgang. Die Gruppe hat keine Klassenleitung; der Mahnlauf lässt sie aus.
- **Was hinausgeht:** der Mahnbrief je Schüler aus der Auswahl („Mahnbriefe drucken", `POST /api/admin/mahnungen/bulk-print`): ein Brief an die Eltern für das Fensterkuvert mit Anschrift und dem Text der Vorlage `MAHNUNG_ELTERN`; einen volljährigen Schüler schreibt er selbst an, mit der Vorlage `MAHNUNG_VOLLJAEHRIG`, wie Bescheid und Ersatzforderung (`sqlVolljaehrig`, ohne Geburtsdatum minderjährig). Ehemalige bekommen ihn wie alle über die Auswahl. Daneben, ohne zu zählen: je Klasse eine Mail an die Klassenleitung mit einem Blatt je Kind („Alle anmahnen", `POST /api/mail/send-bulk-overdue`) und der Ausdruck der Liste als Tabelle („Liste drucken"); er entsteht im Browser aus der geladenen Liste (`utils/mahnlisteDruck.js`).
- **Mahnstufe:** Nur der Druck der Mahnbriefe zählt sie (`ErhoeheMahnstufe`), höchstens einmal je Kalendertag der Schule und nur für ein Buch, das auf dem Brief steht: das offene Buch eines Schülers mit abgelaufener Frist. Ein zweiter Druck am selben Tag liefert den Brief und zählt nicht. Rückt die Frist in die Zukunft (Verlängerung, Termin aus dem LMF-Plan, Datum von Hand), fällt sie auf null. Sie steht an der Ausleihe (`ausleihen.mahnstufe`, `letztes_mahndatum`). Die Mahnliste zeigt je Schüler die höchste Zahl und den jüngsten Tag über seine Bücher („2× gemahnt, zuletzt 26.09.2026"), das Fenster für den Bescheid je Buch; eine aus Littera übernommene Mahnstufe hat keinen Tag. Das Programm knüpft weder eine Gebühr noch eine Sperre an sie.
- **Rechte:** Die Liste sehen und drucken verlangt `view_students`. Mahnbriefe drucken und der Mahnlauf verlangen `send_dunning` („Mahnungen versenden“): Der Druck zählt die Mahnung, der Lauf verschickt.
- **Sperre und Geld:** Was ein Kind an der Theke anhält, steht in §2.2 (überfällige Bücher nach Zahl und Dauer, offene Forderung, Sperre von Hand). Eine Forderung entsteht mit einer Verlust- oder Schadensmeldung und dem Bescheid (§14); Mahngebühren gibt es nicht.

---

## 4. Vormerkungen und Klassensatz-Reservierungen

Das System verwaltet den Mangel an verfügbaren Büchern durch zwei Konzepte:

### 4.1. Einzel-Vormerkungen

- Ein Schüler kann ein Buch vormerken, wenn kein Exemplar mehr frei ist; steht eins verfügbar im Regal, lehnt die Tür mit 409 ab („ausleihen statt vormerken“, `ErrVormerkungExemplarFrei`). Ein bestelltes oder ganz verliehenes Buch lässt sich vormerken. Vormerken können nur Schüler (entschieden am 24.09.2026): Die Oberfläche bietet zum Vormerken nur Schüler an, und die Tür lehnt einen Kollegen mit 409 ab (`ErrVormerkungNurFuerSchueler`).
- **Rückgabe-Match:** Wird ein Exemplar dieses Titels zurückgegeben, prüft das System, ob eine Vormerkung vorliegt.
- **Abholbereitschaft:** Das Buch wird nicht freigegeben, sondern direkt dem wartenden Schüler zugeteilt (Status `abholbereit`). Es landet physisch im Bereitstellungsregal.
- **Abholfrist** (entschieden am 28.09.2026): Das Buch liegt drei Tage bereit, gerechnet ab der Zuteilung — bei der Rückgabe oder wenn der Nächste nachrückt. Fällt das Ende auf ein Wochenende, einen Feiertag oder in die Ferien, gilt wie bei der Leihfrist der nächste Schultag, bis Tagesende (`repository.BestimmeAbholfrist`, §2.1). Kommt ein Buch kurz vor den Ferien zurück, liegt es also bis nach den Ferien bereit. Eine Einstellung dafür gibt es nicht. Danach verfällt die Vormerkung beim nächsten stündlichen Lauf, und das Buch geht an den Nächsten in der Warteschlange (`LasseAbgelaufeneVerfallen`).
- **Anderes Exemplar genommen:** Nimmt der Schüler ein Exemplar aus dem Regal statt des bereitgelegten, ist seine Vormerkung erfüllt. Das bereitgelegte bekommt der Nächste in der Warteschlange und bleibt im Fach; die Theke nennt Nummer und Nachrücker. Wartet niemand, nennt sie die Nummer, und das Exemplar geht zurück ins Regal (`repository.RueckeNachMitExemplar`).
- _Schutz:_ Es ist technisch unmöglich, dass ein Schüler ein Buch vormerkt, das er aktuell selbst ausleiht (Vermeidung von Monopolisierung).

### 4.2. Klassensatz-Reservierungen

**Reservieren heißt Anstellen, nicht Sperren** (entschieden am 16.08.2026):

- Eine Lehrkraft reserviert im Kollegiums-Portal Titel + Klasse + Anzahl. Die einzige
  harte Grenze: nicht mehr, als die Bibliothek physisch besitzt.
- Die Reservierung **blockiert nichts** — weder Exemplare noch Rückgaben. Sie ist ein
  Arbeitsauftrag an die Bibliothek und reiht sich in eine **sichtbare Warteschlange**
  ein (älteste zuerst).
- Das Portal zeigt bestehende Reservierungen **vor** dem Klick am Treffer
  („28 reserviert für 8a“) und verrechnet sie mit dem Regal
  („28 vorgemerkt · 2 rechnerisch frei“ — das OPAC-Abzeichen „N von M verfügbar“ sinkt
  durch eine Reservierung nicht, weil sie nichts bucht). Übersteigt die Wunschanzahl
  die rechnerisch freie Zahl, warnt das Formular vor dem Absenden („Reicht aktuell
  nicht — du stellst dich hinter 8a an“); wer trotzdem reserviert, erfährt in der
  Bestätigung, hinter wem sein Satz an der Reihe ist.
- **Gesucht wird im Katalog des Kollegiums** (`GET /api/reservierungen/klassensatz/katalog`,
  Recht `create_reservations`; `internal/api/portal/katalog_kollegium.go`): dieselbe Abfrage wie der
  öffentliche Katalog (§16.1, `repository.FindeImKatalog`), aber mit
  `repository.SQLKollegiumSichtbar` — jeder Titel mit einem Exemplar, das nicht ausgesondert ist.
  Das schließt Lernmittel ein und Titel, deren Exemplare bestellt und noch nicht eingetroffen
  sind: Reserviert werden vor allem Schulbücher, und der öffentliche Katalog verbirgt sie.
  Ein Titel, von dem nichts im Regal steht und etwas bestellt ist, trägt am Treffer
  „N bestellt" und lässt sich reservieren; stehen Exemplare im Haus und sind weitere bestellt,
  steht „N bestellt" neben dem Bestand. Zur Tür gehört der Filter nach Schlagwort
  (`?schlagwort_id=`, Liste über `…/katalog/filter`, §13); gibt es mehr als 50 Treffer, sagt
  das Portal über den Kopf `X-Treffer-Gesamt` „50 von 312 Treffern. Ein Suchwort grenzt ein.“
- Die Bibliothek arbeitet die Schlange unter Bestellungen → Klassensatz-Reservierungen
  ab: je Zeile Anfragende(r) mit Namen und der aktuelle Regal-Bestand („N verfügbar“).
  „Abschließen“ beendet den Vorgang nach der physischen Übergabe — die Ausleihe selbst
  läuft über die normalen Wege (Theke je Schüler oder Lehrer-Handapparat).
- **Die Abschluss-Notiz hat zwei Rückwege neben der Bereit-Mail** (scheitert die Mail oder
  fehlt die Adresse, bliebe die Zusage sonst unsichtbar): Die Theke zeigt „Zuletzt bereitgestellt“ (die jüngsten zehn,
  mit Notiz und Abschlussdatum), und das Portal zeigt der Lehrkraft unter
  „Bereitgestellt“ die EIGENEN abgeschlossenen Reservierungen samt „Bibliothek: …“
  (`GET /api/reservierungen/klassensatz/eigene`, dasselbe Muster wie beim Anliegen).
  „Deine Reservierungen“ im Portal meint die eigenen — die Warteschlange
  aller bleibt als Chip an den Suchtreffern sichtbar.

### 4.3. Klassensatz-Übersicht: zwei Quellen, zwei Bedeutungen von „Klasse“

Die Übersicht (`ladeKlassensatzGruppen`, `internal/repository/klassensaetze.go`) beantwortet die
Alltagsfrage „welche Bücher hat diese Klasse?“ — gebraucht vor allem für Nachzügler, die
mitten im Jahr dazukommen. Sie speist sich aus zwei Quellen, die verschieden altern:

- **Handliste (`class_books`, Quelle `hand`)** hängt am **Klassennamen**. „Die 7R2 liest
  Mathe 7“ gilt für jede künftige 7R2, deshalb lässt die Versetzung sie bewusst stehen
  (`internal/api/leser/schuljahreswechsel.go`). Sie veraltet nicht von selbst und muss von Hand
  gepflegt werden — dafür sieht sie auch Gruppen unter fünf Kindern.
- **Aus Ausleihen (Quelle `ausleihe`)** hängt an den **Kindern**: bei jedem Aufruf neu
  gerechnet über `klassen_normkey(schueler.klasse)`, nie gespeichert. Halten mehr als die
  Hälfte der aktiven Kinder einer Klasse und mindestens `klassensatzMindestLeser` (5)
  denselben Titel, ist er ihr Klassensatz. Gezählt werden die Klassen mit einem Jahrgang von 1
  bis 13 (`repository.JahrgangSQL`), auch die Einführungsphase, deren Name nicht mit einer Zahl
  beginnt: ET1 ist Jahrgang 11 und steht vor 12T1.

**Die Ableitung wandert mit den Menschen, die Handliste bleibt am Namen:** Nach dem Büchertausch
zeigt die abgeleitete Liste die neuen Bücher der 7R2 sofort unter _07R2_ (richtig für den
Nachzügler), nach der Versetzung ohne Zutun unter _08R2_; dazwischen stehen beide untereinander
(`internal/repository/klassensatz_ableitung_pg_test.go`).

Ein Titel erscheint je Klasse **einmal**, unabhängig von der Zahl der Exemplare; die
Leserzahl steht als Abzeichen an der Kachel. Ist ein Titel von Hand zugeordnet UND
abgeleitet, gewinnt `hand`.

**Auflagen (§13).** Die Ableitung zählt am Buch, nicht an der Auflage (`COALESCE(werk_id, id)` wie
der Bestellbedarf); sonst verschwände ein Klassensatz, der auf zwei Auflagen verteilt ist. Die
Kachel zeigt die Auflage, die die meisten Kinder haben, bei Gleichstand die neueste; der
Bestellbedarf zeigt dagegen die neueste, weil sie bestellt wird. Eine Zuordnung von Hand deckt das
ganze Buch ab. Haben Kinder eine andere Auflage als die der Kachel, trägt sie die Aufschlüsselung
(`auflagen`, die meisten zuerst; `internal/repository/klassensatz_auflagen_pg_test.go`).

### 4.4. Meldungen der Lehrkräfte

Bewusst schlank, kein Ticketsystem (entschieden am 18.08.2026):

- **Die Meldung, dass etwas nicht stimmt** („die 8G3 hat die falschen Bücher bekommen"). Im
  Kollegiums-Portal steht „Problem melden" am Treffer der Suche: Das Buch ist dann gewählt,
  sein Titel steht in der Meldung; dazu Klasse/Kurs und was nicht stimmt. Findet die
  Suche nichts, steht „Problem melden" ohne Buch unter „Keine Titel gefunden"; dort nennt die
  Lehrkraft selbst, worum es geht, und was im Suchfeld steht, ist als Vorgabe eingetragen. Die
  Beschreibung ist Pflicht, im Formular und am Server (`internal/api/portal/anliegen.go`). Gespeichert wird
  Text, kein Verweis auf einen Titel.
- **Einen Buchwunsch gibt es nicht.** Gemeint wäre damit in aller
  Regel eine Reservierung („Markl 2 für die 8G3"); sie geht über die Suche (§4.2), die auch
  Lernmittel und bestellte Titel findet. `POST /api/anliegen` nimmt nur Meldungen an.
  Ältere Wünsche stehen in der Liste der Bibliothek, bis sie abgehakt sind.
- Die Bibliothek arbeitet die Liste unter Bestellungen → „Meldungen" in
  Ruhe ab: Meldungen stehen über den Wünschen, je die ältesten zuerst. **Abhaken**
  schließt das Anliegen und schickt der Lehrkraft automatisch eine Mail — mit der
  optionalen Notiz („bestellt, kommt Anfang September"). Ein Doppelklick von zwei
  Arbeitsplätzen löst keine zweite Mail aus (gleiches Muster wie die
  Klassensatz-Bereit-Mail).
- Die Lehrkraft sieht ihre Meldungen samt Status und Erledigungs-Notiz im Portal, unter der
  Suche neben ihren Reservierungen. Einen eigenen Reiter dafür gibt es nicht; das Portal hat
  vier.
- Bewusst NICHT gebaut: Prioritäten, Kommentar-Threads, Genehmigungsketten,
  Deckungsprüfung, Packlisten — erst nachrüsten, wenn der Alltag sie vermisst.

---

## 5. Geräteausleihe (Hardware)

- **Verwaltung** im Medienkatalog, Bereich „Geräte“ (kein eigener Menüpunkt —
  ausgebucht wird ohnehin an der Theke): anlegen (Barcode zwingend mit `G-`-Präfix,
  sonst findet der Scan an der Theke das Gerät nicht), Liste mit aktuellem Ausleiher,
  Zubehör- und Stammdaten-Pflege, Defekt-Schalter (`ist_ausleihbar`).
- **Checklisten-Zwang an der Theke:** Trägt ein Gerät Zubehör (kommaseparierte Liste),
  unterbricht der Scan mit einem Bestätigungs-Dialog, der jedes Teil nennt —
  bei Ausleihe UND Rückgabe. Erst „Alles vollständig“ schickt den Scan mit
  Bestätigung erneut; Abbrechen bucht nichts.
- **Fristen:** „Leihfrist Geräte“ (`frist_geraete_tage`, Vorgabe 14 Tage), auf das Tagesende in
  der Schul-Zeitzone normalisiert wie Buch-Fristen; im Ferien-Leseclub wie ein Buch bis zu dessen Datum. Eine Verlängerung rechnet mit derselben Leihfrist (§2.1). Lehrkräfte leihen Geräte
  als Dauerleihe mit der Jahresfrist wie beim Buch.
- **Zustands-Sperre:** Defekte (`ist_ausleihbar = false`) und ausgesonderte Geräte
  verweigern die Ausleihe an der Theke.
- **Sperren des Lesers:** derselbe Prüfweg wie beim Buch (§2.2). Die Hinweise lassen
  sich auch am Gerät übergehen; die Zubehör-Liste reicht das Übergehen weiter, und
  protokolliert wird erst mit der Ausleihe.
- Fehlendes Zubehör bei der Rückgabe: Das Personal bricht im Dialog ab.
- **Verlust oder Schaden:** „Verlust oder Schaden melden“ an der Gerätezeile der Leserakte
  beendet die Ausleihe. Das Gerät bleibt in der Geräteverwaltung und ist nicht ausleihbar:
  beschädigt steht es dort als defekt und lässt sich nach der Reparatur wieder freigeben,
  verloren steht es als verloren; taucht es wieder auf, holt _Wieder freigeben_ es in den Schrank
  zurück und storniert die Forderung „nicht zurückgegeben“ wie die Rückkehr eines Buchs
  (`repository.HoleVerlorenesGeraetZurueck`, Eintrag im Logbuch). Die Forderung entsteht wie
  beim Buch mit dem eingetragenen Betrag, beim Kollegium nicht; bezahlt wird in der Akte. Auf den
  Bescheid des Landes kommt ein Gerät nicht.

---

## 6. Inventur-System

Die Inventur findet im laufenden Betrieb statt, ohne dass die Bibliothek zwingend schließen muss.

- **Session-basiert:** Jede Inventur erhält einen Bereich — die ganze Bibliothek, eine Signatur oder Fach und Klasse (im Code `scope`) — und läuft in einer eigenen Session (`inventur_sessions`). Mehrere Mitarbeiter können mit Handscannern parallel inventarisieren, ohne sich gegenseitig zu überschreiben. Ein Buch, das zu zwei zugleich offenen Läufen gehört (ein Band für die Jahrgänge 5 und 6), muss nur in einem gescannt werden; ein Lauf, der vor dem Beginn des anderen abgeschlossen war, zählt dafür nicht. Ein abgeschlossener Lauf, dessen Scans ein offener noch mitzählt, lässt sich erst nach dessen Abschluss löschen (409).
- **Fehlmengen-Ausbuchung:** Wird die Session beendet (`Finish`), vergleicht das System alle gescannten Exemplare mit dem theoretischen Bestand in diesem Bereich.
- **Erwartet ist, was seit dem Beginn im Regal stehen musste:** vorhanden und ausleihbar, nicht für eine Vormerkung im Abholfach, beim Beginn nicht verliehen und seitdem weder verliehen noch zurückgegeben, entsperrt, wieder aufgenommen oder neu angelegt (`repository.InventurBereich`, eine Bedingung für Zählung und Abschluss). Ein Buch beim Schüler kann niemand scannen, eins im Abholfach sucht niemand im Regal; ein Buch, das erst während der Inventur ins Regal kam, beweist nicht gescannt keinen Verlust. Nicht erfasst ist ein Titel, dessen Signatur, Fach oder Jahrgang während der Inventur in den Bereich wechselt: Seine ungescannten Bücher bucht der Abschluss als Verlust, und sie stehen im Fehlbestandsbericht wie jedes übersehene Buch; ein Kreuz unter „Gefunden" oder ein Scan an der Theke holt sie zurück. Eine Liste der erwarteten Bücher beim Beginn, die den Fall schlösse, ist bewusst nicht gebaut: Er braucht eine solche Änderung genau während der Inventur, und der Bericht fängt ihn auf.
- Fehlende erwartete Exemplare erhalten automatisch den Status `VERLUST`. Fortschritt („gefunden / erwartet") und Rückfrage vor dem Abschluss zählt der Server nach derselben Bedingung (`ZaehleStand`); die Rückfrage nennt die Zahl, die der Abschluss bucht. Ein gescanntes Buch, das der Abschluss nicht erwartet (gesperrt, verliehen), zählt nicht als gefunden.
- **Verliehenes Buch im Regal:** Der Scan bucht es zurück wie die Theke, über denselben Dienst (`AusleihDienst.NimmZurueck`: Ausleihe beenden, Vormerkung bedienen, Protokoll, Meldung an die Arbeitsplätze). Ist der Titel vorgemerkt, sagt die Karte „ins Abholfach legen"; einen Alarm, der das Scanfeld sperrt wie an der Theke, gibt es hier nicht, weil der nächste Scan schon eintrifft. Wer an der Theke nicht zurücknehmen darf (`perform_actions`), bekommt nur den Hinweis.

---

## 7. Bestellwesen und Wareneingang

Die Software verwaltet Bedarfe und Lieferungen:

- **Bestellbedarf:** Ein Schulbuch steht auf der Liste, wenn sein Bestand nicht für die Kinder reicht,
  die es im nächsten Schuljahr brauchen; die Menge ist der Unterschied (`fehlen`), und das Plus der
  Zeile legt genau sie in den Warenkorb. Eine feste Schwelle oder Reserve gibt es nicht. **Bedarf:**
  Welcher Jahrgang ein Buch braucht, zeigen die Ausleihen dieses Schuljahres, offen oder zurückgegeben;
  das Schuljahr einer Ausleihe ist das ihrer Frist, so zählt der Büchertausch im Juni die alten Bücher
  weiter und die neuen erst im nächsten Schuljahr. Ein Mehrjahresband zählt, solange das Kind in seiner
  Spanne ist. Je Jahrgang gilt im nächsten Schuljahr derselbe Anteil wie heute, gerechnet auf die
  Kinder, die aus dem Jahrgang darunter nachrücken (ohne Abschlussklassen, `AbschlussklasseSQL`); hat
  der Jahrgang darunter kein Kind (Jahrgang 5), gilt die Zahl von heute. So zählt ein Buch für einen
  Zweig oder ein Wahlfach mit seinem Anteil; ein neues Lehrwerk kommt erst nach seiner ersten Ausgabe
  in die Rechnung, es wird ohnehin in genügender Zahl bestellt. **Bestand:** alle nicht ausgesonderten
  Exemplare, auch verliehene und bestellte; gezählt werden nur Lernmittel. Die Antwort von
  `GET /api/bestellungen` trägt neben den Zeilen die Zahl der aktiven Kinder (`kinder`), der Kinder mit
  einem Schulbuch in diesem Schuljahr (`kinder_mit_schulbuch`) und `ausgeschaltet`; daran unterscheidet
  die Seite „Kein Schulbuch fehlt“ von „Noch keine Grundlage“ und „Bestellbedarf ausgeschaltet“, und
  der Kopf nennt, mit wie vielen Kindern gerechnet wurde, denn ohne Ausgabe über das Programm ist der
  Bedarf zu niedrig. Auflagen desselben Werks bilden eine Zeile, gezeigt wird die neueste, weil sie
  bestellt wird (§13). Die Bestellliste als PDF liest dieselbe Quelle (`bestellbedarfMitEinstellungen`).
  Belegt: `internal/repository/bestellbedarf_schuelerzahl_pg_test.go`,
  `internal/api/bestellung/bestellbedarf_test.go`, `internal/api/bestellung/bestellbedarf_pdf_pg_test.go`,
  `BedarfZeile.test.js`, `frontend/e2e/auflagen-bestellbedarf.spec.js`.
- **Fokus auf Lernmittelfreiheit (LMF):** Die Bedarfsvorschläge sind standardmäßig auf LMF-Medien (Schulbücher) gefiltert. Freihand-Exemplare (Lese-Einzelstücke) werden in der Regel nicht nachbestellt.
- **Eine Bestellung = ein Topf:** Jede Bestellung trägt, aus welchen Mitteln sie bezahlt wird — Lernmittelfreiheit (Land), Schülerbücherei (Schulträger) oder Spende (ohne Topf, etwa Geld des Fördervereins; Migration 182). Der Händler gewährt darauf verschiedene Nachlässe, und die Rechnungen gehen getrennte Wege; beide gehen an die Schule (Auskunft der EDV-Servicestelle für Schulbibliotheken, 10.09.2026). Der Warenkorb gruppiert seine Positionen nach dem Vorschlag aus `ist_lernmittel`, eine Position lässt sich in jeden anderen Topf schieben (falsch gekennzeichneter Titel, Spende), und „Bestellung auslösen" erzeugt je Gruppe eine Bestellung an denselben Händler, je mit eigener Mail und eigenem Anschreiben. Eine Spende gilt beim Händler, auf dem Etikett und beim Ersatz wie die Schülerbücherei (`mitteltopf.WieSchuelerbuecherei`): dieselbe Kundennummer, die Rechnung an die Schule, derselbe Eigentumsvermerk und kein Bescheid; eigen sind ihr Vermerk und ihre Liste im Zugangs- und im Abgangsbuch. Der Vermerk steht in der Betreffzeile und im Text des Anschreibens (`internal/pkg/mitteltopf`) und in der Händler-Mail (Platzhalter `{{.Mittel}}`; fehlt er in der Vorlage, hängt der Versand den Vermerk an). Ein Lieferant kann eine zweite Kundennummer für die Schülerbücherei führen; sie gilt auch für die Spende. Alt-Bestellungen sind zugeordnet, wo es aus den Positionen eindeutig war; gemischte stehen als „ohne Zuordnung". Ein über die DNB neu angelegter Titel wird im Staging-Fenster gefragt, ob er Lernmittel ist. Der Topf einer bestehenden Bestellung lässt sich im Bestelldetail mit Pflicht-Grund korrigieren (`PUT /api/bestellungen/{id}/mittel`, protokolliert im Verwaltungsprotokoll); das Anschreiben beim Händler ändert sich dadurch nicht — die Korrektur gilt der eigenen Zuordnung, aus der die Berichte rechnen. Belegt: `internal/api/bestellung/bestellung_mittel_pg_test.go`, `internal/api/bestellung/bestellanschreiben_test.go` (Inhaltsstrom des Anschreibens), `internal/db/bestellung_mittel_nachtrag_pg_test.go`, `internal/api/bestellung/bestellung_mittel_korrektur_pg_test.go`.
- **Exemplare entstehen beim BESTELLEN, nicht beim Eintreffen:** Mit dem Absenden legt das System die Exemplare samt Barcode-Nummern an und markiert sie als „Im Zulauf". Nur so kann der Barcodebogen mit der Bestellung mitgehen. Der Wareneingang bucht diese Exemplare später frei — er erzeugt sie nicht.
- **Hauptlieferant** (Lieferantenverwaltung, ein Schalter; höchstens ein Händler, die Datenbank erzwingt es): Er ist im Bestellformular vorausgewählt, beklebt die Bücher selbst und bekommt statt der reinen Bestellmail einen Link (nächster Punkt). Seine Exemplare entstehen als etikettiert und stehen nicht auf der Nachdruck-Liste. Ein Merkmal statt drei Schaltern, weil alle drei Folgen denselben Händler beschreiben (`IstHauptlieferant` in `internal/repository/lieferant.go`). Wird ein anderer Händler Hauptlieferant, ist der bisherige ein gewöhnlicher; alle anderen Lieferanten bekommen nur die Bestellmail.
- **Bestätigungs-Link an den Hauptlieferanten:** Mit der Bestellmail kommt ein Link auf eine Seite
  ohne Anmeldung, auf der der Lieferant die Etiketten dieser Bestellung klein oder groß druckt (wie im
  Mailanhang) und die Bestellung **einmalig** bestätigt. Die Bestätigung kommt ohne Mail an:
  Bestellhistorie, geöffnete Bestellung und Wareneingang zeigen denselben Chip (`BestellStandChip`):
  „Mail nicht versendet“, „Bestätigt“ mit Tag oder „Wartet auf Händler“; der Altbestand ohne Bestellung
  trägt keinen. Bestätigt der Händler oder trägt die Bibliothek die Zusage nach, wechselt der Stand in
  allen offenen Fenstern von selbst, ohne Meldung, und ein Bildschirmleser liest den Wechsel vor (WCAG
  4.1.3); was bei unterbrochener Leitung geschah, holt das Fenster beim Wiederaufbau nach, ein
  gescheitertes Holen nach höchstens 15 Sekunden. Andere Wechsel des Chips, etwa nach „Erneut senden“,
  stehen an einem zweiten Arbeitsplatz beim nächsten Öffnen des Reiters da.
- **Bestellmail nicht versendet:** Scheitert die Mail an den Lieferanten oder fehlt der Mailserver, ist
  die Bestellung trotzdem gespeichert; die Bestellhistorie nennt „Mail nicht versendet“, die Bestellung
  den letzten Versuch und den Knopf „Erneut senden“ (`create_orders`). Der erneute Versand schickt
  dieselbe Mail aus der gespeicherten Bestellung an die heutige Adresse des Lieferanten; mit
  Bestätigungsschritt bekommt sie einen neuen Link, der alte wird ungültig. Ohne den Vermerk lässt sich
  nichts erneut senden, damit keine Bestellung doppelt beim Händler ankommt. Der Vermerk gilt, bis der
  Händler bestätigt, die Bibliothek seine Zusage nachträgt oder ein Exemplar aus der Bestellung den
  Wareneingang durchläuft; dass der Link in der gescheiterten Mail stand, nennt die Bestellung auch nach
  der Lieferung. Hat der Händler auf anderem Weg bestellt, entfernt „Auf anderem Weg bestellt“ den
  Hinweis nach einer Rückfrage, ohne zu senden (entschieden am 07.10.2026; Protokoll), nur an
  Bestellungen ohne Bestätigungs-Link, mit Link wird die Zusage nachgetragen. Danach lässt sich die
  Bestellung nicht mehr aus dem Programm senden; die Rückfrage sagt das, ihr Fokus steht auf
  „Abbrechen“. „Gesamtanzahl Exemplare“ in der Mail ist die bestellte Menge, auch für Positionen ohne
  Vorab-Barcode.
  - _Klein:_ Bogen für vorgestanztes Etikettenmaterial. Der Lieferant **wählt das Bogenraster selbst** (Zweckform L4760 3×7, Avery 3475 3×8, Kleine Barcodes 4×13) — er druckt auf sein eigenes Material, und davon gibt es verschiedene. Das gewählte Raster steht anschließend in der Bestellhistorie der Schule.
  - _Groß:_ Lernmittel-Etikett mit Ausleihtabelle, **vier Stück auf einem A4-Blatt** (2×2, mit Schnittlinien) — wird ausgeschnitten, nicht auf vorgestanztes Material gedruckt. Die Bestätigung erscheint automatisch in der Bestellhistorie und ist dort von einem manuellen Nachtrag aus der Bibliothek unterscheidbar. Voraussetzung ist die **Öffentliche Adresse** in den Einstellungen — fehlt sie, geht die Bestellung ohne Link raus. Sicherheitszuschnitt: 256-Bit-Token, in der Datenbank nur als Hash, 21 Tage gültig (Vorgabe; einstellbar unter Bestellwesen, `bestelllink_gueltigkeit_tage`, ein laufender Link behält sein Ablaufdatum), jederzeit durch einen neuen ersetzbar (der alte stirbt dabei). Details in [SECURITY.md](SECURITY.md).
- **Historie zeigt die neuesten 200 Bestellungen** (Grenze serverseitig, max. 500). Die Kennzahlen im Kopf — Gesamtausgaben, Exemplare, wartende Bestätigungen — zählen dagegen **alle** Bestellungen (`/api/bestellhistorie/uebersicht`), damit aus einer Teilsumme keine vermeintliche Gesamtsumme wird. Ohne die Grenze lieferte die Historie auf einer gewachsenen Datenbank 2,45 MB in 3,9 Sekunden, mit ihr 0,10 MB in 0,17 Sekunden.
- **Nach Topf getrennt:** Der Bestellbericht (`GET /api/bestellhistorie/bericht`) steht in Blöcken — Lernmittelfreiheit (Land), Schülerbücherei (Schulträger), Spende (ohne Topf), zuletzt die Alt-Bestellungen ohne Zuordnung —, je mit eigener Summe und der Gesamtsumme darunter; `mittel=` beschränkt ihn auf einen Topf, und die Überschrift nennt ihn. Die Historie filtert serverseitig nach Topf (`GET /api/bestellhistorie?mittel=land|schultraeger|spende`), ein unbekannter Wert ist 400; einen Filter für Bestellungen ohne Topf gibt es nicht, jede Bestellung wird mit Topf angelegt. `/api/bestellhistorie/uebersicht` liefert die Kennzahlen zusätzlich je Topf (immer alle vier, auch mit Null); der Kopf der Historie zeigt sie, sobald mehr als ein Topf Bestellungen hat. Bericht und Historie lesen den Parameter über dieselbe Stelle (`internal/repository/mittel_filter.go`).
- **Preis laut Rechnung:** Den Einzelpreis einer Position ändert, wer bestellen darf, jederzeit nach der Bestellung (`PUT /api/bestellungen/{id}/positionen/{position_id}/einzelpreis`, „Preis ändern" in der Bestellung). Mit ihm ändern sich der Einkaufspreis der Exemplare dieses Titels aus der Bestellung und die Summe der Bestellung, in einer Transaktion (`repository.AenderePositionspreis`); der Eintrag `BESTELLPREIS_GEAENDERT` im Protokoll nennt alten und neuen Preis. Anlass: Der Preis beim Bestellen stammt oft vom Fachbereich und ist manchmal veraltet, und der Händler rechnet auf einer Papierrechnung mit Rabatt oder neuem Preis ab. Der Listenpreis am Titel bleibt; er ist der Neupreis für Ersatz.
- **Preise sind optional:** Ist „Preise erfassen" aus, arbeitet das ganze Bestellwesen ohne Geldbeträge — kein Preisfeld, keine Betragsspalten, Berichte zählen Exemplare statt Euro zu summieren.

---

## 8. LUSD-Synchronisation & Datenschutz (DSGVO)

### 8.1. Der LUSD-Import

Die LUSD ist das führende System für Schülerdaten; der Import überschreibt Namen, Klassen und
LUSD-IDs. Wie er Schüler ohne Schüler-ID wiedererkennt (drei Stufen, die die Datei bestimmt),
welche Dateien und Kopfzeilen er liest, wie er Umbenennungen paart und wie zwei Datensätze von
Hand zusammengeführt werden (`merge_students`), steht in [LUSD.md](LUSD.md) §1 bis §5. Hier steht,
was dort fehlt:

- In der Stufe Name + Geburtsdatum ist das Datum in jeder Zeile Pflicht; fehlt es, bricht der
  Import ab, statt still zu überspringen. Bestandsschüler ohne Geburtsdatum meldet die Vorschau als
  „nicht abgleichbar“. Im ID-Modus übernimmt der Import Bestandsschüler ohne ID (Handanlage und
  Littera-Übernahme, `lusd_id='littera:…'`) über Name und Geburtsdatum, statt sie zu verdoppeln.
  Blätter ohne Kopfzeile (Deckblatt) überspringt er, die Spaltenfolge darf je Blatt abweichen.
- **Klassen-Vokabular:** Jede Klasse steht genau einmal in `klassen`; Schüler, Bücherlisten,
  Klassenleitungen und LMF-Termine verweisen darauf, Schreibvarianten („5 a“, „05A“) laufen per
  Trigger auf denselben Eintrag. Die Anzeigeform ist fest: Jahrgang zweistellig, Rest groß
  („05F1“); Sonderwerte (`lehrer` als Handapparat-Entleiher, `ABG`) und Kursnamen bleiben, wie sie
  sind. Die Klasse einer Klassensatz-Reservierung ist Freitext: Ein Tippfehler im Portal legt keine
  Klasse an, und eine Lehrkraft kann einen Kurs eintragen.
- **Handanlage:** Das Geburtsdatum ist Pflicht (Oberfläche und Tür), denn es ist die einzige
  Brücke zum späteren Import; ohne Datum entstünde ein Duplikat.
- **Kontaktdaten:** Anschriften und Eltern-Mails, soweit die Datei sie trägt, dienen ausschließlich
  Rechnung und Mahnung. Ein gefüllter Exportwert überschreibt den Bestand, ein leerer lässt ihn
  stehen; Handkorrekturen überleben den nächsten Import nicht (Entscheidung 02.09.2026).

### 8.2. DSGVO und Lösch-Routinen (Ehemalige)

- Fehlt ein bestätigter Schüler im LUSD-Export, gehört er zu den **Ehemaligen**
  (`ist_abgaenger = true`): Er ist weg. „Abgänger“ heißen nur die Abschlussklassen, die noch da
  sind (Abgänger-Ansicht, §8.x); der Code sagt für beide „abgaenger“.
- **Karenzzeit und Sperre:** Ein Ehemaliger wird gesperrt, nicht sofort anonymisiert; der nächtliche
  Job anonymisiert ihn nach der Karenzzeit (`abgaenger_karenz_tage`, Vorgabe 90 Tage, 0 = sofort),
  sofern nichts offen ist. Uhr (`repository.sqlKarenzUhr`), Gründe der Sperre und warum 90 Tage:
  [LUSD.md](LUSD.md) §4. Die
  Sperre lässt sich in der Akte aufheben (§2.2); ein späterer Import sperrt den Ehemaligen nicht
  wieder, denn er behandelt nur aktive Schüler als neue Ehemalige (`sammleAbgaenger`), und die
  Karenz läuft unabhängig von der Sperre weiter. Die endgültige Löschung ab dem 30. Januar des
  Folgejahres trifft nur anonymisierte Datensätze; im Cron läuft die Anonymisierung deshalb vor der
  Löschung (`fuehreNaechtlicheDSGVOAus`).
- **Was die Anonymisierung leert** (Stammdaten, Foto, Ausweisnummer → `ANON-…`, die Spuren in den
  Neben-Tabellen) und die übrigen Lösch-Läufe stehen in [SECURITY.md](SECURITY.md), „Automatisierte
  Löschroutinen“; eine `A-`-Nummer bleibt als bloße Zahl ohne Person gesperrt (§12).
- **Papierkorb:** Löschen in der Akte (Recht „Leser löschen“; bei einem Kollegen mit Zugang zu „Mein
  Portal“ zusätzlich „Benutzer & Rechte verwalten“, für ein Admin-Konto ein Admin) legt den Leser in
  den Papierkorb; Ausleihhistorie und Name bleiben für ein Zurückholen. Nach 180 Tagen löscht ihn der
  nächtliche Lauf endgültig, einen Schüler in derselben Nacht vorher anonymisiert; offene Ausleihen
  und unbezahlte Forderungen halten beides. Für einen Ehemaligen im Papierkorb läuft die Karenz weiter
  und kann früher enden; der Löschdialog sagt deshalb „bis zu 180 Tage“.
- **Lesehistorie ist befristet** (entschieden am 29.09.2026): Eine zurückgegebene Ausleihe verliert
  ihren Leser nach der Frist, Schülerbücherei 1 Tag, Lernmittel 730 Tage, 0 = aus; Gründe, Quellen
  und Ausnahmen in [SECURITY.md](SECURITY.md), „Automatisierte Löschroutinen“. Schülerprofil und Titel-Historie zeigen ältere
  Ausleihen als „anonym“; ein später bemerkter Schaden an einem Buch der Bücherei lässt sich keinem
  Ausleiher mehr zuordnen. Dieselbe Frist der Schülerbücherei nimmt der Protokollzeile einer
  Vormerkung, die mit ihrem Titel gelöscht wurde, den Leser
  (`repository.praedikatLesehistorieVormerkspur`); die Zeile einer gelöschten Forderung behält ihn.

---

### 8.x Abgänger-Ansicht und Kontoauszug

Die Abgänger-Ansicht (`/abgaenger`) zeigt die **Abschlussklassen mit noch offenen Ausleihen**:
H-Zweig ab Jahrgang 9 (9H, auch das freiwillige 10H), R-Zweig ab 10, sonst ab 13 —
`repository.AbschlussklasseSQL`, dieselbe Regel, mit der die Versetzung Ehemalige markiert und
ihre Klassenleitungs-Zuordnung entfernt (Paar-Gate `TestAbgaengerliste_UndVersetzungSehenDieselbeMenge`,
Regel-Tabelle `TestAbschlussklasseSQL_Regel`). Eine Klasse, die mit keinem Jahrgang von 1 bis 13
beginnt (14A, eine Buchnummer im Feld Klasse, die Einführungsphase ET1 bis ET3), ist keine
Abschlussklasse: Die Versetzung lässt ihre Schüler stehen und nennt sie in der Vorschau. Sie sind noch an der Schule (`ist_abgaenger = false`)
und leihen weiter aus. Sichtbar ist die Liste in der **Saison 01.05.–31.07.** (fester Wert,
`internal/api/mahnwesen/abgaenger_fenster.go`; `Server.Uhr` nur für Tests): außerhalb antwortet `GET /api/abgaenger`
mit `fenster.offen = false` und leerer Liste, der Druck mit `404`, der Versand mit `409`. Je Klasse
lässt sich ein **Kontoauszug** drucken (`ladeAbgaengerKontoauszuege`) oder per Mail an die
Klassenleitung schicken (`POST /api/abgaenger/mail`); die Empfängeradresse kommt aus den
Klassenleitungen (§17, Einstellungen → Klassenleitungen), Klassen ohne Zuordnung werden im Versand-Dialog als „keine E-Mail"
markiert, statt still übersprungen zu werden.

Wer weg ist (`ist_abgaenger = true`, weg laut LUSD oder durch die Versetzung), steht in der Leserdatei unter
„Ehemalige" (`GET /api/schueler?status=ehemalige`, `LeserAbfragen.LadeEhemaligenListe` — dieselbe
Liste und Serversuche wie „Aktive Schüler" mit umgekehrtem Vorzeichen), mit der Zahl seiner offenen und überfälligen Bücher,
auch ohne Bücher, bis er anonymisiert wird. Eine anonymisierte Zeile steht in keiner Liste und findet auch die Suche der
Theke nicht (`FindeUnscharf`); sie wartet nur auf die endgültige Löschung. Im
Mahnwesen steht er, sobald die Frist eines Buchs abgelaufen ist (`MahnwesenAbfragen.LadeUeberfaelligeNachKlasse`),
in der Gruppe der Ehemaligen (§3); das Mahnwesen kennt nur die Frist.

## 9. Druck-Center und Ausweise

Das Druck-Center hat vier Reiter (`DruckCenter.svelte`): _Buch-Etiketten_, _Fehlende
Etiketten_, _Ausweis-Design_ und _Ausweise je Klasse_.

- **Buch-Etiketten:** PDF-Bögen mit Strichcode (Code 128) oder QR-Code für vorgestanzte
  Etikettenbögen, in den drei Rastern aus §7, mit Startposition für einen angebrochenen
  Bogen (`POST /api/print/labels`). „Barcodes drucken" in der Maske eines Titels öffnet
  diesen Reiter mit dem Titel und seinen Exemplaren, wie der Etikett-Knopf der Exemplarkarte.
- **Fehlende Etiketten:** die Exemplare, deren Etikett noch nicht gedruckt ist
  (`etikett_gedruckt = false`, §13); ihre Zahl steht am Menüpunkt und am Reiter.
- **Das große Lernmittel-Etikett** mit Ausleihtabelle entsteht mit der Bestellung, als
  Mailanhang an den Hauptlieferanten und auf der Seite des Händlers (§7), und für Bücher aus
  jedem Topf: Wer druckt, wählt die Größe, auch für einen Klassensatz der Schülerbücherei oder
  einer Spende. Den Eigentumsvermerk wählt es nach dem Topf des Exemplars. Aufbau von oben nach unten (`internal/pdf/etikett_lernmittel.go`): Schulname, Titel, Anschaffungsjahr mit
  Signatur, Strichcode, Exemplarnummer, Eigentumsvermerk, darunter die Tabelle Schuljahr ·
  Name des Schülers · Klasse · Zustand mit sechs Zeilen zum Ausfüllen von Hand.
- **Nachdruck für ein Buch aus Littera** (entschieden am 30.09.2026): das Etikett dieses
  Programms, Code 128 mit den 13 Ziffern der Nummer darunter. Ein Nachbau des alten Etiketts
  ist es nicht; der Scanwert ist derselbe. Der Etikett-Knopf der Exemplarkarte steht bei jeder
  Nummer außer den Platzhaltern `AUTO-…` und `SYS-…` und übergibt an das Druck-Center.
- **Lernmittel mit LMF-Nummer** (entschieden am 22.10.2026): Jedes Exemplar trägt eine kurze
  Nummer wie in Littera, auch ein Lernmittel mit LMF-Nummer (`LMF-`, ISBN-13, laufende Nummer).
  Die LMF-Nummer bleibt als altes Etikett am Exemplar (`lmf_barcode`): Das Etikett scannt weiter,
  an der Theke, im Medienkatalog, in der Inventur und in der Theken-Auskunft. Die Bestandsliste
  als CSV nennt sie in der Spalte „LMF-Nummer": Bis zum Überkleben fände ein anderes Programm
  das Buch nur über sie. Das Druck-Center
  zeigt und druckt die kurze Nummer aus `barcode_seq` ohne Rückfrage; ist ein Etikett kaputt, wird
  es neu gedruckt und überklebt. Der Bestands-Import legt solche Exemplare gleich so an (§13). Als
  Nummer eines Exemplars nimmt die Datenbank keine LMF-Nummer an; der Barcode-Editor nennt die Ablehnung.
- **Ausweis-Design:** Der Reiter ist der Designer — Vorlagen, Vorder- und
  Rückseite, Elemente wie Name, Gültigkeit, Logo, Passbild, Strichcode und Farbfläche. Das
  Design gilt für alle Arbeitsplätze: Es liegt am Server (`/api/ausweis-layout`), speichern
  verlangt `manage_settings`. Wer dieses Recht nicht hat, sieht den Designer gesperrt und kann
  ihn ansehen und testdrucken. **Sicherer Rand:** Die Kartendrucker der Schule (Datacard Sigma DS1
  und DS2) drucken bis an den Rand, lassen dort aber einen schmalen weißen Streifen frei; für
  Strichcodes verlangt Datacard 6,3 mm Abstand zum Rand. Der Designer zeigt 3 mm innen eine
  gestrichelte Linie, die nicht mitdruckt (`designer/kartenRand.js`); verschieben lässt sich
  weiter bis an den Rand. Beim Ziehen rastet eine Kante ein, die ihrer Linie auf 1 mm nahe kommt,
  beim Strichcode an der Grenze seiner Striche, die die Linie dann zeigt; weiter gezogen läuft
  das Element frei darüber hinaus. Steht ein Element zu nah am Rand, ist die Linie rot, und der
  Satz unter der Karte nennt den Grund; Farbflächen und Motive über die ganze Breite zählen
  nicht. Die Linie folgt dem Rand eines Textfelds in Material 3: Outline, beim Einrasten Primary
  in doppelter Stärke, sonst Error (`designer/randLinie.js`). Was in Standard-Design und Vorlagen außen steht, steht bündig an der
  Linie, damit alle denselben Rand haben; ein Strichcode unten steht 0,3 mm darüber, so enden
  seine Striche 6,3 mm über der Kante (`designer/kartenRand.test.js`). **Der Designer legt fest,
  wie gedruckt wird, die Leserdatei, wer.** Die
  Betriebsart ist _Karte_ (Kartendrucker, Druck aus dem Browser) oder _Etikett_
  (Klebeetiketten mit Name, Klasse und Strichcode als PDF vom Server,
  `POST /api/print/schueler-etiketten`; die Anfrage trägt nur Kennungen, die Namen holt der
  Server). Das Passbild kommt aus der Theke (§18.2; die LUSD liefert keine), die Aufschrift
  richtet sich nach der Art des Lesers (§12.3). Einen A4-Bogen mit acht Karten zum Ausschneiden
  gibt es nicht (§20).
- **Ausweise je Klasse:** Klasse wählen, und die Ausweise all ihrer Schüler gehen in den Druck,
  als Karte oder auf dem Etikettenbogen ab einem wählbaren Feld, wie unter _Ausweis-Design_
  eingestellt. Wer keine Ausweisnummer hat, fällt heraus; die Zeile unter der Auswahl nennt, wie
  viele Schüler es sind und wie vielen die Nummer fehlt. Gedruckt wird über denselben Baustein
  und mit derselben Steuerung wie in der Leserdatei (`ausweisdruck.svelte.js`,
  `AusweisDruckSteuerung.svelte`): Warnung ohne Ablaufjahr, erstes Feld des Bogens, Druckknopf.
  Einzelne Schüler druckt die Leserdatei.

Wie die Karte auf Bildschirm und Papier gezeichnet wird und wie die Wege zum selben Etikett
zusammengehalten werden, steht in [ARCHITEKTUR.md](ARCHITEKTUR.md) 8.9.

---

## 10. System-Audit & Protokollierung

Um Nachvollziehbarkeit bei sensiblen Schuldaten zu garantieren, gibt es ein Audit-Log:

- Jede administrative Aktion (Benutzer gelöscht, Schadensfall storniert, Leser gesperrt oder entsperrt) wird in der Tabelle `audit_logs` mit `Akteur`, `Zeitstempel`, `IP` und Vorher-/Nachher-Details protokolliert. Sperren und Entsperren heißen dort `LESER_GESPERRT` und `LESER_ENTSPERRT`.
- Das Audit-Log ist **append-only als Konvention**: Kein Bedien- oder Codepfad ändert oder löscht Einträge außer der DSGVO-Tilgung (siehe unten). Einen Trigger, der jede Änderung sperrt, gibt es nicht, weil die Löschpflicht (Art. 17) das Protokoll ändern muss, um den Personenbezug zu entfernen.
- Die Daten dienen der Fehlerbehebung und DSGVO-Rechenschaftspflicht.
- **Theken-Auskunft** (`internal/api/system/theken_auskunft.go`): der EINE
  zweckgebundene Leseweg in die Detail-Spalte des fachlichen Audit-Logs. Ein
  zurückgebrachtes Buch, dessen Exemplar (oder ganzer Titel) gelöscht ist, wäre sonst
  niemandem mehr zuzuordnen, obwohl das Protokoll es weiß. Zuschnitt bewusst eng:
  nur Barcode-Suche, eigenes Recht `audit_details` (ab Werk Admin und Leitung), Stufe 2 der
  PII-Matrix, jeder Abruf wird selbst mit IP protokolliert (`TRESEN_AUSKUNFT`). Alle
  drei Löschwege (Exemplar ausbuchen, Verlust-Löschen, Titel löschen) hinterlassen
  dafür je Exemplar einen Barcode-Snapshot; nach DSGVO-Tilgung oder
  Lesehistorie-Befristung zeigt auch dieser Weg bewusst nichts mehr.
- **Aufbewahrungsfrist:** Auch Protokolle brauchen ein „wie lange“ —
  IP-Adressen und Bearbeiter-Bezüge sind personenbezogen (Speicherbegrenzung, Art. 5).
  Ein nächtlicher Job (03:00 UTC, nach dem Backup um 02:30 UTC) löscht Einträge beider Tabellen jenseits
  der Frist aus `audit_aufbewahrung_monate` (Vorgabe 24 Monate, Untergrenze 6 gegen
  Fehlkonfiguration) und hinterlässt die Löschung selbst als eine Meta-Zeile mit den
  Zahlen — sonst sähe eine spätere Prüfung nur ein Protokoll mit unerklärlicher
  Vorderkante.

---

## 11. Statistiken & Dashboards

Für die Schulleitung und Bibliotheks-Administration zählt das System, ganz oder nur Freihand oder
nur LMF:

- Bestand, aktuell verliehen, Zirkulationsquote, Wiederbeschaffungswert; Ausleihen pro Monat;
  Überfällige Bücher nach Dauer in vollen Tagen wie im Mahnwesen (überfällige Geräte stehen dort).
- Hitlisten: Renner (meistausgeliehen) und Ladenhüter, mit Detailseite und Filter.

Gezählt werden Ausleihen, nicht Personen. Eine Auswertung je Jahrgang oder Klasse, eine
Warnübersicht für Schäden und Mahnungen und einen Export der Statistik gibt es nicht.

---

### 11.x Renner und Ladenhüter

Zwei Listen mit Detailseite (`/statistiken/renner`, `/statistiken/ladenhueter`; Abfragen in
`internal/repository/statistik.go`): **Renner** sind die meistausgeliehenen Titel des gewählten
Zeitraums; **Ladenhüter** sind Titel, die seit mehr als zwei Jahren nicht mehr ausgeliehen
wurden — oder nie, wenn ihr ältestes Exemplar vor mehr als zwei Jahren angelegt wurde. Ein
Titel ohne Exemplar im Bestand ist schon ausgesondert, einer mit offener Ausleihe in Gebrauch
(Mehrjahresband); beide stehen nicht unter den Ladenhütern. Beide Listen lassen sich
nach Fachbereich und Systematik filtern und clientseitig nach Titel/Autor durchsuchen; das
`?limit=`-Argument ist serverseitig gedeckelt. Beide Listen enthalten keine Schülerdaten —
sie zählen Ausleihen, nicht Ausleiher (siehe PII_MATRIX).

## 12. Authentifizierung & Rollenmodell (RBAC)

Der Zugang zum System ist strikt reglementiert und wird durch ein Role-Based Access Control (RBAC) System gesteuert.

### 12.1. Login & Sicherheit

- **Verfahren:** E-Mail und Passwort **gegen den Mailserver der Schule (IMAP)** oder, je Konto eingestellt, LDAP-Kennung oder E-Mail und Passwort **gegen den Verzeichnisdienst (LDAP)**, nie gegen die eigene Datenbank: Eine Passwortspalte gibt es nicht; wer ein Konto anlegt, setzt eine Adresse, die auf dem Schulserver existiert. Gesucht wird zuerst das Konto, dann fragt der Server, den es nennt ([ARCHITEKTUR.md](ARCHITEKTUR.md) 6.1, A2, A30). Ist `SELBSTANMELDUNG_DOMAIN` gesetzt, genügt der Name vor dem @; der Server ergänzt die Domain, und die Anmeldeseite nennt sie unter dem Feld. Für die Dauer einer Anmeldung hält der Server einen Prüfwert des Passworts, damit die Sperre auch bei einem Ausfall des Mailservers aufgeht ([SECURITY.md](SECURITY.md)).
- **Folge für die Kontoverwaltung:** Die E-Mail **ist** die Identität, bei einem LDAP-Konto auch die LDAP-Kennung. Wer die Spalte `benutzer.email` oder `benutzer.ldap_kennung` schreiben darf, übernimmt damit ein Konto; ein Rechte-Audit, das nur auf `rolle` schaut, sieht diesen Weg nicht.
- **Anmeldeverfahren je Konto:** Unter Benutzer & Rechte steht an jedem Konto „Anmeldung über“: Mailserver der Schule (Vorgabe) oder Verzeichnisdienst mit der LDAP-Kennung, dem Namen, den die Person beim Anmelden tippt (etwa ihr Kürzel). Die Rolle kommt aus dieser Verwaltung, nicht aus Gruppen des Verzeichnisses.
- **Sitzung:** ein JWT im HttpOnly-Cookie; zu jeder Anmeldung gehört eine Zeile in `sitzungen`, und der Server liest bei jeder Anfrage, ob sie gesperrt ist (`internal/auth/sitzungen.go`). Gesperrt ist sie auch 10 Stunden nach der letzten Passworteingabe, gleich wie oft das Token dazwischen erneuert wurde; das Passwort am Sperrbildschirm beginnt die 10 Stunden neu.
- **Inaktivität:** Nach 5 Minuten ohne Bedienung leert sich die Theke (kein geladener Schüler mehr), nach 15 Minuten kommt der Sperrbildschirm — Entsperren mit dem eigenen Passwort oder Abmelden. Beide Fristen stehen in den Einstellungen („Datenschutz & Sitzung", 0 = aus). Die Sitzung läuft dabei weiter, der Server sperrt sie aber: Bis das Passwort eingegeben ist, beantwortet er keine Anfrage, auch nicht nach dem Neuladen oder in einem neuen Tab (entschieden am 01.10.2026). Ist der Mailserver nicht erreichbar, schließt das Passwort der Anmeldung trotzdem auf.
- **Brute-Force-Schutz:** Fehlversuche werden je Eingabe und Adresse gezählt und sperren die Anmeldung für eine Weile ([SECURITY.md](SECURITY.md)).
- **Selbstanmeldung des Kollegiums (`SELBSTANMELDUNG_DOMAIN`, `internal/auth/selbstanmeldung.go`):** damit niemand rund 160 Lehrkräfte vorab von Hand anlegt. Meldet sich ein Postfach der eingetragenen Schuldomain an, das noch kein Konto hat, entsteht ein **inaktiver** Eintrag mit Rolle `kollegium` (Name aus `vorname.nachname@…` geraten, `zugang_beantragt_am` gesetzt, Audit-Zeile `SELBSTANMELDUNG`); die Lehrkraft liest „Zugang beantragt. Die Bibliothek muss ihn noch freischalten.“, kein Fehlversuch wird gezählt. Unter Benutzer & Rechte steht der Eintrag als „Zugang beantragt“ mit Zähler oben; „Aktiv“ setzen schaltet frei, danach sieht die Person nur „Mein Portal“. Ein Antrag verfällt nicht von selbst (entschieden am 22.09.2026): Er bleibt sichtbar, bis jemand freischaltet oder löscht; ein stiller Verfall nähme eine echte Anfrage weg, ohne dass sie je jemand gesehen hätte. Wird das aus Datenschutzgründen anders gewollt, gehört die Frist zu den Löschfristen, nicht als Sonderregel hierher. IMAP beantwortet „wer bist du“, nicht „darfst du rein“ — die Freischaltung bleibt bewusst bei der Schule (ein Schülerpostfach derselben Domain würde sonst ebenfalls hereinkommen). Ist die Variable leer, ist der Weg zu: richtige Zugangsdaten enden dann in „Anmeldung fehlgeschlagen“, die Selbstprüfung meldet das als Warnung. Wer in Bibliothek oder LMF mitarbeitet (Mitarbeiter, Helfer), wird weiterhin von Hand angelegt.

### 12.2. Vier Rollen und ein Grundzustand

**Der Admin vergibt vier Rollen** — Admin, Leitung, Mitarbeiter, Helfer —, deren genaue
Rechte (z.B. `view_students`, `manage_settings`, `perform_actions`) er in der Rechte-Matrix
einstellt. **Kollegium ist keine davon**, sondern der Grundzustand jeder Lehrkraft; es steht
unten als Nummer 5, weil es technisch derselbe Enum-Wert ist, und aus keinem anderen Grund.

Als Spalte der Rechte-Matrix neben Leitung und Mitarbeiter sähe Kollegium aus wie eine Stufe
in einer Rangfolge. Es ist keine: Jede Lehrkraft
meldet sich über „Mein Portal" selbst an, wird freigeschaltet und ist damit erst einmal
niemand Besonderes. Eine Rolle bekommt nur, wen der Admin an seiner E-Mail-Adresse dazu
erhebt. Was das Kollegium darf, steht deshalb fest in `internal/db/rechte_vorgabe.go` und ist kein Schalter je
Schule — das ist eine Produktentscheidung.

1. **Admin (`admin`):** Uneingeschränkter Zugriff auf alle Systembereiche, Einstellungen, Audits und Datenschutz-Routinen.
2. **Leitung (`leitung`, Migration 121/122):** Die Rechte des Admins **minus** `manage_users` (Benutzer & Rechte) und `manage_settings` (Einstellungen) — die Rolle für die Person, die die Bibliothek führt, ohne die Systempflege zu übernehmen.

   **Das Soll wird abgeleitet, nicht abgeschrieben:** Migration 122 erzeugt die Zeilen aus den Admin-Zeilen und setzt genau die zwei Ausnahmen auf `false`; eine abgeschriebene Liste liefe beim nächsten neuen Recht auseinander, und niemand merkte es (Gate `internal/db/rolle_leitung_test.go`, [ARCHITEKTUR.md](ARCHITEKTUR.md) A12).

   **Warum ausgerechnet `manage_users` fehlt:** Mit dem Recht ändert man die E-Mail-Adresse eines Kontos, und die Anmeldung erkennt eine Person allein an ihrer E-Mail. Die Rechtevergabe wäre damit der Weg in jedes Konto der Anlage. Der Admin kann das Recht erteilen; ein Admin-KONTO bleibt der Leitung auch dann verschlossen (`internal/api/benutzer/admin_schutz.go`).
3. **Mitarbeiter (`mitarbeiter`):** Das Personal für das Tagesgeschäft. Hat Zugriff auf die Scanner-Omnibox, Buchkatalog, Mahnwesen und Leserdatei, darf aber keine Systemeinstellungen ändern.
4. **Helfer (`helfer`):** siehe unten.
5. **Kollegium (`kollegium`) — der Grundzustand, keine vergebene Rolle:** Zugang zum Kollegiums-Portal mit vier Reitern: _Reservieren & Melden_ (suchen, Klassensatz reservieren, ein Problem melden; darunter die eigenen Reservierungen und Meldungen, §4.2 und §4.4), _Klassensätze_ (welche Klasse hat welche Bücher — Handliste `class_books` plus live aus den Ausleihen abgeleitet: mehr als die Hälfte der Klasse und mindestens `klassensatzMindestLeser` Kinder halten den Titel; `ladeKlassensatzGruppen`, Quelle `hand`/`ausleihe`, nie gespeichert), _LMF-Plan_ (Rückgabe- und Ausgabetermine je Klasse, §2.3) und _Schulbücher_ (Suche über Titel, ISBN, Autor und Fach; Filter Jahrgang und Schulzweig; je Fach eine aufklappbare Zeile mit Exemplaren, Titeln und Verliehenen; Export je Fach als **PDF** mit Coverbildern, Jahrgang, Schulzweig und Zähldatum; nur Titel mit `ist_lernmittel`; Portal-Routen `/api/portal/lernmittel[/export]`). Erteilt ist ein einziges Recht, `create_reservations`: Suche (Katalog des Kollegiums, §4.2), Reservierung und Anliegen laufen über `create_reservations`; die Klassensatz-Sicht hängt an einer eigenen Portal-Route (`/api/portal/klassensaetze`), für die die Anmeldung genügt — bewusst kein `view_books`, das der Rolle den ganzen Medienkatalog öffnen würde. Nichts davon fasst Personendaten an.

   **„Mein Portal“ hängt am Recht `create_reservations`, nicht an der Rolle** (entschieden): Eine Lehrkraft, die in Bibliothek oder LMF mitarbeitet und deshalb als Mitarbeiter angelegt ist, sieht das Portal ebenfalls und reserviert dort für die eigene Klasse.

   **Die Tabelle der Rechte entscheidet auch an der Route:** `role_permissions` steuert das Menü und `verlangeRecht`; ein Recht zu viel ist keine Frage der Anzeige. Das Kollegium hält nur `create_reservations`. Wer einer Lehrkraft gezielt mehr geben will, tut das unter Benutzer & Rechte — Migrationen laufen nur einmal, eine spätere Vergabe wird nicht zurückgedreht.

**Helfer (`helfer`):** Stark limitierte Rolle für studentische Hilfskräfte oder Eltern. Theken-Ansicht (Omnibox) für Ausleihe und Rückgabe, dazu **lesender Katalogzugriff** (entschieden am 30.07.2026): Ein Helfer an der Theke ist die erste Anlaufstelle für „Habt ihr Band 3 noch da?" und musste die Frage sonst weiterreichen. Die Grenze zur Akte zieht weiterhin `view_students`.

   **Ein Helfer braucht ein Postfach auf dem Schul-Mailserver oder ein LDAP-Konto** (§12.1); einen Code- oder Barcode-Anmeldeweg gibt es nicht. Wer eine Hilfskraft aufnimmt, lässt zuerst das Postfach anlegen und trägt die Person dann unter System → Benutzer & Rechte mit der Rolle „Helfer“ ein.

   Erteilt sind genau zwei Rechte (`internal/db/rechte_vorgabe.go`): `perform_actions` (Scannen, Ausleihe, Rückgabe) und `view_books` (Katalog). Erreichbar sind damit Ausleihe, Medienkatalog, Signaturen und Schulklassen. Die Pflege-Aktionen auf diesen Seiten hängen an `edit_books` und bleiben dem Helfer verborgen. Nach dem Scan eines Ausweises zeigt die Theke, was die Antwort des Servers trägt: Name, Klasse, Ausweisnummer und Sperre. Die Akte mit Ausleihen und Gebühren bleibt zu, und die Akte eines Titels zeigt nur die Exemplare (`view_students` öffnet beides, `manage_vormerkungen` die Warteliste). Gebucht wird trotzdem, jede Buchung nennt eine kurze Meldung.

   **Welche Seiten eine Rolle erreicht, entscheidet `darfSehen()` in `frontend/src/lib/menue.js` — und nur diese Funktion.** Der Router fragt dieselbe; eine zweite Liste liefe beim nächsten Wechsel eines Rechts auseinander, und ein Menüpunkt führte beim Klick wortlos an die Theke zurück. Wer eine Rolle oder ein Recht ändert, fasst deshalb `menue.js` an und sonst nichts. Abgesichert durch `e2e/menue-fuehrt-irgendwohin.spec.js`, das für Helfer, Mitarbeiter und Lehrkraft jeden sichtbaren Menüpunkt anklickt.

   **Welche Aktion jemand sieht, entscheidet das Recht der Route, nicht die Rolle.** Am Server hängt jede Route an einem Recht, keine an einer Rolle; die Oberfläche fragt deshalb `hatRecht(Benutzer, Recht)` aus `menue.js` mit demselben Recht, das die Route verlangt. Ein Rollenvergleich an dieser Stelle machte den Schalter in der Rechte-Matrix wirkungslos: Der Server erlaubte, der Knopf fehlte — oder umgekehrt. Ratsche: `frontend/src/lib/frontend-hygiene-rechte.test.js`, mit den begründeten Ausnahmen. Der Menüpunkt „Einstellungen" ist ein Sammelpunkt über Kategorien mit verschiedenen Rechten (§17): Er öffnet sich mit jedem Recht, das eine Kategorie verlangt (`ALLE_KATEGORIE_RECHTE`, Gate `menue.test.js`). Dass eine Rolle etwas nicht darf, zeigt deshalb die Antwort 403 des Servers und nicht ein fehlender Menüpunkt.

### 12.3. Leserdatei: eine Tabelle, sieben Arten (Migration 123–125)

**Rolle und Art sind zwei verschiedene Fragen.** Die Rolle (§12.2) sagt, was jemand im
Programm DARF. Die Art sagt, WER an der Theke Bücher bekommt. Ein Mensch kann beides haben,
eines von beidem oder keines: Eine Lehrkraft ohne Konto steht in der Leserdatei und darf
nichts im Programm; ein Admin muss nicht in der Leserdatei stehen.

Schüler und Kollegium stehen in einer Tabelle `leser` mit der Spalte `art`:

| Art           | Wort in der Oberfläche | Woher                                                              |
| ------------- | ---------------------- | ------------------------------------------------------------------ |
| `schueler`    | Schüler                | LUSD-Import oder von Hand                                          |
| `lehrkraft`   | Lehrkraft              | Selbstanmeldung („Mein Portal"), Handanlage, Littera-Bestand       |
| `liv`         | LiV                    | von Hand, oder eine Lehrkraft wird dazu umgestellt                 |
| `praktikum`   | Praktikum              | von Hand, Littera-Bestand („Praktikant", „Praktikantin")           |
| `sekretariat` | Sekretariat            | von Hand, Littera-Bestand („Sekretärin")                           |
| `uplus`       | U-plus                 | von Hand, Littera-Bestand („U-plus", die Vertretungskräfte)        |
| `fachbereich` | Fachbereich            | von Hand, Littera-Bestand („Fachbereich …", Sammelkonto des Fachs) |

Die Art entscheidet **keine Rechte**; ausleihen darf jeder aktive Leser. Ändern lässt sie
sich nur innerhalb des Kollegiums: Ein Schüler kommt aus der LUSD und bleibt Schüler, in
beide Richtungen (`chk_leser_nur_schueler_werden_abgaenger`).

**Die Sonderkonten** (entschieden am 30.09.2026): Praktikum, Sekretariat, U-plus
und Fachbereich sind die Lesergruppen, die Littera neben Klassen und Lehrern führt. Sie stehen
als Art da und nicht als zweites Feld daneben — eine Gruppe neben der Art hätte Paare wie „LiV
und Sekretariat" erlaubt und eine Pflegeseite gebraucht. Das Programm unterscheidet an jeder
Stelle nur `art = 'schueler'` gegen den Rest; für die vier gilt deshalb alles, was für das
Kollegium gilt (Dauerleihe, keine Mahnung, keine Forderung). Praktikum und Fachbereich haben
**kein Zugangskonto** (`leserart.HatKonto`): Ein Fachbereich ist keine Person, sondern ein
Sammelkonto, das die Kollegen des Fachs benutzen; ein Praktikant leiht aus, braucht aber
„Mein Portal" nicht.

**`schueler` ist eine Sicht**, keine Tabelle: `WHERE art = 'schueler'`
mit `WITH CHECK OPTION`. Das trägt die alte Bedeutung weiter — Klassenlisten,
LUSD-Abgleich, Mahnlauf und die DSGVO-Löschfristen lesen sie und bekommen das Kollegium
nicht zu sehen. Die Kehrseite ist eine eigene Bugklasse; sie steht in
[sweeps.md](sweeps.md) („Schreibpfad gegen gefilterte Sicht").

**Eine Maske für jeden.** Akte und Formular zeigen für jede Art dieselben Felder an
derselben Stelle. Einem Kollegen sind verschlossen, und zwar nicht aus Geschmack:
Klasse und Abgangsjahr (eine Klasse hat nur ein Schüler, und das Abgangsjahr leitet der
Server aus ihr ab), die LUSD-Kennung (die Datenbank verbietet sie einem Nicht-Schüler) sowie
Anschrift und Eltern-E-Mail: Für das Kollegium führt das Programm keine Anschrift
(`chk_leser_anschrift_nur_schueler`). Geburtsdatum und
Ausweisnummer stehen jedem offen. Beim Kollegen bleibt das Geburtsdatum leer, ein eingetragenes
lässt sich in der Akte wieder löschen (beim Schüler nicht, es ist der Schlüssel des LUSD-Abgleichs); eine
Ausweisnummer hat jeder mit freigeschaltetem Zugang, das Programm vergibt sie.

**Die Schul-E-Mail ist beim Anlegen von Lehrkraft, LiV, Sekretariat und U-plus Pflicht.**
Bei Praktikum und Fachbereich muss sie leer bleiben — dort entsteht kein Konto, weder beim
Anlegen noch beim Nachtragen in der Akte (`api.pruefeEmailZurArt`). Die Adresse ist keine
Kontaktangabe, sondern der Schlüssel: Mit ihr entsteht sofort das Anmeldekonto, und weil der
Anmeldeweg eine Zugangsanfrage nur anlegt, wenn zu der Adresse GAR KEIN Konto existiert,
findet die spätere Selbstanmeldung genau diesen Eintrag. Ohne sie stand die Person danach
zweimal in der Leserdatei — Ausweis und Ausleihen am ersten Eintrag, die Anmeldung am
zweiten. Freigeschaltet wird das Konto nur, wenn der Anlegende `manage_users` hat; sonst
entsteht eine Zugangsanfrage wie bei der Selbstanmeldung. Die Adresse steht dabei an genau
EINER Stelle, am Konto (`benutzer.email`, `UNIQUE lower(email)`) — eine zweite Spalte an der
Leserzeile gibt es bewusst nicht. Die Akte liest sie vom Konto; fehlt sie, lässt sie sich dort
nachtragen, ändern lässt sie sich nur in „Benutzer & Rechte" (`internal/api/leser/leser_schul_email.go`).

**Ausweise:** Alle Leser ziehen ihre Nummer aus einem Nummernkreis, und
alle neuen Nummern tragen die Vorsilbe `A-` (§1). Eine `A-`-Nummer vergibt das Programm nur
einmal, auch nachdem sie verschwunden ist — endgültig gelöscht, anonymisiert, zusammengeführt,
geleert oder umgeschrieben: Die Tabelle `ausweisnummern_ausgeschieden` hält sie
als bloße Zahl fest, ohne Person, und der Generator zählt über sie hinweg. Eine alte Karte
findet danach an der Theke niemanden. Von Hand bleibt eine frühere Nummer eintragbar, etwa die
alte Karte eines Schülers, der zurückkommt. Die Aufschrift der gedruckten Karte
richtet sich nach der Art — „Schülerausweis", „Lehrerausweis" (Lehrkraft und LiV) oder
„Leserausweis" (die Sonderkonten); eine Gültigkeit trägt nur
der Schülerausweis, weil ein Ausweis im Kollegium mit keinem Schuljahr abläuft. Ihr Jahr rechnet
das Programm bei jedem Druck aus der Klasse (`internal/ausweis`), der Tag ist der 31.07., das
Ende des Schuljahres: im Hauptschulzweig bis zum Ende von Jahrgang 9, in der übrigen Mittelstufe
bis zum Ende von 10, auch im Gymnasialzweig, weil offen ist, wer in die Oberstufe geht. Ab
Jahrgang 11 ist jede Klasse Oberstufe, gleich wie sie heißt, und bekommt einen neuen Ausweis bis
zum Ende von 13. Lässt sich aus der Klasse nichts ableiten, wird nichts geraten; in der Akte ist
das Jahr für einen Druck änderbar.

---

## 13. Katalogisierung & Medienverwaltung

Das System bietet umfassende Werkzeuge zur Pflege des Buchkatalogs:

- **Titel ohne Exemplar stehen in keinem Katalog** (Antwort der Schule auf die Sichtung vom
  16.09.2026): Katalogliste (`GET /api/books`), Theken-Suche und Aktionssuche der Omnibox zeigen nur
  Titel mit mindestens einem nicht ausgesonderten Exemplar, der Zulauf zählt als vorhanden; ein
  Prädikat, `repository.sqlTitelHatExemplar`. Der Titel bleibt in der Tabelle, sonst legte ihn jemand
  ein zweites Mal an: Die Titel-Verwaltung erreicht ihn über „Ohne Exemplare“ (`?bestand=ohne`,
  dasselbe Prädikat verneint). Öffentlicher Katalog und Monitor haben eine engere Regel
  (`OeffentlichSichtbar`: kein Lernmittel, kein Zulauf); _Mein Portal_ zeigt auch Lernmittel und
  bestellte Titel (§4.2).
  - **Anlegen:** Trägt die ISBN schon ein Titel, antwortet die Dublettenkontrolle mit 409 und nennt ihn
    (`vorhanden`), beim Anlegen und beim Ändern. Die Maske fragt schon bei der Eingabe der ISBN, mit
    derselben Abfrage wie das Speichern (`GET /api/books/vorhanden`), und bietet „Vorhandenen Titel
    öffnen?“ an; dort wird der Bestand eingetragen. Für eine vergebene ISBN werden die Katalogdienste
    nicht gefragt. Lässt sich der Katalog nicht fragen, arbeitet die Maske weiter, und das Speichern
    prüft noch einmal.
  - **Ohne ISBN:** Pflicht ist der Titel, die ISBN darf fehlen (Zeitschriften, Spiele, alte Bücher; die
    Littera-Übernahme lässt falsche und doppelte Nummern weg). Beim Ändern prüft die Tür die ISBN nur,
    wenn sie sich ändert. Beim Anlegen ohne ISBN nennt sie einen Titel mit gleichem Titel, Autor und
    Auflage (409 mit `gleicherTitel`); die Maske fragt „Ist es dasselbe Medium?“, weil Hefte und Bände
    gleich heißen, und legt nach „Anderes Medium“ an.
  - Ein neues Buch beginnt mit einem Exemplar; steht dort 0, sagen Hinweis und Meldung, dass der Titel
    nur unter „Ohne Exemplare“ steht. Mit der Kamera scannt man die ISBN im Feld der Maske; einen
    eigenen Knopf oder ein zweites Fenster dafür gibt es nicht.
  - Belegt: `internal/repository/titel_bestand_pg_test.go`, `titel_ohne_exemplar_pg_test.go`,
    `internal/api/katalog/dublette_nennt_den_titel_pg_test.go`, `buch_vorhanden_pg_test.go`,
    `titel_ohne_isbn_pg_test.go`, `frontend/e2e/buch-anlegen-auffindbar.spec.js`, `buch-ohne-isbn.spec.js`
    und die Tests der Maske in `frontend/src/inventur/lib/` (`buch_speichern`, `AdminBuchAktionen`,
    `IsbnFeld`, `BuchFormular`, `buchFormularRegeln`).
- **Mehrjahresband** (Antwort der Schule auf die Sichtung vom 16.09.2026): Das Kästchen am Titel (nur
  bei Lernmitteln) lässt das Buch beim Kind bis zum Ende von `jahrgang_bis` der Spanne „Im Unterricht
  von … bis“; eine zweite Zahl gibt es nicht. Die Frist ist der Stichtag des Schuljahres, in dem das
  Kind `jahrgang_bis` beendet (`internal/service/frist.go`); der Rückgabetermin der Klasse geht das Buch
  nichts an. Hat der Büchertausch begonnen, rechnet die Frist vom folgenden Schuljahr aus, und eines
  der Jahre steckt in diesem Sprung. Der Schalter gilt nur an einem Lernmittel mit einer Spanne über
  mehr als einen Jahrgang, an beiden Türen (`internal/repository/titel_spanne.go`) und in der Datenbank
  (`chk_mehrjahresband_spanne`). Belegt: `internal/service/lmf_frist_termintag_pg_test.go`,
  `internal/repository/mehrjahresband_am_titel_pg_test.go`.
- **Jahrgang am Titel: „unbekannt“ ist eine eigene Angabe** (entschieden am 24.09.2026): „Im Unterricht von Jahrgang … bis“ ist ganz eingetragen oder gar nicht; ohne Eintrag ist der Jahrgang unbekannt (`jahrgang_von` und `jahrgang_bis` NULL). Eine Vorgabe gibt es nicht, sonst wäre ein Titel ohne Angabe von einem für diese Jahrgänge nicht zu unterscheiden. Die Inventur nach Klasse und der Filter „Jahrgang“ im Portal treffen nur Titel mit Spanne. Regel an beiden Türen: `repository.PruefeJahrgangsSpanne`, in der Datenbank `chk_jahrgang_spanne`. Eine „Klasse“ am Titel gibt es nicht (entschieden am 08.10.2026: Was die Zahl meinte, war nicht belegt, und mehrjährige Bände trugen ein einziges Jahr). In der Maske geht „bis“ mit „von“ mit, solange es leer ist oder dieselbe Zahl zeigt. Jeder Weg, auf dem ein Jahrgang ins Programm kommt, schreibt die Spanne: Die Littera-Übernahme liest sie aus Signatur („LMF Bio 7“) und Interessenkreis („Sekundarstufe 1“ → 5 bis 10), der Listenimport die Spalte „klasse“ als N bis N, und die ISBN-Abfrage schlägt eine Spanne nur vor, wenn der Titel sie eindeutig als Schulstufe nennt („Klasse 7“, „Klassen 7–10“, „5./6. Schuljahr“), nicht bei „Band 2“, „Level 9“, einer Grenze („ab Klasse 5“), einer Aufzählung oder einer Zahl außerhalb von 5 bis 13 (`stufenAusText`, die Fälle in `internal/katalogdienste/fach_und_jahrgang_test.go`); die Maske trägt nur beide Zahlen zusammen ein. Gemessen am 08.10.2026 an 1.216 Titeln, die die DNB zu den Stufenwörtern liefert: 259 bekommen eine Spanne, 337 einen einzelnen Jahrgang. Belegt außerdem: `internal/api/katalog/jahrgang_vorgabe_pg_test.go`, `internal/repository/jahrgang_unbekannt_pg_test.go`, `internal/repository/inventurlauf_pg_test.go`, `internal/api/katalog/isbn_suche_test.go`, `internal/api/portal/schulbuecher_pg_test.go`, `frontend/e2e/feld-roundtrip.spec.js`, `frontend/e2e/portal-schulbuecher.spec.js`, `isbnAbfrage.test.js`, `BuchEingabefelderKategorisierung.jahrgang.svelte.test.js`.
- **Auflagen eines Schulbuchs:** Eine neue Auflage ist ein eigener Titel mit eigener ISBN (andere
  Seitenzahlen), und jedes Exemplar bleibt an seiner Auflage. Auflagen desselben Buchs lassen sich
  zusammenfassen, nur Lernmittel: Sie tragen dieselbe `werk_id` (Tabelle `werke`, ohne Namen; jede
  Ansicht zeigt die neueste Auflage), gelesen über `COALESCE(werk_id, id)`. Geschrieben wird nur in
  `internal/repository/auflagen.go`, nicht beim Speichern der Titelmaske: Gehören zwei Titel schon zu
  verschiedenen Büchern, werden alle zu einem; ein Werk mit weniger als zwei Titeln fällt weg, auch
  wenn ein Titel gelöscht wird. „Die neueste Auflage“ ist das jüngste Erscheinungsjahr, bei gleichem
  oder fehlendem Jahr die zuletzt angelegte (`repository.sqlNeuesteAuflageZuerst`). Zusammenfassen und
  Lösen verlangen `edit_books`.
  - **Titelmaske:** Der Abschnitt _Auflagen_ steht bei jedem Lernmittel und bei jedem Titel, der
    schon zu einem Buch gehört, sonst ließe er sich nicht lösen. Lösen fragt nicht nach, weil es sich
    zurücknehmen lässt und nichts löscht. _Andere Auflage zuordnen_ findet auch Titel ohne Exemplar,
    denn eine alte Auflage hat oft keins mehr, und sagt vor dem Klick, wenn weitere Auflagen mitkommen.
  - **Neue Auflage bestellen** (Zeilenmenü im Bestellbedarf): Die ISBN holt den Titel aus dem Katalog
    oder der DNB; erst die Bestätigung führt ihn als Lernmittel, fasst ihn mit der Zeile zusammen und
    legt ihn in den Warenkorb („Der Besteller legt das Werk an“, entschieden am 23.09.2026; Recht
    `create_orders`).
  - **Gezählt wird am Buch:** im Bestellbedarf (§7), in der Klassensatz-Übersicht (§4.3), im
    Medienkatalog (eine Kachel je Buch, oben die getroffene oder die neueste Auflage, Bestand als
    Summe, Aufschlüsselung beim Zeigen; erst die Anzeige fasst zusammen, weil Titel-Verwaltung und
    Zuordnen-Dialog jede Auflage einzeln brauchen, `buecherJeBuch`) und im Portal-Reiter
    „Schulbücher“ (die Fach-Zahlen zählen Bücher; das PDF setzt die Aufschlüsselung unter den Titel,
    `auflagenBeschriftung` in `internal/pdf/schulbuecher.go` ist der Zwilling der Browser-Regel, beide
    lesen die Prüffälle aus `auflagenText.faelle.json`).
  - **An der Theke:** Bekommt ein Kind eine andere Auflage, als Kinder seiner Klasse sie gerade haben
    (nur offene Ausleihen aktiver Kinder), trägt die Antwort `auflagen_hinweis`: Klasse und Auflagen
    mit der Zahl der Kinder, keine Namen. Die Theke blitzt orange und zeigt die Zeile über dem Konto
    (entschieden am 25.09.2026: wie die Fremdrückgabe, kein Dialog; §18.6). Gefragt wird nach dem
    Commit; ein Fehler dabei hält die Buchung nicht auf.
  - **Dieselbe ISBN in der anderen Länge:** Eine zehnstellige ISBN mit richtigem Prüfzeichen wird an
    jeder Tür dreizehnstellig gespeichert (`isbn_normalform`, in Go `isbnutil.Normalform`); Titelsuche,
    Maske und Suchfelder finden beide Längen. Mit falschem Prüfzeichen bleibt sie zehnstellig und wird
    mit keiner anderen gepaart: Die Rechnung führte auf die ISBN-13 eines anderen Buchs.
  - **Keine Vorschlagsliste zum Zusammenfassen:** Am Testserver standen am 25.09.2026 nur zwei
    Lernmittel mit gleichem Titel und Verlag, und höchstens eines der beiden hatte Exemplare;
    zusammengefasst wird deshalb beim Nachbestellen und von Hand. Die Zählung sah nur gleiche Titel,
    eine Neuauflage unter geändertem Titel nicht.
  - Belegt: die Tests mit „auflage“ im Namen (Go, Vitest, `frontend/e2e/auflagen-*.spec.js`), dazu
    `KlassenBuchKachel.test.js`, `BuchKarte.test.js`, `startseiten_api.test.js`, `BestellSuche.test.js`,
    `internal/api/portal/schulbuecher_pg_test.go`, `internal/pdf/schulbuecher_test.go` und für die
    ISBN-Längen `internal/pkg/isbnutil/normalform_test.go`, `internal/repository/isbn_normalform_pg_test.go`,
    `internal/api/katalog/isbn_eine_laenge_pg_test.go`.
- **Titel zusammenführen** (entschieden am 22.10.2026): Steht dasselbe Buch zweimal im Katalog, etwa aus einer Liste ohne ISBN und aus dem Bestands-Import mit ISBN, macht die Titelmaske einen Titel daraus (_Mit anderem Titel zusammenführen_; Recht `delete_books`, weil danach einer der beiden fehlt). Es bleibt der Titel mit ISBN, hat der andere keine, der offene; der Dialog fragt nicht danach (entschieden am 23.10.2026). Regeln in `internal/repository/titel_zusammenfuehren.go`: Der bleibende behält Kennung und Angaben, was ihm fehlt, füllt der andere auf, auch die ISBN (leer heißt unbekannt, außer beim Schulzweig, wo leer „für alle Zweige“ heißt; ist einer ein Lernmittel, ist es der bleibende auch; der Mehrjahresband kommt mit dem Jahrgang). Exemplare samt Ausleihen, Vormerkungen (von zweien desselben Lesers bleibt die fortgeschrittenere, sonst die ältere), Reservierungen, Bestellpositionen, Meldungen, Schlagworte und Klassenzuordnungen wandern; war der andere Titel Auflage eines Buchs, tritt der bleibende an seine Stelle. Das Protokoll hält beide Stände und alle gewanderten Zeilen (`TITEL_ZUSAMMENGEFUEHRT`). Tragen beide eine ISBN, sind es zwei Ausgaben: 409 mit dem Hinweis auf _Andere Auflage zuordnen_. Die Vorschau läuft durch denselben Weg und nimmt ihn zurück; sie nennt den Titel danach mit ISBN und Zahl der Exemplare, und eine verdoppelte Zahl zeigt, dass beide dieselben Bücher trugen. Belegt: `internal/repository/titel_zusammenfuehren_pg_test.go`, `titel_zusammenfuehren_felder_test.go`, `internal/api/katalog/titel_zusammenfuehren_pg_test.go`, `TitelZusammenfuehrenDialog.test.js`, `zusammenfuehrenVorschau.test.js`, `BuchFormular.zusammenfuehren.svelte.test.js`, `frontend/e2e/titel-zusammenfuehren.spec.js`.
- **Systematiken & Signaturen:** Bücher können hierarchisch nach Systematiken (Kategorien/Themen) und spezifischen Signaturen (Regal-/Standort-Kennung) klassifiziert werden. Die DNB liefert weder Signatur noch Systematik; das Fach schlägt das Programm aus dem Text des Titels vor (`automatischeKategorisierung` in `internal/katalogdienste/fach_und_jahrgang.go`), die Übernahme aus Littera leitet es aus Signatur und Schlagworten ab (siehe unten).
- **Standort am Exemplar:** Der Standort sagt, wo ein Exemplar steht, wenn nicht an seinem Platz nach
  der Signatur („Bibliothek, Regal 3B“); er gehört zum Exemplar. Ein neues Exemplar erbt ihn, wenn alle
  Exemplare des Titels im Bestand denselben tragen (`repository.sqlGeerbterStandort`, eine Regel für
  jeden Weg, der ein Exemplar anlegt). Die Littera-Übernahme bringt ihn je Exemplar mit. Geändert wird
  er in der Buchakte für die markierten Exemplare, ohne Pflicht-Grund, weil er auf keine Rechnung und
  kein Papier wirkt; alter und neuer Wert stehen im Protokoll (`internal/repository/exemplar_standort.go`).
  Die Bestandsliste nennt ihn je Exemplar. Bedienung: [HANDBUCH.md](HANDBUCH.md).
- **Titel ändern: nur die genannten Felder:** `PUT /api/books/{id}` schreibt am Titel nur die Felder, die die Anfrage nennt; ein Feld, das fehlt, bleibt, wie es ist (`titelFelder` und `sqlTitelAendern` in `internal/repository/titel_aendern.go`, `leseAenderung`). Die Maske schickt nur, was geändert wurde. Haben zwei Plätze denselben Titel offen, überschreibt keiner, was er nicht angefasst hat; ändern beide dasselbe Feld, gilt der spätere Eintrag (§20). Ein neues Feld am Titel braucht einen Eintrag in `titelFelder`. Belegt: `internal/repository/titel_aendern_test.go`, `frontend/e2e/buchmaske-zwei-plaetze.spec.js`.
- **Schlagworte:** frei eintragbar, mehrere je Titel, mit Vorschlägen aus dem Bestand (beim Öffnen
  die 500 häufigsten, beim Tippen die Treffer über alle Wörter). Buchformular und Bestellkorb schreiben
  über einen Pfad, `repository.SetzeSchlagworte`: Leerraum wird zusammengezogen, Doppelte fallen,
  höchstens 300 Wörter je Titel (Littera führt bis zu 282) und 80 Zeichen je Wort, das Eingabefeld
  sperrt an denselben Grenzen (`ui/ChipFeld`, `internal/repository/schlagworte_grenzen_zwilling_test.go`);
  eine vorhandene Schreibweise gewinnt, denn die Identität eines Worts ist seine Kleinschreibung
  (eindeutiger Index auf `lower(wort)`).
  - **Pflege** unter _Einstellungen → Schlagworte_ (`edit_books`): Gesucht wird am Server über Wörter
    und Verweise, höchstens 200 Zeilen je Antwort mit der Zahl der Treffer (Littera bringt 13.207 mit).
    Umbenennen und Zusammenführen ändern alle Titel auf einmal und lassen die alte Schreibweise
    wahlweise als **Verweis** stehen (Pflichtfeld `alte_als_verweis`, das Kästchen steht auf
    „behalten“). Ein Verweis leitet eine Schreibweise auf ein Wort („Tierfantasy“ → „Fantasy“); wer ihn
    am Titel einträgt, bekommt das Ziel. Löschen nimmt ein Wort aus allen Titeln, Verweise darauf
    fallen mit, einzeln oder mehrere, alle oder keins. Die Datenbank hält die Regeln der Verweise per
    Trigger mit Sperre: kein Verweis auf sich selbst, keine Kette, kein Titel an einem Verweis, kein
    Verweis als Filter.
  - **Suche:** Medienkatalog, öffentlicher Katalog und _Mein Portal_ finden einen Titel auch über seine
    Schlagworte und die Verweise darauf, als Teilstring ohne Rücksicht auf Groß- und Kleinschreibung.
    Die Regel steht einmal in `internal/repository/schlagworte_suche.go` und löst ein Wort auf wie der
    Schreibpfad (`coalesce(verweis_auf, id)`): als Bedingung am Server und als Wörter je Titel in der
    Katalogliste (`suchwoerter`), weil „Suche & Filter“ im Browser sucht.
  - **Filter im Portal:** Die markierten Wörter stehen in _Mein Portal → Reservieren & Melden_ als
    Filter-Chips unter der Suche (eins wählbar, ein zweiter Klick nimmt es zurück), nur solche, zu
    denen der Katalog des Kollegiums einen Titel zeigt. Gefiltert wird über `schlagwort_id` (§4.2):
    ohne Suchtext alle Titel des Worts, mit Suchtext die, die beides treffen. Der öffentliche Katalog
    nimmt nur einen Suchtext.
  - **Vorschläge aus der DNB:** Entsteht ein Titel beim Bestellen neu aus der DNB, gleicht der Server die Gattungsbegriffe (655 $a) und die freien Verlagswörter (653 $a ohne Vorsatz wie „(Zielgruppe)“) gegen die eigene Liste ab (`repository.ladeSchlagworteAusStichwoertern`). Vorgeschlagen wird nur ein ganzes Wort, das schon ein Titel trägt, direkt oder über einen Verweis, sonst schlüge „Kriegsende“ „Krieg“ vor. Dazu kommen die Schlagwörter der Gemeinsamen Normdatei (MARC 600–651 mit `$2 gnd`) in der Schreibweise der Liste („Kafka <Franz>“, „Geschichte <1918-1933>“), gesucht unter beiden Formen; was die Liste nicht kennt, steht in eigener Zeile als _Neue Schlagworte aus der DNB_, nie vorbelegt: Die Normdatei ist ein festes Vokabular ohne Werbewörter, Verlagswörter kommen nur über die Liste. Eingetragen wird nur, was jemand anklickt, denn ein Satz nennt etwa „Deutsch“ für die Sprache. Für vorhandene Titel und im Buchformular gilt dieselbe Regel (`repository.LadeSchlagwortVorschlagAusDNB`, `GET /api/schlagworte/dnb-vorschlag?isbn=`): Die Tür fragt nur die DNB und schreibt nichts; angeboten wird nur, was der Titel noch nicht trägt, und nur zum Hinzufügen, Wort für Wort. Trägt ein vorhandener Titel kein Schlagwort, fragt die Oberfläche die DNB beim Öffnen von selbst, einmal je Fenster (`SchlagwortDnbVorschlag`, `vonSelbst`); trägt er welche, auf Knopfdruck. Gemessen am 30.09.2026 an Stichproben des Katalogisats: Von 50 Bücherei-Titeln ohne Schlagwort bekämen 26 einen Vorschlag; von 60 Titeln mit Littera-Schlagworten hätte die DNB für 26 weitere Wörter, im Median zwei bis drei je Titel. Kein Abgleich über den ganzen Katalog (entschieden am 30.09.2026).
  - **Aus Littera:** Die Übernahme bringt alle Schlagworte der Titel mit (über
    `repository.SetzeSchlagworte`, [SCRIPTS.md](SCRIPTS.md) Abschnitt 1) und leitet das Fach aus ihnen
    ab, wo die Signatur keins nennt und die Schlagworte genau eins (`lmf.FachAusSchlagworten`, dieselbe
    Regel wie der Katalogisat-Import); Verweise zählt sie nur. Litteras Interessenkreise kommen als
    Schlagworte hinter denen des Titels (entschieden am 30.09.2026): Zielgruppen wie „Lehrer“ oder
    „Sekundarstufe 2“, die die Bücherei als Filter in _Mein Portal_ markieren kann; ihre Stufe ergibt
    die Jahrgangsspanne, wo die Signatur keine nennt, das Fach nie. Der Katalogisat-Import schreibt
    Schlagworte nur an Titel, die noch keine tragen, damit ein erneuter Import keine Pflege
    überschreibt ([SCRIPTS.md](SCRIPTS.md) Abschnitt 1a).
  - Belegt: die Tests mit „schlagwort“ im Namen (Go, Vitest, Playwright), dazu
    `internal/api/portal/katalog_kollegium_pg_test.go`, `portalSuche.svelte.test.js`,
    `internal/katalogdienste/dnb_stichwoerter_test.go`, `internal/katalogdienste/dnb_nur_dnb_test.go`,
    `BuchEingabefelder.test.js` und `BestellSuche.test.js`.
- **Automatische Cover-Synchronisation:** Ein Hintergrund-Worker (`Cover-Sync`) sucht über ISBNs automatisch in externen Buch-APIs (z.B. Google Books) nach Buchcovern, lädt diese herunter und speichert sie datensparsam im WebP-Format.
- **Legacy-Import-Engine:** Für die initiale Einrichtung oder Datenübernahme bietet das System eine Import-Schnittstelle (`POST /api/import/littera`, Recht `manage_inventory`) für Altbestände aus _Littera_. Sie nimmt drei Dateiarten: das Katalogisat als MAB2-XML (aktualisiert die Angaben bestehender Titel) und eine Tabelle als CSV oder XLSX mit den Spalten Titel und Exemplarnummer (legt Titel und Exemplare an).
- **Drei Importwege in der Datenverwaltung, unterschieden danach, ob sie Exemplare übernehmen oder erzeugen:** Der _Katalog-Import_ (Littera-XML) aktualisiert Metadaten bestehender Titel. Der _Bestands-Import_ (Kombi-CSV, `/api/admin/import-bestand`, Recht `manage_inventory`) ÜBERNIMMT vorhandene Exemplare samt ihrer Nummer aus der Datei — sie tragen schon ein Etikett, `etikett_gedruckt = true`. Eine LMF-Nummer wird zum alten Etikett (`lmf_barcode`), das Exemplar bekommt eine kurze Nummer aus `barcode_seq` (§9), gezogen erst, wenn die übrigen Exemplare der Datei angelegt sind: So trifft keine kurze Nummer die Nummer eines Buchs derselben Datei. Ein zweiter Lauf derselben Datei legt nichts doppelt an und zieht keine Nummer. Ist eine Nummer der Datei der Ausweis eines Lesers, importiert er nichts und nennt die Zeilen (eine Nummer ist entweder ein Buch oder ein Ausweis). Der _Listenimport_ (`POST /api/books/import`, Recht `edit_books`) verlangt nur ISBN und Stückzahl und ERZEUGT neue Exemplare mit frischen Nummern aus `barcode_seq` — ohne Etikett, `etikett_gedruckt = false`, also auf der Nachdruck-Liste. Sein Knopf steht in der Datenverwaltung (`ListenImport.svelte`, E2E `datenverwaltung-importe.spec.js`). Weil er additiv ist, benennt das Widget jeden Lauf mit einem Idempotenz-Schlüssel je Dateiauswahl (`X-Idempotency-Key`, `internal/api/katalog/listenimport_schluessel.go`, Tabelle `idempotency_keys` wie bei den Theken-Aktionen): Nach einer verlorenen Antwort holt der zweite Klick das Ergebnis des ersten Laufs ab, statt die Exemplare doppelt anzulegen; solange der Lauf noch offen ist, antwortet der Server 409. Der Pfad steht in der 5-Minuten-Liste der Langläufer (`internal/api/middleware.go`).
- **Exemplarnummern — eine Quelle:** Jede Nummer, die das System selbst vergibt, kommt aus `barcode_seq` (Präfix `B-`, `internal/repository/barcode_vergabe.go`, mit Bestandsabgleich gegen von Hand eingetippte Nummern), in einer Form: `B-` und mindestens fünf Ziffern, eine längere Zahl steht ganz da (`B-100000`): Bestellwesen, Handvergabe in der Exemplarkarte, Littera-Import, Bestandskorrektur in der Buchmaske („Aktueller Bestand") und Listenimport der Titel-Verwaltung. Eine Nummer wird nie umgeschrieben oder recycelt — sie klebt physisch am Buch. Vorhandene `SYS-`/`AUTO-`-Exemplare (Platzhalter älterer Nummernkreise) bleiben deshalb, wie sie sind, und zeigen in der Exemplarkarte „Barcode scannen". Das Feld „Aktueller Bestand" nimmt höchstens 5.000 an, ebenso eine Zeile des Listenimports, wo eine größere Zahl als leer gilt (`repository.HoechsterBestandJeTitel`); eine Liste oder eine Bestellung legt zusammen höchstens 200.000 Exemplare an, geprüft vor der ersten Nummer, und keine Vergabe zieht mehr Nummern (`repository.HoechstensNummernJeVergabe`): Eine ISBN im Feld „Bestand" zöge sonst Millionen Nummern, die der Server nicht im Speicher hält und die verbraucht blieben. Belegt: `internal/api/katalog/barcode_vergabe_pg_test.go`, `internal/repository/bestandskorrektur_nummernkreis_pg_test.go`, `internal/repository/barcode_vergabe_grenze_test.go`, `internal/api/katalog/titel_aendern_test.go`, `internal/api/katalog/listenimport_zeilen_test.go`, `internal/api/bestellung/bestellung_abgelehnt_pg_test.go`.

- **Titel löschen:** ein Weg für einen und für mehrere Titel, `DELETE /api/books` mit Titeln, Grund
  und den bestätigten verliehenen Exemplaren (`repository.TitelAbfragen.LoescheTitel`); Buchakte,
  Buchmaske und Mehrfachauswahl öffnen denselben Dialog (`components/buchakte/titelLoeschen.js`). Er
  fragt nach dem Grund für die Exemplare im Haus: „Alter Bestand“ (vorgewählt: In Littera standen 10.275
  von 14.158 Abgängen mit Grund unter „Alter Bestand“, „veraltet“ und „al“, und M3 verlangt bei
  Radio-Buttons eine Vorauswahl),
  „Beschädigt“, „Verlust“, „Falsch angelegt“. Freitext gibt es nicht: Littera hatte 22 Schreibweisen
  für vier Bedeutungen, ein Viertel ohne Grund, und das Abgangsbuch filtert nach dem Grund. Die Gründe
  stehen in `frontend/src/lib/utils/aussonderungsgruende.js`, ihr Text in
  `internal/repository/aussonderungsgrund.go`; beide prüfen gegen `aussonderungsgruende.faelle.json`,
  der Go-Test auch gegen die CHECK-Bedingungen. Verliehene Exemplare nennt der Dialog mit Leser und
  Klasse: Ohne Kästchen bleiben sie beim Leser und ihr Titel, bis sie zurück sind; erst ein eigenes
  Kästchen löscht sie mit, und der Knopf nennt dann die Zahl aller Exemplare. Ein zweiter Dialog wäre
  unsicherer, sein Bestätigungsknopf stünde, wo eben geklickt wurde (M3, Dialogs). Der Server löscht
  nur verliehene Exemplare, die der Dialog bestätigt hat; wurde eines danach ausgeliehen, bleibt sein
  Titel oder das Löschen endet mit 409. „Exemplar löschen“ in der Buchakte fragt nach demselben Grund
  und verweist bei einem verliehenen Exemplar auf „Verlust oder Schaden melden“. Belegt:
  `internal/api/katalog/titel_loeschen_pg_test.go`, `internal/repository/titel_loeschen_samt_ausleihen_pg_test.go`,
  `internal/repository/aussonderungsgrund_zwilling_pg_test.go`.
- **Zugangsbuch** (Berichte → „Bestandsbücher“): Welche Exemplare in einem Zeitraum in den Bestand
  kamen, mit Eingangsdatum, Titel, Anzahl, Barcodes und Lieferant, wie die Arbeitshilfe es verlangt
  ([mittel_konzept.md](mittel_konzept.md) 7.1; die Inventarnummer ist der Barcode). Eine Zeile fasst
  zusammen, was in allen Spalten gleich ist (im Abgangsbuch: derselbe Tag, derselbe Grund); fortlaufende
  Barcodes ab drei stehen als „von – bis“, auf dem Blatt laufen sie um, statt gekürzt zu werden
  (`internal/bestandsbuch`). Ohne gewähltes Datum steht das vorige Schulhalbjahr da, weil „zu Beginn
  jedes Schulhalbjahres ein Ausdruck der Neuanschaffungen des vorhergehenden Halbjahres“ gedruckt wird
  (Leitfaden Lernmittelfreiheit, 11.5); jede Seite trägt „Seite x von y“. Zugangs- und Abgangsbuch
  teilen Bauteil und Abschnitts-Bildung (`components/bestand/Bestandsbuch.svelte`,
  `internal/api/katalog/bestandsbuch.go`). Am Bildschirm zeichnet ein Abschnitt je 50 Zeilen nach „Mehr
  laden“, weil ein Listenimport Zehntausende Zugänge an einem Tag anlegen kann; Stückzahl und Ausdruck
  umfassen immer alle. Der Topf kommt aus einem Beleg, nicht aus dem Titel: Eigentum am Exemplar,
  sonst die Bestellung (`repository.exemplarTopfBelegtSQL`); ohne beides steht ein Exemplar unter „ohne
  Zuordnung“ (Altbestand ohne Littera-Vermerk, Handanlage, Bestandskorrektur). **Das Zugangsdatum ist
  die Lieferung, nicht die Bestellung** (`zugang_am`): Ein Trigger setzt es, sobald ein Exemplar in den
  Bestand kommt (Wareneingang, Freigeben, Zurückholen); im Zulauf steht es in keinem Zugangsbuch. Ohne
  Bestellung ist es der Tag der Anlage, bei der Littera-Übernahme das Datum aus Littera. Ein später
  ausgesondertes oder gelöschtes Exemplar bleibt im Zugangsbuch; Titel löschen und „Verluste endgültig
  löschen“ legen beim Löschen seine Zeile für beide Bücher an (`geloeschte_exemplare`,
  `repository.loescheExemplareMitBuchzeile`). Belegt: `internal/api/katalog/zugangsbuch_pg_test.go`,
  `internal/repository/zugangsdatum_pg_test.go`, `frontend/e2e/zugangsbuch.spec.js`.
- **Abgangsbuch** (Berichte → „Bestandsbücher“): Welche Exemplare in einem Zeitraum aus dem Bestand
  gingen, mit Datum, Titel, Signatur, Grund (Alter Bestand, Beschädigt, Verlust, Falsch angelegt;
  `internal/repository/aussonderungsgrund.go`), Anzahl und Barcodes. Ein Abgang ist nur, was im Bestand
  war: Ein bestelltes Exemplar, das nie eintraf und gelöscht oder ausgesondert wurde, steht in keinem
  der beiden Bücher (`sqlIstAbgang`). Ein gelöschtes Exemplar steht am Tag des Löschens darin, ein schon
  ausgesondertes mit Tag und Grund von damals. Die Statistik zählt nur, was im Bestand ist oder war
  (`repository.sqlWarImBestand`); Verluste, Verlustquote und Wiederbeschaffungswert folgen der Grenze
  des Abgangsbuchs und zählen wie dieses gelöschte Exemplare mit (`geloeschte_exemplare` trägt dafür
  Einkaufspreis und Art des Titels). Die Quote teilt durch alles, was im Bestand war; der Gesamtbestand
  nennt die Exemplare im System. Vorbelegt ist das laufende Schulhalbjahr mit
  den Stichtagen der Bestandskartei (15.3. und 15.9.). Der Zeitraum wird am Server bestimmt, damit
  Bildschirm und Ausdruck denselben meinen, und umfasst höchstens ein Jahr (entschieden am 16.10.2026):
  Jede Zeile steht in der Antwort, weil jemand den Nachweis unterschreibt; mehrere Jahre gehen als
  mehrere Ausdrucke (`bestandsbuchZeitraum`). Drei Abschnitte mit eigener Stückzahl, Land,
  Schulträger und Spende, weil die Finanzen der Töpfe getrennt geführt werden; der Topf folgt dem Eigentum wie am
  Etikett (`repository.ExemplarTopfSQL`). Das Blatt kommt über `GET /api/bestand/abgangsbuch/pdf`. **Was
  nicht darauf steht, steht darunter:** Was sich keinem Zeitraum zuordnen lässt (Abgänge ohne Datum,
  Löschungen ohne Buchzeile), nennt das Abgangsbuch als Zahl, statt Vollständigkeit zu behaupten. Das
  Datum setzt ein Trigger am Zustandswechsel, nicht jeder Aussonder-Weg. Wer ausgesondert hat, nennt das
  Abgangsbuch nicht; das steht mit Weg und Grund im Protokoll, auf jedem Weg (Status in der Buchakte,
  kleinerer Bestand, Abschluss der Inventur, Verlust oder Schaden; `repository.protokolliereAussonderung`).
  Belegt: `internal/api/katalog/abgangsbuch_pg_test.go`, `internal/api/katalog/bestandsbuch_tuer_pg_test.go`,
  `internal/api/katalog/abgangsdatum_pg_test.go`, `internal/repository/titel_loeschen_bestand_pg_test.go`,
  `internal/repository/eigentum_bestandsbuch_pg_test.go`, `internal/api/system/statistik_verlust_pg_test.go`,
  `internal/api/aussondern_protokoll_pg_test.go`, `frontend/e2e/abgangsbuch.spec.js`.

---

## 14. Schadensmanagement

- **Eine Forderung entsteht, wenn jemand in der Leserakte ein verliehenes Buch als verloren oder
  beschädigt meldet** („Verlust oder Schaden melden“, `POST /api/damage/report`, Recht `edit_students`).
  Die Inventur legt keine an, sie bucht nur Bücher aus dem Regal aus (§6). Offene Forderungen halten
  die DSGVO-Löschung eines Schülers auf und lassen sich als PDF-Rechnung drucken.
- **Gegen einen Kollegen entsteht KEINE Forderung.** „Verlust oder Schaden melden" steht in seiner
  Akte und bucht, was den Bestand angeht — das Exemplar wird ausgesondert, die Ausleihe endet, eine Vormerkung
  darauf wird gelöst —, aber ohne Schadensfall. Grund: Der Weg einer Forderung endet im Schadensersatz-Bescheid,
  und der ist ein Schreiben an Erziehungsberechtigte; er braucht Klasse und Anschrift, die von einer Lehrkraft
  nirgends stehen. Ersatz von einer Lehrkraft zu verlangen ist Sache der Schulleitung gegenüber dem Dienstherrn
  und dort auch nur bei Vorsatz oder grober Fahrlässigkeit.
- **Gelöscht ist gelöscht, auch im Bescheid:** Wer einen Titel löscht, nimmt die Forderungen
  seiner Exemplare mit, auch eine, die schon in einem Schadensersatz-Bescheid steht; „Verluste
  endgültig löschen" nach der Inventur nimmt die bezahlten und stornierten mit. Der Bescheid behält
  Nummer und Gesamtbetrag, verliert die Position und gilt danach als erledigt
  (`repository.bescheidHatOffenePosition`). Eine offene Forderung hinterlässt ihre Spur im
  Protokoll (`internal/repository/titel_loeschen_spur.go`). Ein verliehenes Exemplar geht nur mit, wenn der
  Mensch es im Löschdialog bestätigt (Abschnitt 13, „Titel löschen").
- **Was ein Buch heute noch wert ist, steht am Exemplar** (`internal/pkg/ersatzwert`): Die Buchakte
  zeigt den Ersatzwert samt Herleitung („3. Verleihjahr → 60 % von 24,90 €“). Für ein Buch des Landes
  gilt die Staffel der Arbeitshilfe (1. Verleihjahr voller Preis, dann 80/60/40/20 %, ab dem 6. Jahr
  10 %), für eines der Schülerbücherei der Neuwert ohne Altersabschlag; Herkunft und Rechnung in
  [mittel_konzept.md](mittel_konzept.md) 1 und 4.2. Wem ein Buch gehört, sagt die Regel des Etiketts
  (`repository.ExemplarTopfSQL`), nicht `ist_lernmittel`: Eine Lektüre aus LMF-Mitteln gehört dem Land.
  Ein beschädigtes Exemplar trägt einen Prozentwert für seinen Zustand (`zustand_abwertung_prozent`, im
  Zustands-Dialog der Buchakte, nicht an der Theke): Er wirkt nach der Staffel und mindert jeden
  künftigen Ersatzbetrag in beiden Töpfen, zieht das Buch aber nicht aus dem Verkehr. Grundlage ist der
  heutige Listenpreis oder, mit der Einstellung „Immer mit dem Einkaufspreis rechnen“ (Kategorie
  Schadensersatz), der damals bezahlte; die Herleitung nennt, welcher galt, und unterscheidet „kein
  Listenpreis hinterlegt“ von „so eingestellt“. In jedem Fall ein Vorschlag, den der Mensch bestätigt
  oder überschreibt; ohne Preis bleibt das Feld bei 0 mit einem Satz, der es sagt.
- **Wohin gezahlt wird, steht in der Rechnung** (`internal/pdf/zahlungsweg.go`): Für ein
  Buch des Landes (Eigentum nach `repository.ExemplarTopfSQL`) nennt sie die Zahlstelle und
  die Bankverbindung aus den Einstellungen — dieselbe Angabe wie im Schadensersatz-Bescheid. Für ein Buch der
  Schülerbücherei steht dort, dass der Weg des Schulträgers noch nicht hinterlegt ist; er ist
  eine offene Frage an den Träger. Trägt eine Rechnung beides, stehen beide Wege mit ihrer
  Teilsumme da. Bar in der Bibliothek zu zahlen ist für Landeseigentum die Ausnahme mit
  Quittung, nicht der Weg, den die Rechnung vorsieht.
- **Erledigt wird eine Gebühr in der Leserakte** auf genau zwei Wegen:
  **„Bezahlt“** (Barzahlung an der Theke) oder **„Stornieren“** mit Pflicht-Begründung
  (Erlass, Buch wiedergefunden, Kulanz). Beides verlangt das Recht `edit_students` —
  bewusst nicht die Theken-/Helfer-Rolle. Der Betrag im revisionssicheren Audit-Eintrag
  stammt immer aus der Datenbank, nie aus der Anfrage; eine bereits erledigte Gebühr
  meldet dem zweiten Bearbeiter einen Konflikt (409) statt einer Doppelbuchung. Die getippte
  Begründung steht an der Forderung und im Eintrag; mit der Anonymisierung oder dem
  endgültigen Löschen des Lesers fällt sie an beiden Stellen, Betrag und Zeitpunkt bleiben.
  Stornierte wie bezahlte Gebühren geben Schülerlöschung, LUSD-Abgleich und Ausleihe
  wieder frei; die DSGVO-Auskunft weist Stornierungszeitpunkt und -grund transparent aus.

---

## 15. Selbstprüfung der Betriebsbereitschaft

Erreichbar unter **System → Einstellungen → Kategorie „Betriebsbereitschaft“** (Recht `manage_settings`,
`GET /api/admin/system/betriebsbereitschaft`). Die Seite beantwortet **eine** Frage:
_Was ist eingerichtet, aber nicht in Betrieb?_

Der Anlass ist eine Fehlerart: Eine Funktion ist fertig und verdrahtet und tut nichts, weil eine
Einstellung fehlt, ohne Fehler, Statuscode oder Logeintrag.

**Geprüft werden zweiundzwanzig Bereiche** (`internal/bereitschaft`), darunter Auslagerung der
Backups, Geheimnisse, Schlüssel und Bestand (lässt sich je verschlüsselter Spalte ein Wert mit dem
laufenden `APP_ENCRYPTION_KEY` entschlüsseln, für die Wiederherstellung mit neuer `.env`), Anmeldung,
vertraute Proxys, Bestell-Bestätigungslink, Mailversand, Demo-Daten aus `seed_demo.sql` und
Admin-Konten. Einige mit eigener Regel:
- **Lieferanten:** Reste der drei Beispiel-Lieferanten früher Fassungen sind kritisch, denn eine
  Bestellung an so einen Eintrag ginge an die Beispiel-Adresse und meldete „gesendet“.
- **Rechte-Vorgabe:** Abgleich von `role_permissions` gegen die Vorgabe im Code; der Seed fasst
  bestehende Zeilen nie an. Eine Abweichung ist nur eine Warnung, weil sie eine Entscheidung des
  Admins sein kann.
- **Klassen-Zuordnung:** Klassen bis Jahrgang 10 ohne Lehrkraft, verwaiste Zuordnungen und Bücherlisten
  ohne Klasse; die Oberstufe hat keine Klassenleitung.
- **Ehemalige mit offenen Vorgängen** seit mehr als einem Jahr: Der Vorgang schützt sie vor
  Anonymisierung und Löschung, Name und Anschrift blieben sonst auf Dauer. Warnung mit Abhilfe
  („Verlust melden, Forderung bezahlen oder stornieren“), kein Automatismus.
- **Schadensersatz-Bescheid:** welche Pflichtangaben fehlen, mit derselben Prüfung, mit der das
  Erstellen abweist; Warnung, weil ohne Bescheid-Bedarf nichts ausfällt.
- **Littera-Übernahme:** das Ergebnis des schreibenden Laufs (`littera_uebernahme_ergebnis`), eine Woche
  lang kritisch bei Abbruch oder Lücke, danach erledigt, damit eine hingenommene Lücke nicht täglich
  meldet.

Jeder Befund nennt **Befund**, **Folge** und **Abhilfe**, ohne die letzte landete die Meldung auf einem
Zettel statt in der `.env`, dazu eine **Stufe**: `ok`, `warnung` (läuft, aber nicht wie gedacht) oder
`kritisch` (löst die Alarm-Mail aus). Bewusst nur drei, eine feinere Skala liest niemand. Die Seite
ändert nichts; die Urteile sind eine reine Funktion über eine eingesammelte Lage
(`internal/bereitschaft/bereitschaft.go`).

**Der Wächter meldet sich selbst:** Kritische Befunde gehen per Mail, solange sie bestehen, 3 Minuten
nach jedem Start und danach täglich; ohne neuen Punkt geht die Mail erst 23 Stunden nach der letzten
wieder hinaus ([ARCHITEKTUR.md](ARCHITEKTUR.md) 8.10). Empfänger sind die Adressen aus Einstellungen →
Erreichbarkeit & Alarme (`alarm_empfaenger`); ist das leer, alle aktiven Admin-Konten, denn ein
Alarm, der niemanden erreicht, ist keiner. Der Fußtext nennt, an wen die Mail ging. Warnungen mailt
er nur mit dem Schalter „Auch Warnungen mailen“ (`alarm_warnungen`, Vorgabe aus), sonst stünde
Dauerrauschen in der Mail; dazu zählt der abgewiesene LUSD-Abgleich (ohne Namen). Auf Spielwiesen
(`APP_ENV=local/development/test`) schweigt er, denn die lokale `.env` zeigt auf den echten
Schul-SMTP.

**Fehler im Log** stehen auf einer eigenen Seite darunter (`GET /api/admin/system/fehler-im-log`,
`manage_settings`): Die Betriebsbereitschaft nennt Zustände, das Log Ereignisse. Gezählt werden die
Zeilen der Stufe ERROR je Meldung über 24 Stunden, eine fehlgeschlagene Anfrage je Route; Warnungen
zählt der Server getrennt, die Seite zeigt sie nicht ([ARCHITEKTUR.md](ARCHITEKTUR.md) 8.11). Einen
Absturz, einen harten Abbruch und einen Ausfall des Rechners meldet der nächste Start; Update und
Neustart melden nichts. Schon eine Fehlerzeile ist ein Grund für die Mail: bis zu dreißig Meldungen
einzeln, je mit erster und letzter Zeit und der ganzen letzten Zeile samt Feldern (bei einem Panic mit
Stack, höchstens 4000 Zeichen), damit sich ein Fehler beheben lässt, ohne das Log am Server zu lesen.
Das Log nennt Personen mit ihrer Kennung, nicht mit Namen ([STIL.md](STIL.md), Abschnitt 11);
Mailadressen stehen in der Mail nur mit ihrer Domain. Jede einzelne Zeile zeigt der Befehl in
[DEPLOYMENT.md](DEPLOYMENT.md), „Fehler im Log finden“.

---

## 16. Öffentliche Seiten (ohne Anmeldung)

Drei Adressen sind bewusst **vor** dem Login-Zweig eingehängt (`App.svelte`): Sie rendern, ohne
dass jemand angemeldet ist, und sie liefern ausschließlich Titeldaten — nie Ausleiher, nie
Namen, nie Klassen (Nachweis: `docs/PII_MATRIX.de.md`, Zeilen zu `/api/public/opac/suche`,
`/api/monitor/slides`, `/api/public/bestellung/*`). Keine der drei Seiten hat einen Menüpunkt; ihre
Adressen stehen unter _Einstellungen → Erreichbarkeit & Alarme_ zum Kopieren.

### 16.1 Katalog (OPAC) — `/katalog`

Für Schülerinnen, Schüler, Eltern und Kollegium: Suche nach Titel, Autor, ISBN oder Schlagwort
(die eine Titelsuche des Programms, `repository.sqlTitelTrifft`; ein Verweis findet sein Schlagwort, §13;
`internal/api/portal/opac.go`), Trefferkarte mit Cover und
„N von M verfügbar". Gezählt werden nur ausleihbare, nicht ausgesonderte Exemplare ohne offene
Ausleihe, die nicht für eine Vormerkung im Abholfach liegen, solange die Abholfrist läuft; so
zählen Katalog, Suche, Klassensätze und Lernmittel (`repository.SQLBestandVerfuegbar`,
`sqlFilterVerfuegbar`), mit derselben Bedingung, mit der die Theke das Exemplar einem anderen
verweigert. Die Buchakte nennt ein solches Exemplar „Im Abholfach“ statt „Verfügbar“. Welche Titel überhaupt erscheinen, regelt die **gemeinsame Sichtbarkeitsregel der
öffentlichen Seiten** (`repository.OeffentlichSichtbar`, für Katalog und Monitor
dieselbe): kein Lernmittel (`buecher_titel.ist_lernmittel`) und
mindestens ein Exemplar im Haus (nicht
ausgesondert, nicht nur bestellt). Maximal 50 Treffer; der Kopf `X-Treffer-Gesamt` nennt, wie
viele es insgesamt sind, und die Seite sagt dann „50 von N Treffern. Ein Suchwort grenzt
ein.“ statt „50 Treffer“.
Das Suchfeld ist scannertauglich (Enter löst die Suche aus). Nach einem Update steht mit der
nächsten Suche über dem Suchfeld der Hinweis „Neu laden", wie in der Verwaltung (§16.2). Das
Kollegiums-Portal sucht mit derselben Abfrage über eine eigene Tür hinter der Anmeldung, die
auch Lernmittel und bestellte Titel zeigt (§4.2).

### 16.2 Bibliotheks-Monitor — `/monitor`

Für einen Bildschirm im Flur oder in der Bibliothek: eine Endlos-Slideshow aus drei Folien, an
einem Tag des LMF-Plans aus vier (`internal/repository/oeffentlich.go` baut die Folien,
`internal/api/portal/monitor.go` liefert sie aus, `Monitor.svelte` zeigt sie). Auf die drei
Folien mit Büchern kommt nur, was die gemeinsame Sichtbarkeitsregel der öffentlichen
Seiten (§16.1) zulässt — **kein Lernmittel, mindestens ein Exemplar im Haus**. Ohne sie
wäre auf einem Bestand, der zum größten Teil aus Schulbüchern besteht, das Mathebuch der 7
„Buch des Monats".

| Folie                   | Regel                                                                                                                                                               |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Buch des Monats**     | der Titel **mit Cover**, der in den letzten **30 Tagen** am häufigsten ausgeliehen wurde; gibt es keinen, der zuletzt angelegte Titel mit Cover |
| **Neu eingetroffen**    | die zehn zuletzt angelegten Titel mit Cover                                                                                                                         |
| **Beliebt diese Woche** | die fünf Titel mit den meisten Ausleihen der letzten **7 Tage** — auch ohne Cover (Platzhalter-Kachel)                                                  |
| **Heute Bücherausgabe** / **Heute Büchertausch** | nur an einem Tag mit Zeilen im veröffentlichten LMF-Plan (§2.3): je Zeile Stunde, Klassen und Vermerk; welcher Plan gilt, entscheidet dieselbe Regel wie im Portal (`LMFAbfragen.LadeNaechstenPlan`). Die Folie steht vorn im Kreis; an jedem anderen Tag gibt es weder Folie noch Punkt |

Gezählt werden **Ausleihen, ohne Dauerleihen**. Ein Klassensatz an eine Lehrkraft sind 30
Ausleihzeilen und null freiwillige Leser; er hätte sonst beide Folien beherrscht. Nach Lesern lässt
sich nicht zählen: Die Lesehistorie trennt eine Ausleihe der Schülerbücherei einen Tag nach der
Rückgabe von ihrem Leser (§8), gezählt würde dann fast nur, was gerade noch ausgeliehen ist.

Alle Folien sind im Raster eines Fernsehers gezeichnet, 864 × 450 dp innerhalb des Rands
(Google TV, Layouts: „Always design at MDPI resolution at 960px * 540px"), und wachsen mit dem
Bildschirm, auf Full HD auf das Doppelte; „Neu eingetroffen" behält dabei alle zehn Cover in
einer Reihe. Der Vermerk aus dem Plan steht auf der Folie „Heute" für jeden im Flur lesbar.

Auf „Buch des Monats" und „Neu eingetroffen" kommen Titel ohne Cover bewusst nicht vor — eine
Folie ohne Bild ist auf einem Flurbildschirm wertlos. „Beliebt" zeigt sie mit Platzhalter, sonst
wäre die Liste kürzer, ohne dass jemand etwas gewinnt. Die Seite braucht keine Bedienung und
keine Anmeldung; sie hat nur einen einzigen Lese-Endpunkt (`GET /api/monitor/slides`, öffentlich,
siehe SECURITY.md). Katalog und Monitor urteilen über denselben Titel gleich — das hält
`internal/api/portal/monitor_pg_test.go` fest.

**Kiosk-Verhalten** (`frontend/src/lib/monitor/monitorTakt.svelte.js`, mit gestellter Uhr geprüft):
Folienwechsel alle 15 s, Cover-Lauf alle 2,5 s auf „Neu eingetroffen". Die Daten werden **alle
5 Minuten nachgeladen**; solange noch nichts da ist (Server nach einem Stromausfall noch nicht
oben), wird alle 30 s neu versucht. Ein neuer Stand erscheint erst mit dem nächsten Folienwechsel,
nicht unter den Augen des Betrachters; scheitert ein Abruf, bleibt der alte Stand stehen. **Leere
Folien werden übersprungen** — in sechs Wochen Sommerferien hat „Beliebt diese Woche" keine
Ausleihe, und ein Bildschirm, der ein Drittel der Zeit „Keine Daten verfügbar" zeigt, wird
abgeschaltet; sind alle drei leer, bleibt die aktuelle stehen. **Einmal täglich um 03:00 Uhr lädt
sich die Seite selbst neu** — ein Bildschirm ohne Tastatur navigiert nie, und der Service Worker
prüft nur beim Laden auf eine neue Version; ohne den Neustart bekäme der Monitor ein Deploy erst
mit, wenn jemand den Stecker zieht. Die Theke lädt sich nicht von selbst neu, sobald an ihr
gearbeitet wurde (sie darf sich nie mitten in einer Ausleihe erneuern); nach einem Update steht
über dem Inhalt der Hinweis „Neu laden".

### 16.3 Bestellbestätigung durch den Händler — `/bestellung/<token>`

Der Link aus der Bestellmail (§7). Der Händler sieht die Positionen, druckt bei Bedarf die
Etiketten und bestätigt; der Token steht im Pfad und wird deshalb nie protokolliert (Fund
„Geheimnis im Pfad landet im Log"). Sicherheitszuschnitt in SECURITY.md.

---

## 17. Einstellungen — die 17 Kategorien

Erreichbar unter _System → Einstellungen_ (`/einstellungen`, `components/einstellungen/kategorien.js`).
Jede Kategorie wird **einzeln** gespeichert: `PUT /api/einstellungen` nimmt nur die Felder dieser
Kategorie (`repository.EinstellungenPatch`, jedes Feld ein Zeiger); ein nicht gesendetes Feld
bleibt unangetastet — Prinzip „nil = unverändert" —, und ein Auftrag ohne ein einziges Feld
ist 400. Ein Speichern des ganzen Formulars hätte bei jeder Kategorie die Felder der anderen
mit Leerwerten überschrieben.

| #   | Kategorie                   | Inhalt                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Recht                                                           |
| --- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| 1   | **Schule**                  | Name, Anschrift und zwei Eigentumsvermerke fürs Etikett: Lernmittel (leer = Vorgabe) und Schülerbücherei, für Bücher aus Mitteln des Schulträgers (leer = kein Vermerk)                                                                                                                                                                                                                                                                                                                                                                                                                                                             | `manage_settings`                                               |
| 2   | **Ausleihe & Fristen**      | Tage je Buch / je Medium / je Gerät (Vorgabe 21, 7 und 14), Ausleihlimit je Schüler (Vorgabe 5), Eingangsjahrgänge (`lmf_eingangsjahrgaenge`, Vorgabe „5, 7"), Ferien-Leseclub                                                                                                                                                                                                                                                                                                                                                                                                                              | `manage_settings`                                               |
| 3   | **Mahnwesen**               | automatische Ausleihsperre: ab wie vielen überfälligen Medien und nach wie vielen Toleranztagen (`max_overdue_items`, `max_overdue_days`)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | `manage_settings`                                               |
| 4   | **Klassenleitungen**        | Klasse → E-Mail der Klassenleitung (`klassen_lehrer_mapping`); Empfänger für Sammel-Mahnlauf und Abgänger-Kontoauszüge; wird beim Schuljahreswechsel mit versetzt                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `manage_settings`                                               |
| 5   | **Bestellwesen**            | Bedarfswarnung an/aus (Schulbücher, deren Bestand nicht für die Kinder im nächsten Schuljahr reicht, §7), Preise erfassen, Gültigkeit des Bestätigungs-Links für den Händler in Tagen (`bestelllink_gueltigkeit_tage`, Vorgabe 21)                                                                                                                                                                                                                                                                                                                                                                                              | `manage_settings`                                               |
| 6   | **Lieferanten**             | Händler mit E-Mail und Kundennummer; genau **einer** ist Hauptlieferant und wird im Bestellformular vorausgewählt                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `create_orders`                                                 |
| 7   | **Schlagworte**             | Pflege der Schlagworte am Titel (§13): umbenennen und zusammenführen (alle Titel ändern sich mit, die alte Schreibweise wahlweise als Verweis), Verweise anlegen, löschen — auch mehrere auf einmal —, Filter-Markierung fürs Portal                                                                                                                                                                                                                                                                                                                                                                                                | `edit_books`                                                    |
| 8   | **Schadensersatz**          | Angaben für den Schadensersatz-Bescheid: Nummer des Schulamtsbereichs und Schulnummer (Teil der Referenznummer), Aufsichtsbehörde (Name und Anschrift), Name der Schulleitung, Geschäftszeichen, Bearbeiter, Durchwahl (freiwillig), Zahlstelle und Bankverbindung (vorbelegt aus dem Musterschreiben), Zahlungsfrist in Tagen ab Briefdatum (Vorgabe 28); Schalter **„Immer mit dem Einkaufspreis rechnen"** (`ersatzwert_immer_kaufpreis`): aus gilt ab dem zweiten Verleihjahr der Listenpreis (Arbeitshilfe des Landes), ohne erfassten Listenpreis ersatzweise der Einkaufspreis; an gilt immer der Einkaufspreis              | `manage_settings`                                               |
| 9   | **Datenschutz & Sitzung**   | Löschfristen (Lesehistorie 1 Tag, Lernmittel-Historie 730 Tage, Anliegen 365 Tage, Audit 24 Monate), Karenzzeit für Ehemalige (Vorgabe 90 Tage, §8.2), Theke leeren nach _n_ Minuten (Vorgabe 5), Sperrbildschirm nach _n_ Minuten (Vorgabe 15)                                                                                                                                                                                                                                                                                                                                                                                     | `manage_settings`                                               |
| 10  | **Erreichbarkeit & Alarme** | öffentliche Adresse (Basis für Bestätigungs-Link, Katalog- und Monitor-Adresse; leer = keine Links), Alarm-Empfänger (leer = alle aktiven Admins)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `manage_settings`                                               |
| 11  | **Mail**                    | SMTP-Postausgang mit Verbindungstest, Test-Mail; zweiter Absender mit Häkchen, welche Mails über ihn gehen: Mails aus Mein Portal (ab Werk), Mahnlisten und Abgänger-Auszüge an die Klassenleitungen, Bestellmails; alle anderen, und ohne Adresse alle, gehen über den Postausgang; **Adressen aus dem Webmail**: Zugang zum Roundcube der Schule, monatlicher Abgleich mit dem globalen Adressbuch, der Konten mit Platzhalter-Adresse ihre Adresse der Schule einträgt (Probelauf und Knopf „Jetzt abgleichen“); **Mail-Vorlagen** `MAHNUNG_ELTERN` (gedruckter Elternbrief: `{{.Vorname}} {{.Nachname}} {{.BuchListe}} {{.Frist}}`) und `BESTELLUNG_HAENDLER` (Bestellmail: `{{.Datum}} {{.Kundennummer}} {{.AnzahlTitel}} {{.AnzahlExemplare}} {{.BestaetigungsLink}} {{.LinkGueltigBis}} {{.Mittel}}`) — Platzhalter je Typ, Gate `mail_vorlagen_platzhalter_test.go`. Die SMTP-Daten aus der `.env` werden nur beim ersten Start übernommen; danach gilt die Datenbank | `manage_settings`                                               |
| 12  | **LMF-Aktionen**            | Massenverlängerung: alle offenen Lernmittel-Ausleihen **einer Klasse** auf ein neues Rückgabedatum (`POST /api/ausleihen/global-extend-lmf`), mit Rückfrage vor dem Ausführen. Verlängert wird nur, was früher fällig ist; ein Band für mehrere Jahre behält seine Frist. Ein Datum in der Vergangenheit oder mehr als 13 Jahre voraus weist der Server ab                                                                                                                                                                                                                                                                                                                                                                                                                                                       | `edit_books`                                                    |
| 13  | **Datenverwaltung**         | Katalog-Import (Littera: Katalogisat-XML, CSV oder XLSX), „Bestand mit vorhandenen Nummern“ (CSV mit Kopfzeile), Listenimport (ISBN + Stückzahl, §13), „Fehlende Cover laden“, Daten exportieren (Katalog als CSV), **Offline-Sicherungen einspielen** (siehe §18.4)                                                                                                                                                                                                                                                                                                                                                                | `manage_inventory` / `edit_books`                               |
| 14  | **LUSD & Versetzung**       | LUSD-Abgleich (§8), Klassen-Versetzung zum Schuljahresende und die Sommerferien für den LMF-Plan (§2.3) (Rechte: import_students, manage_students_admin bzw. für die Sommerferien manage_settings)                                                                                                                                                                                                                                                                                                                     | `import_students` / `manage_students_admin` / `manage_settings` |
| 15  | **Sicherung**               | Schalter „Jede Nacht selbst sichern“ (`sicherung_an`), Vorgabe aus: Der Server der Schule wird selbst gesichert. An: Nachtsicherung, Kopie außer Haus, Restore-Probe (§15). Aus: keine eigene Sicherung, keine Probe, ältere Stände werden nach der Aufbewahrung gelöscht (`internal/jobs/sicherung_aus.go`), die Selbstprüfung nennt das ohne Befund                                                                                                                                                                                                                                                                                        | `manage_settings`                                               |
| 16  | **Betriebsbereitschaft**    | Selbstprüfung (§15)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 | `manage_settings`                                               |
| 17  | **Fehler im Log**           | Fehlermeldungen des Servers aus den letzten 24 Stunden (§15)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | `manage_settings`                                               |

---

## 18. Theke: Ergänzungen zur Omnibox (§1)

### 18.1 Kamera als Barcode-Scanner

Der Handscanner ist der Normalfall. Zusätzlich kann die Kamera des Geräts scannen
(`ThekenKamera.svelte`, Bibliothek `html5-qrcode`, wird erst beim Einschalten geladen) —
gedacht für Laptops oder Tablets ohne Scanner. Ein-/Ausschalter neben dem Omnibox-Feld.

### 18.2 Passbild per Webcam

In der Theke lässt sich ein Passbild aufnehmen (`WebcamAufnahme.svelte`,
`POST /api/schueler/{id}/photo`). Das Bild wird **verschlüsselt** gespeichert
(`schueler_fotos.foto_encrypted`, AES). Aufnehmen verlangt das Recht `upload_photos`,
ausgeliefert wird es nur mit `view_students`.

### 18.3 Theke leeren und Sperrbildschirm

Nach `theke_leeren_minuten` ohne Eingabe (Vorgabe 5) schließt sich die offene Leserakte,
nach `sperre_minuten` (Vorgabe 15) erscheint der Sperrbildschirm — die Anwendung ist dann nicht
mehr im DOM; entsperrt wird mit dem Passwort, nicht mit Maus oder Tastatur. Beides ist
Datenschutz an einer Theke, an der Schüler mitlesen können (Kategorie Datenschutz & Sitzung).
Das Abmelden leert die Theke ebenso. Kommt die Antwort auf einen Scan oder eine Suche erst danach
an, öffnet sie keine Akte, keine Rückfrage und keine Trefferliste mehr; was der Server gebucht
hat, bleibt gebucht, und ein Buch, das den Server nicht erreicht hat, geht mit dem Leser oder
Ausweis vom Scan in die Warteschlange.

Der Server sperrt die Anmeldung mit: Neuladen oder ein neuer Tab zeigen wieder den
Sperrbildschirm, und hinter ihm liefert der Server keine Daten. Erreicht ihn die Sperre nicht
(Neustart, kurzer Ausfall), fragt der Bildschirm jede Minute noch einmal, bis er sie bestätigt.
Alle Fenster eines Browsers sperren und öffnen gemeinsam; wird in einem gearbeitet, sperrt
keines. Ist der Mailserver der Schule nicht erreichbar, schließt das Passwort trotzdem auf — der
Server prüft es dann gegen einen Wert, den er sich bei der Anmeldung gemerkt hat und beim
Abmelden löscht.

Denselben Sperrbildschirm zeigt der Server 10 Stunden nach der letzten Passworteingabe, auch
mitten in der Arbeit; er nennt dann diesen Grund („Gesperrt nach 10 Stunden"). Eine offene Maske
bleibt dahinter stehen, und wer gerade auf „Speichern" geklickt hat, klickt nach dem Aufschließen
noch einmal: Gespeichert wurde nichts. Die Frist ist nicht einstellbar; sie deckt den längsten
Schultag mit einer Stunde Puffer.

### 18.4 Die Theke ohne Verbindung

Fällt die Verbindung aus, geht der Betrieb an der Theke weiter. Was dabei geschieht:

**Scannen.** Jeder Scan wird auf dem Theken-Rechner eingeordnet, wie es sonst der Server tut
(`scanEinordnen.js`, der Zwilling von `omnibox.go`): `B-…` und `LMF-…` sind Bücher,
`A-…`/`S-…`/`L-…` sind Ausweise, `G-…` sind Geräte. Eine nackte Ziffernfolge entscheidet die
Buch-Barcode-Liste, die der Rechner bei jeder Anmeldung holt (`GET /api/action/buchbarcodes`,
nur Buchnummern, keine Personendaten); ein Littera-Etikett wird dafür aus der EAN-13
zurückgerechnet. Steht die Nummer nicht auf der Liste, gilt der Scan als **unklar** und wird
nicht gebucht — wer hier riete, schriebe das nächste Buch einer fremden Person zu.

**Personen.** Ein ohne Netz gescannter Ausweis wird als NUMMER gemerkt; die folgenden Bücher
tragen sie, der Server löst sie beim Nachbuchen auf. Vor jedem weiteren Buch fragt die Theke
den Server nach dem gemerkten Ausweis; erreicht sie ihn nicht, geht das Buch mit der Nummer in
die Warteschlange und nie ohne Person hinaus, denn der Server läse es als Rückgabe. Personendaten
liegen bewusst nicht auf dem Rechner — eine Namenssuche gibt es ohne Netz deshalb nicht. Ein
unklarer Scan lässt den Merker fallen: Ab dann wird nichts mehr zugeordnet, bis ein eindeutiger
Ausweis kommt.

**Warteschlange.** Die Scans liegen in einer lokalen IndexedDB (`offlineWarteschlange.js`) mit
Scan-Zeitpunkt und Idempotenz-Schlüssel. Ein Band oben zeigt den Zustand; die Theke wird dabei
nicht verdeckt, und ohne Netz sperrt auch der Inaktivitäts-Wächter nicht (aufgeschlossen wird
am Server, und der ist dann nicht erreichbar). Geleert wird die Theke weiter. War die Sperre
fällig, wenn die Verbindung zurückkommt, überträgt der Rechner erst die Warteschlange und sperrt
dann.

**Nachbuchen.** Kommt die Verbindung zurück, schickt der Rechner die Warteschlange in Portionen
an `POST /api/action/nachbuchen`. Der Server bucht die Wirklichkeit: Lag das Buch bei jemand
anderem, nimmt er es dort zurück und leiht es neu aus; lag es schon vor dem Besuch bei der
Person selbst, nimmt er es zurück wie am Netz (der Rechner schickt mit, wie lange sie schon an
der Theke stand); eine Rückgabe hinter einer neueren Buchung weist er ab. Gebucht wird mit dem Scan-Zeitpunkt (höchstens der Serverzeit), und jeder
Schlüssel genau einmal.

**Meldungen.** Jede Abweichung vom Scan hält der Server fest. An der Theke steht
dafür am Band ein Knopf mit der Zahl der offenen Meldungen; dahinter die Liste mit Zeitpunkt,
Buch, Person, Ergebnis und Grund, quittierbar mit „Erledigt". Die Zahl gehört jeder Theken-Rolle
(`perform_actions`), die Liste verlangt `view_students` — sie nennt Namen und Klassen. Offene
Meldungen erscheinen nach 14 Tagen als Warnung in der Betriebsbereitschaft; quittierte werden
nach der Lesehistorie-Frist gelöscht, spätestens nach 30 Tagen.

**Sicherung.** Bleibt ein Arbeitsplatz dauerhaft ohne Verbindung, lässt sich seine Warteschlange
als Datei sichern und an einem anderen Rechner unter _Datenverwaltung → Offline-Sicherungen
einspielen_ nachbuchen. Der Idempotenz-Schlüssel wandert mit: Dieselbe Datei zweimal eingespielt
bucht nichts doppelt.

Nicht im Umfang: Anmelden ohne Server, Schülerdaten auf dem Rechner, Geräteausgabe ohne Netz,
Umbuchen im Online-Weg und die scannende Person im Protokoll (nachgebucht wird unter dem Konto,
das beim Übertragen angemeldet ist).

Gehalten von den Browser-Tests `frontend/e2e/theke-ohne-netz-*.spec.js` und
`theke-gleicher-schluessel.spec.js`, am Stack mit abgeschaltetem Netz und gestellter Uhr.

### 18.5 Selbstanmeldung fürs Kollegium

Wer sich mit einem Postfach der Schuldomäne (`SELBSTANMELDUNG_DOMAIN`) anmeldet und noch kein
Konto hat, bekommt einen **inaktiven** Eintrag; anmelden kann er sich damit nicht. Die
Anfrage erscheint unter _Benutzer & Rechte → Zugangsanfragen_ und wird dort freigeschaltet
(`internal/auth/selbstanmeldung.go`). Ohne diesen Weg müssten ~160 Lehrkräfte vorab von Hand angelegt
werden — was niemand tut.

### 18.6 Hinweise beim Ausgeben von Schulbüchern

Nach der Buchung eines Schulbuchs an einen Schüler fragt der Dienst drei Dinge
(`AusleihDienst.verarbeiteAusleihe`); keines hält die Ausleihe auf, und ein Fehler dabei landet
im Log:

- **Doppelt** (`doppelt_hinweis`, `repository.HatBuchSchon`): Das Kind hat ein weiteres
  Exemplar desselben Buchs offen, auch in einer anderen Auflage (`COALESCE(werk_id, id)`).
- **Sonst niemand in der Klasse** (`klassen_hinweis`, `repository.FindeBuchFremdInKlasse`):
  Mehr als die Hälfte der Klasse, mindestens fünf Kinder, hat Schulbücher für das Schuljahr, und
  kein anderes davon hat dieses Buch. Klasse und Schwelle sind die der Klassensatz-Übersicht
  (§4.3). Verglichen wird nur mit Schulbüchern desselben Schuljahres, gelesen an der Frist:
  Beim Büchertausch hat die Klasse noch die alten Bücher, während die neuen ausgehen, und ab dem
  ersten Tag des Tauschs gilt jedes neue Schulbuch dem nächsten Schuljahr (§2.3). So bekommt das
  erste Kind mit dem neuen Buch keinen Hinweis, und ein altes Buch im neuen Stapel fällt auf.
  Die Oberstufe fragt es nicht, dort lernen Kurse statt Klassen. Vor der Hälfte und beim
  zweiten Kind mit demselben falschen Buch bleibt es still; ein Wahlfach (Latein, Ethik) bekommt
  beim ersten Kind nach der Hälfte einmal einen Hinweis, der nicht stimmt.
- **Andere Auflage** (`auflagen_hinweis`, §13).

Die Theke zeigt je Hinweis eine Zeile über dem Konto, mit dem Titel (`utils/ausgabeHinweise.js`,
`OmniboxThekeHinweise.svelte`), blitzt orange und hält die Reihe wartender Scans an wie bei der
Fremdrückgabe. Die Zeilen stehen, bis ein anderer Leser geladen oder das Konto geschlossen wird:
Beim Bücherstapel kommt der nächste Scan nach einer Sekunde, eine Zeile bis zum nächsten Scan
läse niemand. Wer das Buch zurücknimmt (erneut scannen), nimmt seine Zeilen mit. Eine Snackbar
ginge nicht: „Only one snackbar may be displayed at a time" (M3, Snackbar, Guidelines), und sie
verschwindet von selbst. Belegt: `internal/repository/buch_in_klasse_pg_test.go`,
`internal/api/theke/auflagen_theke_pg_test.go`, `frontend/src/lib/stores/omniboxAusgabeHinweise.test.js`,
`frontend/e2e/auflagen-theke.spec.js`.

---

## 19. Barrierefreiheit

Die Anwendung ist auf WCAG 2.1 AA gebaut; geprüft wird das mit zwei Browser-Gates.

**Was geprüft wird.**

- **`e2e/barrierefreiheit-axe.spec.js`:** axe-core mit den Regeln WCAG 2.0/2.1 A und AA über
  Anmeldung, Katalog, jede Monitor-Folie und alle 15 internen Hauptansichten, je im
  **Anfangszustand**; Reiter, Dialoge und Unteransichten werden nicht mitgescannt. Dazu
  `<html lang="de">`, genau ein `<main>`, eine Überschrift je Seite, Skip-Link als erstes Fokusziel.
  Gescannt wird mit „Bewegung reduzieren“, weil axe mitten in einer Einblendung gemischte Farben liest.
- **`e2e/barrierefreiheit-dialog.spec.js`:** Ein Dialog nimmt den Fokus, hält ihn über 25 Tab und 5
  Shift+Tab und gibt ihn bei Escape an den Auslöser zurück; Tabellen tragen Beschriftung und `scope`;
  `prefers-reduced-motion` setzt Animationen auf 0 s.

Keine dieser Prüfungen ersetzt einen Durchgang mit einem Screenreader (NVDA, VoiceOver); der hat
nicht stattgefunden.

**Was gebaut ist,** in Bauteilen statt in Seiten: die Fokusfalle (`ui/fokusFalle.js`) in jedem Dialog,
im Sperrbildschirm und in den zwei Theken-Alarmen (`OmniboxSperrHinweis`, `OmniboxVormerkungHinweis`,
als `alertdialog`); `ui/Tabelle.svelte` mit Pflicht
zur `beschriftung` (Ratsche `frontend-hygiene-tabellen`) und `scope`; Skip-Link und das eine `<main>`
mit unsichtbarer `h1` aus dem Menü (`layout/SkipLink.svelte`, `layout/Hauptbereich.svelte`), denn sichtbare Seitenköpfe bleiben abgeschafft. Rahmen von Feld,
Select und Suchfeld stehen in der Rolle `outline` (4,5:1 auf Weiß) statt `outline-variant` (1,7:1),
weil WCAG 1.4.11 für Bedienelemente 3:1 verlangt. Meldungen pausieren unter Maus und Fokus (2.2.1),
Fehler sind `role="alert"`, die Omnibox führt per `aria-activedescendant` durch die Treffer. Zeile und
Kachel sind keine Knöpfe: Der Name bzw. Titel öffnet die Akte, im Medienkatalog das Cover.

**Bekannte Lücken (offen, keine davon im Gate versteckt):**

- Der Ausweis-Designer (Zeichenfläche `designer/CanvasFlaeche`) ist Maus- und Touch-Arbeit ohne
  Tastaturweg.
- Die PDFs (maroto/gofpdf) sind ungetaggt — HTML-Druckweg oder begründete Ausnahme, offen
  ([ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5).
- Nicht gescannte Zustände (Reiter, Dialoge, Unteransichten) können weitere Verstöße tragen;
  gemessen sind sie nicht.

Regel für neue Oberfläche: Ein interaktiver Container (Zeile, Kachel) enthält keine weiteren
Bedienelemente — der Name oder Titel ist der Knopf. Farben kommen nur aus den Rollen in
`frontend/src/styles/rollen.css`; Palettenklassen wie `slate-*` trägt die Oberfläche nicht mehr
(`frontend/src/lib/frontend-hygiene-farben.test.js`).

---

## 20. Bewusst nicht gebaut

Was vorgeschlagen, zum Teil gebaut und dann verworfen wurde. Der Code zeigt es nicht; ohne
diese Liste käme derselbe Vorschlag wieder. Je Zeile: was, wann entschieden, warum nicht.
Technische Alternativen stehen in [ARCHITEKTUR.md](ARCHITEKTUR.md) 4.3.

| Was | Entschieden | Warum nicht, und was gilt |
| --- | --- | --- |
| **Zahlen oder Kacheln auf dem Ruhebildschirm der Theke** (überfällig, Abholfach, Klassensätze, Meldungen) | 02.09.2026, gebaut mit `50464daa`, entfernt mit `1da97db4` | „Überfällig" ist im Schuljahr immer groß, hinter dem Abholfach stand keine Liste, die übrigen Zahlen standen schon als Abzeichen im Menü. Die Theke zeigt im Ruhezustand nur das Scanfeld; Zahlen stehen in Statistik, Mahnwesen und Bestellungen |
| **Sperre für den zweiten Scan desselben Buchs an der Theke** | 06.10.2026 (`639b6414`) | Sie hätte nur den doppelten Scan gefangen und den gewollten zweiten Scan verweigert; Littera hat sie auch nicht. Stattdessen klingt eine Ausleihe anders als eine Rückgabe (`1701418f`); den Stand hält `frontend/e2e/theke-schnellrueckgabe.spec.js` |
| **Eigene oder einstellbare Farbe für die Ausleihe, Schalter für die Töne der Theke** | 06.10.2026 | Ausleihe und Rückgabe bleiben grün, der Unterschied liegt im Ton |
| **Eigene Uhr für die Schnellrückgabe** | 06.10.2026 | Sie endet mit dem Leeren der Theke (Einstellungen → Datenschutz & Sitzung); eine zweite Frist gibt es nicht |
| **Farben je Fach** | 06.10.2026 | Die Oberfläche trägt nur die Farbrollen; ein Titel ohne Cover zeigt einen neutralen Platzhalter (`ui/BuchCover`) |
| **Auswahlfelder für „von" und „bis" am Jahrgang des Titels** | 08.10.2026, am Bild der Maske | Tippen ist schneller: „7 bis 10" sind im Zahlenfeld vier Tasten, im Auswahlfeld acht, und dessen Liste zeigt sechs von zehn Einträgen. Die zwei Zahlenfelder bleiben, „bis" geht mit „von" mit |
| **Enddatum am Leser, Liste der Gegangenen aus dem Kollegium** | 30.09.2026 (`2074db30`) | Wer geht und noch Bücher hat, klärt bei Bedarf das Sekretariat |
| **Meldung, wenn zwei Plätze dasselbe Feld eines Titels ändern** | 06.10.2026 (`ddeba157`) | Die Maske schickt nur geänderte Felder; damit verliert niemand, was er nicht angefasst hat. Ändern beide dasselbe Feld, gilt der spätere Eintrag (`frontend/e2e/buchmaske-zwei-plaetze.spec.js`) |
| **Seite zum Umbenennen, Zusammenlegen und Löschen von Klassen** | 30.09.2026 (`160b4cf2`) | Die Klasse wird an den Eingabestellen ausgewählt statt getippt; falsche Klassen entstehen so nicht, und die Klassen selbst kommen aus der LUSD |
| **Sammelseite „Schuljahreswechsel"** mit LMF-Plan, Abgängern und LUSD/Versetzung als Reitern | 05.09.2026 | Sie stand einen Abend und verwirrte. Der LMF-Plan steht unter System → Schuljahreswechsel, die Abgänger unter Verwaltung, LUSD und Versetzung in den Einstellungen (`frontend/src/lib/menue.js`) |
| **Unterteilung der Ausleihliste der Leserakte nach Bücherei und Lernmittel** | 01.10.2026 | Abgelehnt; das Kennzeichen „Lernmittel" an der Zeile bleibt |
| **Mail an den Besteller, wenn der Händler bestätigt** | 01.11.2026 | Der Stand steht im Programm (Bestellhistorie, Wareneingang, geöffnete Bestellung), und die Bestellung merkt sich nicht, wer bestellt hat |
| **Zugang ohne Mailserver für Tester** | 08.09.2026, gebaut mit `cb6dd13b`, zurückgenommen mit `94577bbe` | Er wurde nicht mehr gebraucht. Angemeldet wird nur gegen den Mailserver der Schule ([ARCHITEKTUR.md](ARCHITEKTUR.md) A2); die fertige Fassung liegt im ersten Commit |
| **A4-Bogen mit acht Ausweisen zum Ausschneiden** | 24.08.2026 (`6bc2829c`) | Als Notbehelf gedacht, im Alltag ohne Nutzen. Gedruckt wird auf dem Kartendrucker oder als Klebeetikett mit Name, Klasse und Strichcode (§9) |
| **Feld „Beschreibung / Klappentext" am Titel, Klappentext aus der DNB** | 22.09.2026 (keine Übernahme); Feld entfernt am 03.10.2026 (`bed921ce`, Migration 156) | Ein Klappentext gehört dem Verlag; frei nutzbar sind die Titeldaten und die Normdaten der DNB. Das Feld zeigte nur die Maske „Titel bearbeiten", und es war leer |
| **Google Books als Quelle für Titeldaten streichen** | 24.09.2026 | Die Reihenfolge bleibt DNB, Google Books, OpenLibrary (`internal/katalogdienste/katalogdienste.go`). Für Schlagworte wird nur die DNB gefragt (§13) |
| **Nachbau des Littera-Etiketts beim Nachdruck** | 30.09.2026 (`bb5e3104`) | Der Scanwert ist derselbe, und Bildschirm und Mahnliste zeigen die 13 Ziffern. Nachgedruckt wird das Etikett dieses Programms (§9) |
| **Reihe neuer Nummern ohne Exemplar im Druck-Center** („Neue Barcodes" ab einer Start-Ziffer, auch aus `barcode_seq` gezogen) | 22.10.2026 | Jede Nummer entsteht mit ihrem Exemplar: bei der Bestellung (§7), in der Maske des Titels und bei der Übernahme aus Littera. Eine Nummer, die nur auf Papier steht, gehört keinem Exemplar, und eine Bestellung könnte sie ein zweites Mal vergeben |
| **Seitenzahlen, die über alle Ausdrucke des Zugangsbuchs weiterlaufen** („fortlaufend paginiert" im Leitfaden Lernmittelfreiheit, 11.5) | 29.10.2026 | Jeder Ausdruck steht für sich |

An anderer Stelle beschrieben, weil die Entscheidung dort zum Ablauf gehört: keine
seitenübergreifende Suchleiste ([HANDBUCH.md](HANDBUCH.md), „Suchen"), kein Buchwunsch und kein
Ticketsystem für Meldungen (§4.4), keine Mahnung an Lehrkräfte ([HANDBUCH.md](HANDBUCH.md),
„Mahnwesen"), keine Mahnung per E-Mail an Eltern ([LUSD.md](LUSD.md) §1), kein Sortiertitel aus
Littera ([littera_schema_befund.md](littera_schema_befund.md)), keine Liste der erwarteten Bücher
beim Beginn einer Inventur (§6), keine Ausgabe von Lesern und Ausleihen für ein anderes Programm
([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md) §8).
