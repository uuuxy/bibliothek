<script>
	import WebcamAufnahme from './WebcamAufnahme.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import SchadenMeldenDialog from './SchadenMeldenDialog.svelte';
	import LeserSperreDialog from './LeserSperreDialog.svelte';
	import LeserBearbeiten from './LeserBearbeiten.svelte';
	import LeserakteKopf from './LeserakteKopf.svelte';
	import Ausweiskarte from './Ausweiskarte.svelte';
	import LeserLoeschenDialog from './LeserLoeschenDialog.svelte';
	import LeserGefahrenzone from './LeserGefahrenzone.svelte';
	import LeserakteStammdaten from './LeserakteStammdaten.svelte';
	import LeserakteAusleihen from './LeserakteAusleihen.svelte';
	import LeserakteGebuehren from './LeserakteGebuehren.svelte';
	import LeserakteAktionen from './LeserakteAktionen.svelte';
	import AusleihQuittung from './AusleihQuittung.svelte';
	import { useLeserakte } from './useLeserakte.svelte.js';
	import { Info } from '@lucide/svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { schuelerRechte, darfAuskunftUeber } from './schuelerRechte.js';
	import { druckeAusweis } from './ausweisDruck.js';
	/**
	 * @typedef {Object} Props
	 * @property {any} student - The selected student object
	 * @property {() => void} onDeselect - Callback when profile is closed
	 * @property {(barcode: string) => void} [onReturnClick] - Callback for returning a book
	 * @property {(zielId: string) => void} [onMerged] - Nach dem Zusammenführen: der Aufrufer hängt seinen aktiven Schüler auf das Ziel um
	 * @property {import('svelte').Snippet} [leftActions] - Optional slot for left card actions
	 * @property {import('svelte').Snippet} [rightTop] - Optional slot for right content top
	 * @property {'ausleihen'|'gebuehren'|'stammdaten'} [defaultTab] - Reiter, der beim Öffnen oben liegt
	 */
	/** @type {Props} */
	let {
		student,
		onDeselect,
		onReturnClick = undefined,
		onMerged = undefined,
		leftActions,
		rightTop,
		defaultTab = 'ausleihen'
	} = $props();

	const st = useLeserakte();
	// Aktionen folgen dem Recht ihrer Route, nicht der Rolle — Zuordnung in schuelerRechte.js.
	const rechte = $derived(schuelerRechte(authStore.currentUser));
	// Anzahl und Summe kommen mit dem Profil vom Server, nicht aus der Liste des Reiters.
	const offen = $derived(st.akte?.offene_forderungen ?? { anzahl: 0, summe: 0 });

	// Der Aufrufer bestimmt den Reiter, der beim Öffnen oben liegt. Bei jedem Wechsel des
	// Lesers zurück auf ihn, sonst speichert ein offenes Blatt auf den falschen Leser.
	$effect(() => {
		if (!student?.id) return;
		st.aktiverReiter = defaultTab;
		st.schliesseAlleBlaetter();
		st.ladeAkte(student.id);
	});

	export const ladeNeu = () => st.ladeAkte(st.akte?.id ?? student?.id);

	// Vom Bediener gesetztes Ablaufjahr für genau diesen Ausdruck; null = Vorschlag des Servers.
	// Nicht gespeichert, beim Wechsel des Lesers fällt es zurück.
	let gueltigBisOverride = $state(/** @type {number|null} */ (null));
	let zuletztGezeigteId = $state(/** @type {string|null} */ (null));
	$effect(() => {
		// Nur beim Wechsel des Lesers: Ein Neuladen des Profils lässt die getippte Jahreszahl stehen.
		const id = st.akte?.id ?? null;
		if (id !== zuletztGezeigteId) {
			zuletztGezeigteId = id;
			gueltigBisOverride = null;
		}
	});
	const gueltigBisEffektiv = $derived(gueltigBisOverride ?? st.akte?.ausweis_gueltig_bis ?? null);
</script>

