<script>
	import LusdImportView from '../students/LusdImportView.svelte';
	import PromoteStudentsView from '../students/PromoteStudentsView.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menu.js';

	// Zwei Werkzeuge, zwei Rechte — jedes folgt seiner Route:
	//   LUSD-Abgleich  POST /api/lusd/preview|import  → import_students
	//   Versetzung     POST /api/students/promote     → manage_students_admin
	// Wer nur eines davon hat, sieht nur dieses; der Reiter „LUSD & Versetzung" der
	// Seite „Schuljahreswechsel" (Schuljahr.svelte) öffnet sich mit einem der Rechte.
	const darfLusd = $derived(hatRecht(authStore.currentUser, 'import_students'));
	const darfVersetzen = $derived(hatRecht(authStore.currentUser, 'manage_students_admin'));
</script>

{#if darfLusd || darfVersetzen}
	<!-- Reiter „LUSD & Versetzung" der Seite „Schuljahreswechsel" (bis 05.09.2026 eine
	     Einstellungs-Kategorie). -->
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
