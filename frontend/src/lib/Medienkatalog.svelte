<script>
	import KatalogSuche from '../inventur/routes/KatalogSuche.svelte';
	import TitelVerwaltung from '../inventur/routes/admin/TitelVerwaltung.svelte';
	import { appState } from '../inventur/lib/store.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import GeraeteVerwaltung from './components/GeraeteVerwaltung.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import { darfVerwerfen } from './stores/verlassenSchutz.svelte.js';

	// Wer mit einem Titel zum Bearbeiten kommt, beginnt in der Titel-Verwaltung: Der Katalog
	// stünde sonst einen Takt lang da und lüde dabei seine ganze Liste.
	let aktiverReiter = $state(appState.requestAdminView ? 'admin' : 'catalog'); // "catalog" | "admin" | "geraete"

	$effect(() => {
		if (appState.requestAdminView) {
			aktiverReiter = 'admin';
			appState.requestAdminView = false;
		}
	});

	// Ein anderer Reiter baut die Titel-Verwaltung ab. Ungespeichertes ihrer Maske fragt vorher,
	// über denselben Wächter wie der Wechsel der Seite (uiStore.verlassenSperre).
	/** @param {string} id */
	async function waehleReiter(id) {
		if (id === aktiverReiter || !(await darfVerwerfen())) return;
		aktiverReiter = id;
	}
</script>

<Seitenrahmen>
	<Reiter
		etikett="Medienkatalog Navigation"
		aktiv={aktiverReiter}
		onwahl={waehleReiter}
		reiter={[
			{ id: 'catalog', label: 'Suche & Filter' },
			{ id: 'admin', label: 'Titel-Verwaltung' },
			{ id: 'geraete', label: 'Geräte' }
		]}
	/>

	<!-- Content -->
	<div class="w-full">
		{#if aktiverReiter === 'catalog'}
			<KatalogSuche />
		{:else if aktiverReiter === 'admin'}
			<TitelVerwaltung />
		{:else if aktiverReiter === 'geraete'}
			<GeraeteVerwaltung />
		{/if}
	</div>
</Seitenrahmen>
