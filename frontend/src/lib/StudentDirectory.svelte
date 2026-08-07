<script>
	import { apiFetch } from './apiFetch.js';
	import { onMount } from 'svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import StudentProfile from './StudentProfile.svelte';
	import StudentCreateModal from './StudentCreateModal.svelte';
	import Graduates from './Graduates.svelte';
	import ActiveStudentList from './components/students/ActiveStudentList.svelte';
	import DeletedStudentList from './components/students/DeletedStudentList.svelte';
	import StudentDirectoryToolbar from './components/students/StudentDirectoryToolbar.svelte';
	import PageContainer from './components/layout/PageContainer.svelte';
	import AuswahlAktionsleiste from './components/students/AuswahlAktionsleiste.svelte';
	import StudentBatchPrint from './components/students/StudentBatchPrint.svelte';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import { SvelteSet } from 'svelte/reactivity';

	// Props (Svelte 5)
	let { role = '' } = $props();

	// State Runes (Svelte 5)
	let activeTab = $state('active');

	/** @type {any[]} */
	let students = $state.raw([]);
	let loading = $state(false);
	let searchQuery = $state('');
	/** @type {any} */
	let activeStudent = $state(null);

	/** @type {any[]} */
	let readerGroups = $state.raw([]);
	let showCreateModal = $state(false);

	// Gesucht wird auf dem SERVER. Vorher filterte diese Ansicht im Browser über die
	// gelieferte Liste — und die ist bei 500 Zeilen gekappt. Bei 875 Schülern waren 375
	// über die Suche schlicht nicht erreichbar, welche genau hing an der alphabetischen
	// Reihenfolge der Klassennamen. Für den Benutzer sah das nach Zufall aus.
	//
	// Nebeneffekt, der den Ausschlag gab: Die Serversuche ist dieselbe wie an der Theke
	// (suchnorm) — "Muller" findet Müller, "Hoffmann Lena" dasselbe wie "Lena Hoffmann".
	// Der Browser-Filter konnte beides nicht.
	let sucheLaeuft = $state(false);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let sucheTimer;
	/** Muss zu ListStudentsWithStatsLimit im Backend passen: Erreicht die ungefilterte
	 *  Liste diese Länge, ist sie gekappt und die Ansicht sagt das auch. */
	const LISTEN_GRENZE = 500;

	// Markierte Schüler für den Ausweis-Stapeldruck. Set statt Array: Das Ankreuzen
	// fragt bei jeder Zeile "ist die dabei?" — das ist der Zugriff, den ein Set kann.
	//
	// SvelteSet statt Set: Ein einfaches Set ist für Svelte 5 ein undurchsichtiger Wert;
	// .add()/.delete() lösten kein Neuzeichnen aus, und die Haken blieben beim Klicken
	// stehen. SvelteSet macht die Mitgliedschaft selbst reaktiv.
	const auswahl = new SvelteSet();
	const markierte = $derived(students.filter((/** @type {any} */ s) => auswahl.has(s.id)));
	// Karten ohne ableitbares Ablaufjahr würden "31.07.–" tragen. Der Balken sagt das
	// VOR dem Druck, nicht der fertige Stapel hinterher.
	const ohneDatum = $derived(
		markierte.filter((/** @type {any} */ s) => s.ausweis_gueltig_bis == null).length
	);

	/** @param {string} id */
	function toggle(id) {
		if (!auswahl.delete(id)) auswahl.add(id);
	}

	function toggleAlle() {
		// Bezugsgröße ist die ANGEZEIGTE Liste, nicht der Gesamtbestand: Wer nach "7H"
		// sucht und "alle" ankreuzt, meint die Treffer vor sich — nicht 875 Schüler.
		const alle = markierte.length === students.length;
		auswahl.clear();
		if (!alle) {
			for (const s of students) auswahl.add(s.id);
		}
	}

	// Druckt den markierten Stapel. Karten- oder A4-Modus kommt aus dem gespeicherten
	// Ausweis-Design (idStore.printMode) — der Designer legt fest WIE, diese Ansicht WER.
	function druckeAuswahl() {
		const a4 = idStore.printMode === 'a4';
		const style = document.createElement('style');
		style.textContent = a4
			? '@media print { @page { size: A4; margin: 0; } }'
			: '@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }';
		document.head.appendChild(style);
		document.body.setAttribute('data-print-mode', a4 ? 'a4' : 'card');
		document.body.setAttribute('data-print-side', 'front');
		window.print();
		document.head.removeChild(style);
		document.body.removeAttribute('data-print-mode');
		document.body.removeAttribute('data-print-side');
	}

	async function loadStudents() {
		loading = true;
		try {
			const q = searchQuery.trim();
			const res = await apiFetch(`/api/schueler${q ? `?q=${encodeURIComponent(q)}` : ''}`);
			if (res.ok) {
				students = (await res.json()) || [];
			}
		} catch (err) {
			console.error('Fehler beim Laden des Schülerverzeichnisses:', err);
		} finally {
			loading = false;
			sucheLaeuft = false;
		}
	}

	// Tippen wird entprellt, damit nicht jeder Tastendruck eine Abfrage auslöst. 300 ms
	// wie in der Omnibox — dieselbe Eingabegeschwindigkeit, dieselbe Wartezeit.
	function sucheAngestossen() {
		sucheLaeuft = true;
		clearTimeout(sucheTimer);
		sucheTimer = setTimeout(loadStudents, 300);
	}

	async function loadClasses() {
		try {
			const res = await apiFetch('/api/readergroups');
			if (res.ok) {
				readerGroups = (await res.json()) || [];
			}
		} catch (err) {
			console.error('Fehler beim Laden der Lesergruppen:', err);
		}
	}

	function handleStudentCreated() {
		showCreateModal = false;
		loadStudents();
		loadClasses(); // Klassenliste aktualisieren
	}

	onMount(() => {
		loadStudents();
		loadClasses();
	});

	// Reiter nach Absicht (siehe StudentProfile): Wer hier selbst gesucht hat, will
	// Stammdaten — Elternkontakt, Adressabgleich, Abgangsjahr. Wer aus Mahnwesen oder
	// Abgängern kommt, fragt nach Büchern und darf nicht im Adressformular landen.
	let profilReiter = $state(/** @type {'ausleihen'|'stammdaten'} */ ('stammdaten'));

	// Öffnet ein Profil, das aus einer anderen Ansicht (Mahnwesen/Abgänger) angefordert
	// wurde: ID einmalig abgreifen, Request sofort zurücksetzen (kein Wiederöffnen), dann
	// per { id } laden — StudentProfile holt den Rest selbst über GET /api/schueler/{id}.
	$effect(() => {
		const id = uiStore.requestedStudentId;
		if (!id) return;
		uiStore.requestedStudentId = null;
		profilReiter = 'ausleihen';
		activeStudent = { id };
	});
