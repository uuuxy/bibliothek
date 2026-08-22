# Verzeichnis von Verarbeitungstätigkeiten — Entwurf (Art. 30 DSGVO, § 5 SchDSV)

**Status:** Entwurf aus dem Projekt (22.08.2026), nach dem HBDI-Muster
„Verfahrensverzeichnis Schulbibliothek" gegliedert. **Beschließen und unterschreiben muss
die Schule** (Schulleitung als Verantwortliche, Beteiligung des schulischen
Datenschutzbeauftragten). Eckige Klammern sind von der Schule auszufüllen; alles andere
beschreibt das System, wie es gebaut ist (Belege: [SECURITY.md](../SECURITY.md),
[PII_MATRIX.de.md](../PII_MATRIX.de.md), [FACHKONZEPT.md](../FACHKONZEPT.md)).

Drei Tätigkeiten, weil drei Rechtsgrundlagen: **(1) Lernmittelausleihe** (öffentlich-
rechtlich), **(2) Schülerbücherei** (Einwilligung), **(3) Benutzerkonten und
Protokollierung des Personals** (Beschäftigtendaten). Das HBDI-Muster kennt nur (2); die
Trennung von (1) ist nötig, weil Lernmittel weder freiwillig noch per Einwilligung laufen.

## Verantwortlicher (für alle drei Tätigkeiten)

| | |
|---|---|
| Verantwortliche Stelle | [Name der Schule, Anschrift], vertreten durch die Schulleitung [Name] |
| Kontakt | [Telefon, E-Mail der Schule] |
| Schulischer Datenschutzbeauftragter | [Name, Kontakt] — Pflicht nach § 5 SchDSV, Beteiligung vor Inbetriebnahme |
| Schulträger | [Name] — IT-Sicherheitskonzept im Benehmen mit dem Schulträger (§ 6 Abs. 3 SchDSV) |
| Betrieb | Schuleigener Server im Schulnetz (on-prem), kein Cloud-Hosting. Reverse-Proxy (Caddy, TLS), Go-Backend, PostgreSQL, Docker. |
| Auftragsverarbeiter | **keine**, solange der Server von der Schule selbst betrieben wird. Wartet eine externe Person (Entwickler) mit Admin-Zugang, braucht es eine Vereinbarung nach Art. 28 DSGVO (siehe [offene Punkte B6](../datenschutz_offene_punkte.md)). |
| Datenschutz-Folgenabschätzung | Schwellwertanalyse dokumentieren (Minderjährige, Fotos). Voraussichtlich nicht erforderlich: keine Profilbildung, kein Scoring, keine systematische Überwachung; Lesehistorie befristet. Ergebnis schriftlich festhalten (B4). |

---

## Tätigkeit 1 — Lernmittelausleihe (Lernmittelfreiheit)

