# 003 — Was könnte weg? Umfang gegen Nutzen prüfen

Stand: 18.09.2026

**Idee.** Das Programm daraufhin durchgehen, was ersatzlos verschwinden kann. Nicht
umbauen — **weglassen**.

**Warum.** 65.883 Zeilen Go im Produktivcode, 206 Routen, 134 Migrationen, 42 Tabellen für
eine Bibliothek mit rund 1.900 Lesern und acht Arbeitsplätzen. Der Verdacht, dass das viel
ist, ist berechtigt genug, um ihn zu prüfen. Die Gegenrechnung steht in
[review/002](../review/002-umfang-gegen-anwendungsfall.md): Der Anwendungsfall ist nicht
„ausleihen und zurücknehmen", sondern trägt ein Dutzend eigener Regelwerke. Beides kann
gleichzeitig wahr sein — deshalb die Prüfung statt eines Urteils.

Konkrete Kandidaten, die heute schon benannt sind:

| Kandidat | Warum er in Frage kommt |
| --- | --- |
| Littera-Übernahme (`internal/littera`, `cmd/littera-altbestand`, 3.041 Zeilen) | Ein Einmal-Werkzeug. Nach abgeschlossener Übernahme ist es Ballast, der mitgetestet und mitgepflegt wird |
| Stapel-Tür `POST /api/action/batch` | Läuft seit A21 nur noch „eine Version länger" mit — zwei Offline-Wege nebeneinander |
| Swagger (`docs/docs.go`, 5.292 generierte Zeilen) | Deckt 63 von 206 Routen; das vollständige Verzeichnis ist ohnehin `api_inventar.md` |
| Geräteausleihe | Eigene Tabelle, eigener Dienst, eigener Sperrpfad — wie viele Geräte sind es wirklich? |
| S3-Offsite-Backup | Gebaut, standardmäßig aus, Wirksamkeit unbelegt |

**Was dagegen spricht.** Löschen ist nicht umsonst: Jede entfernte Funktion, die jemand
doch benutzt, ist ein Betriebsausfall. Und der Umfang liegt zum großen Teil **nicht** in
Funktionen, sondern in der Sorgfalt — Gates, Ratschen, PG-Tests. Die sind der Grund, warum
das System hält; sie einzusparen wäre der falsche Schnitt.

**Nächster Schritt.** Eine einzige Zahl je Kandidat besorgen, bevor diskutiert wird: Wie oft
wurde er im letzten Halbjahr benutzt? Für die Theke beantwortet das der Audit-Trail, für
Routen `api_inventar.md` samt Frontend-Aufrufern. Was nachweislich nie lief, ist die
einfachste Entscheidung.
