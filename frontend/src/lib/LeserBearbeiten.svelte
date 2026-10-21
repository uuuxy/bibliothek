<script>
	import LeserakteFelder from './components/leser/LeserakteFelder.svelte';
	import { erzeugeKlassenVorschlaege } from './components/leser/klassenVorschlaege.svelte.js';
	import { onMount } from 'svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import Button from './components/ui/Button.svelte';
	import { snackbars } from './stores/snackbars.svelte.js';
	import { verwerfenBestaetigen } from './stores/bestaetigung.svelte.js';
	import { schuetzeUngespeichertes } from './stores/verlassenSchutz.svelte.js';
	import { useLeserakteFormular } from './useLeserakteFormular.svelte.js';
	import { leserArtText } from './leserArt.js';
	import { Check, ChevronLeft } from '@lucide/svelte';

	// Die Klasse wird gewählt, nicht getippt, damit kein Tippfehler eine Klasse anlegt.
	const klassenListe = erzeugeKlassenVorschlaege();
	onMount(klassenListe.lade);

	/**
	 * @type {{
	 *   leser: any,
	 *   onClose: () => void,
	 *   onSave: () => void,
	 * }}
	 */
	let { leser, onClose, onSave } = $props();

	// Getter statt Werte: So liest der Hook bei jedem Zugriff das aktuelle Prop. Direkt
	// übergeben wären es Schnappschüsse vom Aufbau der Komponente — `speichere()` hätte das
	// PATCH dann an den zuvor geöffneten Schüler geschickt. Siehe useLeserakteFormular.
	//
	// Das Hook-Objekt wird GEHALTEN und nicht ganz auseinandergenommen: `eingaben` ist ein
	// Proxy und überlebt die Destrukturierung, ein einfacher Wert nicht. `speichert` und
	// `kontoVorhanden` sind Getter auf $state — einmal herausdestrukturiert, stünde für
	// immer der Wert vom Aufbau der Komponente da (der Speichern-Knopf hätte nie
	// „Speichert…" gezeigt). Gelesen werden sie deshalb über `formular.`, dort wo sie
	// gebraucht werden.
	//
	// Die Meldungen gehen an die gemeinsame Anzeige (SnackbarAnzeige.svelte): Die Maske schließt
	// mit dem Speichern, eine eigene Meldung verschwände mit ihr.
	const formular = useLeserakteFormular({
		getStudent: () => leser,
		onSave: () => onSave(),
		showSnackbar: (msg, type) => snackbars.addToast(msg, type)
	});
	const { eingaben, uebernimmLeser, speichere, hatUngespeichertes } = formular;

	// Der Effekt verfolgt `student` über den Getter in uebernimmLeser — wechselt das Prop,
	// wird das Formular neu befüllt statt die alten Werte zu behalten.
	$effect(() => {
		uebernimmLeser();
	});

	// Ungespeichertes geht nur nach Rückfrage verloren, wie an der Buchmaske: am Zurück-Knopf,
	// beim Wechsel der Seite und beim Neuladen oder Schließen des Fensters.
	schuetzeUngespeichertes(hatUngespeichertes);
	async function zurueck() {
		if (hatUngespeichertes() && !(await verwerfenBestaetigen())) return;
		onClose();
	}
</script>

<!-- Full Page View (Replaces the side sheet) -->
<div class="w-full h-full bg-surface-container-lowest flex flex-col animate-fade-in">
	<!-- ── Header ─────────────────────────────────────────────────────────── -->
	<header
		class="shrink-0 flex items-center justify-between gap-4 px-8 py-5 border-b border-outline-variant"
	>
		<div class="flex items-center gap-4 min-w-0">
			<!-- Back Button -->
			<button
				onclick={zurueck}
				aria-label="Zurück"
				data-tip="Zurück"
				class="icon-btn h-10 w-10 shrink-0 text-on-surface-variant"
			>
				<ChevronLeft class="w-5 h-5" aria-hidden="true" />
			</button>

			<div class="min-w-0">
				<!-- Der Titel nennt die Art statt fest „Schüler": Hier wird jeder Leser bearbeitet,
				     auch eine Lehrkraft. -->
				<h2 class="text-xl font-black text-on-surface leading-tight">
					{leserArtText(leser?.art)} bearbeiten
				</h2>
				<p class="text-xs text-on-surface-variant font-medium mt-0.5">
					{leser?.vorname}
					{leser?.nachname}{leser?.barcode_id ? ` · ${leser.barcode_id}` : ''}
				</p>
			</div>
		</div>

		<div class="flex items-center gap-3 shrink-0">
			<Button size="lg" onclick={speichere} disabled={formular.speichert} class="px-6">
				{#if formular.speichert}
					<Ladekreis size="sm" farbe="aktuell" />
					Speichert…
				{:else}
					<Check class="w-4 h-4" aria-hidden="true" />
					Speichern
				{/if}
			</Button>
		</div>
	</header>

	<!-- ── Scrollable Body ────────────────────────────────────────────────── -->
	<div class="flex-1 overflow-y-auto px-8 py-6 space-y-8">
		<LeserakteFelder
			{eingaben}
			klassen={klassenListe.liste}
			klassenFehler={klassenListe.ladefehler}
			lusdVerknuepft={!!leser?.lusd_id}
			kontoVorhanden={formular.kontoVorhanden}
		/>

		<!-- Bottom spacing -->
		<div class="h-4"></div>
	</div>
</div>
