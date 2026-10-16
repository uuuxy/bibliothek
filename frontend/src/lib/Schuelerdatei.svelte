<script>
	import { onMount } from 'svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import Leserakte from './Leserakte.svelte';
	import LeserAnlegenDialog from './LeserAnlegenDialog.svelte';
	import EhemaligeListe from './components/leser/EhemaligeListe.svelte';
	import AktiveLeserListe from './components/leser/AktiveLeserListe.svelte';
	import Papierkorb from './components/leser/Papierkorb.svelte';
	import SchuelerdateiLeiste from './components/leser/SchuelerdateiLeiste.svelte';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import AuswahlAktionsleiste from './components/leser/AuswahlAktionsleiste.svelte';
	import AusweisStapeldruck from './components/leser/AusweisStapeldruck.svelte';
	import { erzeugeAusweisdruck } from './components/leser/ausweisdruck.svelte.js';
	import { erzeugeSchuelerSuche } from './components/leser/schuelerSuche.svelte.js';
	import { erzeugeKlassenVorschlaege } from './components/leser/klassenVorschlaege.svelte.js';
	import { erzeugeLeserAuswahl } from './components/leser/leserAuswahl.svelte.js';
	import Reiter from './components/ui/Reiter.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { schuelerRechte } from './schuelerRechte.js';
	import { schuelerdateiReiter } from './schuelerdateiReiter.js';

	const rechte = $derived(schuelerRechte(authStore.currentUser));
	const reiterListe = $derived(schuelerdateiReiter(rechte));

	let aktiverReiter = $state('active');

	let activeStudent = $state(/** @type {any} */ (null));

	// Die Klassen des Anlegen-Dialogs (eigene Datei, Größen-Ratsche).
	const klassen = erzeugeKlassenVorschlaege();
	let showCreateModal = $state(false);

	// Markierung für den Ausweis-Stapeldruck (leserAuswahl.svelte.js, Größen-Ratsche).
	const gewaehlt = erzeugeLeserAuswahl(() => suche.students);

	// Serversuche & Laden der Liste liegen in schuelerSuche.svelte.js (Größen-Ratsche);
	// die erste Ladung stößt das Modul selbst an. Der Rückruf läuft nach dem Sprung aus
	// dem Druck-Center („Klassenweise drucken"): Die Klasse ist dann gesucht und
	// geladen, hier werden die Treffer markiert.
	const suche = erzeugeSchuelerSuche(() => {
		aktiverReiter = 'active';
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

	function beiLeserAngelegt() {
		showCreateModal = false;
		suche.lade();
		klassen.lade(); // Klassenliste aktualisieren
	}

	onMount(klassen.lade);

	// Reiter nach Absicht (siehe Leserakte): Wer hier selbst gesucht hat, will
	// Stammdaten — Elternkontakt, Adressabgleich, Abgangsjahr. Wer aus Mahnwesen oder
	// Abgängern kommt, fragt nach Büchern oder nach dem Geld und nennt seinen Reiter selbst.
	let profilReiter = $state(/** @type {'ausleihen'|'gebuehren'|'stammdaten'} */ ('stammdaten'));

	// Öffnet ein Profil, das aus einer anderen Ansicht (Mahnwesen/Abgänger) angefordert
	// wurde: ID einmalig abgreifen, Request sofort zurücksetzen (kein Wiederöffnen), dann
	// per { id } laden — Leserakte holt den Rest selbst über GET /api/schueler/{id}.
	$effect(() => {
		const id = uiStore.requestedStudentId;
		if (!id) return;
		profilReiter = uiStore.requestedStudentReiter ?? 'ausleihen';
		uiStore.requestedStudentId = null;
		uiStore.requestedStudentReiter = null;
		activeStudent = { id };
	});
</script>

<div class="w-full h-full flex flex-col text-on-surface">
	{#if activeStudent}
		<div class="animate-fade-in flex-1 overflow-y-auto">
			<Leserakte
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
		<Seitenrahmen>
			<Reiter
				etikett="Schülerdatei"
				reiter={reiterListe}
				aktiv={aktiverReiter}
				onwahl={(id) => (aktiverReiter = id)}
			/>

			<!-- Tab Content -->
			{#if aktiverReiter === 'active'}
				<div class="w-full no-print animate-fade-in">
					<SchuelerdateiLeiste
						bind:searchQuery={suche.query}
						bind:jahrgang={suche.jahrgang}
						jahrgaenge={suche.jahrgaenge}
						jahrgaengeFehler={suche.jahrgaengeFehler}
						darfAnlegen={rechte.anlegen}
						trefferzahl={suche.students.length}
						suchend={suche.suchend}
						gekuerzt={suche.gekuerzt}
						onsearch={() => suche.angestossen()}
						oncreate={() => {
							// Frisch laden: Die Liste ist die einzige Wahl, und ein Abruf, der beim
							// Öffnen der Leserdatei scheiterte, bliebe sonst bis zum Neuladen stehen.
							klassen.lade();
							showCreateModal = true;
						}}
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
						<AktiveLeserListe
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
			{:else if aktiverReiter === 'graduates'}
				<EhemaligeListe onSelect={oeffneStammdaten} />
			{:else if aktiverReiter === 'deleted'}
				<div class="w-full animate-fade-in space-y-6">
					<Papierkorb
						darfEndgueltigLoeschen={rechte.endgueltigLoeschen}
						onRestoreSuccess={() => {
							suche.lade();
							klassen.lade();
						}}
					/>
				</div>
			{/if}
		</Seitenrahmen>
	{/if}
</div>

<LeserAnlegenDialog
	open={showCreateModal}
	klassen={klassen.liste}
	klassenFehler={klassen.ladefehler}
	onclose={() => (showCreateModal = false)}
	onsuccess={beiLeserAngelegt}
/>

<!-- Die Druckfläche steht AUSSERHALB des .no-print-Wrappers oben, sonst blendet die
     Druck-CSS sie mit dem Rest der Ansicht aus und der Ausdruck bliebe leer (dieselbe
     Anordnung wie Ausweiskarte im Profil).
     Nur rendern, wenn wirklich markiert ist: Sonst hinge an jeder Schülerdatei ein
     unsichtbarer Kartensatz im DOM. Im Etikettenmodus gar nicht: Der Bogen kommt als
     PDF vom Server, und jede Karte hier zöge ein Barcode-Bild über die Leitung, das
     niemand zu sehen bekommt. -->
{#if gewaehlt.markierte.length > 0 && !druck.etikettModus}
	<AusweisStapeldruck students={gewaehlt.markierte} />
{/if}
