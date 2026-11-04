<!-- @component NummerKopieren — ISBN oder EAN in Festbreite mit Kopier-Symbol. Ein Klick legt sie
     in die Zwischenablage; „Kopiert!" steht erst, wenn es gelungen ist (zwischenablage.js). Der
     Kopf der Buchakte und das Seitenblatt in Mein Portal nehmen ihn von hier, damit die Nummer
     an beiden Stellen gleich aussieht und gleich kopiert wird. -->
<script>
	import { Copy } from '@lucide/svelte';
	import { kopiere } from '../../utils/zwischenablage.js';

	/** @type {{ nummer?: string, art?: string }} */
	let { nummer = '', art = 'ISBN' } = $props();

	let kopiert = $state(false);

	async function kopieren() {
		if (!nummer || !(await kopiere(nummer))) return;
		kopiert = true;
		setTimeout(() => (kopiert = false), 2000);
	}
</script>

<button
	class="flex w-fit cursor-pointer items-center gap-1.5 font-mono transition-colors hover:text-primary"
	onclick={kopieren}
	title="{art} kopieren"
>
	<span>{art}: {nummer || '–'}</span>
	{#if kopiert}
		<span class="font-sans text-xs font-medium text-primary">Kopiert!</span>
	{:else if nummer}
		<Copy class="h-3.5 w-3.5" aria-hidden="true" />
	{/if}
</button>
