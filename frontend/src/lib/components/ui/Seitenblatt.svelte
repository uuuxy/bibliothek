<!-- @component Seitenblatt — Angaben zu dem, was man in einem Raster angetippt hat, in einem Blatt
     am rechten Rand (Material 3, Side sheets: „Standard side sheets display content that
     complements the screen's primary content. They remain visible while people interact with
     primary content."). Ab lg steht es neben dem Inhalt, der dafür mit SEITENBLATT_PLATZ
     einrückt; schmaler füllt es die Breite. Maße nach den Specs: 24 px Rand, 12 px zwischen den
     Elementen oben, Fläche „Surface", Überschrift und Kreuz in „On surface variant". Es schließen
     das Kreuz, Escape (escapeSchliesst, wie bei den Dialogen) und beim Aufrufer ein zweiter
     Klick auf dieselbe Kachel. -->
<script module>
	/** Rand, um den der Inhalt einrückt, solange das Blatt offen ist: so breit wie das Blatt. */
	export const SEITENBLATT_PLATZ = 'lg:pr-90';
</script>

<script>
	import { X } from '@lucide/svelte';
	import { escapeSchliesst } from './escapeSchliesst.js';

	/** @type {{ titel: string, onschliessen: () => void, children: import('svelte').Snippet }} */
	let { titel, onschliessen, children } = $props();
</script>

<aside
	aria-label={titel}
	use:escapeSchliesst={onschliessen}
	class="fixed inset-0 z-40 overflow-y-auto bg-surface lg:left-auto lg:w-90 lg:border-l lg:border-outline-variant"
	data-seitenblatt
>
	<div class="flex min-h-16 items-center gap-3 pt-3 pr-3 pl-6">
		<h2 class="min-w-0 flex-1 text-lg text-on-surface-variant">{titel}</h2>
		<button
			class="icon-btn shrink-0 text-on-surface-variant"
			onclick={onschliessen}
			aria-label="Schließen"
			data-tip="Schließen"
		>
			<X class="h-5 w-5" aria-hidden="true" />
		</button>
	</div>
	<div class="flex flex-col gap-5 px-6 pt-3 pb-6">
		{@render children()}
	</div>
</aside>
