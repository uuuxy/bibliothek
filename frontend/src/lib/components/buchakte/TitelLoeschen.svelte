<!-- @component TitelLoeschen — „Titel löschen" im Kopf der Buchakte und in der Buchmaske. Ein
     Auftrag zur Zeit: Ein zweiter vor der Antwort schickte ein zweites DELETE, und nach dem Erfolg
     des ersten stand „Titel nicht gefunden". Gesperrt mit Ladekreis bis zur Antwort (M3, States
     und Progress indicators). -->
<script>
	import Button from '../ui/Button.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { Trash } from '@lucide/svelte';

	/** @type {{ onDelete: () => Promise<unknown> | void, class?: string, title?: string }} */
	let { onDelete, class: klasse = '', title = undefined } = $props();

	let loescht = $state(false);

	async function loesche() {
		if (loescht) return;
		loescht = true;
		try {
			await onDelete();
		} finally {
			loescht = false;
		}
	}
</script>

<Button variant="ghost" onclick={loesche} disabled={loescht} class="text-error {klasse}" {title}>
	{#if loescht}
		<Ladekreis size="sm" farbe="aktuell" />
	{:else}
		<Trash class="h-4 w-4" aria-hidden="true" />
	{/if}
	Titel löschen
</Button>