| Feld | Inhalt |
|---|---|
| **Bezeichnung** | Verwaltung der Ausleihe und Rückgabe von Lernmitteln (Schulbücher) im Rahmen der Lernmittelfreiheit, einschließlich Mahnung und Schadensersatz. |
| **Zweck** | Nachweis, welches Exemplar wann an welche Schülerin/welchen Schüler ausgegeben und wann zurückgenommen wurde (Bestandskartei, HKM-Leitfaden Lernmittelfreiheit 11.3); Erinnerung und Mahnung bei fehlender Rückgabe; Feststellung von Verlust/Beschädigung als Grundlage eines Schadensersatzverfahrens (Leitfaden 12.3–12.7, Leistungsbescheid durch die Schulaufsicht); Bestandsplanung (Bestellbedarf ohne Personenbezug). |
| **Rechtsgrundlage** | Art. 6 Abs. 1 lit. e DSGVO i. V. m. § 83 HSchG (Datenverarbeitung durch Schulen) und § 153 HSchG (Lernmittelfreiheit) mit der zugehörigen Verordnung; § 2 SchDSV mit Anlage 1 (Anschrift 1.4, Geburtsdatum 1.7, E-Mail der Eltern 1.14, Schüler-ID 1.20). Öffentlich-rechtliches Nutzungsverhältnis — **kein** Vertrag, **keine** Einwilligung. |
| **Betroffene** | Schülerinnen und Schüler der Schule; Erziehungsberechtigte (Anschrift, E-Mail); Lehrkräfte mit Handapparat-Ausleihe. |
| **Datenkategorien** | Stammdaten: Name, Vorname, Klasse, Geburtsdatum (Zweck: Dublettenschutz und Abgleich mit der LUSD), LUSD-Schüler-ID, interne Ausweis-/Barcode-Nummer, Abgangsjahr. — Kontaktdaten: Anschrift, E-Mail der Eltern (nur für gedruckten Brief bzw. manuelle Kontaktaufnahme; **kein automatischer Mailversand an Eltern**). — Ausleihdaten: Exemplar, Ausleih-, Fristen- und Rückgabedatum, Mahnstufe, Sperrstatus mit Grund. — Schadensfälle: Beschreibung, Betrag, Zahlungsstatus. — **Foto** nur, wenn der Schülerausweis mit Foto eingeführt wird (Rechtsgrundlage gesondert zu klären, B3; AES-256-GCM verschlüsselt gespeichert). |
| **Herkunft** | LUSD-Export der Schule (Klassenliste, CSV/XLSX), manuelle Anlage durch die Bibliothek, Ausleihvorgang am Scanner. |
| **Empfänger** | Intern: Bibliothekspersonal (Rollen Admin/Mitarbeiter), Helfer am Kiosk (nur Name, Klasse, Sperrstatus — Stufe 1), Klassenleitungen (Mahnliste ihrer Klasse per E-Mail an die dienstliche Adresse). Extern: Schulaufsicht nur im Schadensersatzverfahren, durch die Schule außerhalb des Systems. Lieferanten erhalten **keine** Schülerdaten (Bestellungen führen nur Titel/Exemplare). |
| **Drittland** | Keine Übermittlung. Betriebsregel: `SENTRY_DSN` leer lassen (sonst Fehlerberichte in die USA), kein S3-Offsite außerhalb EU/Schulträger. |
| **Löschfristen** | Ausleihvorgang bleibt der Person **730 Tage nach Rückgabe** zugeordnet (Bestandskartei, Schadensersatz über das Folgeschuljahr), danach automatisch getrennt (nächtlicher Job; Frist einstellbar). Bearbeitende Person der Ausleihe nach 14 Tagen entfernt. Schülerdatensatz: beim Abgang (nicht mehr in der LUSD) ab 30. Januar des Folgejahres gelöscht, sofern keine offene Ausleihe / kein unbezahlter Schadensfall; Altfälle nach 360 Tagen anonymisiert (Name, Adresse, Geburtsdatum, LUSD-ID, Eltern-E-Mail, Foto; Spuren in Audit und Vormerkungen getilgt). Protokolle 24 Monate. Verschlüsselte Backups 14 Tage. |
| **TOM** | siehe Anhang. |

## Tätigkeit 2 — Schülerbücherei (freiwillige Ausleihe)

