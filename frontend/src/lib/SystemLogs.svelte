<script>
	import Logbuch from './Logbuch.svelte';
	import VerwaltungsProtokoll from './VerwaltungsProtokoll.svelte';
	import TresenAuskunft from './TresenAuskunft.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import { hatRecht } from './menue.js';

	let activeTab = $state('system');
	// Das Admin-Audit-Log liest GET /api/admin/auditlog, und die Route verlangt
	// manage_users — der Reiter folgt demselben Recht, nicht der Rolle.
	const darfAdminLog = $derived(hatRecht(authStore.currentUser, 'manage_users'));
	// Tresen-Auskunft: GET /api/audit/tresen-auskunft verlangt audit_details.
	const darfTresen = $derived(hatRecht(authStore.currentUser, 'audit_details'));

	const reiter = $derived([
		{ id: 'system', label: 'Allgemeines Logbuch' },
		...(darfAdminLog ? [{ id: 'admin', label: 'Admin-Audit-Log' }] : []),
		...(darfTresen ? [{ id: 'tresen', label: 'Tresen-Auskunft' }] : [])
	]);
</script>

<Seitenrahmen>
	<Reiter etikett="System-Logs" {reiter} aktiv={activeTab} onwahl={(id) => (activeTab = id)} />

	<div class="flex-1 overflow-y-auto">
		{#if activeTab === 'system'}
			<div class="animate-fade-in h-full">
				<Logbuch />
			</div>
		{:else if activeTab === 'admin' && darfAdminLog}
			<div class="animate-fade-in h-full">
				<VerwaltungsProtokoll />
			</div>
		{:else if activeTab === 'tresen' && darfTresen}
			<div class="animate-fade-in h-full p-6">
				<TresenAuskunft />
			</div>
		{/if}
	</div>
</Seitenrahmen>
