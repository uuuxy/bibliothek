<script>
	import LeserEditFelder from './components/students/LeserEditFelder.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import Snackbar from './components/ui/Snackbar.svelte';
	import Button from './components/ui/Button.svelte';
	import { useStudentEditForm } from './useStudentEditForm.svelte.js';
	import { leserArtText } from './leserArt.js';
	import { Check, ChevronLeft } from '@lucide/svelte';

	/**
	 * @type {{
	 *   student: any,
	 *   onClose: () => void,
	 *   onSave: () => void,
	 * }}
	 */
	let { student, onClose, onSave } = $props();

	/** @type {{ msg: string, type: 'success' | 'error' } | null} */
	let snackbar = $state(null);
	/** @type {ReturnType<typeof setTimeout> | null} */
	let snackbarTimer = null;

	/**
	 * Show a self-dismissing snackbar.
	 * @param {string} msg
	 * @param {'success'|'error'} type
	 */
	function showSnackbar(msg, type = 'success') {
		if (snackbarTimer) clearTimeout(snackbarTimer);
		snackbar = { msg, type };
		snackbarTimer = setTimeout(() => {
			snackbar = null;
		}, 3000);
	}

	// Getter statt Werte: So liest der Hook bei jedem Zugriff das aktuelle Prop. Direkt
	// übergeben wären es Schnappschüsse vom Aufbau der Komponente — `save()` hätte das
	// PATCH dann an den zuvor geöffneten Schüler geschickt. Siehe useStudentEditForm.
	//
	// Das Hook-Objekt wird GEHALTEN und nicht ganz auseinandergenommen: `formData` ist ein
	// Proxy und überlebt die Destrukturierung, ein einfacher Wert nicht. `saving` und
	// `kontoVorhanden` sind Getter auf $state — einmal herausdestrukturiert, stünde für
	// immer der Wert vom Aufbau der Komponente da (der Speichern-Knopf hätte nie
	// „Speichert…" gezeigt). Gelesen werden sie deshalb über `form.`, dort wo sie
	// gebraucht werden.
	const form = useStudentEditForm({
		getStudent: () => student,
		onSave: () => onSave(),
		showSnackbar
	});
	const { formData, syncData, save } = form;

	// Der Effekt verfolgt `student` über den Getter in syncData — wechselt das Prop,
	// wird das Formular neu befüllt statt die alten Werte zu behalten.
	$effect(() => {
		syncData();
	});
</script>

<!-- Snackbar -->
<Snackbar {snackbar} />

<!-- Full Page View (Replaces the side sheet) -->
<div class="w-full h-full bg-white flex flex-col animate-fade-in">
	<!-- ── Header ─────────────────────────────────────────────────────────── -->
	<header
		class="shrink-0 flex items-center justify-between gap-4 px-8 py-5 border-b border-slate-100"
	>
		<div class="flex items-center gap-4 min-w-0">
			<!-- Back Button -->
			<button
				onclick={onClose}
				aria-label="Zurück"
				class="w-10 h-10 shrink-0 flex items-center justify-center rounded-xl bg-slate-50
               text-slate-500 hover:text-slate-800 hover:bg-slate-100 transition-colors cursor-pointer"
			>
				<ChevronLeft class="w-5 h-5" aria-hidden="true" />
			</button>

			<div class="min-w-0">
				<!-- Der Titel nennt die Art statt fest „Schüler": In dieser Datei steht seit
				     dem 16.09.2026 jeder Leser, und „Schüler bearbeiten" über der Akte einer
				     Lehrkraft ist schlicht eine falsche Auskunft. -->
				<h2 class="text-xl font-black text-slate-900 leading-tight">
					{leserArtText(student?.art)} bearbeiten
				</h2>
				<p class="text-xs text-slate-500 font-medium mt-0.5">
					{student?.vorname}
					{student?.nachname}{student?.barcode_id ? ` · ${student.barcode_id}` : ''}
				</p>
			</div>
		</div>

		<div class="flex items-center gap-3 shrink-0">
			<Button size="lg" onclick={save} disabled={form.saving} class="px-6">
				{#if form.saving}
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
		<LeserEditFelder
			{formData}
			lusdVerknuepft={!!student?.lusd_id}
			kontoVorhanden={form.kontoVorhanden}
		/>

		<!-- Bottom spacing -->
		<div class="h-4"></div>
	</div>
</div>
