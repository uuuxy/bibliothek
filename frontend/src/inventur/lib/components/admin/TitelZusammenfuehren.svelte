<!-- @component TitelZusammenfuehren — der Knopf „Mit anderem Titel zusammenführen" in der Maske
     eines Titels; er öffnet TitelZusammenfuehrenDialog. Die Maske zeigt ihn nur mit dem Recht
     delete_books: Danach gibt es einen der beiden Titel nicht mehr, wie nach „Titel löschen".
     Der Dialog hängt nicht hier, sondern an der Wurzel der Maske: In der stehenden Spalte
     (sticky) läge er unter der Exemplarliste. -->
<script>
	import { Merge } from '@lucide/svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { uiStore } from '../../../../lib/stores/uiStore.svelte.js';
	import { verwerfenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';

	/** @type {{ offen: boolean }} */
	let { offen = $bindable(false) } = $props();

	// Nach dem Zusammenführen lädt die Maske den Titel neu, und was darin noch nicht gespeichert
	// ist, ginge verloren. Deshalb fragt der Knopf vorher wie jeder Wechsel der Ansicht.
	async function oeffnen() {
		if (uiStore.verlassenSperre?.() && !(await verwerfenBestaetigen('Verwerfen und weiter')))
			return;
		offen = true;
	}
</script>

<Button
	variant="secondary"
	onclick={oeffnen}
	class="w-full"
	title="Ein Buch, das zweimal im Katalog steht, zu einem Titel machen"
>
	<Merge class="w-4 h-4" aria-hidden="true" />
	Mit anderem Titel zusammenführen
</Button>
