<script>
	/**
	 * @component PortalUeberblick
	 * Was unter der Suche des Kollegiums-Portals steht, solange nichts gesucht wurde:
	 * die eigenen Reservierungen — die Antwort auf die Frage, mit der eine Lehrkraft
	 * das Portal öffnet („ist mein Satz durch?").
	 *
	 * Vorher (23.08.2026) stand hier zusätzlich ein Auszug „Deine Anliegen" mit „Alle
	 * ansehen" — damals gab es nur zwei Reiter. Seit dem 25.08. ist „Meine Anliegen" ein
	 * eigener Primary Tab mit Zähler; derselbe Inhalt in einem Geschwister-Reiter
	 * verstößt gegen M3 („each tab contains distinct content"), und ein Knopf, der zum
	 * Nachbarreiter springt, ist Navigation im Reiter. Peters Frage am 25.08.: „sollte
	 * unter den oberen Reitern … offene Klassensätze und meine Anliegen liegen?"
	 *
	 * Vokabular: „Reservierung", nicht „Klassensatz". Der Reiter „Klassensätze" meint die
	 * Zuordnung Klasse → Bücher; die Warteschlange hier ist etwas anderes, und ein Wort
	 * darf auf einer Oberfläche nur eine Bedeutung haben.
	 *
	 * Ein Leerzustand für die Ansicht, Überschrift nur, wenn die Liste Einträge hat
	 * (M3-Subheader gruppieren vorhandene Einträge), keine Trennlinie (eine Gruppe).
	 *
	 * @prop {{ titel_id: string, titel: string, klasse: string, anzahl: number, erstellt_am: string }[]} reservierungen
	 */

	/** @type {{ reservierungen: { titel_id: string, titel: string, klasse: string, anzahl: number, erstellt_am: string }[] }} */
	let { reservierungen } = $props();
</script>

<section class="flex w-full max-w-3xl flex-col gap-3 pt-4">
	{#if reservierungen.length === 0}
		<p class="text-sm text-on-surface-variant">
			Zurzeit wartet keine Reservierung — suche oben einen Titel, um einen Klassensatz zu
			reservieren.
		</p>
	{:else}
		<h2 class="text-base font-medium text-on-surface">Deine Reservierungen</h2>
		<ul class="divide-y divide-outline-variant">
			{#each reservierungen as r, _i (_i)}
				<li class="flex items-center justify-between gap-4 py-2.5 text-sm">
					<span class="min-w-0 flex-1 truncate text-on-surface">
						{r.titel || 'Titel nicht mehr im Katalog'}
						<span class="text-on-surface-variant">· Klasse {r.klasse}</span>
					</span>
					<span class="shrink-0 text-on-surface-variant"
						>{r.anzahl} Stück · seit {r.erstellt_am}</span
					>
				</li>
			{/each}
		</ul>
		<p class="text-xs text-on-surface-variant">
			Reservieren sperrt nichts — wer denselben Titel will, stellt sich an.
		</p>
	{/if}
</section>
