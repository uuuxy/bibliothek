<script>
	import { Check } from '@lucide/svelte';

	/**
	 * @component Segmente — der Segmented Button aus Material 3 (einfachauswahl).
	 *
	 * Wofür: ZWEI BIS FÜNF Ansichten desselben Inhalts, die einander ausschließen und
	 * zusammen alles abdecken — „Offen | Erledigt | Alle". Nicht zu verwechseln mit
	 * den Nachbarn:
	 *   - Reiter (ui/Reiter) trennen BEREICHE einer Seite, nicht Filter eines Inhalts.
	 *   - Filter-Chips sind einzeln an- und abwählbar und dürfen alle aus sein.
	 *
	 * Vorher stand im Druck-Center ein handgebauter Umschalter, der aus einer anderen
	 * Designsprache stammte: 38 px hoch statt 40, Ecken 12 px statt voll gerundet, ohne
	 * Trennstriche, und das gewählte Segment lag auf `inverse-surface` (fast schwarz).
	 * Das verstieß gegen die Regel, die seit dem 04.08.2026 in styles/rollen.css steht:
	 * „In M3 markiert NICHT die Primärfarbe eine Auswahl, sondern der secondary-container."
	 *
	 * Das Häkchen ist Teil der Bauform, kein Schmuck — es sagt bei getönten Flächen,
	 * WELCHES Segment gewählt ist, auch wenn die Tönung schwach wiedergegeben wird.
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
			class="flex cursor-pointer items-center justify-center gap-2 px-4 text-sm font-medium transition-colors {i >
			0
				? 'border-l border-outline'
				: ''} {gewaehlt ? 'bg-secondary-container text-on-secondary-container' : 'text-on-surface'}"
		>
			<!-- Immer im Fluss, nur unsichtbar: hält die Breite über alle Zustände fest. -->
			<Check class="h-[18px] w-[18px] shrink-0 {gewaehlt ? '' : 'invisible'}" aria-hidden="true" />
			{o.text}
		</button>
	{/each}
</div>