{#if st.laedt}
	<div class="w-full py-12 flex justify-center items-center">
		<Ladekreis size="lg" />
	</div>
{:else if st.akte}
	{#if st.fehlermeldung}
		<div
			class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-error text-on-error flex items-center gap-2"
		>
			<Info class="h-5 w-5" aria-hidden="true" />
			{st.fehlermeldung}
		</div>
	{/if}

	{#if !st.bearbeitenOffen}
		<div
			class="w-full grid grid-cols-1 lg:grid-cols-[320px_minmax(0,1fr)] items-stretch text-on-surface animate-fade-in no-print print:hidden font-sans"
		>
			<LeserakteKopf
				bind:profile={st.akte}
				{rechte}
				timestamp={st.fotoStand}
				bind:showWebcam={st.kameraOffen}
				{onDeselect}
				{leftActions}
				{offen}
				onLock={rechte.bearbeiten ? () => (st.sperreOffen = true) : undefined}
			/>

			<!-- pt-4: Die Knopfzeile steht auf der Höhe des Schließen-Knopfs der linken Spalte. -->
			<div class="lg:col-span-1 space-y-6 flex flex-col h-full px-6 pt-4 pb-4">
				{#if rechte.einsehen}
					<LeserakteAktionen
						profile={st.akte}
						darfAuskunft={darfAuskunftUeber(rechte, st.akte)}
						kontoauszugPdfLoading={st.kontoauszugLaedt}
						rechnungPdfLoading={st.rechnungLaedt}
						downloadKontoauszugPDF={st.ladeKontoauszugHerunter}
						downloadRechnungPDF={st.ladeRechnungHerunter}
						onPrint={druckeAusweis}
						gueltigBis={gueltigBisEffektiv}
						onGueltigBis={(jahr) => (gueltigBisOverride = jahr)}
					/>
				{/if}

				<Reiter
					etikett="Leserakte"
					reiter={[
						{ id: 'ausleihen', label: 'Ausleihen & Vormerkungen' },
						{ id: 'gebuehren', label: 'Gebühren & Schäden', anzahl: offen.anzahl },
						{ id: 'stammdaten', label: 'Stammdaten & Adresse' }
					]}
					aktiv={st.aktiverReiter}
					onwahl={(id) => (st.aktiverReiter = id)}
				/>

				<div class="flex-1 relative">
					{#if st.aktiverReiter === 'ausleihen'}
						<LeserakteAusleihen
							buecher={st.akte.entliehene_buecher || []}
							bind:vormerkungen={st.vormerkungen}
							fehlendeListen={st.fehlendeListen.filter((l) => l === 'Vormerkungen')}
							{onReturnClick}
							onDamageClick={rechte.bearbeiten ? st.oeffneSchaden : undefined}
							onChanged={() => st.ladeAkte(st.akte.id)}
							{rightTop}
						/>
					{:else if st.aktiverReiter === 'gebuehren'}
						<LeserakteGebuehren
							schuelerId={st.akte.id}
							gebuehren={st.gebuehren}
							bescheide={st.bescheide}
							fehlendeListen={st.fehlendeListen.filter((l) => l !== 'Vormerkungen')}
							canEdit={rechte.bearbeiten}
							onChanged={() => st.ladeAkte(st.akte.id)}
						/>
					{:else if st.aktiverReiter === 'stammdaten'}
						<LeserakteStammdaten
							profile={st.akte}
							darfBearbeiten={rechte.bearbeiten}
							darfZusammenfuehren={rechte.zusammenfuehren}
							onEdit={() => (st.bearbeitenOffen = true)}
							onMerged={(id) => (st.ladeAkte(id), onMerged?.(id))}
						/>

						<!-- Gefahrenzone am unteren Ende des Stammdaten-Reiters, seit 16.09.2026 auch beim Kollegium -->
						{#if rechte.loeschen}
							<LeserGefahrenzone profile={st.akte} onDelete={() => (st.loeschfrageOffen = true)} />
						{/if}
					{/if}
				</div>
			</div>
		</div>
	{:else}
		<LeserBearbeiten
			student={st.akte}
			onClose={() => (st.bearbeitenOffen = false)}
			onSave={() => st.nachBearbeiten(st.akte.id)}
		/>
	{/if}
{:else}
	<LadeFehler
		onerneut={ladeNeu}
		titel="Akte nicht geladen"
		text="Die Daten dieses Lesers konnten nicht abgerufen werden."
	/>
{/if}

{#if st.kameraOffen}
	<WebcamAufnahme
		studentId={st.akte.id}
		onCapture={() => st.nachFoto(st.akte.id)}
		onClose={() => (st.kameraOffen = false)}
	/>
{/if}

{#if st.akte}
	<!-- Die Abweichung liegt auf dem Profil, nicht in Kartenseite: So drucken Profil, Klassensatz
	     und Designer dieselbe Karte. -->
	<Ausweiskarte
		profile={{ ...st.akte, ausweis_gueltig_bis: gueltigBisEffektiv }}
		timestamp={st.fotoStand}
	/>
{/if}

<LeserLoeschenDialog
	open={st.loeschfrageOffen}
	profile={st.akte}
	onclose={() => (st.loeschfrageOffen = false)}
	onsuccess={() => st.nachLoeschen(onDeselect)}
/>

{#if st.schadenOffen && st.schadenExemplar}
	<SchadenMeldenDialog
		book={st.schadenExemplar}
		isSubmitting={st.meldetSchaden}
		onCancel={() => (st.schadenOffen = false)}
		onSubmit={(r, a, art) => st.meldeSchaden(st.akte.id, r, a, art)}
	/>
{/if}

<LeserSperreDialog bind:open={st.sperreOffen} profile={st.akte} onsuccess={st.nachSperre} />

{#if st.akte}
	<AusleihQuittung profile={st.akte} />
{/if}
