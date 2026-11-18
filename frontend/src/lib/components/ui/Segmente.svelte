<script>
	import Icon from './Icon.svelte';

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
	 * @prop {boolean} [disabled] - sperrt die ganze Gruppe. Schrift in on-surface zu 38 %, Rahmen
	 *   und gewählte Fläche zu 12 %, wie jeder gesperrte Zustand in M3 (Interaction states).
	 */
	/** @type {{ optionen: { wert: string, text: string }[], wert: string, onwahl: (wert: string) => void, etikett: string, klasse?: string, disabled?: boolean }} */
	let { optionen, wert, onwahl, etikett, klasse = '', disabled = false } = $props();
	const rahmen = $derived(disabled ? 'border-on-surface/12' : 'border-outline');
</script>

<!-- auto-cols-fr: M3 gibt allen Segmenten dieselbe Breite. Mit `inline-grid` richtet
     sich diese Breite nach dem längsten Segment, statt die ganze Zeile zu füllen. h-9: M3 nennt
     40 dp bei Dichte 0 („Each step down in density removes 4dp"), das Haus steht auf -1. -->
<div
	role="group"
	aria-label={etikett}
	class="inline-grid h-9 grid-flow-col auto-cols-fr overflow-hidden rounded-full border {rahmen} {klasse}"
>
	{#each optionen as o, i (o.wert)}
		{@const gewaehlt = o.wert === wert}
		{@const zustand = disabled
			? `cursor-not-allowed text-on-surface/38 ${gewaehlt ? 'bg-on-surface/12' : ''}`
			: `cursor-pointer ${gewaehlt ? 'bg-secondary-container text-on-secondary-container' : 'text-on-surface'}`}
		<button
			type="button"
			aria-pressed={gewaehlt}
			{disabled}
			onclick={() => onwahl(o.wert)}
			class="flex items-center justify-center gap-2 px-4 text-label-large transition-colors {i > 0
				? `border-l ${rahmen}`
				: ''} {zustand}"
		>
			<!-- Immer im Fluss, nur unsichtbar: hält die Breite über alle Zustände fest. -->
			<Icon name="check" groesse={18} class={gewaehlt ? '' : 'invisible'} />
			{o.text}
		</button>
	{/each}
</div>
