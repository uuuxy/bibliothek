<script>
	import { apiFetch } from './apiFetch.js';
	import { onMount } from 'svelte';
	import { uiStore } from './stores/uiStore.svelte.js';

	/** Öffnet das Profil des Abgängers in der Schülerdatei (zentraler Request im uiStore). */
	function openProfile(student) {
		uiStore.requestedStudentId = student.id;
		uiStore.activeTab = 'students_dir';
	}

	// State Runes
	/** @type {any[]} */
	let graduates = $state([]);
	let loading = $state(true);

	// Klassenfilter: leerer Wert = alle Klassen. Filtert die Liste UND den Laufzettel-Druck.
	let selectedKlasse = $state('');
	let classes = $derived(
		[...new Set(graduates.map((/** @type {any} */ s) => s.klasse))].sort((a, b) =>
			String(a).localeCompare(String(b), 'de', { numeric: true })
		)
	);
	let filteredGraduates = $derived(
		(selectedKlasse
			? graduates.filter((/** @type {any} */ s) => s.klasse === selectedKlasse)
			: graduates
		)
			.slice()
			// Dringlichkeit zuerst: überfällige oben, dann nach Anzahl offener Bücher, dann Klasse/Name.
			.sort(
				(/** @type {any} */ a, /** @type {any} */ b) =>
					b.ueberfaellig - a.ueberfaellig ||
					b.offene_buecher - a.offene_buecher ||
					String(a.klasse).localeCompare(String(b.klasse), 'de', { numeric: true }) ||
					String(a.nachname).localeCompare(String(b.nachname), 'de')
			)
	);

	// Laufzettel print state
	let loadingLaufzettel = $state(false);

	async function printLaufzettel() {
		loadingLaufzettel = true;
		try {
			// Ist eine Klasse gewählt, druckt der Laufzettel gezielt nur diese Klasse.
			const endpoint = selectedKlasse
				? `/api/abgaenger/pdf?klasse=${encodeURIComponent(selectedKlasse)}`
				: '/api/abgaenger/pdf';
			const response = await apiFetch(endpoint);
			if (!response.ok) {
				throw new Error('Failed to load PDF');
			}

			const blob = await response.blob();
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = selectedKlasse ? `Laufzettel_${selectedKlasse}.pdf` : 'Laufzettel.pdf';
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			a.remove();
		} catch (err) {
			console.error('Laufzettel load error:', err);
		} finally {
			loadingLaufzettel = false;
		}
	}

	// Fetch graduates list from backend api
	async function fetchGraduates() {
		try {
			const res = await apiFetch('/api/abgaenger');
			if (!res.ok) throw new Error('Fehler beim Laden');
			graduates = await res.json();
		} catch (err) {
			console.error('Graduates error:', err);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		// Initial fetch
		fetchGraduates();

		// Listen to Go SSE events for instant UI synchronization
		const source = new EventSource('/events');

		// When a book is returned or transferred via the Omnibox,
		// refetch the graduates list to verify if the student is cleared.
		source.addEventListener('action', (e) => {
			try {
				const actionData = JSON.parse(e.data);
				if (actionData.event === 'rueckgabe' || actionData.event === 'fremdrueckgabe') {
					fetchGraduates();
				}
			} catch (err) {
				console.error('Failed to parse SSE payload:', err);
			}
		});

		return () => {
			source.close();
		};
	});
</script>

