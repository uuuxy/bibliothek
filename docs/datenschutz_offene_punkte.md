# Datenschutz — offene Punkte (Stand 21.08.2026)

Ergebnis der datenschutzrechtlichen Bewertung gegen den **hessischen** Rahmen. Dieses
Dokument hält fest, was **noch zu tun** ist — nicht, was schon gebaut ist (das steht in
[SECURITY.md](SECURITY.md) und [PII_MATRIX.de.md](PII_MATRIX.de.md)).

**Betriebsannahme:** Das System läuft auf einem **schuleigenen Server in der Schule**
(on-prem). Der Hetzner-Server `flasch3.herzog-dupont.de` ist ausschließlich Test und
trägt nie echte Schülerdaten.

## Rechtsrahmen (gelesen, nicht nacherzählt)

| Quelle | Was daraus folgt |
|---|---|
| **SchDSV** (Schul-Datenschutzverordnung, 1.12.2023, ABl. S. 763) § 2 + Anlage 1 | Erlaubter Datenkatalog: Anschrift (1.4), Geburtsdatum (1.7), Eltern-E-Mail (1.14), Schüler-ID (1.20). **Kein Foto.** |
| SchDSV § 3 | Alles außerhalb Anlage 1 braucht eine dokumentierte **Einwilligung**. |
| SchDSV § 5 | Schule führt VVT (Art. 30), schließt AVV bei Fremdverarbeitung, hält Datenschutzhinweise aktuell, löscht regelmäßig, meldet Vorfälle an Schulamt/Schulträger. |
| SchDSV § 6 | TOMs nach Art. 25/32, BSI-Grundschutz beachten, **IT-Sicherheitskonzept im Benehmen mit dem Schulträger**. |
| SchDSV § 13 | Schülerausweis = Teil-B-Datensatz (Name, Vorname, Anschrift, Geburtsdatum, Schulen, Schüler-ID); Näheres per Erlass. |
| SchDSV § 15 | Eltern-E-Mail nur für erforderliche Schulkommunikation; Weitergabe an Dritte nur mit Einwilligung. |
| SchDSV § 17 + Anlage 3 | Löschen, sobald Zweck erreicht. |
| **§ 83a HSchG Nr. 2** | Schule darf eine digitale Anwendung selbst einführen, wenn sie als Verantwortliche Datenschutz und Sicherheit gewährleistet. |
| **HBDI-Muster-VVT „Schulbibliothek"** | Datenumfang: Name, Klasse, ggf. Anschrift, Ausleihdaten. Rechtsgrundlage: Einwilligung (freiwillige Ausleihe). Löschfrist: „unverzüglich, wenn nicht mehr notwendig". |
| **HKM-Leitfaden Lernmittelfreiheit** | 11.3 Bestandskartei weist Ausleihe **und** Rücklauf nach; 12.3–12.7 Schadensersatz ist öffentlich-rechtlich (Leistungsbescheid der Schulaufsicht); 18: Littera zulässig, Datenschutz beachten. |