| Feld | Inhalt |
|---|---|
| **Bezeichnung** | Ausleihe von Büchern und Medien der Schülerbücherei, Vormerkungen, Leseförderung (Ferien-Leseclub). |
| **Zweck** | Abwicklung der Ausleihe (wer hat was bis wann), Rückgabeerinnerung/Mahnung, Vormerkung, Ausleihstatistik **ohne Personenbezug** (Titel, Klassenstufen, Zeiträume). |
| **Rechtsgrundlage** | Einwilligung, Art. 6 Abs. 1 lit. a DSGVO (so das HBDI-Muster „Schulbibliothek": freiwillige Nutzung); bei Minderjährigen durch die Erziehungsberechtigten, § 3 SchDSV (dokumentiert, widerruflich). Alternativ vertretbar: Art. 6 Abs. 1 lit. e i. V. m. § 83 HSchG als schulische Aufgabe (Leseförderung) — **Entscheidung der Schule mit DSB**; der Datenschutzhinweis ist für beide Varianten vorbereitet. |
| **Betroffene** | Schülerinnen und Schüler, die die Bücherei nutzen; Erziehungsberechtigte; Lehrkräfte (Handapparat, Klassensätze). |
| **Datenkategorien** | Wie Tätigkeit 1 (derselbe Schülerdatensatz), zusätzlich Vormerkungen (Titel, Datum, Freitext-Notiz) und Klassensatz-Reservierungen durch Lehrkräfte. |
| **Herkunft / Empfänger / Drittland** | wie Tätigkeit 1. |
| **Löschfristen** | Ausleihvorgang bleibt der Person **90 Tage nach Rückgabe** zugeordnet (Nachfragen zu später bemerkten Schäden, Fremdrückgaben), danach automatisch getrennt (HBDI-Muster: „unverzüglich, wenn nicht mehr notwendig"); Vormerkungen mit Erledigung bzw. Anonymisierung gelöscht. Übrige Fristen wie Tätigkeit 1. |
| **Besonderheit** | Bei Widerruf der Einwilligung: Schülerdatensatz kann sofort in den Papierkorb (Soft-Delete) und wird nach 180 Tagen anonymisiert; laufende Lernmittel-Ausleihen (Tätigkeit 1) bleiben davon unberührt — dort gilt keine Einwilligung. |

## Tätigkeit 3 — Benutzerkonten und Protokollierung (Personal)

| Feld | Inhalt |
|---|---|
| **Bezeichnung** | Konten für Lehrkräfte, Bibliothekspersonal und Helfer; Protokollierung administrativer und fachlicher Aktionen (Audit-Trail). |
| **Zweck** | Zugriffskontrolle (4 Rollen: Admin, Mitarbeiter, Kollegium, Helfer), Nachvollziehbarkeit von Änderungen an Schülerdaten und Einstellungen (Art. 5 Abs. 2 Rechenschaft), Klassensatz-Reservierungen und Wünsche der Lehrkräfte. |
| **Rechtsgrundlage** | Art. 6 Abs. 1 lit. e DSGVO i. V. m. § 83 HSchG; für Beschäftigte § 23 HDSIG (Datenverarbeitung im Beschäftigungsverhältnis). |
| **Datenkategorien** | Name, dienstliche E-Mail (Anmeldung gegen den Schul-Mailserver per IMAP — **kein Passwort im System**), Rolle, Ausweis-Barcode, Aktiv-Status; Protokolleinträge (wer hat wann welche Aktion ausgelöst; administrative Eingriffe im `audit_logs` **mit** IP-Adresse des Arbeitsplatzes — Rechenschaft nach Art. 5 Abs. 2, 24 Monate). Handapparat-Ausleihen der Lehrkraft bleiben dauerhaft zugeordnet (dienstlich); Lehrkräfte werden nicht gemahnt. |
| **Empfänger** | Schulleitung/Admin; keine externen. |
| **Löschfristen** | Protokolle 24 Monate (nächtlicher Job). Bearbeiter-ID an Ausleihen nach 14 Tagen entfernt. Konto bei Ausscheiden deaktivieren/löschen (manuell; Pflicht der Schule). |

---

## Anhang — Technische und organisatorische Maßnahmen (Art. 32 DSGVO, § 6 SchDSV)

Belegt in [SECURITY.md](../SECURITY.md); Kurzfassung für das VVT:

| Bereich | Maßnahme |
|---|---|
| Zutritt/Netz | Server in der Schule; nur über Reverse-Proxy (TLS) erreichbar; Schülerdaten gehören ins Verwaltungsnetz — Netzplatzierung von Theke/Kiosk mit dem Schulträger klären (B5). |
| Zugang | Anmeldung gegen den Schul-Mailserver (IMAP), kein eigenes Passwort; Sitzungs-Cookie HttpOnly/SameSite=Strict, 12 h; Brute-Force-Sperre; **Sperrbildschirm nach 15 Minuten Inaktivität, Theken-Ansicht nach 5 Minuten geleert** (einstellbar). |
| Zugriff | Rollen mit Rechte-Matrix (Admin, Mitarbeiter, Kollegium, Helfer); Helfer sehen nur Kiosk-Niveau (Name, Klasse, Sperrstatus), keine Adressen, keine Geburtsdaten, keine Fotos; jede Route ist in der PII-Matrix eingestuft und per Test an das Recht gebunden. |
| Trennung | Statistik und Dashboard ohne Personenbezug (serverseitig aggregiert). |
| Verschlüsselung | Fotos AES-256-GCM in der Datenbank; Backups `pg_dump → gzip → AES-256-GCM` mit scrypt-Schlüsselableitung, 0600, 14 Tage Rotation; Transport TLS. |
| Integrität | Audit-Trail (fachlich + administrativ), append-only per Konvention mit Quelltext-Ratsche; DSGVO-Tilgung und 24-Monats-Aufbewahrung sind die einzigen Schreibtüren. Administrative Einträge tragen die IP-Adresse des Arbeitsplatzes (Schulnetz), fachliche nicht. |
| Verfügbarkeit | Tägliches verschlüsseltes Backup, wöchentliche Restore-Probe, Selbstprüfung der Betriebsbereitschaft mit Kritisch-Alarm per Mail. |
| Datenminimierung/Löschung | Automatische Jobs: Bearbeiter-ID 14 Tage, Lesehistorie 90/730 Tage, Abgänger-Löschung, Anonymisierung 180/360 Tage, Audit 24 Monate. Betroffenenauskunft (Art. 15) als JSON/PDF für Admins. |
| Eingabekontrolle | Jede Änderung an Schülerdaten mit Bearbeiter und Zeit protokolliert; privilegierte Felder (Sperren, Überschreiben) nur mit eigenem Recht. |
| Organisation | Schulischer DSB beteiligt (B4); Rolle des Wartenden geregelt (B6); Datenschutzhinweis Art. 13 ausgegeben (B2); Löschkonzept gegenüber Littera (B7). |

**Offen (Stand 22.08.2026):** `update.sh`/`scripts/backup.sh` legen unverschlüsselte Dumps ab (A5) — Frist kürzen oder verschlüsseln; Foto-Rechtsgrundlage (B3).
