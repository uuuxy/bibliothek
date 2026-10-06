<script>
	import InventurCatalog from '../inventur/routes/+page.svelte';
	import InventurAdmin from '../inventur/routes/admin/+page.svelte';
	import { appState } from '../inventur/lib/store.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';
	import GeraeteVerwaltung from './components/GeraeteVerwaltung.svelte';
	import Reiter from './components/ui/Reiter.svelte';

	// Wer mit einem Titel zum Bearbeiten kommt, beginnt in der Titel-Verwaltung: Der Katalog
	// stünde sonst einen Takt lang da und lüde dabei seine ganze Liste.
	let activeView = $state(appState.requestAdminView ? 'admin' : 'catalog'); // "catalog" | "admin" | "geraete"

	$effect(() => {
		if (appState.requestAdminView) {
			activeView = 'admin';
			appState.requestAdminView = false;
		}
	});
</script>

<PageShell>
	<Reiter
		etikett="Medienkatalog Navigation"
		aktiv={activeView}
		onwahl={(id) => (activeView = id)}
		reiter={[
			{ id: 'catalog', label: 'Suche & Filter' },
			{ id: 'admin', label: 'Titel-Verwaltung' },
			{ id: 'geraete', label: 'Geräte' }
		]}
	/>

	<!-- Content -->
	<div class="w-full">
		{#if activeView === 'catalog'}
			<InventurCatalog />
		{:else if activeView === 'admin'}
			<InventurAdmin />
		{:else if activeView === 'geraete'}
			<GeraeteVerwaltung />
		{/if}
	</div>
</PageShell>
