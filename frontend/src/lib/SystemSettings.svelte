<script>
	import { apiGet } from './apiFetch.js';
	import { onMount } from 'svelte';
	import MailTemplates from './MailTemplates.svelte';
	import DataManagement from './components/admin/DataManagement.svelte';
	import MailConfig from './components/admin/MailConfig.svelte';
	import PermissionManager from './PermissionManager.svelte';
	import SystemSettingsAllgemein from './SystemSettingsAllgemein.svelte';
	import SystemSettingsRouting from './SystemSettingsRouting.svelte';
	import { authStore } from './stores/authStore.svelte.js';

	// --- STATE ---
	let loading = $state(true);

	const isAdmin = $derived(authStore.currentUser?.rolle === 'admin');

	// Tabs
	const tabs = $derived(
		isAdmin
			? [
					'Allgemein',
					'Team & Rechte',
					'Mahnwesen-Routing',
					'Datenverwaltung',
					'Mail-Server',
					'System'
				]
			: ['Allgemein', 'Team & Rechte', 'Mahnwesen-Routing', 'System']
	);
	let activeTab = $state('Allgemein');

	// Global Settings (Allgemein)
	let ferienLeseclubAktiv = $state(false);
	let ferienLeseclubZieldatum = $state('');
	let lmfStichtag = $state('07-31');
	let maxAusleihenSchueler = $state(5);
	let fristBuchTage = $state(21);
	let fristMedienTage = $state(7);
	let maxOverdueDays = $state(14);
	let maxOverdueItems = $state(1);
	let bestellbedarfWarnungAktiv = $state(true);
	let bestellbedarfSchwelle = $state(3);

	// --- LOGIC ---

	async function loadSettings() {
		try {
			const data = await apiGet('/api/einstellungen');
			ferienLeseclubAktiv = data.ferien_leseclub_aktiv ?? false;
			ferienLeseclubZieldatum = data.ferien_leseclub_zieldatum ?? '';
			lmfStichtag = data.lmf_stichtag ?? '07-31';
			maxAusleihenSchueler = data.max_ausleihen_schueler ?? 5;
			fristBuchTage = data.frist_buch_tage ?? 21;
			fristMedienTage = data.frist_medien_tage ?? 7;
			maxOverdueDays = data.max_overdue_days ?? 14;
			maxOverdueItems = data.max_overdue_items ?? 1;
			bestellbedarfWarnungAktiv = data.bestellbedarf_warnung_aktiv ?? true;
			bestellbedarfSchwelle = data.bestellbedarf_schwelle ?? 3;
		} catch {
			/* use defaults */
		}
	}

	onMount(async () => {
		await loadSettings();
		loading = false;
	});
</script>

<div class="max-w-6xl mx-auto w-full space-y-6 text-slate-800 font-sans antialiased pb-20 pt-6">
	<!-- Header -->
	<div class="space-y-6">
		<!-- Tabs -->
		<div class="flex gap-4 border-b border-slate-200">
			{#each tabs as tab, _i (_i)}
				<button
					onclick={() => (activeTab = tab)}
					class="relative px-2 py-3 text-sm font-semibold transition-colors focus:outline-none {activeTab ===
					tab
						? 'text-blue-600'
						: 'text-slate-500 hover:text-slate-800'}"
				>
					{tab}
					{#if activeTab === tab}
						<div class="absolute bottom-0 left-0 w-full h-1 bg-blue-600 rounded-t-full"></div>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	{#if loading}
		<div class="py-20 flex justify-center items-center">
			<div
				class="w-10 h-10 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"
			></div>
		</div>
	{:else}
		<!-- Tab Content -->
		<div class="pt-2 animate-fade-in">
			<!-- TAB: ALLGEMEIN -->
			{#if activeTab === 'Allgemein'}
				<SystemSettingsAllgemein
					bind:ferienLeseclubAktiv
					bind:ferienLeseclubZieldatum
					bind:lmfStichtag
					bind:maxAusleihenSchueler
					bind:fristBuchTage
					bind:fristMedienTage
					bind:maxOverdueDays
					bind:maxOverdueItems
					bind:bestellbedarfWarnungAktiv
					bind:bestellbedarfSchwelle
				/>

				<!-- TAB: TEAM & RECHTE -->
			{:else if activeTab === 'Team & Rechte'}
				<section class="w-full">
					<h3 class="text-lg font-bold text-slate-900 mb-6">Account- und Rollenverwaltung</h3>
					<PermissionManager />
				</section>

				<!-- TAB: MAHNWESEN-ROUTING -->
			{:else if activeTab === 'Mahnwesen-Routing'}
				<SystemSettingsRouting />

				<!-- TAB: DATENVERWALTUNG -->
			{:else if activeTab === 'Datenverwaltung' && isAdmin}
				<DataManagement />

				<!-- TAB: MAIL-SERVER -->
			{:else if activeTab === 'Mail-Server' && isAdmin}
				<MailConfig />

				<!-- TAB: SYSTEM -->
			{:else if activeTab === 'System'}
				<section class="w-full">
					<h3 class="text-lg font-bold text-slate-900 mb-6">Mail-Templates</h3>
					<MailTemplates />
				</section>
			{/if}
		</div>
	{/if}
</div>
