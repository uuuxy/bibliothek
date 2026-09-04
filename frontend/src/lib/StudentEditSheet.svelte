<script>
	import Feld from './components/ui/Feld.svelte';
	import Snackbar from './components/ui/Snackbar.svelte';
	import Button from './components/ui/Button.svelte';
	import { useStudentEditForm } from './useStudentEditForm.svelte.js';
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
	const { formData, saving, syncData, save } = useStudentEditForm({
		getStudent: () => student,
		onSave: () => onSave(),
		showSnackbar
	});

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
				<h2 class="text-xl font-black text-slate-900 leading-tight">Schüler bearbeiten</h2>
				<p class="text-xs text-slate-500 font-medium mt-0.5">
					{student?.vorname}
					{student?.nachname} · {student?.barcode_id}
				</p>
			</div>
		</div>

		<div class="flex items-center gap-3 shrink-0">
			<Button size="lg" onclick={save} disabled={saving} class="px-6">
				{#if saving}
					<div
						class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
					></div>
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
		<!-- ── Persönliche Daten ──────────────────────────────── -->
		<section>
			<h3 class="text-base font-medium text-slate-500 mb-4 flex items-center gap-2">
				<div class="w-2.5 h-2.5 rounded-full bg-slate-300"></div>
				Persönliche Daten
			</h3>

			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<Feld id="vorname" label="Vorname" bind:value={formData.vorname} feld="font-semibold" />
				<Feld id="nachname" label="Nachname" bind:value={formData.nachname} feld="font-semibold" />
				<Feld
					id="geburtsdatum"
					label="Geburtsdatum"
					type="date"
					bind:value={formData.geburtsdatum}
				/>
				<!-- LUSD-ID ist kontrolliert nachtragbar: nur setzbar, solange sie leer ist
				     (Waise adoptieren). Ist sie bereits gesetzt, bleibt sie schreibgeschützt —
				     der Server lehnt eine Änderung/Leerung ohnehin ab (kontrollierter Pfad). -->
				<Feld
					id="lusd_id"
					label="LUSD-ID"
					bind:value={formData.lusd_id}
					feld="font-mono"
					disabled={!!student.lusd_id}
					hint={student.lusd_id
						? 'Bereits mit der LUSD verknüpft — Änderung nur über den Import.'
						: 'Nachtragbar: verknüpft diesen Schüler mit der LUSD.'}
				/>
			</div>
		</section>

		<!-- ── Schuldaten ─────────────────────────────────────── -->
		<section>
			<h3 class="text-base font-medium text-slate-500 mb-4 flex items-center gap-2">
				<div class="w-2.5 h-2.5 rounded-full bg-slate-300"></div>
				Schuldaten
			</h3>

			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<Feld id="klasse" label="Klasse" bind:value={formData.klasse} feld="font-semibold" />
				<Feld
					id="barcode"
					label="Schüler-ID / Barcode"
					bind:value={formData.barcode_id}
					feld="font-mono"
				/>
				<Feld
					id="abgangsjahr"
					label="Abgangsjahr"
					type="number"
					bind:value={formData.abgaenger_jahr}
					feld="font-semibold"
				/>

				<!-- Kein Status-Dropdown: „status" ist ein abgeleiteter Lesewert
				     (aktiv/gesperrt/abgaenger aus ist_gesperrt/ist_abgaenger) ohne eigene
				     DB-Spalte. Sperren läuft übers Lock-Modal, Abgänger über das Abgangsjahr —
				     ein editierbares Feld hier wurde vom Backend still verworfen. -->
			</div>
		</section>

		<!-- ── Kontaktdaten ────────────────────────────────────── -->
		<section>
			<h3 class="text-base font-medium text-slate-500 mb-4 flex items-center gap-2">
				<div class="w-2.5 h-2.5 rounded-full bg-blue-400"></div>
				Kontaktdaten
			</h3>

			<!-- EIN flaches Raster, keine Hülle: Eine <div>-Hülle spannt eine Rasterzeile, das
			     Feld braucht drei (Subgrid) — es rutscht aus der Reihe (feld-huellen.test.js). -->
			<div class="grid grid-cols-4 gap-4 md:grid-cols-8">
				<Feld
					id="strasse"
					label="Straße"
					class="col-span-3"
					bind:value={formData.strasse}
					placeholder="Musterstraße"
				/>
				<Feld id="hausnummer" label="Nr." bind:value={formData.hausnummer} placeholder="12a" />
				<Feld
					id="plz"
					label="PLZ"
					class="col-span-2"
					bind:value={formData.plz}
					placeholder="12345"
					maxlength={5}
					feld="font-mono"
				/>
				<Feld
					id="ort"
					label="Ort"
					class="col-span-2"
					bind:value={formData.ort}
					placeholder="Musterstadt"
				/>
				<Feld
					id="email"
					label="Eltern E-Mail"
					class="col-span-4"
					type="email"
					bind:value={formData.eltern_email}
					placeholder="eltern@schule.de"
				/>
			</div>
		</section>

		<!-- Bottom spacing -->
		<div class="h-4"></div>
	</div>
</div>
