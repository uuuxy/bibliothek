<script>
	import InventurCatalog from '../inventur/routes/+page.svelte';
	import InventurAdmin from '../inventur/routes/admin/+page.svelte';
	import { appState } from '../inventur/lib/store.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';
	import GeraeteVerwaltung from './components/GeraeteVerwaltung.svelte';

	let activeView = $state('catalog'); // "catalog" | "admin" | "geraete"

	$effect(() => {
		if (appState.requestAdminView) {
			activeView = 'admin';
			appState.requestAdminView = false;
		}
	});
</script>

<PageShell>
	<!-- Google-style underline Tabs.
	     `role="tablist"` stand hier nicht, obwohl die Knoepfe darin `role="tab"` tragen —
	     ein Reiterband ohne Band. Nachgetragen am 04.09.2026, als das Startlinien-Gate die
	     Baender ueber den Suchpillen zaehlte und dieses als namenlosen Block sah. Aus dem
	     <nav> wurde dabei ein <div>: Ein Reiterband ist keine Navigations-Landmarke, und
	     Svelte weist `role="tablist"` auf <nav> zu Recht zurueck. Die Trennlinie sass in
	     einer zusaetzlichen Huelle darum; sie gehoert ans Band selbst, so wie in
	     ui/Reiter.svelte — sonst steht ueber der Suchpille formal ein namenloser Kasten
	     und kein Reiterband. -->
	<div
		class="flex gap-6 border-b border-slate-200"
		role="tablist"
		aria-label="Medienkatalog Navigation"
	>
		<button
			onclick={() => (activeView = 'catalog')}
			class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {activeView ===
			'catalog'
				? 'text-blue-600'
				: 'text-slate-500 hover:text-slate-700'}"
			role="tab"
			aria-selected={activeView === 'catalog'}
		>
			Suche & Filter
			{#if activeView === 'catalog'}
				<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
			{/if}
		</button>
		<button
			onclick={() => (activeView = 'admin')}
			class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {activeView ===
			'admin'
				? 'text-blue-600'
				: 'text-slate-500 hover:text-slate-700'}"
			role="tab"
			aria-selected={activeView === 'admin'}
		>
			Titel-Verwaltung
			{#if activeView === 'admin'}
				<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
			{/if}
		</button>
		<!-- Neues Markup nutzt die M3-Rollen (Paletten-Ratsche) — die Nachbar-Tabs sind
			     Bestand und werden bei ihrer eigenen Umstellung nachgezogen. -->
		<button
			onclick={() => (activeView = 'geraete')}
			class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {activeView ===
			'geraete'
				? 'text-primary'
				: 'text-on-surface-variant hover:text-on-surface'}"
			role="tab"
			aria-selected={activeView === 'geraete'}
		>
			Geräte
			{#if activeView === 'geraete'}
				<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-primary rounded-full"></span>
			{/if}
		</button>
	</div>

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
