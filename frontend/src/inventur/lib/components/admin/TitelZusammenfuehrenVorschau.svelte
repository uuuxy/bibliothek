<!-- @component TitelZusammenfuehrenVorschau — was das Zusammenführen täte: was zum bleibenden
     Titel kommt, wo er seine Angaben behält, und dass der andere danach gelöscht ist. Die
     Zahlen und Felder rechnet der Server (zusammenfuehrenVorschau.svelte.js). -->
<script>
	import { CircleAlert } from '@lucide/svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import { kommtHinzu } from './zusammenfuehrenVorschau.svelte.js';

	/** @type {{ vorschau: any | null, laedt: boolean, fehler: string }} */
	let { vorschau, laedt, fehler } = $props();

	const zeilen = $derived(vorschau ? kommtHinzu(vorschau) : []);
</script>

{#if laedt}
	<div class="flex justify-center py-4"><Ladekreis size="md" /></div>
{:else if fehler}
	<div
		role="alert"
		class="flex items-start gap-2 rounded-xl bg-error-container p-3 text-on-error-container"
	>
		<CircleAlert class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
		<p class="text-xs font-bold leading-tight">{fehler}</p>
	</div>
{:else if vorschau}
	<div class="space-y-3 text-sm text-on-surface">
		<div>
			<p>Zu „{vorschau.titel}“ kommt:</p>
			{#if zeilen.length > 0}
				<ul class="mt-1 list-disc space-y-0.5 pl-5">
					{#each zeilen as zeile (zeile)}
						<li>{zeile}</li>
					{/each}
				</ul>
			{:else}
				<p class="mt-1 text-on-surface-variant">Nichts außer dem Titel selbst.</p>
			{/if}
		</div>
		{#if vorschau.abweichend?.length > 0}
			<div>
				<p>Bleibt wie bei „{vorschau.titel}“:</p>
				<ul class="mt-1 list-disc space-y-0.5 pl-5">
					{#each vorschau.abweichend as a (a.bezeichnung)}
						<li>
							{a.bezeichnung}: {a.hier}
							<span class="text-on-surface-variant">(beim anderen: {a.dort})</span>
						</li>
					{/each}
				</ul>
			</div>
		{/if}
		<p class="font-bold">
			„{vorschau.quelle_titel}“ wird danach gelöscht. Das lässt sich nicht rückgängig machen; was
			gewandert ist, steht im Protokoll.
		</p>
	</div>
{/if}
