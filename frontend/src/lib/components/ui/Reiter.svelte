<script>
	/**
	 * @component Reiter — die Reiterleiste der Anwendung (Material 3, Primary Tabs), der eine
	 * Ort für Reiter: Von Hand gebaute Leisten laufen auseinander
	 * (`frontend-hygiene-reiter.test.js`).
	 *
	 * M3 kennt Reiter für drei bis fünf gleichrangige Bereiche. Mehr Bereiche oder
	 * ungleichgewichtige gehören in eine Liste (siehe KategorieListe der Einstellungen).
	 *
	 * `anzahl` ist der Zähler am Reiter: Er sagt vor dem Klick, ob dort etwas wartet. Er wird
	 * von ui/Zaehlerpille gezeichnet, derselben Pille wie in der Seitenleiste, damit dieselbe
	 * Zahl nicht zwei Aussagen macht.
	 *
	 * Zwei M3-Maße, im Browser gemessen und nicht aus dem Klassennamen gelesen:
	 *
	 *   Label-Gewicht: M3 gibt dem Reiter `title-small`, dessen Gewicht ist `weight-medium`
	 *   = 500. Die Klasse `font-medium` zeigt in diesem Haus aber auf 400, theme-mass.css
	 *   biegt die Skala um (medium→400, semibold/bold→500). Wer hier 500 will, schreibt
	 *   `font-semibold`.
	 *
	 *   Indikator: primary-navigation-tab hat 3 px und `shape (3px 3px 0 0)`,
	 *   secondary-navigation-tab 2 px ohne shape.
	 *
	 * Zwei Ränge, wie M3 sie kennt: `primaer` (Bereiche einer Seite, Indikator in
	 * Primärfarbe) und `sekundaer` (Unteransichten desselben Inhalts, unter einer
	 * Primärleiste; gewählter Text in on-surface statt primary, Indikator ohne Rundung).
	 * Zwei Leisten im selben Stil übereinander ließen die Rangfolge unsichtbar.
	 *
	 * @prop {{ id: string, label: string, anzahl?: number, steuert?: string }[]} reiter
	 *   `steuert` = aria-controls (ids der Inhaltsflächen); der Reiter selbst heißt tab-<id>.
	 * @prop {string} aktiv
	 * @prop {(id: string) => void} onwahl
	 * @prop {string} etikett - Name der Leiste für Screenreader.
	 * @prop {'primaer'|'sekundaer'} [variante]
	 * @prop {string} [klasse] - zusätzliche Klassen der Leiste (z. B. justify-center).
	 */
	import Zaehlerpille from './Zaehlerpille.svelte';

	/** @type {{ reiter: { id: string, label: string, anzahl?: number, steuert?: string }[], aktiv: string, onwahl: (id: string) => void, etikett: string, variante?: 'primaer'|'sekundaer', klasse?: string }} */
	let { reiter, aktiv, onwahl, etikett, variante = 'primaer', klasse = '' } = $props();
	const sekundaer = $derived(variante === 'sekundaer');
</script>

<!-- no-print: Eine Reiterleiste ist Bedienung. Auf einer Ausweiskarte schöbe sie die Karte
     von der Seite (e2e/ausweis-druckseite.spec.js). -->
<div
	role="tablist"
	aria-label={etikett}
	class="no-print flex gap-6 border-b border-outline-variant overflow-x-auto {klasse}"
>
	{#each reiter as r (r.id)}
		{@const gewaehlt = r.id === aktiv}
		<button
			type="button"
			role="tab"
			id="tab-{r.id}"
			aria-selected={gewaehlt}
			aria-controls={r.steuert}
			onclick={() => onwahl(r.id)}
			class="relative shrink-0 cursor-pointer text-sm font-semibold whitespace-nowrap transition-colors {sekundaer
				? 'pb-2.5'
				: 'pb-3'} {gewaehlt
				? sekundaer
					? 'text-on-surface'
					: 'text-primary'
				: 'text-on-surface-variant hover:text-on-surface'}"
		>
			{r.label}
			{#if r.anzahl}
				<Zaehlerpille anzahl={r.anzahl} klasse="ml-2" />
			{/if}
			{#if gewaehlt}
				<span
					class="absolute inset-x-0 bottom-0 bg-primary {sekundaer
						? 'h-0.5'
						: 'h-0.75 rounded-t-[3px]'}"
				></span>
			{/if}
		</button>
	{/each}
</div>
