<script>
	import StudentProfile from '../StudentProfile.svelte';
	import OmniboxTeacherCard from '../OmniboxTeacherCard.svelte';
	import OmniboxThekeHinweise from './OmniboxThekeHinweise.svelte';
	import { omniboxStore } from '../stores/omnibox.svelte.js';

	let { profil = $bindable(null) } = $props();

	// Seit Migration 125 steht in activeStudent EIN Leser — Schüler oder Kollege. Was die
	// Theke zeigt, entscheidet seine Art, nicht mehr zwei getrennte Plätze im Store.
	//
	// Ein Kollege bekommt die schmale Karte, kein Schülerprofil: Die Akte einer Lehrkraft
	// (Ausleihen, Ausweis drucken) ist der nächste Schritt; bis dahin liefe StudentProfile
	// in ein 404, weil es die Sicht `schueler` liest.
	const leser = $derived(omniboxStore.activeStudent);
	const istKollege = $derived(!!leser?.art && leser.art !== 'schueler');

	function abwaehlen() {
		omniboxStore.activeStudent = null;
		omniboxStore.lastFremdrueckgabe = null;
	}
</script>

{#if leser}
	<!-- Fremdrückgabe- und Abholfach-Banner (200-Zeilen-Regel: eigene Datei). -->
	<OmniboxThekeHinweise />
	{#if istKollege}
		<OmniboxTeacherCard teacher={leser} onDeselect={abwaehlen} />
	{:else}
		<StudentProfile
			bind:this={profil}
			student={leser}
			defaultTab="ausleihen"
			onMerged={omniboxStore.uebernimmZusammengefuehrt}
			onDeselect={abwaehlen}
			onReturnClick={(barcode) => omniboxStore.gibZurueck(barcode, () => profil?.reloadProfile())}
		/>
	{/if}
{/if}
