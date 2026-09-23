# Fachkonzept & Feature-Spezifikation

Beschreibt die funktionale Business-Logik der Bibliothekssoftware auf Basis der Go-Implementierung. Zielgruppe: Administratoren, Betreiber und Entwickler.

---

## 1. Zentrale Scanner-Omnibox (Kiosk)

Die Software nutzt ein eingabefokussiertes Kiosk-Design für die Ausleihe und Rückgabe. Ein
einziges Eingabefeld (Omnibox) verarbeitet alle Scans.

**Der Normalfall ist ohne Präfix.** Die Ausweise und Buchetiketten aus dem Littera-
Altbestand tragen nackte Nummern und dürfen nicht neu gedruckt werden; ein Scan wird
deshalb der Reihe nach aufgelöst:

```
Buch-Exemplar → Schülerausweis → Lehrerausweis → Volltextsuche im Katalog
```

Das geht auf, weil die Formen verschieden sind: Buchetiketten liefern eine 13-stellige
EAN-13, Schülerausweise die Nummer ihres Kartenherstellers. Die Vorgängersoftware Littera
kennt überhaupt keine Präfixe — sie hat getrennte Suchfelder, wir haben eines für alles.

**Präfixe sind eine Abkürzung**, kein Muss. Sie überspringen die Auflösung und sprechen
den Bereich direkt an:

- **`A-[Barcode]` (Leser):** Lädt das Konto eines Lesers (inkl. offener Ausleihen, Mahnungen und Sperren).
- **`S-[Barcode]`, `L-[Barcode]`:** dasselbe für Nummern von früher. Vergeben werden sie seit dem 16.09.2026 nicht mehr.
- **`B-[Barcode]` (Buch-Exemplar):** Führt eine Aktion mit einem Buch aus.
- **`G-[Barcode]` (Gerät):** Führt eine Aktion mit Hardware (z. B. Laptops, iPads) aus.

**Eine Vorsilbe für alle Ausweise: `A-` (seit 16.09.2026).** Vorher gab es zwei, und beide
behaupteten etwas über die Person: `S-` aus Handanlage und LUSD-Import, `L-` aus dem
Littera-Personenlauf. Seit die Leserdatei alle führt, ist das die falsche Aussage — wer
jemand ist, steht in den Stammdaten und nicht auf seinem Ausweis. Gelesen werden `S-` und
`L-` weiter: Es gibt Karten von früher, und Nummern werden nicht recycelt.

Die Vorsilbe selbst BLEIBT, und der Grund ist nicht offensichtlich: **Ohne Netz ist sie
die einzige Information, an der die Theke einen Buchscan von einem Ausweisscan
unterscheiden kann.** Offline gibt es niemanden zu fragen. Littera kommt ohne aus, weil
dort Nummer und Scanwert zwei verschiedene Felder sind und der Scanwert vom
Kartenhersteller stammt; dieses zweite Feld haben wir nicht.

**Ablauflogik:**

1. Wird ein Buch gescannt, _ohne_ dass ein Schüler/Lehrer aufgerufen ist, wird das Buch **zurückgegeben**.
2. Wird ein Schüler/Lehrer gescannt und danach Bücher gescannt, werden diese an die Person **ausgeliehen**.

---

## 2. Ausleih-Regelwerk und Fristen

Das System unterscheidet zwischen verschiedenen Medien und Leihertypen:

### 2.1. Fristenberechnung

