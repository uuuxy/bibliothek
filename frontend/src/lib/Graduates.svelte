<script>
	import { apiFetch } from './apiFetch.js';
	import { onMount } from 'svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import Button from './components/ui/Button.svelte';
	import KlassenVersandDialog from './components/ui/KlassenVersandDialog.svelte';
	import { Mail, Printer } from '@lucide/svelte';

	/** Öffnet das Profil des Abgängers in der Schülerdatei (zentraler Request im uiStore). */
	function openProfile(student) {
		uiStore.requestedStudentId = student.id;
		uiStore.activeTab = 'students_dir';
	}

	// State Runes
	/** @type {any[]} */
	let graduates = $state([]);
	let loading = $state(true);

	// Klassenfilter: leerer Wert = alle Klassen. Filtert die Liste UND den Ausdruck.
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

	// Kontoauszug-Druck. Das PDF heißt intern noch /abgaenger/pdf, ist aber seit
	// Langem der Kontoauszug mit Freigabezeile — eine Seite je Abgänger.
	let loadingKontoauszuege = $state(false);

	async function printKontoauszuege() {
		loadingKontoauszuege = true;
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
			a.download = selectedKlasse
				? `Kontoauszuege_${selectedKlasse}.pdf`
				: 'Kontoauszuege_Abgaenger.pdf';
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			a.remove();
		} catch (err) {
			console.error('Kontoauszug load error:', err);
		} finally {
			loadingKontoauszuege = false;
		}
	}

	// Versand an die Klassenleitungen: je Klasse eine Mail, darin ein Kontoauszug je
	// Abgänger. Der Dialog davor ist derselbe wie beim Mahnlauf — er entscheidet, WELCHE
	// Klassen laufen und ob die Auszüge ausnahmsweise an eine einzelne Adresse gehen.
	let versandOffen = $state(false);

	// Der Dialog erwartet die Form des Mahnwesens ({ klasse, schueler, lehrer_email }),
	// damit er für beide Aufrufer derselbe bleibt. Die Abgängerliste ist flach, also
	// wird sie hier auf Klassen verdichtet.
	//
	// lehrer_email kommt seit dem Mapping-JOIN in /api/abgaenger mit. Ohne sie stand im
	// Dialog bei JEDER Klasse „keine E-Mail" — auch bei hinterlegter Adresse, weil das
	// Frontend die Adressen schlicht nicht kannte.
	let klassenFuerVersand = $derived(
		classes.map((k) => {
			const schueler = graduates.filter((/** @type {any} */ s) => s.klasse === k);
			return {
				klasse: String(k),
				schueler,
				lehrer_email: schueler[0]?.lehrer_email ?? ''
			};
		})
	);

	/** @param {{ klassen: string[], overrideEmail: string }} auswahl */
	async function sendeKontoauszuege(auswahl) {
		versandOffen = false;
		// Ohne Auswahl gar nicht erst losschicken: Ein fehlendes klassen-Feld bedeutet
		// serverseitig ALLE Klassen — genau der Rundumschlag, den der Dialog verhindert.
		if (!auswahl.klassen.length) {
			showToast('Keine Klasse ausgewählt.', 'error');
			return;
		}
		try {
			const res = await apiFetch('/api/abgaenger/mail', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					klassen: auswahl.klassen,
					override_email: auswahl.overrideEmail ?? ''
				})
			});
			const json = await res.json();
			// Die Server-Meldung wird durchgereicht, nicht selbst formuliert: nur sie weiß,
			// an WEN die Auszüge tatsächlich gingen.
			showToast(
				res.ok
					? (json.message ?? 'Kontoauszüge versendet.')
					: (json.error ?? json.message ?? 'Versand fehlgeschlagen.'),
				res.ok ? 'success' : 'error'
			);
		} catch (e) {
			showToast(`Netzwerkfehler beim Versand: ${e}`, 'error');
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
				<label class="text-xs font-medium text-slate-500 shrink-0" for="grad-klasse">Klasse</label>
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
			<!-- Zwei Wege für dasselbe Dokument: ausdrucken (Papier bleibt der Notweg,
			     wenn keine Lehrer-Adresse hinterlegt ist) oder an die Klassenleitungen mailen. -->
			<Button
				variant="secondary"
				onclick={printKontoauszuege}
				disabled={loadingKontoauszuege || graduates.length === 0}
				class="no-print"
			>
				{#if loadingKontoauszuege}
					<div
						class="w-3.5 h-3.5 border-2 border-slate-400 border-t-transparent rounded-full animate-spin"
					></div>
					Lade Daten…
				{:else}
					<Printer class="h-4 w-4" aria-hidden="true" />
					{selectedKlasse ? `Kontoauszüge ${selectedKlasse}` : 'Kontoauszüge drucken'}
				{/if}
			</Button>
			<Button
				onclick={() => (versandOffen = true)}
				disabled={graduates.length === 0}
				class="no-print"
				title="Je Klasse eine Mail an die Klassenleitung, darin ein Kontoauszug je Abgänger"
			>
				<Mail class="h-4 w-4" />
				An Klassenleitungen mailen
			</Button>
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
					<tr class="border-b border-slate-100 text-slate-500 text-sm">
						<th class="py-2 px-4">Klasse</th>
						<th class="py-2 px-4">Name</th>
						<th class="py-2 px-4">Offene Bücher</th>
						<th class="py-2 px-4">Sperr-Status</th>
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
							<td class="py-2 px-4 text-slate-500">{student.klasse}</td>
							<td class="py-2 px-4 font-medium text-slate-800"
								>{student.vorname} {student.nachname}</td
							>
							<td class="py-2 px-4 text-slate-600">
								{student.offene_buecher}
								{student.offene_buecher === 1 ? 'Buch' : 'Bücher'}
								{#if student.ueberfaellig > 0}
									<span class="font-medium text-rose-600">
										· {student.ueberfaellig} überfällig
									</span>
								{/if}
							</td>
							<td class="py-2 px-4">
								{#if student.ist_gesperrt}
									<span class="text-xs font-medium text-rose-600">Sperre aktiv</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Auf oberster Ebene: Der Dialog ist ein Overlay und gehört nicht in die Werkzeugleiste. -->
<KlassenVersandDialog
	open={versandOffen}
	titel="Kontoauszüge versenden"
	beschreibung="Wähle die Klassen aus, deren Abgänger-Kontoauszüge an die Klassenleitung gehen sollen."
	aktion="senden"
	hinweis="Leer lassen = jede Klasse an ihre eigene Klassenleitung. Der Namensteil genügt, die Schul-Domäne wird ergänzt."
	klassen={klassenFuerVersand}
	onclose={() => (versandOffen = false)}
	onconfirm={sendeKontoauszuege}
/>
