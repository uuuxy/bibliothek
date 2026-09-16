<script>
	import StudentProfile from '../StudentProfile.svelte';
	import OmniboxThekeHinweise from './OmniboxThekeHinweise.svelte';
	import { omniboxStore } from '../stores/omnibox.svelte.js';

	let { profil = $bindable(null) } = $props();

	// Seit Migration 125 steht in activeStudent EIN Leser — Schüler oder Kollege. Seit dem
	// 16.09.2026 bekommen beide dieselbe AKTE: Die Theke muss sehen, welche Bücher der
	// Mensch vor ihr hat, und das ist bei einem Kollegen dieselbe Frage wie bei einem Kind.
	//
	// Vorher stand hier für einen Kollegen eine schmale Karte ohne Ausleihen — nicht aus
	// Absicht, sondern weil GET /api/schueler/{id} die Sicht `schueler` las und ihn mit
	// 404 beantwortete. Was an ihm anders ist (keine Klasse, Frist ein Jahr), sagen die
	// Akte selbst und der Hinweis darüber.
	const leser = $derived(omniboxStore.activeStudent);

	function abwaehlen() {
		omniboxStore.activeStudent = null;
		omniboxStore.lastFremdrueckgabe = null;
	}
</script>

{#if leser}
	<!-- Fremdrückgabe-, Abholfach- und Kollegiums-Banner (200-Zeilen-Regel: eigene Datei). -->
	<OmniboxThekeHinweise />
	<StudentProfile
		bind:this={profil}
		student={leser}
		defaultTab="ausleihen"
		onMerged={omniboxStore.uebernimmZusammengefuehrt}
		onDeselect={abwaehlen}
		onReturnClick={(barcode) => omniboxStore.gibZurueck(barcode, () => profil?.reloadProfile())}
	/>
{/if}
