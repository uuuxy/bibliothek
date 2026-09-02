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
	import PageShell from './components/layout/PageShell.svelte';
	import AuswahlAktionsleiste from './components/students/AuswahlAktionsleiste.svelte';
	import StudentBatchPrint from './components/students/StudentBatchPrint.svelte';
	import { erzeugeAusweisdruck } from './components/students/ausweisdruck.svelte.js';
	import { erzeugeSchuelerSuche } from './components/students/schuelerSuche.svelte.js';
	import { SvelteSet } from 'svelte/reactivity';
	import Reiter from './components/ui/Reiter.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { schuelerRechte } from './schuelerRechte.js';
	import { schuelerdateiReiter } from './schuelerdateiReiter.js';

	const rechte = $derived(schuelerRechte(authStore.currentUser));
	const reiterListe = $derived(schuelerdateiReiter(rechte));

	let activeTab = $state('active');

	let activeStudent = $state(/** @type {any} */ (null));

	/** @type {any[]} */
	let readerGroups = $state.raw([]);
	let showCreateModal = $state(false);

	// Markierte Schüler für den Ausweis-Stapeldruck. Set statt Array: Das Ankreuzen
	// fragt bei jeder Zeile "ist die dabei?" — das ist der Zugriff, den ein Set kann.
	//
	// SvelteSet statt Set: Ein einfaches Set ist für Svelte 5 ein undurchsichtiger Wert;
	// .add()/.delete() lösten kein Neuzeichnen aus, und die Haken blieben beim Klicken
	// stehen. SvelteSet macht die Mitgliedschaft selbst reaktiv.
	const auswahl = new SvelteSet();

	// Serversuche & Laden der Liste liegen in schuelerSuche.svelte.js (Größen-Ratsche);
	// die erste Ladung stößt das Modul selbst an. Der Rückruf läuft nach dem Sprung aus
	// dem Druck-Center („Klassenweise drucken"): Die Klasse ist dann gesucht und
	// geladen, hier werden die Treffer markiert.
	const suche = erzeugeSchuelerSuche(() => {
		activeTab = 'active';
		auswahl.clear();
		for (const s of suche.students) auswahl.add(s.id);
	});

	const markierte = $derived(suche.students.filter((/** @type {any} */ s) => auswahl.has(s.id)));
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
		const alle = markierte.length === suche.students.length;
		auswahl.clear();
		if (!alle) {
			for (const s of suche.students) auswahl.add(s.id);
		}
	}

	// Ausweiskarten oder Klebeetiketten — die Entscheidung steht im zentral
	// gespeicherten Design, die Wege dahinter sind grundverschieden (ausweisdruck.svelte.js).
	const druck = erzeugeAusweisdruck();

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
		suche.lade();
		loadClasses(); // Klassenliste aktualisieren
	}

	onMount(loadClasses);

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

<div class="w-full h-full flex flex-col text-on-surface">
	{#if activeStudent}
		<div class="animate-fade-in flex-1 overflow-y-auto">
			<StudentProfile
				student={activeStudent}
				defaultTab={profilReiter}
				onMerged={(id) => (activeStudent = { id })}
				onDeselect={() => {
					activeStudent = null;
					suche.lade();
					profilReiter = 'stammdaten';
				}}
			/>
		</div>
	{:else}
		<PageShell>
			<Reiter
				etikett="Schülerdatei"
				reiter={reiterListe}
				aktiv={activeTab}
				onwahl={(id) => (activeTab = id)}
			/>

			<!-- Tab Content -->
			{#if activeTab === 'active'}
				<div class="w-full no-print animate-fade-in">
					<StudentDirectoryToolbar
						bind:searchQuery={suche.query}
						darfAnlegen={rechte.anlegen}
						trefferzahl={suche.students.length}
						suchend={suche.suchend}
						gekuerzt={suche.gekuerzt}
						onsearch={() => suche.angestossen()}
						oncreate={() => (showCreateModal = true)}
					/>

					<AuswahlAktionsleiste
						anzahl={markierte.length}
						{ohneDatum}
						etikettModus={druck.etikettModus}
						maxPosition={druck.maxPosition}
						bind:startPosition={druck.startPosition}
						onDrucken={() => druck.drucke(markierte)}
						onLeeren={() => auswahl.clear()}
					/>

					<div class="mt-6">
						<ActiveStudentList
							filteredStudents={suche.students}
							loading={suche.beschaeftigt}
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
						darfEndgueltigLoeschen={rechte.endgueltigLoeschen}
						onRestoreSuccess={() => {
							suche.lade();
							loadClasses();
						}}
					/>
				</div>
			{/if}
		</PageShell>
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
     unsichtbarer Kartensatz im DOM. Im Etikettenmodus gar nicht: Der Bogen kommt als
     PDF vom Server, und jede Karte hier zöge ein Barcode-Bild über die Leitung, das
     niemand zu sehen bekommt. -->
{#if markierte.length > 0 && !druck.etikettModus}
	<StudentBatchPrint students={markierte} />
{/if}