Links: [SchDSV Volltext](https://www.glb-hessen.de/wp-content/uploads/2024/01/DLHRatgeberAktuell_SchDSV_NEU.pdf) ·
[HBDI-Muster Schulbibliothek](https://datenschutz.hessen.de/sites/datenschutz.hessen.de/files/2024-02/verfahrensverzeichnis_schulbibliothek_v1_0.pdf) ·
[HKM-Leitfaden LMF](https://kultus.hessen.de/sites/kultus.hessen.de/files/2021-06/lernmittelfreiheit_in_hessen_-_leitfaden_fuer_das_verfahren.pdf) ·
[§ 83a HSchG](https://gesetze.co/HE/HSchG/83a)

## A. Im Code zu erledigen

| # | Punkt | Beleg | Status |
|---|---|---|---|
| A1 | **Lesehistorie befristen.** Zurückgegebene Ausleihen behalten `schueler_id` bis zur Schüler-Löschung; die Titel-Historie zeigt bis zu 200 Entleiher mit Namen. Vorschlag: Schülerbücherei-Ausleihen N Tage nach Rückgabe vom Schüler trennen; Lernmittel nach Schuljahresende + Frist (Schadensersatz, Bestandskartei). Mit rot gesehenem PG-Gate. | `jobs/cron.go` (kein Job trennt), `api/copy_admin.go:246`, einziges `schueler_id = NULL` in `repository/audit_users.go:169` | offen |
| A2 | **Rechtsgrundlage in SECURITY.md korrigieren.** Dort steht Art. 6 (1) lit. c + lit. b. Richtig: Lernmittel → Art. 6 (1) e i. V. m. § 83/§ 153 HSchG + DVO-LMF (öffentlich-rechtliches Nutzungsverhältnis); Schülerbücherei → Einwilligung (HBDI-Muster). Zwei Verarbeitungstätigkeiten. | `docs/SECURITY.md:428` | offen |
| A3 | **Eltern-E-Mail-Aussage an den Code anpassen.** SECURITY.md sagt „E-Mail für Mahnungen"; tatsächlich: `mailto:`-Link im Profil, Hinweis-Icon im Mahnwesen, Vorlage `MAHNUNG_ELTERN` nur für den **gedruckten** Elternbrief. Mahn-Mails gehen an Klassenleitungen. Entscheiden: Eltern-Mahnmail bauen oder Doku angleichen. | `api/reports_pdf.go:40`, `api/mahnwesen_bulk_mail.go:320`, `StudentProfileStammdaten.svelte:65` | offen |
| A4 | **Theken-Ansicht nach Inaktivität leeren + Sperrbildschirm.** Kein Auto-Leeren vorhanden; JWT 12 h ohne Idle-Lock. Der nächste Schüler an der Theke sieht sonst den vorigen. | `main.go:361`, Omnibox/StudentProfileCard | offen |
| A5 | **Unverschlüsselte Dumps** von `update.sh` (30 Tage) und `scripts/backup.sh` (7 Tage) verschlüsseln oder Frist kürzen. | `docs/DEPLOYMENT.md:281` | offen |
| A6 | `SENTRY_DSN` in Produktion **leer lassen** (sonst Fehlerreports in die USA) oder EU-Instanz; S3-Offsite nur EU/Schulträger. | `.env.example:138`, `jobs/backup.go:255` | Betriebsregel |

## B. Von Schule / DSB / Schulträger zu erledigen (Dokumente kann das Projekt liefern)

| # | Punkt |
|---|---|
| B1 | **VVT** nach HBDI-Muster — zwei Tätigkeiten: (1) Lernmittelausleihe, (2) Schülerbücherei. TOM-Seite aus SECURITY.md/PII_MATRIX füllen. |
| B2 | **Datenschutzhinweis** (Art. 13) — Muster sieht „bei Anmeldung zur Nutzung der Bibliothek" vor; für Lernmittel gehört er in die Schulaufnahme-Information (§ 5 (2) SchDSV). |
| B3 | **Foto auf dem Schülerausweis**: Anlage 1 kennt kein Foto. Entweder deckt es der Schülerausweis-Erlass (prüfen) oder es braucht eine Einwilligung (§ 3). Vor dem ersten Ausweisdruck klären. |
| B4 | **Schulischer DSB** beteiligen; Schwellwertanalyse DSFA dokumentieren (Minderjährige, Fotos — Ergebnis vermutlich „nicht erforderlich", aber schriftlich). |
| B5 | **IT-Sicherheitskonzept im Benehmen mit dem Schulträger** (§ 6 (3)); **Netzplatzierung** klären: Schülerdaten ins Verwaltungsnetz, Theke/Kiosk und Lehrer-Browser sitzen meist im pädagogischen Netz. |
| B6 | **Rolle des Entwicklers/Wartenden**: Wer Admin-Zugang hat und Schülerdaten sehen kann, braucht eine Regelung — als Lehrkraft dienstlich, als Externer AVV/Wartungsvertrag. |
| B7 | **Löschkonzept gegenüber Littera**: „Littera hatte die Daten auch" ist keine Rechtsgrundlage. Beim Übergang nur übernehmen, was aktuell ist und einen Zweck hat. |

## C. Bewusst KEIN Befund

- Geburtsdatum: Anlage 1 (1.7), Zweck Dubletten-Wachhund + LUSD-Abgleich → im VVT begründen.
- Anschrift: „ggf. Anschrift" im HBDI-Muster; Zweck gedruckte Rechnung/Elternbrief.
- Rollen ab Werk: nur ADMIN/MITARBEITER sehen Stufe 3 (Adresse, Eltern-Mail, Foto); KOLLEGIUM/HELFER nicht.
- IP-Adressen werden nicht geloggt; Fotos und Backups verschlüsselt; Auskunft Art. 15 vorhanden; Statistik ohne Personenbezug.
