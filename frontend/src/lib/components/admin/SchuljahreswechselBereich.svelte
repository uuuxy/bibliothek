<script>
	import LusdImportView from '../students/LusdImportView.svelte';
	import PromoteStudentsView from '../students/PromoteStudentsView.svelte';
	import SommerferienKategorie from '../settings/kategorien/SommerferienKategorie.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menu.js';

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	// Drei Werkzeuge, drei Rechte — jedes folgt seiner Route:
	//   LUSD-Abgleich  POST /api/lusd/preview|import  → import_students
	//   Versetzung     POST /api/students/promote     → manage_students_admin
	//   Sommerferien   PUT  /api/einstellungen        → manage_settings
	// Wer nur eines davon hat, sieht nur dieses; die Kategorie „LUSD & Versetzung"
	// selbst öffnet sich mit einem der Rechte (kategorien.js).
	const darfLusd = $derived(hatRecht(authStore.currentUser, 'import_students'));
	const darfVersetzen = $derived(hatRecht(authStore.currentUser, 'manage_students_admin'));
	const darfEinstellen = $derived(hatRecht(authStore.currentUser, 'manage_settings'));
</script>

{#if darfLusd || darfVersetzen || darfEinstellen}
	<!-- Inhalt der Einstellungs-Kategorie „LUSD & Versetzung"; Titel und Beitext kommen
	     vom KategorieRahmen. -->
	<div class="divide-y divide-outline-variant [&>*+*]:pt-8 [&>*+*]:mt-8">
		{#if darfLusd}
			<LusdImportView />
		{/if}

		{#if darfVersetzen}
			<PromoteStudentsView />
		{/if}

		{#if darfEinstellen}
			<SommerferienKategorie {daten} {onSaved} />
		{/if}
	</div>
{/if}
