<!--
  @component PageShell
  Das Seitengeruest fuer JEDE Route. Es besitzt genau eine Sache: den vertikalen Rhythmus.

  KEIN Seitentitel und KEIN Erklaertext. a3e4184 (20.06.2026) hat beides als
  "redundant page headers and description subtitles" aus Druck-Center, Schuelerdatei,
  System-Logs, Einstellungen und Inventur ENTFERNT — die Seitenleiste sagt bereits,
  wo man ist, und ein Satz wie "Bedarf erfassen, bestellen und Zulauf verbuchen"
  erklaert einem Sekretariat nichts, das die Seite taeglich benutzt. Ich hatte beides
  am 07.08. auf alle 14 Seiten eingebaut; zurueckgenommen am 08.08.

  KEIN `aktionen`-Slot mehr (04.09.2026). Er war der Rest des Seitentitels: eine
  rechtsbuendige Knopfzeile GANZ OBEN, die neben einer Ueberschrift stand, die es seit
  68c4810 nicht mehr gibt. Von sechzehn Routen hat ihn genau eine benutzt — Mahnwesen —
  und dort begann die Suchpille dadurch 84 px tiefer als auf jeder anderen Seite. Peter
  am 04.09. an zwei Bildschirmfotos: „hier ist die Suchleiste immer an anderen
  Positionen … ich empfinde es an dieser Stelle als Stilbruch".

  Die Hausordnung ueber der Suche lautet seitdem: Reiterband (wenn es eins gibt), dann
  die 48-px-Suchpille ueber die volle Breite, dann Filter und Knoepfe auf der
  36-px-Grundlinie. Das ist auch die Reihenfolge, die Material 3 vorgibt — die
  Suchleiste ist „a persistent and prominent search field at the top of the screen" und
  nimmt den Platz der App-Leiste ein, Reiter sitzen „at the top of the content pane
  under an app bar". Wer eine Knopfzeile braucht, stellt sie UNTER die Pille (Vorbild:
  StudentDirectoryToolbar, MahnwesenSuchleiste). Gate: e2e/suchpille-einheitlich.spec.js
  misst die Startlinie.

  Warum es das Geruest ueberhaupt gibt: Am 07.08.2026 brachte jede Route ihr eigenes mit —
  drei Breiten (keine / max-w-4xl / max-w-6xl), fuenf Polsterungen und DREI Routen
  malten mit `bg-slate-50` eine zweite Flaeche ueber die Leinwand. Auf dem Bildschirm
  sah dadurch jede Seite anders aus. Die Flaeche gehoert der App-Huelle (App.svelte) —
  hier drin steht keine Farbe.

  KEINE Breitenbegrenzung. Ich hatte am 07.08. ein `breite="inhalt"` (max-w-6xl)
  eingefuehrt; das widerspricht f2320e1, wo die Einengung auf max-w-3xl ausdruecklich
  zugunsten von `w-full` aufgehoben wurde. Ein Verwaltungswerkzeug am Schreibtisch
  nutzt den Bildschirm, den es hat.

  Die aeussere Polsterung (px-4 md:px-8 py-6) sitzt in App.svelte und gilt fuer alle;
  eine Route, die hier nochmal p-6 setzt, polstert doppelt.
-->
<script>
	let { children } = $props();
</script>

<!-- `animate-fade-in` ist global in App.svelte definiert und stand bisher an 36 Stellen
     verstreut. Im Geruest tritt jede Seite gleich auf, statt manche gar nicht. -->
<div class="animate-fade-in flex w-full flex-col gap-6">
	{@render children()}
</div>
