# Antwort auf die Sichtung vom 16.09.2026 — Entwurf

**Status:** Entwurf aus dem Projekt (17.09.2026), zum Abschicken durch die Schule. Er beantwortet
das Sichtungsprotokoll vom 16.09.2026 (Teilnehmende CN, SW, HMP; Protokoll HMP) Punkt für Punkt
und stellt zwei Fragen zurück. Der Stand der Technik dahinter steht in
[OFFEN.md](OFFEN.md), Abschnitt 9.

---

Sehr geehrte Damen und Herren,

vielen Dank für die Sichtung am 16.09.2026 und das Protokoll. Die Beobachtungen waren
durchweg berechtigt; der größte Teil ist inzwischen abgestellt. Nachstehend der Stand zu
jedem Punkt, dazu zwei Fragen an Sie und drei Stellen, an denen wir bewusst anders arbeiten
als die Anforderungsliste es beschreibt — mit Begründung und der Bitte um Rückmeldung.

## 1. Stand zu den Beobachtungen

**Zu 1 — Vorgaben des Landes**

- _„Weder Einkaufs- noch Listenpreise, keine Beschädigungsgrade, fehlende
  Restwertberechnung, Handeingabe."_ Abgestellt. Beide Preise sind am Titel bzw. am Exemplar
  hinterlegt; welcher von beiden die Grundlage der Berechnung ist, stellt die Schule ein. Der
  Listenpreis wird beim Anlegen über die Nummer des Buches automatisch ermittelt und ist von
  Hand überschreibbar. Jedes Exemplar kann einen Beschädigungsgrad in Prozent tragen, der den
  Wert mindert. Beim Melden eines Verlusts oder Schadens schlägt das Programm den Restwert
  selbst vor und schreibt die Herleitung darunter („3. Verleihjahr, 60 % von 41,50 €"); die
  bisherige feste Voreinstellung ist entfallen. Gerechnet wird die Staffel der Arbeitshilfe:
  erstes Verleihjahr voller Preis, dann 80, 60, 40, 20 Prozent, ab dem sechsten Jahr 10
  Prozent. Der Betrag bleibt überschreibbar, weil er im Ermessen der Schule liegt.
- _„Bei offener Bearbeitung erfolgt die Sperrung. LMF untersagt dies."_ Unverändert — dazu
  unsere Frage 1 weiter unten.
- _„Ein Abgangs- und ein Zugangsbuch fehlen."_ Das Abgangsbuch gibt es seit dem 17.09.2026:
  Es listet je Zeitraum, welche Exemplare aus dem Bestand gegangen sind, mit Datum, Nummer,
  Titel, Signatur und Grund, getrennt nach Lernmitteln des Landes und Beständen des
  Schulträgers, als Ausdruck zum Abheften. Vorbelegt ist das laufende Schulhalbjahr mit den
  Stichtagen 15. März und 15. September. Eines sagt der Ausdruck ausdrücklich: Exemplare, die
  vor Einführung dieser Führung ausgebucht wurden, tragen kein Abgangsdatum; ihre Zahl steht
  unter der Liste. Ein Datum nachträglich zu erfinden hätte einen Nachweis erzeugt, der wie
  eine Tatsache aussieht und keine ist. Das Zugangsbuch folgt aus denselben Daten; der
  Wareneingang ist je Exemplar erfasst.
- _„Das Mahnwesen entspricht noch nicht den Vorgaben (Standardabwertung, zeitabhängiger
  Wertverlust)."_ Abgestellt, siehe oben. Drei Abweichungen bleiben und sind unter Punkt 3
  beschrieben.

**Zu 2 — Scannen weder mit Handscanner noch mit Rechnerkamera.** Abgestellt. Die Ursache war
für beide Geräte dieselbe und lag nicht an der Auflösung: Der gedruckte Strichcode trug ein
Prüfzeichen, das im Code steckt, aber nicht in der aufgedruckten Nummer. Das Lesegerät lieferte
also eine Nummer, die es in keiner Datei gibt. Neu gedruckt wird jetzt ein Code ohne dieses
Zeichen. Zusätzlich liest das Programm einen Scan, der nichts findet, ein zweites Mal ohne das
Prüfzeichen — damit funktionieren **alle vor dem 17.09.2026 gedruckten Ausweise und Etiketten
weiter**, die Schule muss nichts neu bekleben. An der Kamera fehlte außerdem das Strichcode-
Format in der Liste der zu erkennenden Formate.

**Zu 3 — Können Littera-Barcodes eingelesen werden?** Ja. Littera druckt die Mediennummer im
Klartext und verschlüsselt sie zusätzlich in einer 13-stelligen Nummer; das Programm rechnet
sie zurück, an der Ausleihe und auch dann, wenn gerade kein Netz da ist.

**Zu 4 — Bücher mit 0 Exemplaren.** Abgestellt, in beiden Ansichten. Ein Titel ohne Exemplare
bleibt sichtbar — er entsteht regulär, etwa beim Anlegen ohne Bestandsangabe oder bei der
Übernahme von Altbestand —, aber die Trefferliste sagt es jetzt: Neben jedem Buch steht, wie
viele Exemplare vorhanden und wie viele davon frei sind, und bei einem Titel ohne Bestand steht
das abgesetzt daneben. Niemand geht mehr ins Regal, um etwas zu suchen, das dort nie stand.

**Zu 5 — Schülerdatei ohne Sortierung und Filter.** Abgestellt. Die Liste lässt sich nach Name,
Klasse und Zahl der geliehenen Bücher sortieren und nach Jahrgang filtern. Beides geschieht auf
dem Server, nicht erst im Bildschirmausschnitt — sonst ordnet eine Sortierung nur die gerade
geladenen Zeilen und verdeckt den Rest.

**Zu 5, zweiter Punkt — „Bei den Ausleihfristen fehlt das Jahr. Mehrjahresbände lassen sich
nicht abbilden."** Dazu unsere Frage 2.

**Zu 6 — Auswahl nach Jahrgängen statt nach Klassen.** Abgestellt; die Auswahl gibt es jetzt
nach Jahrgang, einschließlich der Bildungsgänge, die nicht mit einer Zahl beginnen.

## 2. Zwei Fragen an Sie

**Frage 1 — Sperre bei offener Forderung.** Das Programm weist heute eine weitere Ausleihe ab,
solange ein Schadensfall unbezahlt ist; die Abweisung lässt sich von Hand übergehen und wird
dann protokolliert. Ihr Protokoll hält fest, dass die Lernmittelfreiheit das untersagt. Wir
haben die Arbeitshilfe zum Erlass vom 17.12.2014 und die Verfahrensbeschreibung zum Mahnwesen
im Original gelesen; zur Sperre steht dort nichts. Deshalb die Bitte um Klarstellung:

1. Gilt das Verbot nur für Lernmittel des Landes oder für jede Ausleihe, also auch für die
   Bestände der Schülerbücherei?
2. Worauf stützt es sich — gibt es dazu eine Fassung des Verfahrens, die uns nicht vorliegt?
3. Zählt eine Abweisung, die die Bibliothekskraft im Einzelfall übergehen kann, in Ihrem Sinne
   bereits als „Sperrung"?

Bis zu Ihrer Antwort bleibt der heutige Stand unverändert.

**Frage 2 — Mehrjahresbände.** Wir möchten sicher sein, dass wir Ihren Satz richtig verstehen.
Gemeint ist vermutlich ein Buch, das über mehrere Schuljahre bei demselben Kind bleibt, mit
einer Rückgabefrist am Ende dieser Zeit statt am Ende des laufenden Schuljahres. Die Rechnung
dafür ist im Programm vorhanden; es fehlt die Eingabe, mit der ein Titel als mehrjährig
gekennzeichnet wird. Bevor wir sie bauen: Ist das gemeint — und wenn ja, wird die Dauer am
Titel festgelegt (dieses Lehrwerk läuft über zwei Jahre) oder bei der Ausgabe an die Klasse?

## 3. Drei Stellen, an denen wir bewusst anders arbeiten

Wir nennen sie ausdrücklich, damit sie nicht als Versehen gelten.

1. **Versand nur per Post.** Die Anforderungsliste sieht Versand per Post, E-Mail oder App vor.
   Das Programm druckt Briefe und verschickt keine Zahlungsaufforderungen per E-Mail. Grund ist
   die Schriftform und eine Datenschutz-Entscheidung der Schule vom 22.08.2026: An
   Erziehungsberechtigte geht nichts unverschlüsselt per E-Mail, was einen Schadensfall nennt.
2. **Ein bezahltes, nicht zurückgegebenes Buch wird ausgesondert, nicht gelöscht.** Die
   Anforderungsliste spricht vom Löschen. Ein gelöschter Datensatz nimmt den Nachweis mit, dass
   es das Exemplar gab und wie es aus dem Bestand ging — genau das, was die Bestandskartei und
   das Abgangsbuch zeigen müssen. Das Exemplar verschwindet deshalb aus Katalog und Ausleihe,
   bleibt aber mit Grund und Datum in der Bestandsführung.
3. **Mahnfrist vier Wochen.** Die Anforderungsliste nennt sechs Wochen, die Arbeitshilfe und
   das Musterschreiben nennen vier — mit Datum im Brief. Wir rechnen mit vier und haben die
   Frist einstellbar gemacht. Wenn in Ihrem Haus sechs Wochen gelten, stellen wir sie um; uns
   fehlt dafür nur die verbindliche Angabe.

## 4. Zu den beiden Sätzen Ihrer Einschätzung

**„Ein Nachweis der DSGVO-Konformität liegt nicht vor."** Das ist richtig, und wir liefern ihn
nach. Vorhanden sind bereits: ein Entwurf des Verzeichnisses der Verarbeitungstätigkeiten, ein
Entwurf der Information für Erziehungsberechtigte, eine vollständige Übersicht, welche Auskunft
des Programms welche Schülerdaten enthält, sowie Löschfristen, die im Programm eingestellt und
nächtlich ausgeführt werden (die Zuordnung „wer hat was gelesen" wird nach Ablauf getrennt).
Was fehlt, ist die Beschlussfassung auf Schulseite und die Prüfung durch die schulische
Datenschutzbeauftragte oder den schulischen Datenschutzbeauftragten. Wir stellen Ihnen die
Unterlagen gern vorab zur Verfügung.

**„Hosting- und Programmpflegekonzepte sind nicht geplant. Dies könnte ein Ausschlusskriterium
sein."** Diesen Punkt nehmen wir ernst, weil er über die Mängelliste hinausgeht. Der Betrieb
ist heute beschrieben (Installation, Aktualisierung, tägliche Sicherung mit Verschlüsselung,
Wiederherstellung, Überwachung), aber als Anleitung für den Betreiber, nicht als Konzept, das
eine prüfende Stelle vorgelegt bekommt. Wir stellen beides zusammen: wo das Programm läuft, wer
es betreibt, wie gesichert und wiederhergestellt wird, wie Aktualisierungen und Fehlerbehebungen
ablaufen und was gilt, wenn die Pflege nicht fortgeführt werden kann. Für die Frage, was daraus
eine tragfähige Lösung für weitere Schulen machen würde, sind wir für Ihre Anforderungen
dankbar — sie ist die eigentliche Frage hinter Ihrem Satz, und sie lässt sich nicht allein aus
der Schule heraus beantworten.

Über eine Rückmeldung zu den Fragen 1 und 2 sowie zu den drei Abweichungen würden wir uns
freuen; danach melden wir den vollständigen Stand.

Mit freundlichen Grüßen
