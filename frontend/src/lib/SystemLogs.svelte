<script>
	import AuditLog from './AuditLog.svelte';
	import AdminAuditLog from './AdminAuditLog.svelte';
	import TresenAuskunft from './TresenAuskunft.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';
	import { hatRecht } from './menu.js';

	let activeTab = $state('system');
	// Das Admin-Audit-Log liest GET /api/admin/auditlog, und die Route verlangt
	// manage_users — der Reiter folgt demselben Recht, nicht der Rolle.
	const darfAdminLog = $derived(hatRecht(authStore.currentUser, 'manage_users'));
	// Tresen-Auskunft: GET /api/audit/tresen-auskunft verlangt audit_details.
	const darfTresen = $derived(hatRecht(authStore.currentUser, 'audit_details'));
</script>

<PageShell>
	<!-- Reiter auf der Leinwand, nicht in einem eigenen weissen Balken — wie Mahnwesen,
	     Medienkatalog und Schuelerdatei. -->
	<div class="border-outline-variant shrink-0 border-b">
		<div class="mx-auto flex max-w-6xl gap-6">
			<button
				onclick={() => (activeTab = 'system')}
				class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'system'
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}"
			>
				Allgemeines Logbuch
			</button>
			{#if darfAdminLog}
				<button
					onclick={() => (activeTab = 'admin')}
					class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'admin'
						? 'border-blue-600 text-blue-700'
						: 'border-transparent text-slate-500 hover:text-slate-800'}"
				>
					Admin-Audit-Log
				</button>
			{/if}
			{#if darfTresen}
				<button
					onclick={() => (activeTab = 'tresen')}
					class="border-b-2 pb-3 text-sm font-semibold transition-colors {activeTab === 'tresen'
						? 'border-primary text-primary'
						: 'border-transparent text-on-surface-variant hover:text-on-surface'}"
				>
					Tresen-Auskunft
				</button>
			{/if}
		</div>
	</div>

	<div class="flex-1 overflow-y-auto">
		{#if activeTab === 'system'}
			<div class="animate-fade-in h-full">
				<AuditLog />
			</div>
		{:else if activeTab === 'admin' && darfAdminLog}
			<div class="animate-fade-in h-full">
				<AdminAuditLog />
			</div>
		{:else if activeTab === 'tresen' && darfTresen}
			<div class="animate-fade-in h-full p-6">
				<TresenAuskunft />
			</div>
		{/if}
	</div>
</PageShell>
