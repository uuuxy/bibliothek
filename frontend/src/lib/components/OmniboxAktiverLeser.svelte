<script>
	import Leserakte from '../Leserakte.svelte';
	import OmniboxThekeHinweise from './OmniboxThekeHinweise.svelte';
	import { omniboxStore } from '../stores/omnibox.svelte.js';

	let { profil = $bindable(null) } = $props();

	// In aktiverLeser steht ein Leser, Schüler oder Kollege, und beide bekommen dieselbe Akte:
	// Die Theke muss sehen, welche Bücher der Mensch vor ihr hat, und das ist bei einem
	// Kollegen dieselbe Frage wie bei einem Kind. Was an ihm anders ist (keine Klasse, Frist
	// ein Jahr), sagen die Akte selbst und der Hinweis darüber.
	const leser = $derived(omniboxStore.aktiverLeser);

	function abwaehlen() {
		omniboxStore.aktiverLeser = null;
		omniboxStore.letzteFremdrueckgabe = null;
	}
</script>

{#if leser}
	<!-- Fremdrückgabe-, Abholfach- und Kollegiums-Banner (200-Zeilen-Regel: eigene Datei). -->
	<OmniboxThekeHinweise />
	<Leserakte
		bind:this={profil}
		{leser}
		startReiter="ausleihen"
		onMerged={omniboxStore.uebernimmZusammengefuehrt}
		onDeselect={abwaehlen}
		onReturnClick={(barcode) => omniboxStore.gibZurueck(barcode, () => profil?.ladeNeu())}
	/>
{/if}
