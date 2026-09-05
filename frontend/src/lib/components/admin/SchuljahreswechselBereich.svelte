<script>
	import LusdImportView from '../students/LusdImportView.svelte';
	import PromoteStudentsView from '../students/PromoteStudentsView.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menu.js';

	// Zwei Werkzeuge, zwei Rechte — jedes folgt seiner Route:
	//   LUSD-Abgleich  POST /api/lusd/preview|import  → import_students
	//   Versetzung     POST /api/students/promote     → manage_students_admin
	// Wer nur eines davon hat, sieht nur dieses; die Kategorie „Schuljahreswechsel"
	// selbst öffnet sich mit einem der Rechte (kategorien.js).
	const darfLusd = $derived(hatRecht(authStore.currentUser, 'import_students'));
	const darfVersetzen = $derived(hatRecht(authStore.currentUser, 'manage_students_admin'));
</script>

{#if darfLusd || darfVersetzen}
	<!-- Inhalt der Einstellungs-Kategorie „Schuljahreswechsel"; Titel und Beitext kommen
	     vom KategorieRahmen. -->
	<div class="space-y-10">
		{#if darfLusd}
			<LusdImportView />
		{/if}

		{#if darfVersetzen}
			<div class:pt-8={darfLusd} class:border-t={darfLusd} class="border-slate-100">
				<PromoteStudentsView />
			</div>
		{/if}
	</div>
{/if}