<div class="w-full space-y-6 text-slate-800">
	<!-- Header Info: links Klassenfilter, rechts Laufzettel-Druck (der dem Filter folgt). -->
	<div class="flex items-center justify-between gap-4 border-b border-slate-100 pb-5">
		{#if !loading && graduates.length > 0}
			<div class="flex items-center gap-3 min-w-0">
				<label class="text-xs font-bold text-slate-500 uppercase tracking-wider shrink-0" for="grad-klasse"
					>Klasse</label
				>
				<select
					id="grad-klasse"
					bind:value={selectedKlasse}
					class="bg-slate-50 border border-slate-200 rounded-lg text-sm font-bold text-slate-700 px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500/20 cursor-pointer"
				>
					<option value="">Alle Klassen ({graduates.length})</option>
					{#each classes as k (k)}
						<option value={k}>{k}</option>
					{/each}
				</select>
				<span class="text-xs text-slate-400 shrink-0">{filteredGraduates.length} Abgänger</span>
			</div>
		{:else}
			<div></div>
		{/if}

		<div class="flex items-center space-x-4 shrink-0">
			<button
				onclick={printLaufzettel}
				disabled={loadingLaufzettel || graduates.length === 0}
				class="no-print px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-bold rounded-xl text-xs flex items-center gap-1.5 transition-all shadow-xs cursor-pointer"
			>
				{#if loadingLaufzettel}
					<div
						class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"
					></div>
					Lade Daten…
				{:else}
					🖨️ {selectedKlasse ? `Laufzettel ${selectedKlasse}` : 'Laufzettel drucken'}
				{/if}
			</button>
			<div
				class="flex items-center gap-1.5 text-[11px] font-semibold text-emerald-600 shrink-0"
				title="Änderungen an allen Arbeitsplätzen sofort sichtbar (Live-Synchronisation)"
			>
				<span class="h-2 w-2 rounded-full bg-emerald-500 animate-pulse shrink-0"></span>
				Live
			</div>
		</div>
	</div>

	{#if loading}
		<div class="py-12 flex justify-center items-center">
			<div
				class="w-8 h-8 border-2 border-t-blue-600 border-blue-100 rounded-full animate-spin"
			></div>
		</div>
	{:else if graduates.length === 0}
		<!-- Completed clearing UI state -->
		<div class="py-12 text-center space-y-3 animate-fade-in">
			<div
				class="w-16 h-16 rounded-full bg-emerald-50 border border-emerald-100 flex items-center justify-center text-emerald-600 mx-auto"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-8 w-8"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M5 13l4 4L19 7"
					/>
				</svg>
			</div>
			<h3 class="font-bold text-slate-800">Alle Abgänger entlastet!</h3>
			<p class="text-xs text-slate-500 max-w-xs mx-auto">
				Kein Abgänger hat mehr offene Lehrmittel oder unbezahlte Schadensfälle.
			</p>
		</div>
	{:else}
		<!-- Active list of graduates with dues -->
		<div class="overflow-x-auto">
			<table class="w-full text-left text-base border-collapse">
				<thead>
					<tr class="border-b border-slate-100 text-slate-450 text-sm uppercase">
						<th class="py-3 px-4">Klasse</th>
						<th class="py-3 px-4">Name</th>
						<th class="py-3 px-4">Offene Bücher</th>
						<th class="py-3 px-4">Sperr-Status</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-50">
					{#each filteredGraduates as student (student.id)}
						<tr
							onclick={() => openProfile(student)}
							onkeydown={(e) => {
								if (e.key === 'Enter' || e.key === ' ') {
									e.preventDefault();
									openProfile(student);
								}
							}}
							tabindex="0"
							role="button"
							aria-label="Profil von {student.vorname} {student.nachname} (Klasse {student.klasse}) anzeigen"
							class="hover:bg-slate-50/85 cursor-pointer transition-colors animate-slide-up focus-visible:outline-2 focus-visible:outline-blue-600 focus-visible:-outline-offset-2"
						>
							<td class="py-3.5 px-4 font-bold text-blue-600">{student.klasse}</td>
							<td class="py-3.5 px-4 text-slate-700 font-semibold"
								>{student.vorname} {student.nachname}</td
							>
							<td class="py-3.5 px-4">
								<span
									class="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-full text-sm font-bold {student.ueberfaellig >
									0
										? 'bg-rose-50 text-rose-600 border border-rose-100'
										: 'bg-slate-100 text-slate-600'}"
									title={student.ueberfaellig > 0
										? `${student.offene_buecher} offen, davon ${student.ueberfaellig} überfällig`
										: `${student.offene_buecher} offen`}
								>
									{student.offene_buecher}
									<span class="text-xs font-medium opacity-70"
										>{student.offene_buecher === 1 ? 'Buch' : 'Bücher'}</span
									>
								</span>
								{#if student.ueberfaellig > 0}
									<span class="ml-2 text-xs font-semibold text-rose-500"
										>{student.ueberfaellig} überfällig</span
									>
								{/if}
							</td>
							<td class="py-3.5 px-4">
								{#if student.ist_gesperrt}
									<span
										class="text-[10px] px-2 py-0.5 rounded bg-rose-50 border border-rose-100 text-rose-600 font-semibold"
										>Sperre aktiv</span
									>
								{:else}
									<span
										class="text-[10px] px-2 py-0.5 rounded bg-slate-100 text-slate-400 font-medium"
										>Bereit</span
									>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
