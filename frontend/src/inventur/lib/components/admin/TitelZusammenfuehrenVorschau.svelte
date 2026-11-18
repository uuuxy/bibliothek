<!-- @component TitelZusammenfuehrenVorschau — eine Zeile: wie der bleibende Titel danach
     aussieht, und welcher Titel gelöscht wird. An der Zahl der Exemplare sieht man, wenn beide
     Titel dieselben Bücher trugen. Gerechnet vom Server (zusammenfuehrenVorschau.svelte.js). -->
<script>
	import Icon from '../../../../lib/components/ui/Icon.svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import { exemplare } from './zusammenfuehrenVorschau.svelte.js';

	/** @type {{ vorschau: any | null, laedt: boolean, fehler: string }} */
	let { vorschau, laedt, fehler } = $props();
</script>

{#if laedt}
	<div class="flex justify-center py-4"><Ladekreis size="md" /></div>
{:else if fehler}
	<div
		role="alert"
		class="flex items-start gap-2 rounded-xl bg-error-container p-3 text-on-error-container"
	>
		<Icon name="error" class="mt-0.5" />
		<p class="text-body-small-emphasized leading-tight">{fehler}</p>
	</div>
{:else if vorschau}
	<div class="space-y-1 rounded-xl bg-surface-container p-3 text-sm text-on-surface">
		<p>
			<span class="text-body-medium-emphasized">Danach:</span>
			{vorschau.titel} · {vorschau.isbn || 'ohne ISBN'} · {exemplare(vorschau.exemplare_danach)}
		</p>
		<p class="text-on-surface-variant">
			„{vorschau.quelle_titel}“ {vorschau.quelle_isbn
				? 'mit ISBN ' + vorschau.quelle_isbn
				: 'ohne ISBN'} wird gelöscht
		</p>
	</div>
{/if}
