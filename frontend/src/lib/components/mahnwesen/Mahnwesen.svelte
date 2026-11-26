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
	import Icon from '../ui/Icon.svelte';
	import { anzahlText } from '../../utils/format.js';

	// Vor „Alle anmahnen" steht ein Dialog statt eines window.confirm, das nur alles oder
	// nichts kennt. Er liegt auf oberster Ebene, nicht in der Aktionszeile: Ein Overlay hat in
	// einem Flex-Container mit print:hidden nichts verloren. Die Knopfzeile steht unter der
	// Suchpille (MahnwesenSuchleiste), damit die Pille beginnt wie auf jeder anderen Seite.
	let mahnlaufOffen = $state(false);
	// Der Bescheid-Dialog trägt die ID des einen markierten Schülers; leer = zu.
	let bescheidFuer = $state('');
	// Schreiben verlangt manage_claims wie die Route (UI entscheidet nach Recht, nicht
	// nach Rolle); LESEN darf jeder, der das Mahnwesen sieht.
	const darfBescheid = $derived(hatRecht(authStore.currentUser, 'manage_claims'));
	// Mahnen verlangt send_dunning, nicht view_students, an dem diese Seite hängt: „Alle
	// anmahnen" (POST /api/mail/send-bulk-overdue) verschickt, „Mahnbriefe drucken"
	// (POST /api/admin/mahnungen/bulk-print) zählt die Mahnung. Sichtbarkeit einer Aktion =
	// hatRecht(user, '<Recht der Route>') — frontend-hygiene-rechte.test.js.
	const darfMahnen = $derived(hatRecht(authStore.currentUser, 'send_dunning'));

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
			<Icon name="warning" groesse={40} class="mt-0.5" />
			<div>
				<h2 class="text-lg">Mahnwesen blockiert</h2>
				<p class="text-sm mt-1">
					Auf diesem Rechner {offlineSync.wartendeAnzahl === 1 ? 'wartet' : 'warten'} noch
					<strong>{anzahlText(offlineSync.wartendeAnzahl, 'Vorgang', 'Vorgänge')}</strong> auf Übertragung
				</p>
				<p class="text-xs mt-2">
					Sobald die Verbindung zum Server zurück ist, geschieht das von selbst. Danach ist das
					Mahnwesen wieder frei.
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
	beschreibung="Wähle die Klassen aus, für die Mahnungen generiert werden sollen"
	aktion="anmahnen"
	hinweis="Leer lassen = an die regulären Klassenleitungen. Der Namensteil genügt, die Schul-Domäne wird ergänzt."
	klassen={mahnwesenStore.versandKlassen}
	onclose={() => (mahnlaufOffen = false)}
	onconfirm={(auswahl) => {
		mahnlaufOffen = false;
		mahnwesenStore.mahnlauf.sende(auswahl);
	}}
/>
