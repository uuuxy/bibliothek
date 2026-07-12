<script>
	import InputField from './components/ui/InputField.svelte';
	import Snackbar from './components/ui/Snackbar.svelte';
	import { useStudentEditForm } from './useStudentEditForm.svelte.js';

	/**
	 * @type {{
	 *   student: any,
	 *   onClose: () => void,
	 *   onSave: () => void,
	 *   role?: string
	 * }}
	 */
	let { student, onClose, onSave, role = '' } = $props();

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

	const { formData, saving, syncData, save } = useStudentEditForm({
		student,
		onSave,
		showSnackbar
	});

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
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2.5"
						d="M15 19l-7-7 7-7"
					/>
				</svg>
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
			<button
				onclick={save}
				disabled={saving}
				class="px-6 py-2.5 text-sm font-bold text-white bg-blue-600 hover:bg-blue-700
               rounded-xl transition-all shadow-sm hover:shadow-md cursor-pointer disabled:opacity-50
               flex items-center gap-2.5"
			>
				{#if saving}
					<div
						class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
					></div>
					Speichert…
				{:else}
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2.5"
							d="M5 13l4 4L19 7"
						/>
					</svg>
					Speichern
				{/if}
			</button>
		</div>
	</header>

	<!-- ── Scrollable Body ────────────────────────────────────────────────── -->
	<div class="flex-1 overflow-y-auto px-8 py-6 space-y-8">
		<!-- ── Persönliche Daten ──────────────────────────────── -->
		<section>
			<h3
				class="text-[10px] font-black text-slate-500 uppercase tracking-[0.12em] mb-4 flex items-center gap-2"
			>
				<div class="w-2.5 h-2.5 rounded-full bg-slate-300"></div>
				Persönliche Daten
			</h3>

			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<InputField
					id="vorname"
					label="Vorname"
					bind:value={formData.vorname}
					extraClasses="font-semibold"
				/>
				<InputField
					id="nachname"
					label="Nachname"
					bind:value={formData.nachname}
					extraClasses="font-semibold"
				/>
				<InputField
					id="geburtsdatum"
					label="Geburtsdatum"
					type="date"
					bind:value={formData.geburtsdatum}
				/>
				<InputField
					id="lusd_id"
					label="LUSD-ID"
					bind:value={formData.lusd_id}
					extraClasses="font-mono"
				/>
			</div>
		</section>

		<!-- ── Schuldaten ─────────────────────────────────────── -->
		<section>
			<h3
				class="text-[10px] font-black text-slate-500 uppercase tracking-[0.12em] mb-4 flex items-center gap-2"
			>
				<div class="w-2.5 h-2.5 rounded-full bg-slate-300"></div>
				Schuldaten
			</h3>

			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<InputField
					id="klasse"
					label="Klasse"
					bind:value={formData.klasse}
					extraClasses="font-semibold"
				/>
				<InputField
					id="barcode"
					label="Schüler-ID / Barcode"
					bind:value={formData.barcode_id}
					extraClasses="font-mono"
				/>
				<InputField
					id="abgangsjahr"
					label="Abgangsjahr"
					type="number"
					bind:value={formData.abgaenger_jahr}
					extraClasses="font-semibold"
				/>

				<div>
					<label
						for="status"
						class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5"
						>Status</label
					>
					<select
						id="status"
						bind:value={formData.status}
						class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-semibold text-slate-800 focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
					>
						<option value="aktiv">Aktiv</option>
						<option value="inaktiv">Inaktiv</option>
						<option value="abgaenger">Abgänger</option>
					</select>
				</div>
			</div>
		</section>

		<!-- ── Kontaktdaten ────────────────────────────────────── -->
		<section>
			<h3
				class="text-[10px] font-black text-slate-500 uppercase tracking-[0.12em] mb-4 flex items-center gap-2"
			>
				<div class="w-2.5 h-2.5 rounded-full bg-blue-400"></div>
				Kontaktdaten
			</h3>

			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<div class="lg:col-span-2 grid grid-cols-4 gap-4">
					<div class="col-span-3">
						<InputField
							id="strasse"
							label="Straße"
							bind:value={formData.strasse}
							placeholder="Musterstraße"
						/>
					</div>
					<div class="col-span-1">
						<InputField
							id="hausnummer"
							label="Nr."
							bind:value={formData.hausnummer}
							placeholder="12a"
						/>
					</div>
				</div>

				<InputField
					id="plz"
					label="PLZ"
					bind:value={formData.plz}
					placeholder="12345"
					maxlength="5"
					extraClasses="font-mono"
				/>
				<InputField id="ort" label="Ort" bind:value={formData.ort} placeholder="Musterstadt" />

				<div class="lg:col-span-2">
					<InputField
						id="email"
						label="Eltern E-Mail"
						type="email"
						bind:value={formData.eltern_email}
						placeholder="eltern@schule.de"
					/>
				</div>
			</div>
		</section>

		<!-- Bottom spacing -->
		<div class="h-4"></div>
	</div>
</div>
