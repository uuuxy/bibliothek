<script>
	import { onMount } from 'svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import StudentProfile from './StudentProfile.svelte';
	import StudentCreateModal from './StudentCreateModal.svelte';
	import EhemaligeListe from './components/students/EhemaligeListe.svelte';
	import ActiveStudentList from './components/students/ActiveStudentList.svelte';
	import DeletedStudentList from './components/students/DeletedStudentList.svelte';
	import StudentDirectoryToolbar from './components/students/StudentDirectoryToolbar.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import AuswahlAktionsleiste from './components/students/AuswahlAktionsleiste.svelte';
	import StudentBatchPrint from './components/students/StudentBatchPrint.svelte';
	import { erzeugeAusweisdruck } from './components/students/ausweisdruck.svelte.js';
	import { erzeugeSchuelerSuche } from './components/students/schuelerSuche.svelte.js';
	import { erzeugeKlassenVorschlaege } from './components/students/klassenVorschlaege.svelte.js';
	import { erzeugeLeserAuswahl } from './components/students/leserAuswahl.svelte.js';
	import Reiter from './components/ui/Reiter.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { schuelerRechte } from './schuelerRechte.js';
	import { schuelerdateiReiter } from './schuelerdateiReiter.js';

	const rechte = $derived(schuelerRechte(authStore.currentUser));
	const reiterListe = $derived(schuelerdateiReiter(rechte));

	let activeTab = $state('active');

	let activeStudent = $state(/** @type {any} */ (null));

	// Vorschlagsliste des Anlegen-Dialogs (eigene Datei, Größen-Ratsche).
	const klassen = erzeugeKlassenVorschlaege();
	let showCreateModal = $state(false);

	// Markierung für den Ausweis-Stapeldruck (leserAuswahl.svelte.js, Größen-Ratsche).
	const gewaehlt = erzeugeLeserAuswahl(() => suche.students);

	// Serversuche & Laden der Liste liegen in schuelerSuche.svelte.js (Größen-Ratsche);
	// die erste Ladung stößt das Modul selbst an. Der Rückruf läuft nach dem Sprung aus
	// dem Druck-Center („Klassenweise drucken"): Die Klasse ist dann gesucht und
	// geladen, hier werden die Treffer markiert.
	const suche = erzeugeSchuelerSuche(() => {
		activeTab = 'active';
		gewaehlt.alleSichtbarenMarkieren();
	});

	/** Selbst gesucht und angeklickt = Datenpflege-Absicht: Profil öffnet Stammdaten. @param {any} s */
	function oeffneStammdaten(s) {
		profilReiter = 'stammdaten';
		activeStudent = s;
	}

	// Ausweiskarten oder Klebeetiketten — die Entscheidung steht im zentral
	// gespeicherten Design, die Wege dahinter sind grundverschieden (ausweisdruck.svelte.js).
	const druck = erzeugeAusweisdruck();

	function handleStudentCreated() {
		showCreateModal = false;
		suche.lade();
		klassen.lade(); // Klassenliste aktualisieren
	}

	onMount(klassen.lade);

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
						bind:jahrgang={suche.jahrgang}
						jahrgaenge={suche.jahrgaenge}
						jahrgaengeFehler={suche.jahrgaengeFehler}
						darfAnlegen={rechte.anlegen}
						trefferzahl={suche.students.length}
						suchend={suche.suchend}
						gekuerzt={suche.gekuerzt}
						onsearch={() => suche.angestossen()}
						oncreate={() => (showCreateModal = true)}
					/>

					<AuswahlAktionsleiste
						anzahl={gewaehlt.markierte.length}
						ohneDatum={gewaehlt.ohneDatum}
						etikettModus={druck.etikettModus}
						maxPosition={druck.maxPosition}
						bind:startPosition={druck.startPosition}
						onDrucken={() => druck.drucke(gewaehlt.markierte)}
						onLeeren={gewaehlt.leeren}
					/>

					<div class="mt-6">
						<ActiveStudentList
							filteredStudents={suche.students}
							loading={suche.beschaeftigt}
							ladefehler={suche.ladefehler}
							onErneut={() => suche.lade()}
							auswahl={gewaehlt.auswahl}
							onToggle={gewaehlt.umschalten}
							onToggleAlle={gewaehlt.alleUmschalten}
							onSelectStudent={oeffneStammdaten}
							sortierung={suche.sortierung}
							onsortiere={suche.sortiere}
						/>
					</div>
				</div>
			{:else if activeTab === 'graduates'}
				<EhemaligeListe onSelect={oeffneStammdaten} />
			{:else if activeTab === 'deleted'}
				<div class="w-full animate-fade-in space-y-6">
					<DeletedStudentList
						darfEndgueltigLoeschen={rechte.endgueltigLoeschen}
						onRestoreSuccess={() => {
							suche.lade();
							klassen.lade();
						}}
					/>
				</div>
			{/if}
		</PageShell>
	{/if}
</div>

<StudentCreateModal
	open={showCreateModal}
	klassen={klassen.liste}
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
{#if gewaehlt.markierte.length > 0 && !druck.etikettModus}
	<StudentBatchPrint students={gewaehlt.markierte} />
{/if}
