<script>
	import Logbuch from './Logbuch.svelte';
	import VerwaltungsProtokoll from './VerwaltungsProtokoll.svelte';
	import ThekenAuskunft from './ThekenAuskunft.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import { hatRecht } from './menue.js';

	let aktiverReiter = $state('system');
	// Das Admin-Audit-Log liest GET /api/admin/auditlog, und die Route verlangt
	// manage_users — der Reiter folgt demselben Recht, nicht der Rolle.
	const darfAdminLog = $derived(hatRecht(authStore.currentUser, 'manage_users'));
	// Theken-Auskunft: GET /api/audit/tresen-auskunft verlangt audit_details.
	const darfThekenAuskunft = $derived(hatRecht(authStore.currentUser, 'audit_details'));

	const reiter = $derived([
		{ id: 'system', label: 'Allgemeines Logbuch' },
		...(darfAdminLog ? [{ id: 'admin', label: 'Admin-Audit-Log' }] : []),
		...(darfThekenAuskunft ? [{ id: 'thekenauskunft', label: 'Theken-Auskunft' }] : [])
	]);
</script>

<Seitenrahmen>
	<Reiter
		etikett="System-Logs"
		{reiter}
		aktiv={aktiverReiter}
		onwahl={(id) => (aktiverReiter = id)}
	/>

	<div class="flex-1 overflow-y-auto">
		{#if aktiverReiter === 'system'}
			<div class="animate-fade-in h-full">
				<Logbuch />
			</div>
		{:else if aktiverReiter === 'admin' && darfAdminLog}
			<div class="animate-fade-in h-full">
				<VerwaltungsProtokoll />
			</div>
		{:else if aktiverReiter === 'thekenauskunft' && darfThekenAuskunft}
			<div class="animate-fade-in h-full p-6">
				<ThekenAuskunft />
			</div>
		{/if}
	</div>
</Seitenrahmen>
