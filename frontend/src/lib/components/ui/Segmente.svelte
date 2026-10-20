<script>
	import { Check } from '@lucide/svelte';

	/**
	 * @component Segmente — der Segmented Button aus Material 3 (einfachauswahl).
	 *
	 * Wofür: zwei bis fünf Ansichten desselben Inhalts, die einander ausschließen und
	 * zusammen alles abdecken, etwa „Offen | Erledigt | Alle". Nicht zu verwechseln mit den
	 * Nachbarn:
	 *   - Reiter (ui/Reiter) trennen Bereiche einer Seite, nicht Filter eines Inhalts.
	 *   - Filter-Chips sind einzeln an- und abwählbar und dürfen alle aus sein.
	 *
	 * Das gewählte Segment liegt auf `secondary-container`, wie styles/rollen.css es für jede
	 * Auswahl festlegt: In M3 markiert der secondary-container eine Auswahl, nicht die
	 * Primärfarbe.
	 *
	 * Das Häkchen ist Teil der Bauform, kein Schmuck: Es sagt bei getönten Flächen, welches
	 * Segment gewählt ist, auch wenn die Tönung schwach wiedergegeben wird.
	 * Sein Platz bleibt in jedem Segment reserviert: Sonst rücken beim Umschalten alle
	 * Beschriftungen, und der Umschalter zappelt unter dem Finger.
	 *
	 * @prop {{ wert: string, text: string }[]} optionen
	 * @prop {string} wert - der gewählte Wert.
	 * @prop {(wert: string) => void} onwahl
	 * @prop {string} etikett - Name der Gruppe für Screenreader.
	 * @prop {string} [klasse]
	 */
	/** @type {{ optionen: { wert: string, text: string }[], wert: string, onwahl: (wert: string) => void, etikett: string, klasse?: string }} */
	let { optionen, wert, onwahl, etikett, klasse = '' } = $props();
</script>

<!-- auto-cols-fr: M3 gibt allen Segmenten dieselbe Breite. Mit `inline-grid` richtet
     sich diese Breite nach dem längsten Segment, statt die ganze Zeile zu füllen. -->
<div
	role="group"
	aria-label={etikett}
	class="inline-grid h-10 grid-flow-col auto-cols-fr overflow-hidden rounded-full border border-outline {klasse}"
>
	{#each optionen as o, i (o.wert)}
		{@const gewaehlt = o.wert === wert}
		<button
			type="button"
			aria-pressed={gewaehlt}
			onclick={() => onwahl(o.wert)}
			class="flex cursor-pointer items-center justify-center gap-2 px-4 text-sm font-semibold transition-colors {i >
			0
				? 'border-l border-outline'
				: ''} {gewaehlt ? 'bg-secondary-container text-on-secondary-container' : 'text-on-surface'}"
		>
			<!-- Immer im Fluss, nur unsichtbar: hält die Breite über alle Zustände fest. -->
			<Check class="h-4.5 w-4.5 shrink-0 {gewaehlt ? '' : 'invisible'}" aria-hidden="true" />
			{o.text}
		</button>
	{/each}
</div>
