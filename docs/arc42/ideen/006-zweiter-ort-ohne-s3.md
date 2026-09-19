# 006 — Ein zweiter Ort für die Sicherungen, der kein S3 sein muss

Stand: 19.09.2026

**Idee.** `OFFEN.md` 7.3 führt die Auslagerung der Sicherungen als S3-Aufgabe („nur EU oder
Schulträger, der Code ist fertig"). Der zweite Ort muss aber kein Objektspeicher sein —
und die Entscheidung hängt seit dem 13.09.2026 an einer Vertragsfrage, während alles auf
**einer** Platte liegt. Drei einfachere Wege, jeder ohne neue Betriebskomponente:

| Weg | Vorteil | Preis |
| --- | --- | --- |
| `rsync`/`scp` der `.enc`-Dateien auf einen zweiten Rechner in der Schule, per Cron | Kein Konto, kein Anbieter, keine Vertragsfrage. Die Dateien sind ohne Passphrase wertlos, das Ziel muss also nicht besonders geschützt sein | Ein zweiter Rechner muss laufen und gepflegt werden |
| Externe Platte am Server, wöchentlich getauscht | Eine **offline** liegende Kopie überlebt auch einen Verschlüsselungstrojaner — S3 mit hinterlegten Zugangsdaten nicht | Jemand muss die Platte tauschen |
| S3 einschalten (Code fertig, vier Variablen) | Kein Handgriff im Betrieb | Anbieter- und Vertragsfrage, offen seit dem 13.09.2026 |

**Warum.** Die Verschlüsselung ist gelöst, die Rotation ist gelöst, die Probe ist gelöst.
Was fehlt, ist allein eine **zweite Kopie an einem anderen Ort** — und das ist ein
Kopierbefehl, keine Architektur.

**Die `.env` gehört mit dazu, aber getrennt.** Siehe
[review/003](../review/003-restore-jenseits-der-datenbank.md): Ohne sie ist die Sicherung
nicht zu öffnen. Sie darf nur **nicht** unverschlüsselt neben den Sicherungen liegen —
sonst liegt der Schlüssel neben der Tür. Praktikabel: einmal mit `age` oder `gpg -c` unter
einer Passphrase, die im Passwortspeicher der Schule steht; neu kopiert wird nur, wenn die
`.env` sich ändert, und das ist selten.

**Was dagegen spricht.**

- Drei Wege sind drei Wege — gewählt wird **einer**, sonst entstehen zwei Wahrheiten
  darüber, wo die gültige Kopie liegt.
- Ein zweiter Rechner in der Schule ist auch nur ein Gerät im selben Brandabschnitt; gegen
  Wasser, Feuer und Diebstahl hilft erst ein Ort außer Haus.
- Ohne Wiederherstellungs-Probe ist auch die zweite Kopie nur eine Vermutung
  (`OFFEN.md` 7.4).

**Nächster Schritt.** Einen Weg wählen — das ist eine Betriebsentscheidung, keine
Codefrage. Danach wandert die Aufgabe nach `OFFEN.md` (7.3 ergänzen oder ersetzen) und
dieser Eintrag wird gelöscht.
