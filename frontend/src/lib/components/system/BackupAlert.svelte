<!-- @component BackupAlert — der Backup-Wächter als Inline-Alert über dem Inhalt.
     Vorher saß diese Meldung als flächig gefüllte Warnkarte im Sidebar-Fuß, direkt
     über „Abmelden". Drei Dinge sind daran anders:

     1. Der Ort. Die Sidebar ist Navigation; ein dauerhafter Konfigurationsfehler wird
        dort zwischen Menüpunkten nach drei Tagen zur Tapete. Der Alert steht jetzt über
        dem Inhalt, wo er zum Bildschirm gehört und nicht zur Wegweisung.
     2. Die Form. Kein Farbblock mit Emoji, sondern ein 3-px-Schweregrad-Streifen links
        auf ruhigem Grund plus ein Icon — auffällig durch Kante und Position.
     3. Die Handlung. Ein Alert ohne Weg zur Behebung ist eine Sackgasse. Der Text sagt,
        was zu tun ist, und der Knopf führt direkt auf den richtigen Reiter. -->
<script>
	import { onMount } from 'svelte';
	import { AlertTriangle, ArrowRight } from '@lucide/svelte';
	import { backupStatus } from '../../stores/backupStatus.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';

	onMount(() => backupStatus.load());

	const critical = $derived(backupStatus.data?.status === 'critical');

	function openDatenverwaltung() {
		uiStore.requestedSettingsTab = 'Datenverwaltung';
		uiStore.activeTab = 'settings';
	}
</script>

{#if backupStatus.needsAction}
	<div
		role="alert"
		class="no-print mb-5 flex items-start gap-3 rounded-md border border-slate-200 border-l-[3px] bg-white py-3 pr-4 pl-3.5 shadow-xs
			{critical ? 'border-l-rose-600' : 'border-l-amber-500'}"
	>
		<AlertTriangle
			class="mt-0.5 h-4 w-4 shrink-0 {critical ? 'text-rose-600' : 'text-amber-600'}"
			aria-hidden="true"
		/>
		<div class="min-w-0 flex-1">
			<p class="text-sm font-semibold text-slate-800">{backupStatus.message}</p>
			<p class="mt-0.5 text-xs leading-relaxed text-slate-500">{backupStatus.hint}</p>
		</div>
		<Button variant="secondary" size="sm" onclick={openDatenverwaltung} class="mt-0.5 shrink-0">
			Datenverwaltung öffnen
			<ArrowRight class="h-3.5 w-3.5" />
		</Button>
	</div>
{/if}
