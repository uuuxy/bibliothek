# Runbook Bibliothek — Erste Hilfe fürs Sekretariat

> Zielgruppe: Ausleih-Pult, ohne Technik-Vorwissen. Bei allem, was hier nicht
> steht oder nicht klappt: **Eskalation ganz unten**. Stand: 11.07.2026.

---

## 1. Tagesgeschäft

### Ausleihe
1. **Schülerausweis scannen** → das Konto des Schülers öffnet sich.
   - Funktioniert mit den **alten Littera-Ausweisen** genauso wie mit neuen.
2. **Buch scannen** → fertig, das Buch ist ausgeliehen.
   - Schulbücher (LMF) laufen automatisch bis zum Schuljahres-Stichtag (31.07.),
     normale Bücher bekommen die Standard-Leihfrist. Nichts einstellen.
3. Nächster Schüler: einfach dessen Ausweis scannen — die alte Sitzung schließt sich.

### Rückgabe
- **Einfach das Buch scannen** (ohne vorher einen Ausweis zu scannen).
  Das System bucht es beim richtigen Schüler aus — egal, wer es abgibt.
- Wenn gerade ein Schülerkonto offen ist und ein *fremdes* Buch gescannt wird:
  Der erste Scan gibt es beim Vorbesitzer zurück (Info-Banner erscheint),
  ein **zweiter Scan** leiht es an den aktuellen Schüler aus.

### Ausweis vergessen
- Namen des Schülers ins Scan-/Suchfeld **tippen** → Schüler aus der Liste wählen.
  Danach normal Bücher scannen.

### Gesperrter Schüler
- Die Sperre meldet sich **beim Buch-Scan** mit einem roten Hinweis.
- Ausnahme im Einzelfall: Knopf **„Einmalig ignorieren"** — bitte sparsam nutzen,
  der Vorgang wird protokolliert.

---

## 2. Sonderfälle

| Situation | Vorgehen |
|---|---|
| **Buch beschädigt abgegeben** | Buch in der Buchansicht öffnen → **Schadensfall melden**. Die Ausleihe endet, eine Forderung wird angelegt. Beschädigte Bücher können nicht gelöscht werden, solange die Forderung offen ist. |
| **Neues Buch erfassen** | Inventar → Neues Buch → **ISBN scannen**. Titel, Autor, Cover und meist ein Signatur-Vorschlag kommen automatisch (DNB). **Signatur ist Pflicht** — sie steht auf dem Rücken-Etikett. Findet die ISBN nichts: Felder von Hand ausfüllen. |
| **Etiketten drucken** | Im Buch-Formular → **„Barcodes drucken"** (A4-Zweckform-Bogen). |
| **Klassensatz-Anfrage einer Lehrkraft** | Lehrkräfte reservieren selbst im Lehrerportal. Bearbeitung: Bestellungen → **Klassensatz-Reservierungen** → „Abschließen", wenn erledigt. |
| **Mahnungen** | Menü **Mahnwesen**: Liste der Überfälligen pro Klasse, PDF drucken oder per Mail an die Lehrkraft senden. Läuft **nicht automatisch** — einmal pro Woche reinschauen. In den Ferien ist das Mahnwesen automatisch pausiert. |
| **Alle Schulbücher einer Klasse verlängern** | Menü **LMF-Aktionen**: Klasse + neues Datum eintragen → „Global verlängern". Vorsicht: ändert hunderte Ausleihen auf einmal, Bestätigung genau lesen. |
| **Abgänger am Schuljahresende** | Menü **Abgänger**: zeigt nur Schüler, die noch Bücher schulden. **„Laufzettel drucken"** erzeugt die Liste zum Abhaken. Wer nichts mehr schuldet, verschwindet automatisch aus der Liste. |

---

## 3. Störungen

### 🔌 Internet/Netzwerk weg
- **Weiterscannen!** Das System puffert Ausleihen und Rückgaben lokal
  (Offline-Anzeige erscheint) und trägt alles automatisch nach, sobald die
  Verbindung zurück ist. Nichts doppelt scannen.

### 📷 Scanner defekt
- Die Barcode-Nummer steht als Zahl unter dem Strichcode: einfach **per Tastatur
  ins Scan-Feld eintippen** und Enter — verhält sich exakt wie ein Scan.

### 🖥️ Seite reagiert nicht / sieht kaputt aus
1. Browser-Seite neu laden (F5).
2. Abmelden und neu anmelden.
3. Hilft beides nicht → Eskalation (unten).

### 🔴 Rotes Backup-Symbol
- Bedeutet: Die letzte automatische Datensicherung ist zu alt.
- Kein Notfall für den laufenden Betrieb, aber **noch am selben Tag melden**
  (Eskalation unten). Nicht wegklicken und vergessen.

### 📕 ISBN-Suche findet nichts
- Kommt vor (alte Bücher, Schenkungen ohne ISBN). Buch von Hand anlegen,
  Signatur eintragen. Kein Fehler im System.

### 🖨️ Mahn-Mails kommen nicht an
- Prüfen, ob gerade Ferien sind (dann ist das Mahnwesen absichtlich pausiert).
- Sonst: Eskalation — vermutlich Mail-Server-Konfiguration.

---

## 4. Einmal pro Woche (5 Minuten)

- [ ] **Mahnwesen** öffnen, überfällige Klassenlisten drucken/versenden.
- [ ] **Klassensatz-Reservierungen** prüfen (roter Punkt an „Bestellungen").
- [ ] Backup-Symbol kurz anschauen (grün = alles gut).

---

## 5. Eskalation

| Stufe | Wer | Wann |
|---|---|---|
| 1 | Peter Flasch | Alles Technische: Server, rotes Backup, Login-Probleme, Mail-Versand |
| 2 | Seite komplett weg | Erst Stufe 1 versuchen; Server-Neustart macht **nur** Stufe 1 |

**Wichtig:** Niemals selbst am Server neu starten oder Kabel ziehen — die
Offline-Pufferung am Pult überbrückt Ausfälle, Datenverlust droht nur durch
unkoordinierte Neustarts.
