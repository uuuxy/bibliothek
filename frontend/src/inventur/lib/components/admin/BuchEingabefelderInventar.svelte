<!-- @component Bestand und Zähldatum eines Titels. Sie stehen im Abschnitt „Exemplare"
     (BuchExemplareListe): Die Zahl im Feld ist der Bestand, den die Liste darunter zeigt. -->
<script>
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	import { bestandHinweis } from './buchFormularRegeln.js';

	let { formular = $bindable() } = $props();
</script>

<div class="grid grid-cols-2 gap-4">
	<Feld
		id="buch-bestand"
		etikett="Aktueller Bestand"
		type="number"
		bind:value={formular.stock}
		hint={bestandHinweis(formular.id, formular.stock)}
		feld="pr-14"
	>
		{#snippet nachlaufend()}Stück{/snippet}
	</Feld>
	<Feld id="buch-zaehldatum" etikett="Zähldatum" type="date" bind:value={formular.lastCounted} />
</div>
