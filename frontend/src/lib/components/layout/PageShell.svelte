<!--
  @component PageShell
  Das Seitengerüst für jede Route. Es besitzt genau eine Sache: den vertikalen Rhythmus.

  Kein Seitentitel und kein Erklärtext: Die Seitenleiste sagt, wo man ist.

  Über der Suche gilt: Reiterband (wenn es eins gibt), dann die Suchpille von 48 px über die
  volle Breite, dann Filter und Knöpfe auf der Grundlinie von 36 px. So gibt es auch
  Material 3 vor — die Suchleiste ist „a persistent and prominent search field at the top of
  the screen", Reiter sitzen „at the top of the content pane under an app bar". Wer eine
  Knopfzeile braucht, stellt sie unter die Pille (Vorbild: StudentDirectoryToolbar,
  MahnwesenSuchleiste). Gate: e2e/suchpille-einheitlich.spec.js misst die Startlinie.

  Das Gerüst gibt es, damit nicht jede Route eigene Breite, Polsterung und Fläche mitbringt.
  Die Fläche gehört der App-Hülle (App.svelte) — hier drin steht keine Farbe.

  Keine Breitenbegrenzung: Ein Verwaltungswerkzeug am Schreibtisch nutzt den Bildschirm, den
  es hat.

  Die äußere Polsterung (px-4 md:px-8 py-6) sitzt in App.svelte und gilt für alle; eine
  Route, die hier nochmal p-6 setzt, polstert doppelt.
-->
<script>
	let { children } = $props();
</script>

<!-- `animate-fade-in` ist global in App.svelte definiert; im Gerüst tritt jede Seite
     gleich auf. -->
<div class="animate-fade-in flex w-full flex-col gap-6">
	{@render children()}
</div>
