# 002 — Einmal-Link per Mail als zweiter Anmeldeweg

Stand: 18.09.2026

**Idee.** Statt eines Passworts einen einmalig gültigen Link an die Schuladresse schicken.
Wer ihn öffnet, ist angemeldet.

**Warum.** Der Vertrauensanker wäre derselbe wie heute: Beide Verfahren beweisen nur
„diese Person verfügt über dieses Postfach". Der Weg wäre aber kürzer — kein Passwort im
Anfragerumpf (siehe [001](001-anmeldung-ueber-einen-idp.md)), kein IMAP-Bind, keine
Ausfallmatrix zwischen „Mailserver weg" und „Passwort falsch". Der SMTP-Versand ist
ohnehin gebaut und gehärtet (`mailservice/`, STARTTLS erzwungen, Kopfzeilen gegen CR/LF).

**Was dagegen spricht.**

- **An der Theke ist er untauglich.** Der Tresen ist ein Scanfeld; eine Anmeldung, die auf
  eine Mail wartet, hält den Betrieb an. Das Verfahren taugt für seltene Anmeldungen, nicht
  für das Tagesgeschäft — also allenfalls als **zweiter** Weg für das Kollegium, nicht als
  Ersatz.
- Zwei Anmeldewege sind zwei Wege zu demselben Zustand — genau die Bugklasse, gegen die
  dieses Projekt antritt (`CLAUDE.md`, Regel 2). Das müsste den Nutzen aufwiegen.
- Der Token stünde im Pfad oder in der Query und damit im Log. Die Lehre dazu gibt es schon:
  Beim Bestätigungslink für den Händler maskiert `maskiereToken` ihn ausdrücklich, weil
  sonst jeder verschickte Link im Klartext in einem Logfile stünde.
- Gültigkeitsdauer, Einmaligkeit, Wiederverwendung nach dem Zurück-Knopf des Browsers — all
  das ist zu entscheiden und zu testen.

**Nächster Schritt.** Erst [001](001-anmeldung-ueber-einen-idp.md) beantworten. Gibt es
einen Identitätsanbieter, ist diese Idee gegenstandslos; gibt es keinen, ist sie die
billigere Antwort auf dasselbe Problem und verdient einen Entwurf.
