<!--
  @component PageShell
  Das Seitengeruest fuer JEDE Route. Es besitzt genau zwei Dinge: den vertikalen
  Rhythmus und einen Platz fuer die Seitenaktionen.

  KEIN Seitentitel und KEIN Erklaertext. a3e4184 (20.06.2026) hat beides als
  "redundant page headers and description subtitles" aus Druck-Center, Schuelerdatei,
  System-Logs, Einstellungen und Inventur ENTFERNT — die Seitenleiste sagt bereits,
  wo man ist, und ein Satz wie "Bedarf erfassen, bestellen und Zulauf verbuchen"
  erklaert einem Sekretariat nichts, das die Seite taeglich benutzt. Ich hatte beides
  am 07.08. auf alle 14 Seiten eingebaut; zurueckgenommen am 08.08.

  Warum es das gibt: Am 07.08.2026 brachte jede Route ihr eigenes Geruest mit —
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
	let {
		/** Aktionen der Seite, rechtsbuendig ueber dem Inhalt (Snippet). Ohne Vorgabewert
		 *  haelt die Typpruefung ein Snippet-Prop fuer PFLICHT. */
		aktionen = undefined,
		children
	} = $props();
</script>

<!-- `animate-fade-in` ist global in App.svelte definiert und stand bisher an 36 Stellen
     verstreut. Im Geruest tritt jede Seite gleich auf, statt manche gar nicht. -->
<div class="animate-fade-in flex w-full flex-col gap-6">
	{#if aktionen}
		<div class="flex shrink-0 flex-wrap items-center justify-end gap-2 print:hidden">
			{@render aktionen()}
		</div>
	{/if}

	{@render children()}
</div>
