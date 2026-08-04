<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Button from '../ui/Button.svelte';
	import KlassenVersandDialog from '../ui/KlassenVersandDialog.svelte';
	import MahnwesenDruckMenue from './MahnwesenDruckMenue.svelte';
	import MahnwesenTabs from './MahnwesenTabs.svelte';
	import MahnwesenSuchleiste from './MahnwesenSuchleiste.svelte';

	// „Alle anmahnen" lief früher gegen ein window.confirm: alles oder nichts, immer
	// an die hinterlegten Klassenleitungen. Der Dialog steht jetzt als Türsteher davor.
	let mahnlaufOffen = $state(false);

	let countAlle = $derived(
		mahnwesenStore.klassen.reduce((/** @type {number} */ sum, /** @type {any} */ k) => sum + k.schueler.length, 0)
	);
</script>

<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
	<div class="min-w-0">
		<h1 class="text-2xl font-bold text-slate-800">Mahnwesen</h1>
		<p class="text-sm text-slate-500 mt-0.5">Überfällige Ausleihen nach Klassen sortiert.</p>
	</div>

	<!-- Rechte Aktionsleiste. Bei einer Auswahl übernimmt sie den Auswahl-Modus (wie Gmail/Drive):
	     Nur noch die auf die Markierung bezogenen Aktionen sind sichtbar. -->
	<div class="flex flex-wrap items-center gap-2 print:hidden shrink-0">
		{#if mahnwesenStore.selectedIds.size > 0}
			<Button
				variant="secondary"
				onclick={mahnwesenStore.deselectAllSchueler}
				aria-label="Auswahl aufheben"
				title="Auswahl aufheben"
				class="px-2 text-slate-500 hover:text-slate-700"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2.5"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</Button>
			<span class="text-sm font-semibold text-slate-700"
				>{mahnwesenStore.selectedIds.size} ausgewählt</span
			>
			<Button onclick={mahnwesenStore.printSelectedMahnungen} disabled={mahnwesenStore.pdfLoading}>
				{#if mahnwesenStore.pdfLoading}
					<div
						class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"
					></div>
				{:else}
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
						/>
					</svg>
				{/if}
				Mahnbriefe drucken
			</Button>
		{:else}
			<div class="flex items-center gap-1 bg-slate-100 p-1 rounded-xl">
				<button
					class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode ===
					'datum'
						? 'bg-white text-slate-800 shadow-sm'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => {
						mahnwesenStore.mahnMode = 'datum';
						mahnwesenStore.fetchData();
					}}
				>
					Datum
				</button>
				<button
					class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode ===
					'jahrgang'
						? 'bg-white text-slate-800 shadow-sm'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => {
						mahnwesenStore.mahnMode = 'jahrgang';
						mahnwesenStore.fetchData();
					}}
				>
					Jahrgang
				</button>
			</div>

			<Button
				variant="secondary"
				onclick={mahnwesenStore.fetchData}
				aria-label="Daten neu laden"
				data-tip="Daten neu laden"
				title="Neu laden"
				class="px-2 text-slate-500 hover:text-slate-700"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
					/>
				</svg>
			</Button>

			<MahnwesenDruckMenue />

			<!-- „Alle anmahnen" ist die EINZIGE echte E-Mail-Aktion → nur hier das Umschlag-Icon. -->
			{#if countAlle > 0}
				<Button
					variant="danger-solid"
					onclick={() => (mahnlaufOffen = true)}
					aria-label="Alle anmahnen – Mahnlauf konfigurieren und per E-Mail versenden"
					class="shrink-0"
				>
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
						/>
					</svg>
					Alle anmahnen
				</Button>
			{/if}
		{/if}
	</div>
</div>

<!-- Auf oberster Ebene, nicht in der Werkzeugleiste: Der Dialog ist ein Overlay und
     hat in einem Flex-Container mit print:hidden nichts verloren. -->
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

<!-- Register nach Dringlichkeit + Filterleiste. Beide hängen an derselben Bedingung:
     ohne geladene Daten gibt es nichts zu filtern. -->
{#if mahnwesenStore.data && !mahnwesenStore.loading}
	<MahnwesenTabs />
	<MahnwesenSuchleiste />
{/if}

