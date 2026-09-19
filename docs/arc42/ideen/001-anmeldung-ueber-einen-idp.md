# 001 — Anmeldung über einen Identitätsanbieter statt IMAP-Bind

Stand: 18.09.2026

**Idee.** Statt Benutzername und Passwort selbst entgegenzunehmen und gegen IMAP zu binden,
die Anmeldung an einen Identitätsanbieter abgeben (SAML oder OIDC). Wo ein Mailserver
steht, steht oft schon ein Verzeichnis dahinter.

**Warum.** Ein Punkt wiegt schwerer als alle anderen und steht so noch nirgends in der
Dokumentation: **Die Anwendung sieht das Passwort im Klartext.** Sie braucht es für den
IMAP-Bind (`auth/imap.go`, `AuthenticateIMAP`), bekommt es also im Anfragerumpf und reicht
es weiter. Dass sie es nicht *speichert* (A2, keine Passwortspalte seit Migration 012), ist
richtig und wenig — sie könnte es. Bei SAML/OIDC bekäme sie es nie zu sehen.

Dazu käme: Abschaltung an einer Stelle statt in zwei Systemen, MFA ohne eigenen Code, und
die Sitzungsdauer läge beim Anbieter statt bei den 12 Stunden aus `main.go`.

**Was dagegen spricht.**

- **Ob es überhaupt einen Anbieter gibt, ist ungeprüft.** Aus dem Code lässt sich das nicht
  beantworten — bekannt ist nur der IMAP-Host. Zu klären, bevor irgendetwas gebaut wird.
- Ein SAML-Dienstanbieter in Go ist kein Nachmittag: Metadaten, Zertifikatswechsel,
  Signaturprüfung, ein Testanbieter für CI. Die Anmeldung ist heute rund 500 Zeilen in
  `auth/handlers.go` samt Ausfallmatrix, die mehrfach am echten Fehler nachgeschärft wurde.
- **Der Notfall wird schlechter:** Fällt der Anbieter aus, kommt niemand mehr herein — heute
  auch, aber der Mailserver ist derselbe Dienst, dessen Ausfall ohnehin alles anhält.
- Die Selbstanmeldung des Kollegiums (`auth/selbstanmeldung.go`) müsste neu gedacht werden:
  Sie hängt an der Domain der Mailadresse.

**Nächster Schritt.** Eine Frage an den Schulträger, keine Zeile Code: Gibt es einen
Identitätsanbieter, und dürfte diese Anwendung ihn nutzen? Lautet die Antwort nein, ist die
Idee erledigt und wird gelöscht.
