<script>
	import { mahnwesenStore } from './stores/mahnwesen.svelte.js';
	import { offlineSync } from './stores/offlineSync.svelte.js';
	import MahnwesenFilters from './components/mahnwesen/MahnwesenFilters.svelte';
	import MahnwesenTable from './components/mahnwesen/MahnwesenTable.svelte';
	import BescheideTabelle from './components/mahnwesen/BescheideTabelle.svelte';
	import BescheidDialog from './components/mahnwesen/BescheidDialog.svelte';
	import { bescheideStore } from './stores/bescheide.svelte.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menu.js';
	import KlassenVersandDialog from './components/ui/KlassenVersandDialog.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import { TriangleAlert } from '@lucide/svelte';

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
	// Der Bescheid-Dialog trägt die ID des einen markierten Schülers; leer = zu.
	let bescheidFuer = $state('');
	// Schreiben verlangt edit_students wie die Route (UI entscheidet nach Recht, nicht
	// nach Rolle); LESEN darf jeder, der das Mahnwesen sieht.
	const darfBescheid = $derived(hatRecht(authStore.currentUser, 'edit_students'));
	// Mahnen verlangt create_orders, nicht view_students, an dem diese Seite hängt: „Alle
	// anmahnen" (POST /api/mail/send-bulk-overdue) verschickt, „Mahnbriefe drucken"
	// (POST /api/admin/mahnungen/bulk-print) zählt die Mahnung. Sichtbarkeit einer Aktion =
	// hatRecht(user, '<Recht der Route>') — frontend-hygiene-rechte.test.js.
	const darfMahnen = $derived(hatRecht(authStore.currentUser, 'create_orders'));

	$effect(() => {
		if (offlineSync.pendingCount === 0) {
			mahnwesenStore.fetchData();
		}
	});

	// Die Zahl am Reiter „Schadensersatz" steht schon vor dem ersten Blick darauf — sonst
	// zeigte er 0, solange niemand hineingesehen hat, und wäre damit eine Falschaussage.
	$effect(() => {
		if (!bescheideStore.geladen) {
			bescheideStore.lade();
		}
	});
</script>

<div class="w-full h-full flex flex-col">
	{#if offlineSync.pendingCount > 0}
		<div
			class="p-4 bg-error-container text-on-error-container flex items-start gap-4 animate-fade-in w-full"
		>
			<TriangleAlert class="mt-0.5 h-8 w-8 shrink-0" aria-hidden="true" />
			<div>
				<h2 class="text-lg font-bold">Mahnwesen blockiert</h2>
				<p class="text-sm mt-1">
					Es befinden sich noch <strong
						>{offlineSync.pendingCount} ungesynchronisierte Offline-Ausleihe(n)/Rückgabe(n)</strong
					> auf diesem Gerät.
				</p>
				<p class="text-xs mt-2">
					Bitte stelle die Internetverbindung wieder her. Das System synchronisiert die Daten
					automatisch im Hintergrund, sobald du wieder online bist. Danach wird das Mahnwesen
					automatisch wieder freigegeben.
				</p>
			</div>
		</div>
	{:else}
		<PageShell>
			<MahnwesenFilters
				onMahnlauf={() => (mahnlaufOffen = true)}
				onBescheid={(id) => (bescheidFuer = id)}
				{darfBescheid}
				{darfMahnen}
			/>
			{#if mahnwesenStore.activeFilter === 'Schadensersatz'}
				<BescheideTabelle darfSchreiben={darfBescheid} onBescheid={(id) => (bescheidFuer = id)} />
			{:else}
				<MahnwesenTable />
			{/if}
		</PageShell>
	{/if}
</div>

{#if bescheidFuer}
	<!-- Auf oberster Ebene wie der Mahnlauf-Dialog: Ein Overlay gehört nicht in den
	     Flex-Container mit print:hidden. {#key}: je Schüler ein frisches Formular. -->
	{#key bescheidFuer}
		<BescheidDialog
			schuelerId={bescheidFuer}
			onclose={() => (bescheidFuer = '')}
			onErstellt={async () => {
				await bescheideStore.lade();
				await mahnwesenStore.fetchData();
			}}
		/>
	{/key}
{/if}

<KlassenVersandDialog
	open={mahnlaufOffen}
	titel="Mahnlauf konfigurieren"
	variant="danger-solid"
	beschreibung="Wähle die Klassen aus, für die Mahnungen generiert werden sollen."
	aktion="anmahnen"
	hinweis="Leer lassen = an die regulären Klassenleitungen. Der Namensteil genügt, die Schul-Domäne wird ergänzt."
	klassen={mahnwesenStore.versandKlassen}
	onclose={() => (mahnlaufOffen = false)}
	onconfirm={(auswahl) => {
		mahnlaufOffen = false;
		mahnwesenStore.sendBulkOverdueMails(auswahl);
	}}
/>
