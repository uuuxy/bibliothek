<!--
  @component PageShell
  Das Seitengeruest fuer JEDE Route. Es besitzt genau drei Dinge: Inhaltsbreite,
  vertikalen Rhythmus und den Seitenkopf.

  Warum es das gibt: Am 07.08.2026 brachte jede Route ihr eigenes Geruest mit —
  drei Breiten (keine / max-w-4xl / max-w-6xl), fuenf Polsterungen und DREI Routen
  malten mit `bg-slate-50` eine zweite Flaeche ueber die Leinwand. Auf dem Bildschirm
  sah dadurch jede Seite anders aus. Die Flaeche gehoert der App-Huelle (App.svelte),
  die Karte gehoert dem Inhalt (Sheet.svelte) — hier drin steht keine Farbe.

  Die aeussere Polsterung (px-4 md:px-8 py-6) sitzt in App.svelte und gilt fuer alle;
  eine Route, die hier nochmal p-6 setzt, polstert doppelt.
-->
<script>
	let {
		/** Seitentitel. Leer = kein Kopf (z. B. Kiosk, das den ganzen Schirm fuellt). */
		titel = '',
		/** Eine Zeile, die sagt, was die Seite zeigt. Nur zusammen mit `titel`. */
		beschreibung = '',
		/** Aktionen rechts im Kopf (Snippet). Ohne Vorgabewert haelt die Typpruefung ein
		 *  Snippet-Prop fuer PFLICHT — acht Seiten meldeten daraufhin "Property 'aktionen'
		 *  is missing", obwohl der Kopf optional ist. */
		aktionen = undefined,
		/** 'voll' fuer Tabellen und Listen, 'inhalt' fuer Formulare und Detailseiten. */
		breite = 'voll',
		children
	} = $props();

	// Zwei Breiten, nicht fuenf. 'inhalt' begrenzt Lesezeilen; Tabellen brauchen die
	// volle Flaeche, sonst entsteht rechts ein toter Streifen.
	const BREITEN = {
		voll: 'w-full',
		inhalt: 'mx-auto w-full max-w-6xl'
	};
</script>

<!-- `animate-fade-in` ist global in App.svelte definiert und stand bisher an 36 Stellen
     verstreut. Im Geruest tritt jede Seite gleich auf, statt manche gar nicht. -->
<div class="animate-fade-in flex flex-col gap-6 {BREITEN[breite] ?? BREITEN.voll}">
	{#if titel}
		<header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
			<div class="min-w-0">
				<h1 class="text-on-surface text-xl font-bold">{titel}</h1>
				{#if beschreibung}
					<p class="text-on-surface-variant mt-0.5 text-sm">{beschreibung}</p>
				{/if}
			</div>

			{#if aktionen}
				<div class="flex shrink-0 flex-wrap items-center gap-2 print:hidden">
					{@render aktionen()}
				</div>
			{/if}
		</header>
	{/if}

	{@render children()}
</div>
