<script>
	import KatalogSuche from '../inventur/routes/KatalogSuche.svelte';
	import TitelVerwaltung from '../inventur/routes/admin/TitelVerwaltung.svelte';
	import { appState } from '../inventur/lib/store.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
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

<Seitenrahmen>
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
			<KatalogSuche />
		{:else if activeView === 'admin'}
			<TitelVerwaltung />
		{:else if activeView === 'geraete'}
			<GeraeteVerwaltung />
		{/if}
	</div>
</Seitenrahmen>