</script>

<div class="w-full h-full flex flex-col text-slate-800 bg-slate-50">
	{#if activeStudent}
		<div class="animate-fade-in flex-1 overflow-y-auto">
			<StudentProfile
				student={activeStudent}
				{role}
				defaultTab={profilReiter}
				onDeselect={() => {
					activeStudent = null;
					loadStudents();
					profilReiter = 'stammdaten';
				}}
			/>
		</div>
	{:else}
		<!-- Tab Navigation Header -->
		<div class="px-8 pt-6 pb-0 border-b border-slate-200 bg-white shrink-0 shadow-sm z-10">
			<div class="max-w-6xl mx-auto flex gap-6">
				{#snippet tabButton(id, label, activeColorClass)}
					<button
						onclick={() => (activeTab = id)}
						class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === id
							? activeColorClass
							: 'border-transparent text-slate-500 hover:text-slate-800'}"
					>
						{label}
					</button>
				{/snippet}

				{@render tabButton('active', 'Aktive Schüler', 'border-blue-600 text-blue-700')}
				{@render tabButton('graduates', 'Abgänger / Archiv', 'border-blue-600 text-blue-700')}
				{#if role === 'admin'}
					{@render tabButton('deleted', 'Papierkorb', 'border-rose-600 text-rose-700')}
				{/if}
			</div>
		</div>

		<!-- Tab Content -->
		<div class="flex-1 overflow-y-auto py-8 w-full">
			<PageContainer>
				{#if activeTab === 'active'}
					<div class="w-full no-print animate-fade-in">
						<StudentDirectoryToolbar
							bind:searchQuery
							{role}
							trefferzahl={students.length}
							suchend={searchQuery.trim().length > 0}
							gekuerzt={!searchQuery.trim() && students.length >= LISTEN_GRENZE}
							onsearch={sucheAngestossen}
							oncreate={() => (showCreateModal = true)}
						/>

						<AuswahlAktionsleiste
							anzahl={markierte.length}
							{ohneDatum}
							onDrucken={druckeAuswahl}
							onLeeren={() => auswahl.clear()}
						/>

						<div class="mt-6">
							<ActiveStudentList
								filteredStudents={students}
								loading={loading || sucheLaeuft}
								{auswahl}
								onToggle={toggle}
								onToggleAlle={toggleAlle}
								onSelectStudent={(s) => {
									// Selbst gesucht und angeklickt = Datenpflege-Absicht.
									profilReiter = 'stammdaten';
									activeStudent = s;
								}}
							/>
						</div>
					</div>
				{:else if activeTab === 'graduates'}
					<div class="w-full animate-fade-in">
						<Graduates />
					</div>
				{:else if activeTab === 'deleted'}
					<div class="w-full animate-fade-in space-y-6">
						<DeletedStudentList
							onRestoreSuccess={() => {
								loadStudents();
								loadClasses();
							}}
						/>
					</div>
				{/if}
			</PageContainer>
		</div>
	{/if}
</div>

<StudentCreateModal
	open={showCreateModal}
	{readerGroups}
	onclose={() => (showCreateModal = false)}
	onsuccess={handleStudentCreated}
/>

<!-- Die Druckfläche steht AUSSERHALB des .no-print-Wrappers oben, sonst blendet die
     Druck-CSS sie mit dem Rest der Ansicht aus und der Ausdruck bliebe leer (dieselbe
     Anordnung wie StudentPrintCard im Profil).
     Nur rendern, wenn wirklich markiert ist: Sonst hinge an jeder Schülerdatei ein
     unsichtbarer Kartensatz im DOM. -->
{#if markierte.length > 0}
	<StudentBatchPrint students={markierte} />
{/if}
