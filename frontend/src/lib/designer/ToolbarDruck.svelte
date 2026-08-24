<!-- @component ToolbarDruck — die oberste Zeile des Ausweis-Designers: womit gedruckt
     wird, und der Testdruck dazu.

     „A4-Bogen" stand hier bis zum 24.08.2026 und meinte acht Kartenabbilder auf einem
     Blatt zum Ausschneiden. Als Notbehelf gedacht, im Alltag nutzlos. An seiner Stelle
     steht der Etikettenbogen: Name, Klasse und Barcode auf Klebeetiketten, vom Server
     erzeugt wie die Buch-Etiketten.

     Die Wahl gilt nicht nur hier — der Stapeldruck der Schülerdatei folgt ihr ebenfalls
     (idStore.printMode). Der Designer legt fest WIE gedruckt wird, die Schülerdatei WER.

     Eigene Datei, weil Toolbar.svelte mit dieser Zeile über die 200 gelaufen wäre. -->
<script>
	import { Printer } from '@lucide/svelte';
	import { idStore } from './idDesignerStore.svelte.js';
	import { ETIKETT_FORMATE } from '../etikettformate.js';
	import Button from '../components/ui/Button.svelte';
	import Select from '../components/ui/Select.svelte';

	/**
	 * @type {{
	 *   printMode: 'card'|'etikett',
	 *   onPrintMode: (m: 'card'|'etikett') => void,
	 *   side: 'front'|'back',
	 *   onPrint: () => void
	 * }}
	 */
	const { printMode, onPrintMode, side, onPrint } = $props();

	const MODI = [
		{ value: 'card', label: 'Kartendrucker' },
		{ value: 'etikett', label: 'Etikettenbogen' }
	];

	// Der Etikettenbogen kennt keine Rückseite — die Beschriftung darf keine versprechen,
	// die der Knopf nicht drucken kann.
	const knopfText = $derived(
		printMode === 'etikett'
			? 'Muster-Etikett drucken'
			: side === 'back'
				? 'Testdruck Rückseite'
				: 'Testdruck Vorderseite'
	);
</script>

<div class="border-outline-variant flex flex-wrap items-center justify-between gap-3 border-b pb-4">
	<div class="flex flex-wrap items-center gap-3">
		<div
			class="border-outline-variant bg-surface-container flex shrink-0 rounded-xl border p-0.5 text-xs"
		>
			{#each MODI as modus (modus.value)}
				<button
					onclick={() => onPrintMode(/** @type {'card'|'etikett'} */ (modus.value))}
					aria-pressed={printMode === modus.value}
					class="cursor-pointer rounded-lg px-3 py-1.5 font-bold transition-all {printMode ===
					modus.value
						? 'bg-surface text-on-surface shadow-xs'
						: 'text-on-surface-variant hover:text-on-surface'}">{modus.label}</button
				>
			{/each}
		</div>

		{#if printMode === 'etikett'}
			<Select
				bind:value={idStore.etikettFormat}
				options={ETIKETT_FORMATE}
				class="w-64"
				aria-label="Etikettenformat"
			/>
		{/if}
	</div>

	<Button onclick={onPrint} class="px-5">
		<Printer class="h-4 w-4" aria-hidden="true" />
		{knopfText}
	</Button>
</div>
