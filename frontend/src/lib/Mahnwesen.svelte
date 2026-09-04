<script>
	import { mahnwesenStore } from './stores/mahnwesen.svelte.js';
	import { offlineSync } from './stores/offlineSync.svelte.js';
	import MahnwesenFilters from './components/mahnwesen/MahnwesenFilters.svelte';
	import MahnwesenTable from './components/mahnwesen/MahnwesenTable.svelte';
	import KlassenVersandDialog from './components/ui/KlassenVersandDialog.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import { Info, TriangleAlert } from '@lucide/svelte';

	// „Alle anmahnen" lief frueher gegen ein window.confirm: alles oder nichts, immer an
	// die hinterlegten Klassenleitungen. Der Dialog steht jetzt als Tuersteher davor —
	// und auf OBERSTER Ebene, nicht in der Aktionszeile: Ein Overlay hat in einem
	// Flex-Container mit print:hidden nichts verloren.
	//
	// Die Knopfzeile selbst sass bis zum 04.09.2026 im `aktionen`-Slot von PageShell und
	// damit UEBER den Reitern — als einzige Seite von sechzehn, mit der Folge, dass die
	// Suchpille hier 84 px tiefer begann als ueberall sonst. Sie steht jetzt unter der
	// Pille, siehe MahnwesenSuchleiste. Den Slot gibt es seitdem nicht mehr.
	let mahnlaufOffen = $state(false);

	$effect(() => {
		if (offlineSync.pendingCount === 0) {
			mahnwesenStore.fetchData();
		}
	});
</script>

{#if mahnwesenStore.globalErrorToast}
	<div
		class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-rose-600 text-white flex items-center gap-2"
	>
		<Info class="h-5 w-5" aria-hidden="true" />
		{mahnwesenStore.globalErrorToast}
	</div>
{/if}

{#if mahnwesenStore.ferienAktiv}
	<div
		class="w-full mb-6 p-4 bg-amber-50 border-b border-amber-200 flex items-start gap-3 animate-fade-in"
	>
		<TriangleAlert class="h-6 w-6 text-amber-600 mt-0.5 shrink-0" aria-hidden="true" />
		<div>
			<h3 class="text-base font-bold text-amber-900">Achtung: Schließzeit / Ferien aktiv!</h3>
			<p class="text-xs text-amber-800 mt-1">
				Das Mahnwesen ist aktuell pausiert. Grund: <strong
					>{mahnwesenStore.ferienBezeichnung}</strong
				>. E-Mails und PDF-Exporte sind währenddessen serverseitig blockiert.
			</p>
		</div>
	</div>
{/if}

<div class="w-full h-full flex flex-col">
	{#if offlineSync.pendingCount > 0}
		<div
			class="p-4 bg-rose-50 border-b border-rose-200 flex items-start gap-4 animate-fade-in w-full"
		>
			<div class="bg-rose-100 p-3 rounded-full shrink-0">
				<TriangleAlert class="h-8 w-8 text-rose-600" aria-hidden="true" />
			</div>
			<div>
				<h2 class="text-lg font-bold text-rose-900">Mahnwesen blockiert</h2>
				<p class="text-sm text-rose-800 mt-1">
					Es befinden sich noch <strong
						>{offlineSync.pendingCount} ungesynchronisierte Offline-Ausleihe(n)/Rückgabe(n)</strong
					> auf diesem Gerät.
				</p>
				<p class="text-xs text-rose-700 mt-2 bg-rose-100/50 p-2 rounded-lg inline-block">
					Bitte stelle die Internetverbindung wieder her. Das System synchronisiert die Daten
					automatisch im Hintergrund, sobald du wieder online bist. Danach wird das Mahnwesen
					automatisch wieder freigegeben.
				</p>
			</div>
		</div>
	{:else}
		<PageShell>
			<MahnwesenFilters onMahnlauf={() => (mahnlaufOffen = true)} />
			<MahnwesenTable />
		</PageShell>
	{/if}
</div>

<KlassenVersandDialog
	open={mahnlaufOffen}
	titel="Mahnlauf konfigurieren"
	variant="danger-solid"
	beschreibung="Wähle die Klassen aus, für die Mahnungen generiert werden sollen."
	aktion="anmahnen"
	hinweis="Leer lassen = an die regulären Klassenleitungen. Der Namensteil genügt, die Schul-Domäne wird ergänzt."
	klassen={mahnwesenStore.klassen}
	onclose={() => (mahnlaufOffen = false)}
	onconfirm={(auswahl) => {
		mahnlaufOffen = false;
		mahnwesenStore.sendBulkOverdueMails(auswahl);
	}}
/>
