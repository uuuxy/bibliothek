<script>
	import { onMount } from 'svelte';
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import { offlineSync } from '../../stores/offlineSync.svelte.js';
	import MahnwesenFilters from './MahnwesenFilters.svelte';
	import MahnwesenTabelle from './MahnwesenTabelle.svelte';
	import BescheideTabelle from './BescheideTabelle.svelte';
	import BescheidDialog from './BescheidDialog.svelte';
	import { bescheideStore } from '../../stores/bescheide.svelte.js';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menue.js';
	import KlassenVersandDialog from '../ui/KlassenVersandDialog.svelte';
	import Seitenrahmen from '../layout/Seitenrahmen.svelte';
	import { TriangleAlert } from '@lucide/svelte';

	// Vor „Alle anmahnen" steht ein Dialog statt eines window.confirm, das nur alles oder
	// nichts kennt. Er liegt auf oberster Ebene, nicht in der Aktionszeile: Ein Overlay hat in
	// einem Flex-Container mit print:hidden nichts verloren. Die Knopfzeile steht unter der
	// Suchpille (MahnwesenSuchleiste), damit die Pille beginnt wie auf jeder anderen Seite.
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
		if (offlineSync.wartendeAnzahl === 0) {
			mahnwesenStore.lade();
		}
	});

	// Die Zahl am Reiter „Schadensersatz" steht schon vor dem ersten Blick darauf — sonst
	// zeigte er 0, solange niemand hineingesehen hat, und wäre damit eine Falschaussage.
	// Bei jedem Besuch wie die Mahnliste: Der Store gilt für die Sitzung und zeigte sonst den
	// Stand des ersten Besuchs, ohne die Forderungen und Bescheide anderer Plätze.
	onMount(() => bescheideStore.lade());
</script>

<div class="w-full h-full flex flex-col">
	{#if offlineSync.wartendeAnzahl > 0}
		<div
			class="p-4 bg-error-container text-on-error-container flex items-start gap-4 animate-fade-in w-full"
		>
			<TriangleAlert class="mt-0.5 h-8 w-8 shrink-0" aria-hidden="true" />
			<div>
				<h2 class="text-lg font-bold">Mahnwesen blockiert</h2>
				<p class="text-sm mt-1">
					Es befinden sich noch <strong
						>{offlineSync.wartendeAnzahl} ungesynchronisierte Offline-Ausleihe(n)/Rückgabe(n)</strong
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
		<Seitenrahmen>
			<MahnwesenFilters
				onMahnlauf={() => (mahnlaufOffen = true)}
				onBescheid={(id) => (bescheidFuer = id)}
				{darfBescheid}
				{darfMahnen}
			/>
			{#if mahnwesenStore.stufenFilter === 'Schadensersatz'}
				<BescheideTabelle darfSchreiben={darfBescheid} onBescheid={(id) => (bescheidFuer = id)} />
			{:else}
				<MahnwesenTabelle />
			{/if}
		</Seitenrahmen>
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
				await mahnwesenStore.lade();
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
		mahnwesenStore.mahnlauf.sende(auswahl);
	}}
/>
