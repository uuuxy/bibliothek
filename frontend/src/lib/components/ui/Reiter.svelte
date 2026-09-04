<script>
	/**
	 * @component Reiter — die Reiterleiste der Anwendung (Material 3, Primary Tabs).
	 *
	 * Reiter waren bis zum 23.08.2026 vier Mal von Hand gebaut (Medienkatalog,
	 * Bestellwesen, Buch-Akte, Inventur-Startseite) und dabei auseinandergelaufen —
	 * dieselbe Geschichte wie bei den Suchfeldern, die in zehn Kopien an sieben Werten
	 * differierten. Diese Komponente ist der eine Ort; die vier Bestandsfälle stehen in
	 * `frontend-hygiene-reiter.test.js` als eingefrorener Rest und werden bei ihrem
	 * nächsten fachlichen Anfassen nachgezogen.
	 *
	 * M3 kennt Reiter für DREI BIS FÜNF gleichrangige Bereiche. Mehr Bereiche oder
	 * ungleichgewichtige gehören in eine Liste (siehe KategorieListe der Einstellungen,
	 * wo sechs Reiter mit einem überfüllten „Allgemein" genau daran scheiterten).
	 *
	 * `anzahl` ist der Zähler am Reiter — er sagt VOR dem Klick, ob dort etwas wartet. Er
	 * wird von ui/Zaehlerpille gezeichnet, derselben Pille wie in der Seitenleiste: Bis zum
	 * 04.09.2026 waren es zwei Fassungen, und im Druck-Center stand links „999+" neben
	 * rechts „30674" — dieselbe Zahl, zwei Aussagen.
	 *
	 * Zwei Ränge, wie M3 sie kennt: `primaer` (Bereiche einer Seite, Indikator in
	 * Primärfarbe) und `sekundaer` (Unteransichten DESSELBEN Inhalts, unter einer
	 * Primärleiste). Bis 24.08.2026 trug der Medienkatalog zwei Leisten im selben
	 * Stil übereinander — die Rangfolge war unsichtbar. Sekundär: gewählter Text in
	 * on-surface statt primary, Indikator ohne Rundung.
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

<div
	role="tablist"
	aria-label={etikett}
	class="flex gap-6 border-b border-outline-variant overflow-x-auto {klasse}"
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
			class="relative shrink-0 cursor-pointer text-sm font-medium whitespace-nowrap transition-colors {sekundaer
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
				<span class="absolute inset-x-0 bottom-0 h-0.5 bg-primary {sekundaer ? '' : 'rounded-full'}"
				></span>
			{/if}
		</button>
	{/each}
</div>