- **Lernmittelfreiheit (LMF) - "Schulbücher":** Haben ein fixes Rückgabedatum: den **31. Juli** des laufenden (oder bei Sommer-Ausleihe des kommenden) Schuljahres (`lmf_stichtag`) — **es sei denn, der LMF-Plan (§2.3) nennt für die Klasse einen Rückgabe-Termin: dann ist der nächste Termin NACH dem Ausleihtag die Frist** (`RueckgabeTerminLage`, nur einjährige Ausleihe; seit 05.09.2026). Hatte die Klasse im laufenden Schuljahr ihren Rückgabe-Termin schon, gilt der Stichtag des FOLGENDEN Schuljahres — wer danach noch ein Schulbuch bekommt, gibt es erst im nächsten Schuljahr zurück (seit 14.09.2026).
- **Freihand-Bestand (Sonderbestände):** CDs, DVDs, Hörbücher etc. haben eine rollierende Frist (z. B. +14 oder +28 Tage ab Ausleihe), keine starre Jahresfrist.
- **Ferien:** Eine automatische Verlängerung „bis zum ersten Schultag nach den Ferien" gibt es NICHT (stand bis 06.09.2026 fälschlich hier; `loan_rules.go` kennt keine Ferien). Das Werkzeug für „Bücher über die Sommerferien mitnehmen" ist der **Ferien-Leseclub** (Kategorie Ausleihe & Fristen, §17): aktiv + Zieldatum → alle Ausleihen bekommen dieses feste Rückgabedatum. Eine Ferien-Pause des Mahnwesens gibt es ebenfalls nicht mehr: Die Tabelle `ferien_schliesszeiten` (Migration 017, Banner + Sperre) hatte nie einen Schreiber und ist mit Migration 102 ausgebaut (entschieden am 06.09.2026: „ausbauen" — das Mahnwesen wird nur von Hand bedient).
- **Lehrer (Handapparat):** Erhalten pauschal eine Frist von einem Jahr — `AddDate(1, 0, 0)`, also ein **Kalenderjahr**, nicht 365 Tage (im Schaltjahr sind es 366). Wie jede andere Frist läuft sie durch `tagesEndeInSchulzeitzone`; eine zweite, rohe Berechnung gibt es bewusst nicht.
- **Verlängerungen:** Ausleihen können verlängert werden, es sei denn, der Schüler ist gesperrt oder hat das Ausleihlimit überschritten.

### 2.2. Blockaden und Limits

- **Ausleihlimit:** Es gibt ein konfigurierbares Maximum an gleichzeitigen Ausleihen pro Schüler (LMF-Bücher ausgenommen).
- **Sperre bei Überfälligkeit:** Hat ein Schüler mehr als `MaxOverdueItems` überfällige Medien, wird das Konto automatisch für neue Ausleihen der Schülerbücherei gesperrt. Dasselbe gilt bei einer unbezahlten Forderung.
- **Lernmittel kennen keine automatische Sperre** (Entscheidung der Schule vom 22.09.2026, Grundlage: Lernmittelfreiheit in Hessen): Ein Schulbuch wird auch bei offener Forderung oder überfälligen Medien ausgegeben, ohne dass jemand etwas übergehen muss. Nur die von Hand gesetzten Schalter am Konto (gesperrt, manuell gesperrt) gelten für beide Töpfe.
- **Manuelle Sperre:** Administratoren können Schüler manuell sperren (z. B. bei massivem Fehlverhalten). Ein verpflichtender Begründungstext (`block_reason`) wird stets verlangt und den Helfern angezeigt.

---

### 2.3. LMF-Plan: Büchertausch vor und Bücherausgabe nach den Sommerferien (seit 05.09.2026)

Zwei Pläne, zwei Zeitpunkte (entschieden am 06.09.2026 — „Rückgabe" und „Ausgabe" allein waren
unklar): **Büchertausch vor den Sommerferien** (`art = rueckgabe`): alle Klassen geben ab
und bekommen direkt die neuen Bücher; Abschlussklassen (`AbschlussklasseSQL`) und Klassen,
deren nächster Jahrgang ein Eingangsjahrgang ist (die 6er → 7H/7R/7G), geben **nur
zurück** (`nurRueckgabeSQL`, Markierung `nur_rueckgabe` in Planer, Portal und PDF).
**Bücherausgabe nach den Sommerferien** (`art = ausgabe`): nur die neu gebildeten Klassen
der **Eingangsjahrgänge** (Einstellung `lmf_eingangsjahrgaenge`, Vorgabe „5, 7",
`EingangsjahrgaengeAus`); der Vorschlag enthält nur sie. **Entwurf und Veröffentlichung**
(Migration 100, `lmf_plaene.veroeffentlicht_am`): Speichern legt einen Entwurf an — zentral,
aber unsichtbar für `GET /api/lmf-termine`, das Portal-PDF und `RueckgabeTerminLage`
(ein Entwurf setzt auch beim Ausleihen keine Frist);
`POST /api/lmf-plan/{art}/veroeffentlichen` stempelt ihn und koppelt die Fristen; danach
gilt jede Speicherung sofort. Das Entwurfs-PDF für die Schulleitung liefert
`GET /api/lmf-termine/entwurf/pdf` (edit_books). Klassen werden weder angelegt noch
gelöscht: Der Name im Plan registriert sich im Vokabular (Trigger, Migration 079), und
eine Klasse ohne aktive Schüler verschwindet von selbst aus jeder Liste — der Planer
markiert sie „ohne Schüler" (`klassen` = Klassen mit Schülern in der GET-Antwort). **Wer
noch keine Zeile hat, steht nur im Planer** über der Tabelle („Noch nicht im Plan",
`LmfPlanVorrat.svelte`); das Portal zeigt den Plan, nicht die Lücken darin. Bis zum
12.09.2026 lieferte `GET /api/lmf-termine` dafür ein zweites Feld — gelesen hat es nie
jemand, und zehn Monate im Jahr nannte es schlicht jede Klasse (gestrichen; die Abfrage
steht in `git log`).

Die Schule führte den Plan als Excel-Tabelle (Wochentag, Datum, Stunde, Klasse(n),
Besonderheiten) und mailte ihn dem Kollegium; Korrekturen kamen als Folge-Mail. Der echte
Plan 2026 zeigte die Form (05.09. abends): Abschlussklassen zuerst, dann JEDER
Schultag Stunde 1–6, eine Klasse je Stunde, die Reihenfolge läuft über die Tage weiter;
manche teilen sich eine Stunde („10R1/10R2"), am Ende Zeilen ohne Klasse („Nachzügler",
„Aufräumen"); zwei Datumsfehler von Hand. **Der Plan ist deshalb eine REIHENFOLGE, die
der Server auf Schultage × Stunden gießt** — nicht eine Liste einzeln angelegter Zeilen
(so gebaut in 1890a4df, am selben Abend ersetzt).

Modell (Migration 096 + 097 + 101): `lmf_plaene` je Art und Schuljahr mit dem Rahmen —
beim Büchertausch das ENDE (`letzter_tag`, `letzte_stunde`: Donnerstag vor den
Sommerferien, 4. Stunde; 06.09.2026: „es endet immer am gleichen Tag"), die
Reihenfolge fließt rückwärts davor und `erster_tag`/`startstunde` sind gerechnet; bei
der Bücherausgabe der Beginn (`erster_tag`, `startstunde`) — plus Stunden je Tag;
die Vorbelegung kommt aus der Ferientabelle Hessen (`pkg/lmfplan/ferien.go`, KMK bis
2030, Horizont-Test als Erinnerung), `lmf_termine` als Zeilen mit `plan_id`, `position`
und den GERECHNETEN Feldern Datum/Stunde (Portal, PDF und Frist-Kopplung lesen sie wie
zuvor), `lmf_termin_klassen` (0..n Klassen aus dem Vokabular; „Bücher setzen" = Zeile
ohne Klasse mit Vermerk), `lmf_plan_ausgelassen` (Klassen, die der Plan bewusst auslässt —
die Oberstufe organisiert sich an dieser Schule selbst; der nächste Plan übernimmt die
Auslassung). Die Verteilung rechnet
`pkg/lmfplan.VerteileMit` bzw. `VerteileRueckwaerts` (Mo–Fr, Feiertage Hessen,
freie Tage des Plans ausgespart, feste Plätze umflossen) —
die EINE Stelle; die Vorschau im Planer ist derselbe Aufruf mit
`"vorschau": true`, kein JavaScript-Zwilling.

Routen (`api/lmf_plan.go`, alle `edit_books`): `GET /api/lmf-plan/{art}` liefert den
neuesten Plan der Art, ob er vorbei ist, und dann den **Vorschlag** für den nächsten —
die Reihenfolge des Vorjahres plus neue Klassen (Klassennamen bleiben Jahr für Jahr gleich,
die Versetzung verschiebt Schüler, nicht Namen), ohne Vorjahr die Regel
`KlassenMitSchuelern` (Abschluss zuerst via `AbschlussklasseSQL`, Jahrgang absteigend,
Jahrgang ≥ 11 oder ohne Ziffer = Oberstufe → ausgelassen). `PUT` rechnet und speichert
(gleiches Schuljahr = ersetzen, anderes = neuer Plan), `DELETE` verwirft den neuesten.
Lesen für Portal und PDF unverändert: `GET /api/lmf-termine[/pdf]` mit Sitzung. Einzel-
Termin-Routen gibt es nicht mehr — eine zweite Tür je Zeile gäbe zwei Wahrheiten.
Seite: Menü _System → Schuljahreswechsel_ (`LmfPlan.svelte`; 05.09.: der Plan
wird ein- bis zweimal im Jahr gebraucht — Abgänger bleiben unter Verwaltung, LUSD und
Versetzung in den Einstellungen). Tests: `pkg/lmfplan/layout_test.go`, `repository/lmf_termine_pg_test.go`,
`api/lmf_termine_frist_pg_test.go`, `frontend/e2e/lmf-plan.spec.js`.

**Kopplung an die Fristen** (`api/lmf_termine_frist.go`, 05.09.2026: „das wäre doch
logisch"): Der Rückgabe-Termin einer Klasse ist die Frist ihrer Lernmittel. Beim Ausleihen
liest `resolveCheckoutDueDate` die Lage der Klasse (`RueckgabeTerminLage`): Steht ein
Rückgabe-Termin nach heute bevor, ist er die Frist (vor dem Stichtag; ein Mehrjahresband
bleibt beim Stichtag plus die verbleibenden Jahre, siehe unten). Lag der Termin der Klasse im laufenden Schuljahr schon heute oder
davor, ist die Frist der Stichtag des folgenden Schuljahres (Entscheidung 13.09.2026): Wer
dann noch ein Schulbuch bekommt, gibt es erst im nächsten Schuljahr zurück — auch wenn die
Klasse noch einen Nachzügler-Termin vor sich hat. Bis zum
14.09.2026 war am Termintag der Termin selbst die Frist und danach der Stichtag des laufenden
Schuljahres — ein Tag in den Ferien. Beim Speichern eines Plans folgt
der Bestand (`koppleLmfPlanFristen`): Klassen, die aus dem Plan fallen, kehren zum Stichtag
zurück — genau die Fristen, die auf ihrem alten Termin-Tag lagen —, jede Klasse des neuen
Plans bekommt ihren Termin (`SetzeLernmittelFristFuerKlassen`: aktive, nicht gesperrte
Schüler, nur Lernmittel, nur Fristen im Schuljahr des Termins, Mahnstufe zurück).
Ausgabe-Pläne setzen keine Frist. PUT/DELETE melden `fristen_angepasst`.

## 3. Mahnwesen

Das Mahnsystem durchläuft einen 3-stufigen, rechtlich bindenden Eskalationsprozess.

- **Stufe 1 (Erinnerung):** Kann als kostenlose E-Mail ("Friendly Reminder") oder als PDF-Ausdruck versendet werden.
- **Stufe 2 & 3 (Kostenpflichtig):** Erzeugen Mahngebühren. Diese **Mahnstufe erhöht sich ausschließlich beim physischen PDF-Druck**, da dies den rechtlichen Verwaltungsakt darstellt. Der reine E-Mail-Versand einer Erinnerung führt _nicht_ zur Erhöhung der Mahnstufe oder zu neuen Gebühren.
- **Sperr-Folge:** Bei Erreichen von Stufe 3 (oder Nichtzahlung) kann der Schüler gesperrt werden.

---

## 4. Vormerkungen und Klassensatz-Reservierungen

Das System verwaltet den Mangel an verfügbaren Büchern durch zwei Konzepte:

### 4.1. Einzel-Vormerkungen

- Ein Schüler kann ein Buch vormerken, wenn kein Exemplar mehr frei ist.
- **Rückgabe-Match:** Wird ein Exemplar dieses Titels zurückgegeben, prüft das System, ob eine Vormerkung vorliegt.
- **Abholbereitschaft:** Das Buch wird nicht freigegeben, sondern direkt dem wartenden Schüler zugeteilt (Status `abholbereit`). Es landet physisch im Bereitstellungsregal.
- _Schutz:_ Es ist technisch unmöglich, dass ein Schüler ein Buch vormerkt, das er aktuell selbst ausleiht (Vermeidung von Monopolisierung).

### 4.2. Klassensatz-Reservierungen

**Reservieren heißt Anstellen, nicht Sperren** (Betreiber-Entscheidung 16.08.2026 — bis
dahin beschrieb dieser Abschnitt ein Blockier-Modell, das nie gebaut war):

- Eine Lehrkraft reserviert im Kollegiums-Portal Titel + Klasse + Anzahl. Die einzige
  harte Grenze: nicht mehr, als die Bibliothek physisch besitzt.
- Die Reservierung **blockiert nichts** — weder Exemplare noch Rückgaben. Sie ist ein
  Arbeitsauftrag an die Bibliothek und reiht sich in eine **sichtbare Warteschlange**
  ein (älteste zuerst).
- Das Portal zeigt bestehende Reservierungen **vor** dem Klick am Treffer
  („28 reserviert für 8a“) und verrechnet sie mit dem Regal (seit 26.08.2026:
  „28 vorgemerkt · 2 rechnerisch frei“ — das OPAC-Abzeichen „N von M verfügbar“ sinkt
  durch eine Reservierung nicht, weil sie nichts bucht). Übersteigt die Wunschanzahl
  die rechnerisch freie Zahl, warnt das Formular vor dem Absenden („Reicht aktuell
  nicht — du stellst dich hinter 8a an“); wer trotzdem reserviert, erfährt in der
  Bestätigung, hinter wem sein Satz an der Reihe ist.
- Die Bibliothek arbeitet die Schlange unter Bestellungen → Klassensatz-Reservierungen
  ab: je Zeile Anfragende(r) mit Namen und der aktuelle Regal-Bestand („N verfügbar“).
  „Abschließen“ beendet den Vorgang nach der physischen Übergabe — die Ausleihe selbst
  läuft über die normalen Wege (Kiosk je Schüler oder Lehrer-Handapparat).
- **Die Abschluss-Notiz hat zwei Rückwege neben der Bereit-Mail** (31.08.2026 — vorher
  existierte sie nur in der Mail; scheiterte die oder fehlte die Adresse, war die Zusage
  für immer unsichtbar): Die Theke zeigt „Zuletzt bereitgestellt“ (die jüngsten zehn,
  mit Notiz und Abschlussdatum, Migration 089), und das Portal zeigt der Lehrkraft unter
  „Bereitgestellt“ die EIGENEN abgeschlossenen Reservierungen samt „Bibliothek: …“
  (`GET /api/reservierungen/klassensatz/eigene`, dasselbe Muster wie beim Anliegen).
  „Deine Reservierungen“ im Portal meint seither wirklich die eigenen — die Warteschlange
  aller bleibt als Chip an den Suchtreffern sichtbar.

### 4.3. Klassensatz-Übersicht: zwei Quellen, zwei Bedeutungen von „Klasse“

Die Übersicht (`GetClassGroups`, `inventur/datenbank_klassen.go`) beantwortet die
Alltagsfrage „welche Bücher hat diese Klasse?“ — gebraucht vor allem für Nachzügler, die
mitten im Jahr dazukommen. Sie speist sich aus zwei Quellen, die verschieden altern:

- **Handliste (`class_books`, Quelle `hand`)** hängt am **Klassennamen**. „Die 7R2 liest
  Mathe 7“ gilt für jede künftige 7R2, deshalb lässt die Versetzung sie bewusst stehen
  (`api/student_promotion.go`, Befund F3). Sie veraltet nicht von selbst und muss von Hand
  gepflegt werden — dafür sieht sie auch Gruppen unter fünf Kindern.
- **Aus Ausleihen (Quelle `ausleihe`)** hängt an den **Kindern**: bei jedem Aufruf neu
  gerechnet über `klassen_normkey(schueler.klasse)`, nie gespeichert. Halten mehr als die
  Hälfte der aktiven Kinder einer Klasse und mindestens `KlassensatzMindestLeser` (5)
  denselben Titel, ist er ihr Klassensatz.

Aus dem Unterschied folgt das Verhalten am Büchertausch (07.09.2026 aufgeschrieben, weil
die Frage im Betrieb aufkam): Die 7R2 gibt im Juni ihre Bücher ab und bekommt die der 8,
steht in der LUSD aber bis nach den Ferien weiter als 07R2. Die abgeleitete Liste zeigt die
neuen Bücher deshalb sofort unter _07R2_ — was für den Nachzügler genau richtig ist, denn
er bekommt dasselbe wie seine Klasse. Schiebt die LUSD (oder die Versetzung) die Kohorte
auf 08R2, steht dieselbe Liste ohne Zutun unter _08R2_, und die nachrückende Klasse
erscheint mit ihren eigenen Büchern unter _07R2_. **Die Ableitung wandert mit den Menschen,
die Handliste bleibt am Namen** — im Fenster zwischen Tausch und Versetzung stehen beide
untereinander. Belegt am echten Postgres:
`inventur/klassensatz_ableitung_pg_test.go` (Schwelle, Normschlüssel, keine Dubletten).

Ein Titel erscheint je Klasse **einmal**, unabhängig von der Zahl der Exemplare; die
Leserzahl steht als Abzeichen an der Kachel. Ist ein Titel von Hand zugeordnet UND
abgeleitet, gewinnt `hand`.

### 4.4. Wünsche & Meldungen der Lehrkräfte

Seit 18.08.2026 (Betreiber-Entscheidung: bewusst schlank, kein Ticketsystem):

- **Ein Mechanismus für zwei Fälle:** „Ich möchte in der 8G3 den Markl 2" (Wunsch)
  und „die 8G3 hat die falschen Bücher bekommen" (Meldung). Die Lehrkraft trägt es
  im Kollegiums-Portal ein — Art, Freitext, Klasse/Kurs, optionale Anmerkung.
- **Wünschen geht immer** — keine Wunschphase, kein Stichtag.
- Die Bibliothek arbeitet die Liste unter Bestellungen → „Wünsche & Meldungen" in
  Ruhe ab (älteste zuerst). **Abhaken** schließt das Anliegen und schickt der
  Lehrkraft automatisch eine Mail — mit der optionalen Notiz („bestellt, kommt
  Anfang September"). Ein Doppelklick von zwei Arbeitsplätzen löst keine zweite
  Mail aus (gleiches Muster wie die Klassensatz-Bereit-Mail).
- Die Lehrkraft sieht ihre Anliegen samt Status und Erledigungs-Notiz im Portal.
- Bewusst NICHT gebaut: Prioritäten, Kommentar-Threads, Genehmigungsketten,
  Deckungsprüfung, Packlisten — erst nachrüsten, wenn der Alltag sie vermisst.

---

## 5. Geräteausleihe (Hardware)

Seit 16.08.2026 vollständig in Betrieb (vorher Backend-Torso ohne Oberfläche —
dieser Abschnitt beschrieb die Absicht als Realität):

- **Verwaltung** im Medienkatalog, Bereich „Geräte“ (kein eigener Menüpunkt —
  ausgebucht wird ohnehin am Kiosk): anlegen (Barcode zwingend mit `G-`-Präfix,
  sonst findet der Kiosk-Scan das Gerät nicht), Liste mit aktuellem Ausleiher,
  Zubehör- und Stammdaten-Pflege, Defekt-Schalter (`ist_ausleihbar`).
- **Checklisten-Zwang am Kiosk:** Trägt ein Gerät Zubehör (kommaseparierte Liste),
  unterbricht der Scan mit einem Bestätigungs-Dialog, der jedes Teil nennt —
  bei Ausleihe UND Rückgabe. Erst „Alles vollständig“ schickt den Scan mit
  Bestätigung erneut; Abbrechen bucht nichts.
- **Fristen:** 14 Tage, auf das Tagesende in der Schul-Zeitzone normalisiert wie
  Buch-Fristen; Lehrkräfte leihen Geräte als Handapparat (Dauerleihe).
- **Zustands-Sperre:** Defekte (`ist_ausleihbar = false`) und ausgesonderte Geräte
  verweigern die Ausleihe am Kiosk.
- Fehlendes Zubehör bei der Rückgabe: Das Personal bricht im Dialog ab und meldet
  den Schaden über den bestehenden Schadensfall-Weg am Profil.

---

## 6. Inventur-System

Die Inventur findet im laufenden Betrieb statt, ohne dass die Bibliothek zwingend schließen muss.

- **Session-basiert:** Jede Inventur erhält einen Scope (z. B. "Raum 2, Regal A") und läuft in einer eigenen Session (`inventur_sessions`). Mehrere Mitarbeiter können mit Handscannern parallel inventarisieren, ohne sich gegenseitig zu überschreiben.
- **Fehlmengen-Ausbuchung:** Wird die Session beendet (`Finish`), vergleicht das System alle gescannten Exemplare mit dem theoretischen Bestand in diesem Scope.
- **Schutz aktiver Ausleihen:** Bücher, die laut Datenbank aktuell _verliehen_ sind, werden vom System bei der Fehlmengenberechnung ignoriert – sie können nicht versehentlich als Verlust ausgebucht werden, nur weil sie nicht im Regal standen.
- Fehlende, nicht verliehene Exemplare erhalten automatisch den Status `VERLUST`.

---

## 7. Bestellwesen und Wareneingang

Die Software verwaltet Bedarfe und Lieferungen:

- **Bestellbedarf:** Ein Titel gilt als Bedarf, wenn sein Gesamtbestand unter der Schwelle aus den Systemeinstellungen liegt (`bestellbedarf_schwelle`, Vorgabe 3). Die Spalte `meldebestand` am Titel wird nur noch **informativ** mitgeliefert und löst nichts mehr aus — ihr pauschaler Default 5 meldete früher fast jeden Titel. Die Warnung selbst lässt sich abschalten (`bestellbedarf_warnung_aktiv`).
- **Fokus auf Lernmittelfreiheit (LMF):** Die Bedarfsvorschläge sind standardmäßig auf LMF-Medien (Schulbücher) gefiltert. Freihand-Exemplare (Lese-Einzelstücke) werden in der Regel nicht nachbestellt.
- **Eine Bestellung = ein Topf (Migration 109, 10.09.2026):** Jede Bestellung trägt, aus welchen Mitteln sie bezahlt wird — Lernmittelfreiheit (Land) oder Schülerbücherei (Schulträger). Der Händler gewährt darauf verschiedene Nachlässe, und die Rechnungen gehen getrennte Wege; beide gehen an die Schule (Auskunft der EDV-Servicestelle für Schulbibliotheken, 10.09.2026). Der Warenkorb gruppiert seine Positionen nach dem Vorschlag aus `ist_lernmittel`, eine Position lässt sich in den anderen Topf schieben (falsch gekennzeichneter Titel), und „Bestellung auslösen" erzeugt je Gruppe eine Bestellung an denselben Händler — zwei Mails, zwei Anschreiben. Der Vermerk steht in der Betreffzeile und im Text des Anschreibens (`api/mittel_vermerk.go`) und in der Händler-Mail (Platzhalter `{{.Mittel}}`; fehlt er in der Vorlage, hängt der Versand den Vermerk an). Ein Lieferant kann eine zweite Kundennummer für die Schülerbücherei führen. Alt-Bestellungen sind zugeordnet, wo es aus den Positionen eindeutig war; gemischte stehen als „ohne Zuordnung". Ein über die DNB neu angelegter Titel wird im Staging-Fenster gefragt, ob er Lernmittel ist — vorher blieb jedes neue Schulbuch ein Bücherei-Titel. Der Topf einer bestehenden Bestellung lässt sich im Bestelldetail mit Pflicht-Grund korrigieren (`PUT /api/bestellungen/{id}/mittel`, protokolliert im Admin-Audit-Log); das Anschreiben beim Händler ändert sich dadurch nicht — die Korrektur gilt der eigenen Zuordnung, aus der die Berichte rechnen. Belegt: `api/bestellung_mittel_pg_test.go`, `api/order_pdf_mittel_test.go` (Inhaltsstrom des Anschreibens), `api/bestellung_mittel_backfill_pg_test.go`, `api/bestellung_mittel_korrektur_pg_test.go`.
- **Exemplare entstehen beim BESTELLEN, nicht beim Eintreffen:** Mit dem Absenden legt das System die Exemplare samt Barcode-Nummern an und markiert sie als „Im Zulauf". Nur so kann der Barcodebogen mit der Bestellung mitgehen. Der Wareneingang bucht diese Exemplare später frei — er erzeugt sie nicht.
- **Lieferanten-Eigenschaften** (Lieferantenverwaltung, je Händler schaltbar):
  - _Händler beklebt die Bücher:_ Der Barcodebogen geht mit; die Exemplare gelten sofort als etikettiert und erscheinen nicht auf der Nachdruck-Liste.
  - _Voreingestellt beim Bestellen:_ Vorauswahl im Bestellformular, höchstens ein Händler (DB-seitig erzwungen).
  - _Lieferant bestätigt Bestellung selbst:_ siehe nächster Punkt.
- **Bestätigungs-Link an den Lieferanten:** Händler, die selbst etikettieren (z. B. Naacher), bekommen mit der Bestellmail einen von Bibliosys erzeugten Link. Dahinter liegt eine Seite ohne Login, auf der der Lieferant die Etiketten dieser Bestellung in klein oder groß druckt (identisch zum Mailanhang) und die Bestellung **einmalig** bestätigt.
  - _Klein:_ Bogen für vorgestanztes Etikettenmaterial. Der Lieferant **wählt das Bogenraster selbst** (Zweckform L4760 3×7, Avery 3475 3×8, Kleine Barcodes 4×13) — er druckt auf sein eigenes Material, und davon gibt es verschiedene. Bis zum 06.08.2026 kam der Bogen immer im Zweckform-Raster; wer andere Bögen im Drucker hatte, bekam einen Ausdruck, der danebenliegt. Das gewählte Raster steht anschließend in der Bestellhistorie der Schule.
  - _Groß:_ Lernmittel-Etikett mit Ausleihtabelle, **vier Stück auf einem A4-Blatt** (2×2, mit Schnittlinien) — wird ausgeschnitten, nicht auf vorgestanztes Material gedruckt. Vorher war jedes Etikett eine eigene A6-Seite: Auf einem A4-Drucker kam damit ein Etikett je Blatt heraus, außer man stellte im Druckdialog von Hand „4 Seiten pro Blatt" ein (telefonische Rückmeldung Naacher, 06.08.2026). Die Bestätigung erscheint automatisch in der Bestellhistorie und ist dort von einem manuellen Nachtrag aus der Bibliothek unterscheidbar. Voraussetzung ist die **Öffentliche Adresse** in den Einstellungen — fehlt sie, geht die Bestellung ohne Link raus. Sicherheitszuschnitt: 256-Bit-Token, in der Datenbank nur als Hash, 180 Tage gültig, jederzeit durch einen neuen ersetzbar (der alte stirbt dabei). Details in [SECURITY.md](SECURITY.md).
- **Historie zeigt die neuesten 200 Bestellungen** (Grenze serverseitig, max. 500). Die Kennzahlen im Kopf — Gesamtausgaben, Exemplare, wartende Bestätigungen — zählen dagegen **alle** Bestellungen (`/api/bestellhistorie/uebersicht`), damit aus einer Teilsumme keine vermeintliche Gesamtsumme wird. Ohne die Grenze lieferte die Historie auf einer gewachsenen Datenbank 2,45 MB in 3,9 Sekunden, mit ihr 0,10 MB in 0,17 Sekunden.
- **Nach Topf getrennt (seit 12.09.2026):** Der Bestellbericht (`GET /api/bestellhistorie/bericht`) steht in Blöcken — Lernmittelfreiheit (Land), Schülerbücherei (Schulträger), zuletzt die Alt-Bestellungen ohne Zuordnung —, je mit eigener Summe und der Gesamtsumme darunter; `mittel=` beschränkt ihn auf einen Topf, und die Überschrift nennt ihn. Die Historie filtert serverseitig nach Topf (`GET /api/bestellhistorie?mittel=land|schultraeger|ohne`; `ohne` findet die Alt-Bestellungen, deren Topf im Bestelldetail nachzutragen ist), ein unbekannter Wert ist 400. `/api/bestellhistorie/uebersicht` liefert die Kennzahlen zusätzlich je Topf (immer alle drei, auch mit Null); der Kopf der Historie zeigt sie, sobald mehr als ein Topf Bestellungen hat. Bericht und Historie lesen den Parameter über dieselbe Stelle (`api/mittel_filter.go`).
- **Preise sind optional:** Ist „Preise erfassen" aus, arbeitet das ganze Bestellwesen ohne Geldbeträge — kein Preisfeld, keine Betragsspalten, Berichte zählen Exemplare statt Euro zu summieren.

---

## 8. LUSD-Synchronisation & Datenschutz (DSGVO)

### 8.1. Der LUSD-Import

- Die Landesschuldatenbank (LUSD) ist das führende System für Schülerdaten.
- Der Import überschreibt Namen, Klassen und LUSD-IDs im Bibliothekssystem.
- **Match-Logik (drei Stufen, die Datei entscheidet):** (1) **LUSD-ID**, wenn der Export eine ID-Spalte mit Werten hat; (2) **Vorname + Nachname + Geburtsdatum**, wenn keine ID, aber ein Geburtsdatum da ist — dann ist das Datum in jeder Zeile Pflicht (harter Abbruch statt stillem Überspringen); (3) **nur Vorname + Nachname**, wenn die Datei beides nicht hat (LANIS-Klassenliste `Nachname;Vorname;Klasse;…`) — dort werden Namensgleiche (in Datei oder Bestand) NIE zugeordnet, sondern als „mehrdeutig" gemeldet. Der Export der Schule enthält keine Schüler-ID; Stufe 2 ist der empfohlene Weg, Stufe 3 der Notweg. Die Vorschau nennt die Stufe. **Format:** CSV (Komma/Semikolon, BOM) oder Excel (.xlsx, Titelzeilen über der Kopfzeile, Datumszellen); altes .xls wird mit Anleitung abgewiesen. **Kopfzeilen** werden in drei LUSD-Stilen erkannt: ohne Präfix („Vorname, Nachname, Klasse“ — LANIS-Klassenliste), mit Tabellenname („Schueler_Vorname, Klassen_Klassenbezeichnung“ — Individueller Bericht) und mit Tabellenkürzel („SLR_Vorname, SLR_Nachname, KLA_Klassennamen, SLR_Strasse, SLR_PLZ, SLR_ORT“ — Kürzelstil, wie er in Klassenlisten mit LUSD-Feldnamen vorkommt; seit 26.08.2026). **Mehrere Blätter:** Eine Arbeitsmappe mit einem Blatt je Klasse (6F1, 6F2, …) wird als eine Tabelle gelesen; Blätter ohne Kopfzeile (Deckblatt) werden übersprungen, abweichende Spaltenreihenfolge je Blatt ist erlaubt. Bis 26.08.2026 zählte nur das erste Blatt — im Nur-Name-Modus hätten die übrigen Klassen als Abgänger gegolten. Was die LUSD liefert: Standardberichte (EXTRAS → BERICHTE) als PDF/CSV/Excel, Individuelle Berichte immer als XLSX; eine Schüler-ID enthält keiner dieser Exporte (offizielle Feldliste des HMKB, geprüft 26.08.2026), Stufe 1 bleibt in der Praxis leer. **Littera-Weg (aus dem Littera-Handbuch, Kapitel „LUSD Leserdaten Import“):** LUSD → Unterricht → Export/Import → Stundenplan/Littera → Export → Programmauswahl „Littera“ → `Littera_Export.txt`. Diese Datei ist **verschlüsselt** (hessischer Datenschutz) und nur von Littera lesbar — für unser System unbrauchbar. Littera gleicht darüber mit Vorname + Zuname + Geburtsdatum ab und meldet Abgänger als druckbare Liste zum Bestätigen; genau dieses Modell haben wir mit Stufe 2 (Name + Geburtsdatum) und der Abgänger-Vorschau nachgebaut. Für uns bleibt der Weg über EXTRAS → BERICHTE (XLSX/CSV, unverschlüsselt) — mit Geburtsdatum im Bericht.
- **Gedächtnis `lusd_bestaetigt_am` (Migration 084):** Jeder Import stempelt jede Zeile, die er im Export wiedergefunden oder angelegt hat. In den Namensmodi gilt als Abgänger nur, wer schon einmal bestätigt wurde und jetzt fehlt — nie bestätigte Handanlagen bleiben unangetastet und werden als „nicht im Export" gemeldet, Schüler ohne Geburtsdatum als „nicht abgleichbar" (Stufe 2). Im ID-Modus werden ID-lose Bestandsschüler (Handanlage **und** Littera-Übernahme `lusd_id='littera:…'`) per Name + Geburtsdatum adoptiert statt dupliziert.
- **Klassen-Vokabular (Migrationen 079, 087):** Jede Klasse existiert genau einmal in `klassen`; Schüler, Bücherlisten, Klassensatz-Reservierungen und Klassenlehrer-Zuordnungen tragen FKs darauf, Schreibvarianten („5 a“, „05A“) laufen per Trigger auf denselben Eintrag. **Anzeigeform seit 26.08.2026 fest:** Jahrgang zweistellig, Rest groß — „05F1“, „09G4“, „10G1“ (wie die LUSD liefert). Vorher galt die zuerst geschriebene Form; ein Klassensatz-Import lieferte so „9G4“ neben „09G1“ und „10g1“ neben „10G2“. Sonderwerte (`lehrer` als Handapparat-Entleiher, `ABG`) und Kursnamen bleiben, wie sie sind.
- **Handanlage:** Das Geburtsdatum ist beim manuellen Anlegen Pflicht (UI + API) — es ist die einzige Brücke zum späteren Import; ohne Datum entstünde zwangsläufig ein Duplikat.
- **Neue Kontaktdaten:** Es werden Anschriften und Eltern-E-Mails importiert, jedoch _ausschließlich_ zum Zweck der Rechnungs- und Mahnungsstellung. **LUSD bleibt führend** (Entscheidung 02.09.2026): Ein gefüllter Exportwert überschreibt den Bestand, ein leerer lässt ihn stehen; Handkorrekturen überleben den nächsten Import nicht.
- **Umbenennung ohne Schüler-ID (seit 02.09.2026, `api/lusd_paarung.go`, Migration 094):** Eine Namensänderung oder Datumskorrektur in der LUSD findet der Schlüssel Name + Geburtsdatum nicht — bis dahin: Abgänger + Neuanlage. Jetzt bildet die Vorschau aus Abgängern und Neuzugängen **Paare** („Vermutlich dieselbe Person"): sicher bei gleichem **Schuleintritt** (`Schueler_Eintritt_AktuelleSchule` → `schueler.schul_eintritt_am`, zweiter Schlüssel, der jede Umbenennung übersteht) plus Name oder plus Geburtsdatum + Vorname; vermutlich bei Geburtsdatum + Name/Klasse/Anschrift oder Name + Klasse/Anschrift bei abweichendem Datum. Ein abweichender Vorname bei gleichem Geburtsdatum (Zwillinge) ist Gegen-Signal, bei Gleichstand zweier Kandidaten wird kein Paar gebildet. Der Admin kreuzt an; bestätigt behält der Datensatz UUID, Barcode, Ausleihen und Historie und bekommt Name bzw. Datum aus dem Export. Der Server nimmt nur selbst vorgeschlagene Paare an. Alles Weitere: [LUSD.md](LUSD.md) §3.
- **Zusammenführen von Hand (`POST /api/schueler/{id}/zusammenfuehren`, eigenes Recht `merge_students` seit 03.09.2026, vorher `manage_students_admin`; bestehende Anlagen erben den alten Wert je Rolle):** Das Sicherheitsnetz, wenn eine Dublette erst später auffällt. Ziel behält UUID und Barcode, die Quelle geht auf (Ausleihen, Schäden, Vormerkungen, Foto, Protokollspuren wandern; Zeile wird endgültig gelöscht), Stammdaten vom zuletzt LUSD-bestätigten Datensatz (ein aktiver schlägt einen Abgänger), Abgänger-Status und automatische Sperre werden bereinigt, eine manuelle Sperre der Quelle geht auf das Ziel über. Rückweg: ein `audit_log`-Eintrag in der Transaktion trägt Stammdaten der Quelle und die Kennungen der gewanderten Zeilen. Schülerakte → Stammdaten → „Doppelter Datensatz?". [LUSD.md](LUSD.md) §5.

### 8.2. DSGVO und Lösch-Routinen (Abgänger)

- Wenn ein Schüler in der LUSD nicht mehr auftaucht, wird er im System zum "Abgänger" im Sinne des Imports (`ist_abgaenger = true`): Er ist **weg**. Die Abgänger-Ansicht (§8.x) meint etwas anderes — die Abschlussklassen, die noch da sind; in der Leserdatei heißt der Reiter der Weggegangenen deshalb „Ehemalige / Archiv".
- **Karenzzeit (seit 02.09.2026):** Ein Abgänger ohne offene Vorgänge wird beim Import **nicht mehr sofort anonymisiert**, sondern nur gesperrt (Grund „Automatisierte Abgänger-Sperre (Karenzzeit vor Anonymisierung)"); der nächtliche Job anonymisiert nach der Karenzzeit aus _Einstellungen → Datenschutz & Sitzung_ (`abgaenger_karenz_tage`, Vorgabe 90, 0 = sofort wie früher). Uhr ist der spätere von zwei Zeitpunkten (`repository.KarenzUhr`): `abgaenger_seit` (Migration 094: gesetzt beim ersten Abgang, geräumt bei Rückkehr) und `letzter_vorgang_am` am Leser (Migration 137) — die letzte Rückgabe einer Ausleihe oder der letzte Abschluss eines Schadensfalls, bezahlt oder storniert. Den Vorgang zählt die Uhr seit 05.09.2026; vorher zählte nur der Abgang, und der Schutz kippte mit der Rückgabe, weil offene Vorgänge die Zeile bis dahin gehalten hatten. Bis Migration 137 rechnete sie ihn aus `ausleihen` und `schadensfaelle` — der Lesehistorie-Lauf trennt aber die Ausleihe nach seiner Frist vom Leser, die Uhr fiel dann auf den Abgang zurück, und die Karenz wurde still kürzer als eingestellt. Seitdem stempeln Trigger den Zeitpunkt am Leser, bei jeder Rückgabe und bei jedem Abschluss eines Schadensfalls; die Uhr geht nie zurück. Grenze: Was der Lesehistorie-Lauf vor 137 schon getrennt hatte, ist nicht mehr auffindbar — diese Leser rechnen weiter ab dem Abgang. Import, Job und Selbstprüfung teilen Schlüssel und Prädikat (`repository.PredikatAnonymisierung`). Die Karenz ist der Raum, in dem eine falsche Zuordnung ohne Schüler-ID noch repariert werden kann; sie ersetzt die frühere feste Frist von 360 Tagen (kürzer, nicht länger). Die endgültige Löschung ab dem 30. Januar des Folgejahres trifft seit 05.09.2026 nur anonymisierte Datensätze — sie wartet die Karenz ab, statt sie abzuschneiden; im Cron läuft die Anonymisierung deshalb vor der Löschung (`RunNaechtlicheDSGVO`).
- Die Anonymisierung leert nicht nur den Schülerdatensatz selbst (Name, Adresse, Geburtsdatum, Schuleintritt, LUSD-ID, Foto), sondern tilgt die Personendaten auch aus den **Neben-Tabellen**: den Klarnamen aus dem fachlichen Audit-Log, die LUSD-ID aus dem Admin-Audit-Log und die offenen Vormerkungen. Sonst überlebte der Personenbezug bis zur Audit-Aufbewahrungsfrist.
- **Retention-Blockade:** Ein Abgänger wird **nicht** gelöscht oder anonymisiert, solange er noch Bücher ausgeliehen hat oder unbezahlte Schadensfälle existieren. In diesem Fall wird der Datensatz eingefroren (`ist_gesperrt = true`, Sperrgrund: "Automatisierte Abgänger-Sperre (offene Vorgänge)"). Falls die offenen Vorgänge geklärt werden und der Abgänger im Folgejahr in der LUSD wieder als aktiver Schüler auftaucht, hebt das System die Sperre automatisch wieder auf.
- **Papierkorb:** Manuelles Löschen von Schülern durch den Admin verschiebt diese in einen Papierkorb (Soft-Delete). Ausleihhistorie und Name bleiben vorerst für einen etwaigen Restore erhalten. Erst der `Purge`-Prozess löscht sie endgültig und anonymisiert historische Ausleihen (`schueler_id = NULL`).
- **Lesehistorie ist befristet (seit 22.08.2026):** Eine zurückgegebene Ausleihe bleibt nicht bis zur Schüler-Löschung dem Schüler zugeordnet. Ein nächtlicher Job trennt sie nach Frist (`schueler_id = NULL`) — **Schülerbücherei 90 Tage**, **Lernmittel 730 Tage** nach Rückgabe; beides in den Einstellungen unter „Datenschutz & Sitzung", 0 = aus. Der Vorgang bleibt für Statistik und Bestandskartei erhalten, nur ohne Person. Ausleihen mit offenem Schadensfall bleiben zugeordnet. Die Frist gilt seit dem 16.09.2026 JEDEM LESER, auch dem Kollegium — was ein Erwachsener gelesen hat, muss die Bücherei nach der Rückgabe so wenig wissen wie bei einem Kind. Eine laufende Dauerleihe ist nicht betroffen: Die Frist beginnt mit der Rückgabe. Die Karenz-Uhr hängt daran seit Migration 137 nicht mehr (siehe oben). Folge für die Oberfläche: Schülerprofil und Titel-Historie zeigen nur noch Ausleihen innerhalb der Frist mit Namen, ältere als „anonym".

---

### 8.x Abgänger-Ansicht und Kontoauszug (Ergänzung 30.08.2026; Bedeutung zurückgestellt 05.09.2026)

Die Abgänger-Ansicht (`/abgaenger`) zeigt die **Abschlussklassen mit noch offenen Ausleihen**:
H-Zweig ab Jahrgang 9 (9H, auch das freiwillige 10H), R-Zweig ab 10, sonst ab 13 —
`repository.AbschlussklasseSQL`, dieselbe Regel, mit der die Versetzung Abgänger markiert und
ihre Klassenleitungs-Zuordnung entfernt (Paar-Gate `TestAbgaengerliste_UndVersetzungSehenDieselbeMenge`,
Regel-Tabelle `TestAbschlussklasseSQL_Regel`). Sie sind noch an der Schule (`ist_abgaenger = false`)
und leihen weiter aus. Sichtbar ist die Liste in der **Saison 01.05.–31.07.** (fester Wert,
`api/abgaenger_fenster.go`; `Server.Uhr` nur für Tests): außerhalb antwortet `GET /api/abgaenger`
mit `fenster.offen = false` und leerer Liste, der Druck mit `404`, der Versand mit `409`. Je Klasse
lässt sich ein **Kontoauszug** drucken (`queryAbgaengerKontoauszug`) oder per Mail an die
Klassenleitung schicken (`POST /api/abgaenger/mail`); die Empfängeradresse kommt aus dem
Mahnwesen-Routing (§17, Kategorie Mahnwesen-Routing), Klassen ohne Zuordnung werden im Versand-Dialog als „keine E-Mail"
markiert, statt still übersprungen zu werden.

Historie: Vom 25.06. bis 05.09.2026 filterte die Ansicht `ist_abgaenger = true` (weg laut LUSD) —
ein Bedeutungswechsel, der als Fix eines echten Fehlers (hartkodierte Klassennamen) durchging
(Register, Entscheidung 2). Wer weg ist und noch Bücher hat, steht im Mahnwesen
(`QueryUeberfaelligeNachJahrgang` schließt `ist_abgaenger` ein) und in der Leserdatei unter
„Ehemalige / Archiv" (`GET /api/schueler?status=ehemalige`, `ListEhemaligeWithStats` — dieselbe
Liste und Serversuche wie „Aktive Schüler" mit umgekehrtem Vorzeichen; bis 05.09.2026 bettete
der Reiter die Abgängerliste ein).

## 9. Druck-Center und Ausweise

Das System bietet einen zentralen Druck-Manager für physische Objekte:

- **Barcode-Etiketten:** Das System generiert PDF-Bögen mit Code-128 oder QR-Codes für neu eingetroffene Bücher. Diese können direkt auf vorgefertigte Etikettenbögen (z. B. Avery Zweckform) gedruckt werden.
- **Schülerausweise:** Mit den (verschlüsselten) LUSD-Fotos generiert das System druckfertige Schülerausweise mit persönlichem Barcode für die Ausleihe am Kiosk.

---

## 10. System-Audit & Protokollierung

Um Nachvollziehbarkeit bei sensiblen Schuldaten zu garantieren, gibt es ein Audit-Log:

- Jede administrative Aktion (Benutzer gelöscht, Schadensfall storniert, Schüler manuell gesperrt) wird in der Tabelle `audit_logs` mit `Akteur`, `Zeitstempel`, `IP` und Vorher-/Nachher-Details protokolliert.
- Das Audit-Log ist **append-only als Konvention**: Kein Bedien- oder Codepfad verändert oder löscht Einträge — mit einer bewussten Ausnahme, der DSGVO-Tilgung (siehe unten). Ein früherer Datenbank-Trigger, der jede Änderung hart sperrte (Migration 003), wurde mit Migration 083 aufgelöst: Er stand im direkten Widerspruch zur Löschpflicht (Art. 17), die das Audit-Log gerade ändern muss, um Personenbezug zu entfernen. Wer echte Manipulationssicherheit braucht, setzt sie über eine eng begrenzte Löschausnahme um, nicht über ein pauschales Änderungsverbot.
- Die Daten dienen der Fehlerbehebung und DSGVO-Rechenschaftspflicht.
- **Tresen-Auskunft** (seit 01.09.2026, `api/audit_tresen_auskunft.go`): der EINE
  zweckgebundene Leseweg in die Detail-Spalte des fachlichen Audit-Logs. Anlass: Ein
  zurückgebrachtes Buch, dessen Exemplar (oder ganzer Titel) gelöscht ist, war sonst
  niemandem mehr zuzuordnen, obwohl das Protokoll es weiß. Zuschnitt bewusst eng:
  nur Barcode-Suche, eigenes Recht `audit_details` (ab Werk nur Admin), Stufe 2 der
  PII-Matrix, jeder Abruf wird selbst mit IP protokolliert (`TRESEN_AUSKUNFT`). Alle
  drei Löschwege (Exemplar ausbuchen, Verlust-Löschen, Titel löschen) hinterlassen
  dafür je Exemplar einen Barcode-Snapshot; nach DSGVO-Tilgung oder
  Lesehistorie-Befristung zeigt auch dieser Weg bewusst nichts mehr.
- **Aufbewahrungsfrist** (seit 16.08.2026): Auch Protokolle brauchen ein „wie lange“ —
  IP-Adressen und Bearbeiter-Bezüge sind personenbezogen (Speicherbegrenzung, Art. 5).
  Ein nächtlicher Job (03:00, nach dem Backup) löscht Einträge beider Tabellen jenseits
  der Frist aus `audit_aufbewahrung_monate` (Vorgabe 24 Monate, Untergrenze 6 gegen
  Fehlkonfiguration) und hinterlässt die Löschung selbst als eine Meta-Zeile mit den
  Zahlen — sonst sähe eine spätere Prüfung nur ein Protokoll mit unerklärlicher
  Vorderkante.

---

## 11. Statistiken & Dashboards

Für die Schulleitung und Bibliotheks-Administration aggregiert das System Echtzeit-Metriken:

- Auswertung von Ausleihen pro Jahrgang/Klasse.
- Hitlisten der beliebtesten Medien (LMF vs. Freihand).
- Warn-Dashboards für offene Schäden und eskalierte Mahnungen.
- Export-Funktionen (CSV/PDF) für die Jahresberichte an die Schulleitung.

---

### 11.x Renner und Ladenhüter (Ergänzung 30.08.2026)

Zwei Listen mit Detailseite (`/statistiken/renner`, `/statistiken/ladenhueter`, `api/stats.go`):
**Renner** sind die meistausgeliehenen Titel des gewählten Zeitraums; **Ladenhüter** sind Titel,
die seit mehr als zwei Jahren nicht mehr ausgeliehen wurden — oder nie. Beide Listen lassen sich
nach Fachbereich und Systematik filtern und clientseitig nach Titel/Autor durchsuchen; das
`?limit=`-Argument ist serverseitig gedeckelt. Beide Listen enthalten keine Schülerdaten —
sie zählen Ausleihen, nicht Ausleiher (siehe PII_MATRIX).

## 12. Authentifizierung & Rollenmodell (RBAC)

Der Zugang zum System ist strikt reglementiert und wird durch ein Role-Based Access Control (RBAC) System gesteuert.

### 12.1. Login & Sicherheit

- **Verfahren:** E-Mail und Passwort **gegen den Schul-Mailserver (IMAP)**, nicht gegen die eigene Datenbank. Eine lokale Passwortspalte gibt es seit Migration 012 nicht, und es wird nirgends ein Passwort gehasht oder gespeichert — wer ein Konto anlegt, legt kein Passwort fest, sondern setzt eine E-Mail-Adresse, die auf dem Schulserver existiert (`auth/handlers.go`, `verifyIMAPCredentials`). Hier stand bis zum 11.08.2026 „E-Mail und Passwort (Bcrypt-gehasht)" — das war nie so, und zwei Absätze weiter unten stand bereits das Gegenteil.
- **Folge für die Kontoverwaltung:** Die E-Mail **ist** die Identität. Wer die Spalte `benutzer.email` schreiben darf, übernimmt damit ein Konto; ein Rechte-Audit, das nur auf `rolle` schaut, sieht diesen Weg nicht.
- **Session-Management:** Stateless via JWT (JSON Web Tokens) in HttpOnly-Cookies.
- **Inaktivität (seit 22.08.2026):** Nach 5 Minuten ohne Bedienung leert sich die Theke (kein geladener Schüler mehr), nach 15 Minuten kommt der Sperrbildschirm — Entsperren mit dem eigenen Passwort oder Abmelden. Beide Fristen stehen in den Einstellungen („Datenschutz & Sitzung", 0 = aus). Die Sitzung läuft dabei weiter; es geht um Sichtschutz am Mehrplatz-/Thekenrechner, nicht um einen Logout.
- **Brute-Force-Schutz:** Strenges Rate-Limiting beim Login (Sperre nach mehreren Fehlversuchen pro IP/E-Mail-Kombination).
- **Selbstanmeldung des Kollegiums (`SELBSTANMELDUNG_DOMAIN`, `auth/selbstanmeldung.go`):** Rund 160 Lehrkräfte legt niemand vorab von Hand an. Meldet sich ein Postfach der eingetragenen Schuldomain an, das noch kein Konto hat, entsteht ein **inaktiver** Eintrag mit Rolle `kollegium` (Name aus `vorname.nachname@…` geraten, `zugang_beantragt_am` gesetzt, Audit-Zeile `SELBSTANMELDUNG`); die Lehrkraft liest „Zugang beantragt — die Bibliothek muss ihn noch freischalten“, kein Fehlversuch wird gezählt. Unter Benutzer & Rechte steht der Eintrag als „Zugang beantragt“ mit Zähler oben; „Aktiv“ setzen schaltet frei, danach sieht die Person nur „Mein Portal“ (Migration 070). Ein Antrag verfällt nicht von selbst (entschieden am 22.09.2026): Er bleibt sichtbar, bis jemand freischaltet oder löscht; ein stiller Verfall nähme eine echte Anfrage weg, ohne dass sie je jemand gesehen hätte. Wird das aus Datenschutzgründen anders gewollt, gehört die Frist zu den Löschfristen, nicht als Sonderregel hierher. IMAP beantwortet „wer bist du“, nicht „darfst du rein“ — die Freischaltung bleibt bewusst bei der Schule (ein Schülerpostfach derselben Domain würde sonst ebenfalls hereinkommen). Ist die Variable leer, ist der Weg zu: richtige Zugangsdaten enden dann in „Anmeldung fehlgeschlagen“, die Selbstprüfung meldet das als Warnung. Wer in Bibliothek oder LMF mitarbeitet (Mitarbeiter, Helfer), wird weiterhin von Hand angelegt.

### 12.2. Vier Rollen und ein Grundzustand

**Der Admin vergibt vier Rollen** — Admin, Leitung, Mitarbeiter, Helfer —, deren genaue
Rechte (z.B. `view_students`, `manage_settings`, `perform_actions`) er in der Rechte-Matrix
einstellt. **Kollegium ist keine davon**, sondern der Grundzustand jeder Lehrkraft; es steht
unten als Nummer 5, weil es technisch derselbe Enum-Wert ist, und aus keinem anderen Grund.

Bis zum 16.09.2026 stand Kollegium als fünfte Spalte in der Rechte-Matrix, neben Leitung und
Mitarbeiter. Dort sah es aus wie eine Stufe in einer Rangfolge. Es ist keine: Jede Lehrkraft
meldet sich über „Mein Portal" selbst an, wird freigeschaltet und ist damit erst einmal
niemand Besonderes. Eine Rolle bekommt nur, wen der Admin an seiner E-Mail-Adresse dazu
erhebt. Was das Kollegium darf, steht deshalb fest in `db/seed.go` und ist kein Schalter je
Schule — das ist eine Produktentscheidung.

1. **Admin (`admin`):** Uneingeschränkter Zugriff auf alle Systembereiche, Einstellungen, Audits und Datenschutz-Routinen.
2. **Leitung (`leitung`, Migration 121/122, seit 16.09.2026):** Die Rechte des Admins **minus** `manage_users` (Benutzer & Rechte) und `manage_settings` (Einstellungen) — die Rolle für die Person, die die Bibliothek führt, ohne die Systempflege zu übernehmen.

   **Das Soll wird abgeleitet, nicht abgeschrieben:** Migration 122 erzeugt die Zeilen aus den ADMIN-Zeilen und setzt genau die zwei Ausnahmen auf `false`. Eine abgeschriebene Rechteliste wäre eine zweite Wahrheit neben `db/seed.go` und liefe beim nächsten neuen Recht auseinander — die Leitung bekäme es nicht, und niemand merkte es. Gate: `db/rolle_leitung_test.go` leitet dasselbe aus der Vorgabe ab.

   **Warum ausgerechnet `manage_users` fehlt:** Mit dem Recht ändert man die E-Mail-Adresse eines Kontos, und die Anmeldung erkennt eine Person allein an ihrer E-Mail. Die Rechtevergabe wäre damit der Weg in jedes Konto der Anlage. Der Admin kann das Recht erteilen; ein Admin-KONTO bleibt der Leitung auch dann verschlossen (`api/user_admin_eskalation.go`).
3. **Mitarbeiter (`mitarbeiter`):** Das Personal für das Tagesgeschäft. Hat Zugriff auf die Scanner-Omnibox, Buchkatalog, Mahnwesen und Leserdatei, darf aber keine Systemeinstellungen ändern.
4. **Helfer (`helfer`):** siehe unten.
5. **Kollegium (`kollegium`) — der Grundzustand, keine vergebene Rolle:** Zugang zum Kollegiums-Portal mit fünf Reitern (Stand 05.09.2026): _Suchen & Reservieren_, _Klassensätze_ (welche Klasse hat welche Bücher — Handliste `class_books` plus live aus den Ausleihen abgeleitet, seit 05.09.2026: mehr als die Hälfte der Klasse und mindestens `KlassensatzMindestLeser` Kinder halten den Titel; `GetClassGroups`, Quelle `hand`/`ausleihe`, nie gespeichert), _LMF-Plan_ (Rückgabe- und Ausgabetermine je Klasse, §2.3), _Schulbücher_ (Suche über Titel, ISBN, Autor und Fach; Filter Jahrgang und Schulzweig; je Fach eine aufklappbare Zeile mit Exemplaren, Titeln und Verliehenen; Export je Fach als **PDF** mit Coverbildern, Jahrgang, Schulzweig und Zähldatum; nur Titel mit `ist_lernmittel`; Portal-Routen `/api/portal/lernmittel[/export]`) und _Meine Anliegen_. Erteilt ist weiterhin ein einziges Recht, `create_reservations` (Migration 070): Die Suche läuft über den öffentlichen OPAC, Reservierung und Anliegen über `create_reservations`; die Klassensatz-Sicht hängt an einer eigenen Portal-Route (`/api/portal/klassensaetze`), für die die Anmeldung genügt — bewusst kein `view_books`, das der Rolle den ganzen Medienkatalog öffnen würde. Nichts davon fasst Personendaten an.

   **„Mein Portal“ hängt seit 26.08.2026 am Recht `create_reservations`, nicht an der Rolle** (entschieden): Eine Lehrkraft, die in Bibliothek oder LMF mitarbeitet und deshalb als Mitarbeiter angelegt ist, sieht das Portal ebenfalls und reserviert dort für die eigene Klasse. Vorher stand der Menüpunkt auf `roles: ['kollegium']`, während der Server sie mit demselben Recht längst hineinließ — zwei Wahrheitsquellen, die nur zufällig einig waren.

   **Der Enum-Wert hieß bis zum 10.08.2026 `lehrer`** (Migration 069). Das Wort war doppelt belegt — als Anmelde-Rolle _und_ als Entleihertyp `schueler.klasse = 'lehrer'` (eigene Behandlung im Mahnwesen). Den zweiten Weg gibt es seit Migration 072 nicht mehr; eine Lehrkraft als Entleiher steht seit Migration 123 in `leser` mit `art = 'lehrkraft'` (§12.3). Die Umbenennung selbst war keine Rechteänderung.

   **Der Rechteumfang war es** (Migration 070): Auf dem Schulserver sah ein Kollegiums-Konto am 10.08.2026 zehn von fünfzehn Menüpunkten, darunter Schülerdatei, Mahnwesen, System-Logs und Einstellungen — `role_permissions` führte `manage_users`, `audit_logs`, `view_stats`, `view_students`, `view_books` und `perform_actions` auf `true`. Das war keine reine Anzeigefrage: Dieselbe Tabelle entscheidet in `RequirePermission`, die API hätte es ebenfalls zugelassen. Alles außer `create_reservations` ist entzogen. Wer einer Lehrkraft gezielt mehr geben will, tut das im PermissionManager — Migrationen laufen nur einmal, eine spätere Vergabe wird nicht zurückgedreht.

**Helfer (`helfer`):** Stark limitierte Rolle für studentische Hilfskräfte oder Eltern. Kiosk-Ansicht (Omnibox) für Ausleihe und Rückgabe, dazu **lesender Katalogzugriff** (Entscheidung vom 30.07.2026, Migration 055): Ein Helfer an der Theke ist die erste Anlaufstelle für „Habt ihr Band 3 noch da?" und musste die Frage sonst weiterreichen. Die Grenze zu Personendaten zieht weiterhin `view_students`.

   **Ein Helfer braucht ein Postfach auf dem Schul-Mailserver.** Das ist die Frage, die in der Praxis zuerst kommt, und sie hatte bis zum 08.08.2026 keine Antwort in dieser Doku. Die Anmeldung läuft ausschließlich über E-Mail + Passwort gegen IMAP (`auth/handlers.go`); eine lokale Passwortspalte gibt es seit Migration 012 nicht, und einen Code- oder Barcode-Anmeldeweg gibt es nicht — die Felder `barcode_id`/`pin` standen einmal im `LoginRequest`, wurden nie ausgewertet und sind entfernt. Wer eine Hilfskraft aufnehmen will, lässt also zuerst ein Postfach anlegen und trägt dann unter System → Benutzer & Rechte (Reiter „Benutzer“) die Person mit der Rolle „Helfer" ein (die Benutzerverwaltung sitzt seit 16.08.2026 dort, nicht mehr in den Einstellungen). Die E-Mail ist dabei die Identität: Wer die Spalte `benutzer.email` schreibt, übernimmt das Konto.

   Erteilt sind genau zwei Rechte (`db/seed.go`): `perform_actions` (Scannen, Ausleihe, Rückgabe) und `view_books` (Katalog). Erreichbar sind damit Ausleihe, Medienkatalog, Signaturen und Schulklassen — Letztere seit dem 08.08.2026, weil der Klassensatz-Reiter im Katalog aufgelöst wurde und der Blick darauf sonst verloren gegangen wäre. Die Pflege-Aktionen auf diesen Seiten hängen an `edit_books` und bleiben dem Helfer verborgen.

   **Welche Seiten eine Rolle erreicht, entscheidet `canSeeItem()` in `frontend/src/lib/menu.js` — und nur diese Funktion.** Der Router fragt dieselbe. Bis zum 08.08.2026 führte er eine zweite, handgepflegte Liste; als das Recht für „Schulklassen" wechselte, liefen beide auseinander, und der Helfer bekam einen Menüpunkt, der ihn beim Klick wortlos an die Theke zurückwarf. Wer eine Rolle oder ein Recht ändert, fasst deshalb `menu.js` an und sonst nichts. Abgesichert durch `e2e/menue-fuehrt-irgendwohin.spec.js`, das für Helfer, Mitarbeiter und Lehrkraft jeden sichtbaren Menüpunkt anklickt.

### 12.3. Leserdatei: eine Tabelle, drei Arten (Migration 123–125, 16.09.2026)

**Rolle und Art sind zwei verschiedene Fragen.** Die Rolle (§12.2) sagt, was jemand im
Programm DARF. Die Art sagt, WER an der Theke Bücher bekommt. Ein Mensch kann beides haben,
eines von beidem oder keines: Eine Lehrkraft ohne Konto steht in der Leserdatei und darf
nichts im Programm; ein Admin muss nicht in der Leserdatei stehen.

Schüler und Kollegium stehen seit Migration 123 in EINER Tabelle `leser` mit der Spalte
`art`:

| Art         | Wort in der Oberfläche | Woher                                                     |
| ----------- | ---------------------- | --------------------------------------------------------- |
| `schueler`  | Schüler                | LUSD-Import oder von Hand                                  |
| `lehrkraft` | Lehrkraft              | Selbstanmeldung („Mein Portal"), Handanlage, Littera-Bestand |
| `liv`       | LiV                    | von Hand, oder eine Lehrkraft wird dazu umgestellt          |

Die Art entscheidet **keine Rechte**; ausleihen darf jeder aktive Leser. Ändern lässt sie
sich nur zwischen Lehrkraft und LiV: Ein Schüler kommt aus der LUSD und bleibt Schüler, in
beide Richtungen (`chk_leser_nur_schueler_werden_abgaenger`).

**`schueler` ist seither eine Sicht**, nicht mehr eine Tabelle: `WHERE art = 'schueler'`
mit `WITH CHECK OPTION` (Migration 124). Das trägt die alte Bedeutung weiter — Klassenlisten,
LUSD-Abgleich, Mahnlauf und die DSGVO-Löschfristen lesen sie und bekommen das Kollegium
nicht zu sehen. Die Kehrseite ist eine eigene Bugklasse; sie steht in
[sweeps.md](sweeps.md) („Schreibpfad gegen gefilterte Sicht").

**Eine Maske für jeden.** Akte und Formular zeigen für jede Art dieselben Felder an
derselben Stelle. Einem Kollegen sind drei verschlossen, und zwar nicht aus Geschmack:
Klasse und Abgangsjahr (eine Klasse gehört keiner Lehrkraft, und das Abgangsjahr leitet der
Server aus ihr ab) sowie die LUSD-Kennung (die Datenbank verbietet sie einem Nicht-Schüler).
Geburtsdatum, Ausweisnummer, Anschrift und Eltern-E-Mail stehen jedem offen und bleiben beim
Kollegen leer.

**Die Schul-E-Mail ist beim Anlegen einer Lehrkraft oder LiV Pflicht.** Sie ist keine
Kontaktangabe, sondern der Schlüssel: Mit ihr entsteht sofort das Anmeldekonto, und weil der
Anmeldeweg eine Zugangsanfrage nur anlegt, wenn zu der Adresse GAR KEIN Konto existiert,
findet die spätere Selbstanmeldung genau diesen Eintrag. Ohne sie stand die Person danach
zweimal in der Leserdatei — Ausweis und Ausleihen am ersten Eintrag, die Anmeldung am
zweiten. Freigeschaltet wird das Konto nur, wenn der Anlegende `manage_users` hat; sonst
entsteht eine Zugangsanfrage wie bei der Selbstanmeldung. Die Adresse steht dabei an genau
EINER Stelle, am Konto (`benutzer.email`, `UNIQUE lower(email)`) — eine zweite Spalte an der
Leserzeile gibt es bewusst nicht.

**Ausweise:** Alle Leser ziehen ihre Nummer aus EINEM Nummernkreis (Migration 125), und
alle neuen Nummern tragen die Vorsilbe `A-` (§1). Die Aufschrift der gedruckten Karte
richtet sich nach der Art — „Schülerausweis" oder „Lehrerausweis"; eine Gültigkeit trägt
nur der Schülerausweis, weil der Ausweis einer Lehrkraft mit keinem Schuljahr abläuft.

---

## 13. Katalogisierung & Medienverwaltung

Das System bietet umfassende Werkzeuge zur Pflege des Buchkatalogs:

- **Titel ohne Exemplar stehen in keinem Katalog (seit 22.09.2026, Antwort der Schule auf Punkt 4 der Sichtung vom 16.09.2026):** Die drei Türen, über die ein Kollegium Titel sieht — die Katalogliste `GET /api/books` (Portal und Titel-Verwaltung), die Theken-Suche (`SearchTitlesFuzzy`) und die Aktionssuche der Omnibox (`SearchTitles`) — zeigen nur Titel mit mindestens einem nicht ausgesonderten Exemplar; der Zulauf zählt als vorhanden. EIN Prädikat, `repository.SQLTitelHatExemplar` (`book_bestand.go`). Der Titel bleibt in der Tabelle, sonst legt ihn jemand ein zweites Mal an: Die Titel-Verwaltung erreicht ihn über die Aufräumsicht `GET /api/books?bestand=ohne` (dasselbe Prädikat mit NOT, Umschalter „Mit Exemplaren | Ohne Exemplare"), die Bestellliste führt ihn unter der Schwelle. Der öffentliche Katalog und der Monitor haben eine engere eigene Regel (`OeffentlichSichtbar`: kein Lernmittel, kein Zulauf). Belegt: `repository/titel_bestand_pg_test.go`, `inventur/titel_ohne_exemplar_pg_test.go`.
- **Mehrjahresband (seit 22.09.2026, Antwort der Schule auf Protokoll 5 der Sichtung vom 16.09.2026; Migration 134):** Am Werk steht der Schalter `mehrjahresband` (Titel-Verwaltung, nur bei Lernmitteln). Die Jahreszahl, bis zu der das Buch beim Kind bleibt, ist `jahrgang_bis` aus der Spanne „Im Unterricht von Jahrgang … bis" — eine zweite Zahl gibt es nicht; die Spalte `ziel_jahrgang` (Migration 030, nie beschrieben, am selben Tag kurz mit einer Tür versehen) ist mit 134 gefallen. Die Frist ist der Stichtag des Schuljahres, in dem das Kind `jahrgang_bis` beendet: `AdditionalYears = jahrgang_bis − Jahrgang der Klasse`, dann `AddDate` am Stichtag (`internal/service/loan_rules.go`). Vor dem Rückgabetermin der Klasse geht der Termin das Buch nichts an; ist er vorbei, rechnet die Frist vom folgenden Schuljahr aus, und eines der Jahre steckt in diesem Sprung. Der Schalter gilt nur an einem Lernmittel und nur mit einer Spanne über mehr als einen Jahrgang — an beiden Türen der Titel-Verwaltung (`inventur/mehrjahresband.go`, 400) und in der Datenbank (`chk_mehrjahresband_spanne`) dieselbe Regel. Das Schuljahr, das Littera getrennt führt, steckt hier im Datum der Frist. Belegt: `internal/service/lmf_frist_termintag_pg_test.go`, `inventur/mehrjahresband_am_titel_pg_test.go`.
- **Systematiken & Signaturen:** Bücher können hierarchisch nach Systematiken (Kategorien/Themen) und spezifischen Signaturen (Regal-/Standort-Kennung) klassifiziert werden.
- **Schlagworte (Migrationen 138, 143, 144; docs/OFFEN.md 4.20):** Frei eintragbar wie in Littera („Schlagworte (Wertehilfe)"), mehrere je Titel, mit Vorschlägen aus dem Bestand (die 500 häufigsten, `GET /api/schlagworte`). Eingetragen wird im Buchformular und im Bestellkorb; beide schreiben über EINEN Pfad, `repository.SetzeSchlagworte`: Leerraum wird zusammengezogen, Doppelte fallen, höchstens 30 Wörter je Titel und 80 Zeichen je Wort, und eine vorhandene Schreibweise gewinnt — die Identität eines Wortes ist seine Kleinschreibung (eindeutiger Index auf `lower(wort)`). Zusammengehalten wird die Liste durch die Pflege unter _Einstellungen → Schlagworte_ (Recht `edit_books`, wie die Systematik): Umbenennen und Zusammenführen ändern alle Titel auf einmal und lassen die alte Schreibweise wahlweise als **Verweis** stehen (Pflichtfeld `alte_als_verweis`, das Kästchen steht auf „behalten"). Ein Verweis leitet eine Schreibweise auf ein Wort („Tierfantasy" → „Fantasy"); wer ihn am Titel einträgt, bekommt das Ziel. Löschen nimmt ein Wort aus allen Titeln, Verweise darauf fallen mit — einzeln oder mehrere auf einmal, alle oder keins (`POST /api/schlagworte/loeschen`; Littera: Dienstprogramme → Datenbearbeitung). Die Regeln der Verweise hält auch die Datenbank (Trigger aus 143, seit 144 mit Sperre gegen gleichzeitige Schreiber): kein Verweis auf sich selbst, keine Kette, kein Titel an einem Verweis, kein Verweis als Filter. Noch nicht gebaut: Suche und Portal nutzen die Schlagworte nicht, auch nicht die als **Filter** markierten Wörter (4.20 Stufe 1), und der Littera-Import leitet aus MAB 710 nur das Fach ab, statt die Wörter zu übernehmen. Belegt: `repository/schlagworte_pg_test.go`, `repository/schlagworte_pflege_pg_test.go`, `repository/schlagworte_loeschen_pg_test.go`, `frontend/e2e/schlagworte-roundtrip.spec.js`, `frontend/e2e/schlagworte-pflege.spec.js`, `frontend/e2e/schlagworte-mehrere-loeschen.spec.js`.
- **Automatische Cover-Synchronisation:** Ein Hintergrund-Worker (`Cover-Sync`) sucht über ISBNs automatisch in externen Buch-APIs (z.B. Google Books) nach Buchcovern, lädt diese herunter und speichert sie datensparsam im WebP-Format.
- **Legacy-Import-Engine:** Für die initiale Einrichtung oder Datenübernahme bietet das System eine dynamische Import-Schnittstelle (`/api/import/littera`), um Altbestände aus Legacy-Programmen (wie z. B. _Littera_) per CSV einzulesen und zu mappen.
- **Drei Importwege in der Datenverwaltung (Stand 07.09.2026), unterschieden danach, ob sie Exemplare übernehmen oder erzeugen:** Der _Katalog-Import_ (Littera-XML) aktualisiert Metadaten bestehender Titel. Der _Bestands-Import_ (Kombi-CSV, `/api/admin/import-bestand`, Recht `manage_inventory`) ÜBERNIMMT vorhandene Exemplare samt ihrer Nummer aus der Datei — sie tragen schon ein Etikett, `etikett_gedruckt = true`. Der _Listenimport_ (`POST /api/books/import`, Recht `edit_books`) verlangt nur ISBN und Stückzahl und ERZEUGT neue Exemplare mit frischen Nummern aus `barcode_seq` — ohne Etikett, `etikett_gedruckt = false`, also auf der Nachdruck-Liste. Sein Knopf saß bis zum 21.06.2026 in der Titel-Verwaltung und ging beim Entschlacken der Toolbar ohne Ersatz verloren; seit dem 07.09.2026 steht er in der Datenverwaltung (`ListenImportWidget.svelte`, E2E `datenverwaltung-importe.spec.js`). Weil er additiv ist, benennt das Widget jeden Lauf mit einem Idempotenz-Schlüssel je Dateiauswahl (`X-Idempotency-Key`, `inventur/import_idempotenz.go`, Tabelle `idempotency_keys` wie bei den Theken-Aktionen): Nach einer verlorenen Antwort holt der zweite Klick das Ergebnis des ersten Laufs ab, statt die Exemplare doppelt anzulegen; solange der Lauf noch offen ist, antwortet der Server 409. Der Pfad steht seit demselben Tag in der 5-Minuten-Liste der Langläufer (`api/middleware.go`).
- **Exemplarnummern — eine Quelle (Migration 068, vervollständigt durch 105 am 07.09.2026):** Jede Nummer, die das System selbst vergibt, kommt aus `barcode_seq` (Präfix `B-`, `repository/barcode_vergabe.go`, mit Bestandsabgleich gegen von Hand eingetippte Nummern): Bestellwesen, Handvergabe in der Exemplarkarte, Littera-Import, Bestandskorrektur in der Buchmaske („Aktueller Bestand") und Listenimport der Titel-Verwaltung. Die letzten beiden zogen bis zum 07.09.2026 aus einer eigenen, nirgends deklarierten Sequenz und prägten `SYS-…` — ein zweiter Nummernkreis, den 068 übersehen hatte; die Omnibox kannte nur `B-`, und der Platzhalter konnte über die Nachdruck-Liste als „SYS-100045" auf ein Etikett gedruckt werden. Eine Nummer wird nie umgeschrieben oder recycelt — sie klebt physisch am Buch. Vorhandene `SYS-`/`AUTO-`-Exemplare bleiben deshalb, wie sie sind, und zeigen in der Exemplarkarte „Barcode scannen". Belegt: `api/barcode_vergabe_pg_test.go`, `inventur/bestandskorrektur_nummernkreis_pg_test.go`.

- **Zugangsbuch (seit 17.09.2026, Berichte → „Bestandsbücher", Reiter „Zugangsbuch"):** Welche Exemplare in einem Zeitraum in den Bestand gekommen sind — Eingangsdatum, Nummer, Titel und Lieferant, die Angaben der Arbeitshilfe (mittel_konzept 7.1). Aufbau, Zeitraum und Ausdruck wie beim Abgangsbuch; beide teilen sich ein Bauteil (`components/bestand/Bestandsbuch.svelte`) und die Abschnitts-Bildung des Servers (`api/bestandsbuch.go`). Der Topf kommt aus der **Bestellung**, nicht aus dem Titel: Beim Zugang zählt, aus welchem Geld das Buch bezahlt wurde (Migration 109, „eine Bestellung = ein Topf"), und ein Titel darf im Warenkorb in den anderen Topf geschoben werden. **Das Zugangsdatum ist die Lieferung, nicht die Bestellung** (`zugang_am`, Migration 129): Im Bestellweg entsteht die Exemplarzeile beim Bestellen; bis zum 17.09.2026 las das Buch `erworben_am` und datierte ein im Dezember bestelltes, im Februar geliefertes Exemplar auf den Dezember — Exemplare im Zulauf standen mit darin. Gesetzt wird das Datum von einem Trigger, sobald ein Exemplar in den Bestand kommt (Wareneingang, Freigeben im Status-Editor, Zurückholen); ein Exemplar im Zulauf trägt NULL und steht in keinem Zugangsbuch. Exemplare ohne hinterlegte Bestellung stehen unter „ohne Zuordnung" — Altbestand, Handanlage, Bestandskorrektur —, und ihr Zugangsdatum ist der Tag der Anlage; bei der Littera-Übernahme das echte Datum aus der Altanwendung. Ein später ausgesondertes Exemplar bleibt im Zugangsbuch stehen: Es ist trotzdem zugegangen, sein Abgang steht im Abgangsbuch. Belegt: `api/zugangsbuch_pg_test.go`, `api/zugangsdatum_pg_test.go` (die Türen aus dem Zulauf), `frontend/e2e/zugangsbuch.spec.js`.
- **Abgangsbuch (seit 17.09.2026, Berichte → „Bestandsbücher", Reiter „Abgangsbuch"):** Welche Exemplare in einem Zeitraum aus dem Bestand gegangen sind — Datum, Nummer, Titel, Signatur und Grund (Verlust, Beschädigung, Aussortiert, Bestandskorrektur). Vorbelegt ist das laufende Schulhalbjahr mit den Stichtagen der Bestandskartei (15.3. und 15.9., `pkg/schulzeit.Halbjahr`); beide Datumsfelder lassen sich überschreiben. Der Zeitraum wird am SERVER bestimmt, damit Bildschirm und Ausdruck denselben meinen. Zwei Abschnitte mit eigener Stückzahl, Lernmittel (Land) und Schülerbücherei (Schulträger), weil die Finanzen der beiden Töpfe getrennt geführt werden. Das Blatt zum Abheften kommt über `GET /api/bestand/abgangsbuch/pdf`. **Was nicht darauf steht, steht ausdrücklich darunter:** Exemplare, die vor Migration 128 ausgesondert wurden, tragen kein Abgangsdatum und lassen sich keinem Zeitraum zuordnen — ihre Zahl nennt der Ausdruck, statt Vollständigkeit zu behaupten. Das Datum selbst setzt ein Trigger am Zustandswechsel, nicht die sechs Aussonder-Wege (Migration 128). Belegt: `api/abgangsbuch_pg_test.go` (Ränder des Zeitraums, Töpfe, Blatt), `api/abgangsdatum_pg_test.go`, `frontend/e2e/abgangsbuch.spec.js`.

---

## 14. Schadensmanagement

Nicht nur bei Hardware, sondern auch bei Büchern greift ein dediziertes Schadensmanagement:

- Wenn ein Buch als "Verlust" oder "Beschädigt" ausgebucht wird (z.B. bei der Inventur oder manuell am Kiosk), kann das System automatisch eine Kostenforderung (Schadensfall) gegen den verursachenden Schüler anlegen.
- Offene Schäden blockieren die DSGVO-Löschung eines Schülers und können per PDF-Rechnung ausgedruckt werden.
- **Gegen einen Kollegen entsteht KEINE Forderung** (seit 16.09.2026). „Verlust/Schaden melden" steht in seiner
  Akte und bucht, was den Bestand angeht — das Exemplar wird ausgesondert, die Ausleihe endet, eine Vormerkung
  darauf wird gelöst —, aber ohne Schadensfall. Grund: Der Weg einer Forderung endet im Schadensersatz-Bescheid,
  und der ist ein Schreiben an Erziehungsberechtigte; er braucht Klasse und Anschrift, die von einer Lehrkraft
  nirgends stehen. Ersatz von einer Lehrkraft zu verlangen ist Sache der Schulleitung gegenüber dem Dienstherrn
  und dort auch nur bei Vorsatz oder grober Fahrlässigkeit. Vorher entstand die Forderung und stand in keiner
  Übersicht: Der Reiter „Schadensersatz" liest die Sicht `schueler`, und „Bescheid erstellen" antwortete
  „Schüler nicht gefunden".
- **Was ein Buch heute noch wert ist, steht am Exemplar** (seit 17.09.2026, `pkg/ersatzwert`,
  Migration 127): Die Buchakte zeigt an jedem Exemplar den Ersatzwert samt Herleitung
  („3. Verleihjahr → 60 % von 24,90 €"). Zwei Regeln, nicht eine — für ein **Lernmittel des
  Landes** die Staffel der Arbeitshilfe (1. Verleihjahr voller Preis, dann 80/60/40/20 %, ab
  dem 6. Jahr 10 %), für ein Buch der **Schülerbücherei** der Neuwert ohne Altersabschlag
  („zuerst Ersatzbeschaffung, sonst Geld in Höhe des Neuwerts", Benutzungsordnung). Ein
  beschädigtes Exemplar trägt zusätzlich einen Prozentwert für seinen Zustand
  (`zustand_abwertung_prozent`, erfasst im Zustands-Dialog der Buchakte, nicht an der Theke):
  Er wirkt NACH der Staffel und mindert jeden künftigen Ersatzbetrag in beiden Töpfen, zieht
  das Buch aber nicht aus dem Verkehr — er ist ein Zustand, keine Forderung. Welcher Preis die
  Grundlage ist, entscheidet die Einstellung „Immer mit dem Einkaufspreis rechnen" (Kategorie
  Schadensersatz): aus = der heutige Listenpreis, an = der damals bezahlte Einkaufspreis; die
  Herleitung nennt, welcher gegolten hat, und unterscheidet „kein Listenpreis hinterlegt" von
  „so eingestellt". In jedem Fall ein Vorschlag, kein Automat: Der Mensch bestätigt oder
  überschreibt, und ohne hinterlegten Preis bleibt das Feld bei 0 mit einem Satz, der es sagt.
- **Wohin gezahlt wird, steht im Brief** (seit 17.09.2026, `pdf/zahlungsweg.go`): Für ein
  Lernmittel des Landes nennen Elternbrief und Rechnung die Zahlstelle und die Bankverbindung
  aus den Einstellungen — dieselbe Angabe wie im Schadensersatz-Bescheid. Für ein Buch der
  Schülerbücherei steht dort, dass der Weg des Schulträgers noch nicht hinterlegt ist; er ist
  eine offene Frage an den Träger. Trägt eine Rechnung beides, stehen beide Wege mit ihrer
  Teilsumme da. Bis dahin verlangten beide Briefe „bar in der Bibliothek" — für Landeseigentum
  ist das die Ausnahme mit Quittung, nicht der Weg, den ein Brief vorsieht.
- **Erledigt wird eine Gebühr in der Schülerakte** (seit 16.08.2026) auf genau zwei Wegen:
  **„Bezahlt“** (Barzahlung am Tresen) oder **„Stornieren“** mit Pflicht-Begründung
  (Erlass, Buch wiedergefunden, Kulanz). Beides verlangt das Recht `edit_students` —
  bewusst nicht die Kiosk-/Helfer-Rolle. Der Betrag im revisionssicheren Audit-Eintrag
  stammt immer aus der Datenbank, nie aus der Anfrage; eine bereits erledigte Gebühr
  meldet dem zweiten Bearbeiter einen Konflikt (409) statt einer Doppelbuchung.
  Stornierte wie bezahlte Gebühren geben Schülerlöschung, LUSD-Abgleich und Ausleihe
  wieder frei; die DSGVO-Auskunft weist Stornierungszeitpunkt und -grund transparent aus.

---

## 15. Selbstprüfung der Betriebsbereitschaft

Erreichbar unter **System → Einstellungen → Kategorie „Betriebsbereitschaft“** (Recht `manage_settings`,
`GET /api/admin/system/betriebsbereitschaft`). Die Seite beantwortet **eine** Frage:
_Was ist eingerichtet, aber nicht in Betrieb?_

Der Anlass ist eine wiederkehrende Fehlerart, und sie ist immer dieselbe: Eine Funktion
ist fertig programmiert, getestet und verdrahtet — und tut nichts, weil eine Einstellung
fehlt. Kein Fehler, kein Statuscode, kein Logeintrag, der jemandem auffiele. Dreimal
gefunden, jedes Mal von Hand: die Auslagerung der Backups (vier leere Variablen), der
nächtliche Backup-Job (Schlüssel stand in der `.env`, kam aber nicht im Container an)
und der Bestell-Bestätigungslink (`oeffentliche_adresse` nie gesetzt — die Mails gingen
raus, nur ohne den Link, um dessentwillen es sie gibt).

**Geprüft werden neunzehn Bereiche**, darunter: Auslagerung der Backups, Geheimnisse, Schlüssel und
Bestand (seit 21.09.2026: lässt sich je verschlüsselter Spalte ein Wert mit dem laufenden
`APP_ENCRYPTION_KEY` entschlüsseln — der Fall dahinter ist die Wiederherstellung mit neuer `.env`), Anmeldung
(IMAP), Bestell-Bestätigungslink, Mailversand (Mahnwesen), Demo-Daten (Schüler UND
Exemplare aus `seed_demo.sql`), Lieferanten (seit 07.09.2026, kritisch: Reste der drei
Beispiel-Lieferanten, die der Programmstart vom 30.05. bis Migration 107 bei leerer Tabelle
anlegte — in der ersten Bauwoche gewollte Startdaten; auf einer echten Anlage aber von
echten Händlern nicht zu unterscheiden, eine Bestellung an so einen Eintrag ging wirklich an
die Beispiel-Adresse und meldete „gesendet"), Admin-Konten
(wer hat Vollzugriff und erhält die Kritisch-Alarme — der Alarm-Mail-Vorfall vom
16.08.2026 zeigte vier aktive Admins, drei davon dem Betreiber unbekannt; Konten-Anlage
und -Änderung werden seither auditiert), Rechte-Vorgabe
(Abgleich der Live-Tabelle `role_permissions` gegen die Code-Vorgabe — der Seed fasst
bestehende Zeilen nie an, eine geänderte Vorgabe erreicht Alt-Anlagen sonst nie; eine
Abweichung ist bewusst nur eine Warnung, denn sie kann eine Admin-Entscheidung sein)
und die Klassen-Zuordnung (seit 18.08.2026, Befund F3: Klassennamen verbinden Schüler,
Klassenlehrer-Zuordnung und Bücherlisten nur als übereinstimmender Text — die Prüfung
benennt Klassen ohne Lehrkraft, verwaiste Zuordnungen und Bücherlisten ohne Klasse); sowie **Ehemalige mit offenen Vorgängen** (seit 05.09.2026, Register Entscheidung 1:
Weggegangene, die seit mehr als einem Jahr ein offenes Buch oder eine unbezahlte Forderung
haben — der Vorgang schützt sie vor Anonymisierung und Löschung, also bleiben Name und
Anschrift sonst auf Dauer; Warnung mit Abhilfe „Verlust melden, Forderung bezahlen oder
stornieren", kein Automatismus; `ZaehleEhemaligeMitOffenenVorgaengen`); und der **Schadensersatz-Bescheid**
(seit 11.09.2026: welche Pflichtangaben fehlen — Nummer des Schulamtsbereichs, Schulnummer,
Aufsichtsbehörde, Schulleitung, Anschrift der Schule; dieselbe Prüfung, mit der das Erstellen
abweist; Warnung, weil ohne Bescheid-Bedarf nichts ausfällt).

Jeder Befund trägt vier Angaben, weil drei nicht reichen: **Befund** („was ist"),
**Folge** („warum das zählt") und **Abhilfe** („was zu tun ist") — ohne die letzte landet
die Meldung auf einem Zettel statt in der `.env`. Dazu eine **Stufe**: `ok` (in Betrieb),
`warnung` (läuft, aber nicht wie gedacht), `kritisch` (vor dem Echtbetrieb zwingend zu
klären). Bewusst nur drei — eine feinere Skala liest niemand.

**Der Wächter meldet sich selbst** (seit 16.08.2026): Kritische Befunde gehen täglich
per Mail — solange sie bestehen, mit Befund, Folge und Abhilfe je Punkt. Der
Empfängerkreis ist seit 17.08.2026 einstellbar (Einstellungen → Erreichbarkeit & Alarme,
`alarm_empfaenger`, mehrere Adressen mit Komma); ist er leer, gehen die Mails als
sicherer Rückfall an alle aktiven Admin-Konten — ein Alarm, der niemanden erreicht,
ist keiner. Die Mail nennt im Fußtext, an wen sie ging und in welchem Modus. Warnungen lösen bewusst keine Mail aus (Dauerrauschen stumpft ab);
ihre Zahl steht als Fußnote. Auf Spielwiesen (`APP_ENV=local/development/test`)
schweigt der Alarm — die lokale `.env` zeigt auf den echten Schul-SMTP.

Die Seite ist eine **reine Prüffunktion**: Sie ändert nichts, sie schaltet nichts frei.
Die Urteile sind als reine Funktion über eine eingesammelte Lage gebaut
(`api/betriebsbereitschaft.go`) und damit vollständig testbar, ohne
Umgebungsvariablen zu verbiegen oder eine Datenbank zu brauchen; das Zusammentragen der
Lage steht daneben im Handler.

---

## 16. Öffentliche Seiten (ohne Anmeldung)

Drei Adressen sind bewusst **vor** dem Login-Zweig eingehängt (`App.svelte`): Sie rendern, ohne
dass jemand angemeldet ist, und sie liefern ausschließlich Titeldaten — nie Ausleiher, nie
Namen, nie Klassen (Nachweis: `docs/PII_MATRIX.de.md`, Zeilen zu `/api/public/opac/suche`,
`/api/monitor/slides`, `/api/public/bestellung/*`). Keine der drei Seiten hat einen Menüpunkt; ihre
Adressen stehen seit 30.08.2026 unter _Einstellungen → Erreichbarkeit & Alarme_ zum Kopieren.

### 16.1 Katalog (OPAC) — `/katalog`

Für Schülerinnen, Schüler, Eltern und Kollegium: Suche nach Titel, Autor oder ISBN
(Volltext über `search_vector` **oder** Teilstring, `api/opac.go`), Trefferkarte mit Cover und
„N von M verfügbar". Gezählt werden nur ausleihbare, nicht ausgesonderte Exemplare ohne offene
Ausleihe. Welche Titel überhaupt erscheinen, regelt die **gemeinsame Sichtbarkeitsregel der
öffentlichen Seiten** (`repository.OeffentlichSichtbar`, seit 30.08.2026 für Katalog und Monitor
dieselbe): kein Lernmittel (`buecher_titel.ist_lernmittel`, seit Migration 093 ein Feld statt
eines „LMF"-Präfixes in Titel oder Signatur) und mindestens ein Exemplar im Haus (nicht
ausgesondert, nicht nur bestellt). Maximal 50 Treffer.
Das Suchfeld ist scannertauglich (Enter löst die Suche aus). Das Kollegiums-Portal benutzt für
seine Suche denselben Endpunkt (§12, Rolle Kollegium).

### 16.2 Bibliotheks-Monitor — `/monitor`

Für einen Bildschirm im Flur oder in der Bibliothek: eine Endlos-Slideshow aus drei Folien
(`repository/oeffentlich.go` baut die Folien, `api/monitor.go` liefert sie aus, `Monitor.svelte`
zeigt sie). Auf jede Folie kommt nur, was die gemeinsame Sichtbarkeitsregel der öffentlichen
Seiten (§16.1) zulässt — **kein Lernmittel, mindestens ein Exemplar im Haus**. Bis zum
30.08.2026 fehlte diese Regel auf dem Monitor: Auf einem Bestand, der zum größten Teil aus
Schulbüchern besteht, wäre das Mathebuch der 7 „Buch des Monats" gewesen.

| Folie                   | Regel                                                                                                                                                               |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Buch des Monats**     | der Titel **mit Cover**, den in den letzten **30 Tagen** die meisten Schülerinnen und Schüler geliehen haben; gibt es keinen, der zuletzt angelegte Titel mit Cover |
| **Neu eingetroffen**    | die zehn zuletzt angelegten Titel mit Cover                                                                                                                         |
| **Beliebt diese Woche** | die fünf Titel mit den meisten Leserinnen und Lesern der letzten **7 Tage** — auch ohne Cover (Platzhalter-Kachel)                                                  |

Gezählt werden **Leser, nicht Exemplare**: Schüler-Ausleihen, je Schüler einmal. Lehrer-Ausleihen
zählen nicht — ein Klassensatz an eine Lehrkraft sind 30 Ausleihzeilen und null freiwillige Leser;
er hätte sonst beide Folien beherrscht. Wer denselben Titel zweimal leiht, ist ein Leser.

Auf „Buch des Monats" und „Neu eingetroffen" kommen Titel ohne Cover bewusst nicht vor — eine
Folie ohne Bild ist auf einem Flurbildschirm wertlos. „Beliebt" zeigt sie mit Platzhalter, sonst
wäre die Liste kürzer, ohne dass jemand etwas gewinnt. Die Seite braucht keine Bedienung und
keine Anmeldung; sie hat nur einen einzigen Lese-Endpunkt (`GET /api/monitor/slides`, öffentlich,
siehe SECURITY.md). Katalog und Monitor urteilen über denselben Titel gleich — das hält
`api/monitor_pg_test.go` fest.

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
mit, wenn jemand den Stecker zieht. Die Theke lädt sich NICHT von selbst neu (sie darf sich nie
mitten in einer Ausleihe erneuern). Bis zum 30.08.2026 lud die Seite genau einmal beim Start — ein
Monitor, der vor dem Server bootete, zeigte „Lade Daten …" bis zum nächsten Neustart.

### 16.3 Bestellbestätigung durch den Händler — `/bestellung/<token>`

Der Link aus der Bestellmail (§7). Der Händler sieht die Positionen, druckt bei Bedarf die
Etiketten und bestätigt; der Token steht im Pfad und wird deshalb nie protokolliert (Fund
„Geheimnis im Pfad landet im Log"). Sicherheitszuschnitt in SECURITY.md.

---

## 17. Einstellungen — die 15 Kategorien

Erreichbar unter _System → Einstellungen_ (`/einstellungen`, `components/settings/kategorien.js`).
Jede Kategorie wird **einzeln** gespeichert (PATCH nur der Felder dieser Kategorie; ein nicht
gesendetes Feld bleibt unangetastet — Prinzip „nil = unverändert"). `/lmf-aktionen` leitet auf
die Kategorie _LMF-Aktionen_ um.

| #   | Kategorie                   | Inhalt                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Recht                                                           |
| --- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| 1   | **Schule**                  | Name, Anschrift und zwei Eigentumsvermerke fürs Etikett: Lernmittel (leer = Vorgabe) und Schülerbücherei, für Bücher aus Mitteln des Schulträgers (leer = kein Vermerk)                                                                                                                                                                                                                                                                                                                                                                                                                                                             | `manage_settings`                                               |
| 2   | **Ausleihe & Fristen**      | Tage je Buch / je Medium, Ausleihlimit je Schüler, LMF-Stichtag (`MM-TT`, Vorgabe 07-31), Eingangsjahrgänge (`lmf_eingangsjahrgaenge`, Vorgabe „5, 7"), Ferien-Leseclub                                                                                                                                                                                                                                                                                                                                                                                                                                                             | `manage_settings`                                               |
| 3   | **Mahnwesen**               | automatische Ausleihsperre: ab wie vielen überfälligen Medien und nach wie vielen Toleranztagen (`max_overdue_items`, `max_overdue_days`)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | `manage_settings`                                               |
| 4   | **Mahnwesen-Routing**       | Klasse → E-Mail der Klassenleitung (`klassen_lehrer_mapping`); Empfänger für Sammel-Mahnlauf und Abgänger-Kontoauszüge; wird beim Schuljahreswechsel mit versetzt                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `manage_settings`                                               |
| 5   | **Bestellwesen**            | Bedarfswarnung an/aus, Bedarfsschwelle (Titel mit weniger nicht ausgesonderten Exemplaren gelten als Bedarf), Preise erfassen                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | `manage_settings`                                               |
| 6   | **Lieferanten**             | Händler mit E-Mail und Kundennummer; genau **einer** ist Hauptlieferant (Migration 066) und wird im Bestellformular vorausgewählt                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `create_orders`                                                 |
| 7   | **Schlagworte**             | Pflege der Schlagworte am Titel (§13): umbenennen und zusammenführen (alle Titel ändern sich mit, die alte Schreibweise wahlweise als Verweis), Verweise anlegen, löschen — auch mehrere auf einmal —, Filter-Markierung fürs Portal                                                                                                                                                                                                                                                                                                                                                                                                | `edit_books`                                                    |
| 8   | **Schadensersatz**          | Angaben für den Schadensersatz-Bescheid: Nummer des Schulamtsbereichs und Schulnummer (Teil der Referenznummer), Aufsichtsbehörde (Name und Anschrift), Name der Schulleitung, Geschäftszeichen, Bearbeiter, Durchwahl (freiwillig), Zahlstelle und Bankverbindung (vorbelegt aus dem Musterschreiben), Zahlungsfrist in Tagen ab Briefdatum (Vorgabe 28); Schalter **„Immer mit dem Einkaufspreis rechnen"** (`ersatzwert_immer_kaufpreis`): aus gilt ab dem zweiten Verleihjahr der Listenpreis (Arbeitshilfe des Landes), ohne erfassten Listenpreis ersatzweise der Einkaufspreis; an gilt immer der Einkaufspreis              | `manage_settings`                                               |
| 9   | **Datenschutz & Sitzung**   | Löschfristen (Lesehistorie 90 Tage, Lernmittel-Historie 730 Tage, Anliegen 365 Tage, Audit 24 Monate), Abgänger-Karenzzeit (Vorgabe 90 Tage, §8.2), Theke leeren nach _n_ Minuten (Vorgabe 5), Sperrbildschirm nach _n_ Minuten (Vorgabe 15)                                                                                                                                                                                                                                                                                                                                                                                        | `manage_settings`                                               |
| 10  | **Erreichbarkeit & Alarme** | öffentliche Adresse (Basis für Bestätigungs-Link, Katalog- und Monitor-Adresse; leer = keine Links), Alarm-Empfänger (leer = alle aktiven Admins)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `manage_settings`                                               |
| 11  | **Mail**                    | SMTP-Postausgang mit Verbindungstest, Test-Mail; **Mail-Vorlagen** `MAHNUNG_ELTERN` (gedruckter Elternbrief: `{{.Vorname}} {{.Nachname}} {{.BuchListe}} {{.Frist}}`) und `BESTELLUNG_HAENDLER` (Bestellmail: `{{.Datum}} {{.Kundennummer}} {{.AnzahlTitel}} {{.AnzahlExemplare}} {{.BestaetigungsLink}}`) — Platzhalter je Typ, Gate `api/mail_vorlagen_platzhalter_test.go` (Migrationen 023, 052; die tote Vorlage `BESTELLUNG_EINGETROFFEN` ist mit 092 ausgetragen, Abholbereit läuft über den Abholfach-Hinweis am Terminal). Die SMTP-Daten aus der `.env` werden nur beim ersten Start übernommen; danach gilt die Datenbank | `manage_settings`                                               |
| 12  | **LMF-Aktionen**            | Massenverlängerung: alle offenen Lernmittel-Ausleihen **einer Klasse** auf ein neues Rückgabedatum (`POST /api/ausleihen/global-extend-lmf`), mit Rückfrage vor dem Ausführen                                                                                                                                                                                                                                                                                                                                                                                                                                                       | `edit_books`                                                    |
| 13  | **Datenverwaltung**         | Katalog-Import (Littera, CSV/XLSX), Finaler Bestands-Import (Kombi-CSV), Listenimport (ISBN + Stückzahl, §13), Cover-Synchronisation für fehlende Cover, Daten exportieren (Katalog als CSV), **Offline-Sicherungen einspielen** (siehe §18.4)                                                                                                                                                                                                                                                                                                                                                                                      | `manage_inventory` / `edit_books`                               |
| 14  | **LUSD & Versetzung**       | LUSD-Abgleich (§8), Klassen-Versetzung zum Schuljahresende und die Sommerferien für den LMF-Plan (§2.3); hieß bis 06.09.2026 „Schuljahreswechsel" — derselbe Name wie der LMF-Plan im System-Menü, deshalb umbenannt (Rechte: import_students, manage_students_admin bzw. für die Sommerferien manage_settings)                                                                                                                                                                                                                                                                                                                     | `import_students` / `manage_students_admin` / `manage_settings` |
| 15  | **Betriebsbereitschaft**    | Selbstprüfung (§15)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 | `manage_settings`                                               |

**Ferien-Leseclub (Kategorie Ausleihe & Fristen):** Ist er aktiv und ein Zieldatum gesetzt, bekommen alle
Ausleihen bis dahin dieses feste Rückgabedatum (`FerienLeseclubZieldatum`, `loan_rules.go`)
statt der rollierenden Frist — der Schalter für „Bücher über die Sommerferien mitnehmen".

---

## 18. Theke: Ergänzungen zur Omnibox (§1)

### 18.1 Kamera als Barcode-Scanner

Der Handscanner ist der Normalfall. Zusätzlich kann die Kamera des Geräts scannen
(`CameraScanner.svelte`, Bibliothek `html5-qrcode`, wird erst beim Einschalten geladen) —
gedacht für Laptops oder Tablets ohne Scanner. Ein-/Ausschalter neben dem Omnibox-Feld.

### 18.2 Passbild per Webcam

In der Theke lässt sich ein Passbild aufnehmen (`WebcamCapture.svelte`,
`POST /api/schueler/{id}/photo`). Das Bild wird **verschlüsselt** gespeichert
(`schueler_fotos.foto_encrypted`, AES) und nur angemeldeten Benutzern ausgeliefert.

### 18.3 Theke leeren und Sperrbildschirm

Nach `theke_leeren_minuten` ohne Eingabe (Vorgabe 5) schließt sich die offene Schülerakte,
nach `sperre_minuten` (Vorgabe 15) erscheint der Sperrbildschirm — die Anwendung ist dann nicht
mehr im DOM; entsperrt wird mit dem Passwort, nicht mit Maus oder Tastatur. Beides ist
Datenschutz an einem Tresen, an dem Schüler mitlesen können (Kategorie Datenschutz & Sitzung).

### 18.4 Die Theke ohne Verbindung

Fällt die Verbindung aus, geht der Betrieb an der Theke weiter. Was dabei geschieht:

**Scannen.** Jeder Scan wird auf dem Theken-Rechner eingeordnet, wie es sonst der Server tut
(`scanEinordnen.js`, der Zwilling von `omnibox_service.go`): `B-…` und `LMF-…` sind Bücher,
`A-…`/`S-…`/`L-…` sind Ausweise, `G-…` sind Geräte. Eine nackte Ziffernfolge entscheidet die
Buch-Barcode-Liste, die der Rechner bei jeder Anmeldung holt (`GET /api/action/buchbarcodes`,
nur Buchnummern, keine Personendaten); ein Littera-Etikett wird dafür aus der EAN-13
zurückgerechnet. Steht die Nummer nicht auf der Liste, gilt der Scan als **unklar** und wird
nicht gebucht — wer hier riete, schriebe das nächste Buch einer fremden Person zu.

**Personen.** Ein ohne Netz gescannter Ausweis wird als NUMMER gemerkt; die folgenden Bücher
tragen sie, der Server löst sie beim Nachbuchen auf. Personendaten liegen bewusst nicht auf dem
Rechner — eine Namenssuche gibt es ohne Netz deshalb nicht. Ein unklarer Scan lässt den Merker
fallen: Ab dann wird nichts mehr zugeordnet, bis ein eindeutiger Ausweis kommt.

**Warteschlange.** Die Scans liegen in einer lokalen IndexedDB (`offlineQueue.js`) mit
Scan-Zeitpunkt und Idempotenz-Schlüssel. Ein Band oben zeigt den Zustand; die Theke wird dabei
nicht verdeckt, und ohne Netz sperrt auch der Inaktivitäts-Wächter nicht (aufgeschlossen würde
gegen den Schul-Mailserver, und der ist dann nicht erreichbar). Geleert wird die Theke weiter.

**Nachbuchen.** Kommt die Verbindung zurück, schickt der Rechner die Warteschlange in Portionen
an `POST /api/action/nachbuchen`. Der Server bucht die Wirklichkeit: Lag das Buch bei jemand
anderem, nimmt er es dort zurück und leiht es neu aus; eine Rückgabe hinter einer neueren
Buchung weist er ab. Gebucht wird mit dem Scan-Zeitpunkt (höchstens der Serverzeit), und jeder
Schlüssel genau einmal.

**Meldungen.** Jede Abweichung vom Scan hält der Server fest (Migration 117). An der Theke steht
dafür am Band ein Knopf mit der Zahl der offenen Meldungen; dahinter die Liste mit Zeitpunkt,
Buch, Person, Ergebnis und Grund, quittierbar mit „Erledigt". Die Zahl gehört jeder Theken-Rolle
(`perform_actions`), die Liste verlangt `view_students` — sie nennt Namen und Klassen. Offene
Meldungen erscheinen nach 14 Tagen als Warnung in der Betriebsbereitschaft; quittierte werden
nach der Lesehistorie-Frist gelöscht, spätestens nach 30 Tagen.

**Sicherung.** Bleibt ein Arbeitsplatz dauerhaft ohne Verbindung, lässt sich seine Warteschlange
als Datei sichern und an einem anderen Rechner unter _Datenverwaltung → Offline-Sicherungen
einspielen_ nachbuchen. Der Idempotenz-Schlüssel wandert mit: Dieselbe Datei zweimal eingespielt
bucht nichts doppelt.

Nicht im Umfang: Anmelden ohne Server, Schülerdaten auf dem Rechner, Geräteausgabe ohne Netz.
Was davon noch offen ist, steht in [OFFEN.md](OFFEN.md), Abschnitt 2.

### 18.5 Selbstanmeldung fürs Kollegium

Wer sich mit einem Postfach der Schuldomäne (`SELBSTANMELDUNG_DOMAIN`) anmeldet und noch kein
Konto hat, bekommt einen **inaktiven** Eintrag; anmelden kann er sich damit nicht. Die
Anfrage erscheint unter _Benutzer & Rechte → Zugangsanfragen_ und wird dort freigeschaltet
(`auth/selbstanmeldung.go`). Ohne diesen Weg müssten ~160 Lehrkräfte vorab von Hand angelegt
werden — was niemand tut.

---

## 19. Barrierefreiheit

Die Anwendung ist auf WCAG 2.1 AA gebaut; geprüft wird das mit zwei Browser-Gates.

**Was geprüft wird.** Zwei Browser-Gates, beide vor dem ersten Fix rot gesehen (17 von 19
bzw. 4 von 4 Tests, 09.09.2026), seither grün:

- **`e2e/barrierefreiheit-axe.spec.js`:** axe-core mit den Regeln WCAG 2.0/2.1 A und AA über
  Anmeldung, Katalog, jede Monitor-Folie und alle 15 internen Hauptansichten — jeweils im
  **Anfangszustand** der Ansicht; Reiter, Dialoge und Unteransichten (etwa die Reiter des
  Bestellwesens) werden nicht mitgescannt. Dazu `<html lang="de">`, genau EIN `<main>`, eine
  Überschrift je Seite, Skip-Link als erstes Fokusziel. Gescannt wird mit „Bewegung
  reduzieren", weil axe mitten in einer Einblendung gemischte Farben liest; der Monitor
  wird auf jeder Folie gescannt (Takt 15 s).
- **`e2e/barrierefreiheit-dialog.spec.js`:** ein Dialog (Schüler anlegen) nimmt den Fokus,
  hält ihn über 25 Tab und 5 Shift+Tab und gibt ihn bei Escape an den Auslöser zurück;
  Tabellen auf drei Seiten tragen Beschriftung und `scope`; `prefers-reduced-motion`
  setzt Animationen auf 0 s.

Keine dieser Prüfungen ersetzt einen Durchgang mit einem Screenreader (NVDA, VoiceOver);
der hat nicht stattgefunden.

**Was gebaut ist (09.09.2026).** Die Fixes sitzen in Bauteilen, nicht in Seiten:

- `ui/fokusFalle.js` in `Modal.svelte` (24 Verwender), im Sperrbildschirm und in den zwei
  Theken-Alarmen (`OmniboxBlockAlert`, `OmniboxVormerkungAlert`, als `alertdialog`); die
  Vorlagen-Galerie des Designers ist ein Popover ohne Falle.
- `ui/Tabelle.svelte`: `beschriftung` ist Pflicht (Ratsche `frontend-hygiene-tabellen`, 26
  Tabellen), `scope` setzt das Bauteil; die sr-only-Tabelle des Trend-Diagramms trägt eine
  `caption`.
- `layout/SkipLink.svelte`, `layout/Hauptbereich.svelte` (das eine `<main>` mit unsichtbarer
  `h1` aus dem Menü — sichtbare Seitenköpfe bleiben abgeschafft), `basis.css` (Bewegung).
- Rahmen von Feld, Select und Suchfeld in der M3-Rolle `outline` (#72777f auf Weiß 4,5:1)
  statt `outline-variant` (#c2c7cf, 1,7:1; WCAG 1.4.11 verlangt 3:1). Sidebar-Gruppentitel
  ohne 70 %-Deckung; Monitor-Folien und der inaktive Schritt im Druck-Center ohne die
  Theme-Töne `slate-400/500`, die dort Dunkelgrau sind.
- Toasts pausieren unter Maus und Fokus (2.2.1), Fehler-Toasts, Snackbar-Fehler und
  Ladefehler sind `role="alert"`; die Omnibox führt per `aria-activedescendant` durch die
  Trefferliste.
- Zeile und Kachel sind keine Knöpfe mehr: In Leserdatei und Signaturen-Regal öffnet der
  **Name bzw. Titel** die Akte, im Medienkatalog der Titel (die Kachelfläche bleibt ein
  Mausziel ohne Rolle).

**Bekannte Lücken (offen, keine davon im Gate versteckt):**

- Der Ausweis-Designer (Zeichenfläche `designer/CanvasArea`) ist Maus- und Touch-Arbeit ohne
  Tastaturweg.
- Die PDFs (maroto/gofpdf) sind ungetaggt — HTML-Druckweg oder begründete Ausnahme, offen
  ([OFFEN.md](OFFEN.md) 8.6).
- Nicht gescannte Zustände (Reiter, Dialoge, Unteransichten) können weitere Verstöße tragen;
  gemessen sind sie nicht.

Regel für neue Oberfläche: Ein interaktiver Container (Zeile, Kachel) enthält keine weiteren
Bedienelemente — der Name oder Titel ist der Knopf. Und `slate-*` ist im Theme auf
M3-Neutraltöne gelegt; `slate-400` ist Dunkelgrau, auf dunklem Grund unlesbar.
