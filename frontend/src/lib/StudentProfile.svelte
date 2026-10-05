<script>
	import WebcamCapture from './WebcamCapture.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import DamageReportModal from './DamageReportModal.svelte';
	import StudentLockModal from './StudentLockModal.svelte';
	import StudentEditSheet from './StudentEditSheet.svelte';
	import StudentProfileCard from './StudentProfileCard.svelte';
	import StudentPrintCard from './StudentPrintCard.svelte';
	import StudentProfileDeleteModal from './StudentProfileDeleteModal.svelte';
	import StudentDangerZone from './StudentDangerZone.svelte';
	import StudentProfileStammdaten from './StudentProfileStammdaten.svelte';
	import StudentProfileAusleihen from './StudentProfileAusleihen.svelte';
	import StudentProfileGebuehren from './StudentProfileGebuehren.svelte';
	import StudentProfileActions from './StudentProfileActions.svelte';
	import StudentPrintReceipt from './StudentPrintReceipt.svelte';
	import { useStudentProfile } from './useStudentProfile.svelte.js';
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

	const st = useStudentProfile();
	// Aktionen folgen dem Recht ihrer Route, nicht der Rolle — Zuordnung in schuelerRechte.js.
	const rechte = $derived(schuelerRechte(authStore.currentUser));
	// Anzahl und Summe kommen mit dem Profil vom Server, nicht aus der Liste des Reiters.
	const offen = $derived(st.profile?.offene_forderungen ?? { anzahl: 0, summe: 0 });

	// Der Aufrufer bestimmt den Reiter, der beim Öffnen oben liegt. Bei jedem Wechsel des
	// Lesers zurück auf ihn, sonst speichert ein offenes Blatt auf den falschen Leser.
	$effect(() => {
		if (!student?.id) return;
		st.activeTab = defaultTab;
		st.schliesseAlleBlaetter();
		st.fetchProfile(student.id);
	});

	export const reloadProfile = () => st.fetchProfile(st.profile?.id ?? student?.id);

	// Vom Bediener gesetztes Ablaufjahr für genau diesen Ausdruck; null = Vorschlag des Servers.
	// Nicht gespeichert, beim Wechsel des Lesers fällt es zurück.
	let gueltigBisOverride = $state(/** @type {number|null} */ (null));
	let zuletztGezeigteId = $state(/** @type {string|null} */ (null));
	$effect(() => {
		// Nur beim Wechsel des Lesers: Ein Neuladen des Profils lässt die getippte Jahreszahl stehen.
		const id = st.profile?.id ?? null;
		if (id !== zuletztGezeigteId) {
			zuletztGezeigteId = id;
			gueltigBisOverride = null;
		}
	});
	const gueltigBisEffektiv = $derived(
		gueltigBisOverride ?? st.profile?.ausweis_gueltig_bis ?? null
	);
</script>

{#if st.loading}
	<div class="w-full py-12 flex justify-center items-center">
		<Ladekreis size="lg" />
	</div>
{:else if st.profile}
	{#if st.globalErrorToast}
		<div
			class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-error text-on-error flex items-center gap-2"
		>
			<Info class="h-5 w-5" aria-hidden="true" />
			{st.globalErrorToast}
		</div>
	{/if}

	{#if !st.showEditModal}
		<div
			class="w-full grid grid-cols-1 lg:grid-cols-[320px_minmax(0,1fr)] items-stretch text-on-surface animate-fade-in no-print print:hidden font-sans"
		>
			<StudentProfileCard
				bind:profile={st.profile}
				{rechte}
				timestamp={st.timestamp}
				bind:showWebcam={st.showWebcam}
				bind:showDeleteConfirm={st.showDeleteConfirm}
				{onDeselect}
				{leftActions}
				{offen}
				onLock={rechte.bearbeiten ? () => (st.showLockModal = true) : undefined}
			/>

			<!-- pt-4: Die Knopfzeile steht auf der Höhe des Schließen-Knopfs der linken Spalte. -->
			<div class="lg:col-span-1 space-y-6 flex flex-col h-full px-6 pt-4 pb-4">
				{#if rechte.einsehen}
					<StudentProfileActions
						profile={st.profile}
						darfAuskunft={darfAuskunftUeber(rechte, st.profile)}
						kontoauszugPdfLoading={st.kontoauszugPdfLoading}
						rechnungPdfLoading={st.rechnungPdfLoading}
						downloadKontoauszugPDF={st.downloadKontoauszugPDF}
						downloadRechnungPDF={st.downloadRechnungPDF}
						onPrint={druckeAusweis}
						gueltigBis={gueltigBisEffektiv}
						onGueltigBis={(jahr) => (gueltigBisOverride = jahr)}
					/>
				{/if}

				<Reiter
					etikett="Leserakte"
					reiter={[
						{ id: 'ausleihen', label: 'Ausleihen & Historie' },
						{ id: 'gebuehren', label: 'Gebühren & Schäden', anzahl: offen.anzahl },
						{ id: 'stammdaten', label: 'Stammdaten & Adresse' }
					]}
					aktiv={st.activeTab}
					onwahl={(id) => (st.activeTab = id)}
				/>

				<div class="flex-1 relative">
					{#if st.activeTab === 'ausleihen'}
						<StudentProfileAusleihen
							buecher={st.profile.entliehene_buecher || []}
							bind:vormerkungen={st.vormerkungen}
							fehlendeListen={st.fehlendeListen.filter((l) => l === 'Vormerkungen')}
							{onReturnClick}
							onDamageClick={rechte.bearbeiten ? st.openDamageModal : undefined}
							onChanged={() => st.fetchProfile(st.profile.id)}
							{rightTop}
						/>
					{:else if st.activeTab === 'gebuehren'}
						<StudentProfileGebuehren
							schuelerId={st.profile.id}
							gebuehren={st.gebuehren}
							bescheide={st.bescheide}
							fehlendeListen={st.fehlendeListen.filter((l) => l !== 'Vormerkungen')}
							canEdit={rechte.bearbeiten}
							onChanged={() => st.fetchProfile(st.profile.id)}
						/>
					{:else if st.activeTab === 'stammdaten'}
						<StudentProfileStammdaten
							profile={st.profile}
							darfBearbeiten={rechte.bearbeiten}
							darfZusammenfuehren={rechte.zusammenfuehren}
							onEdit={() => (st.showEditModal = true)}
							onMerged={(id) => (st.fetchProfile(id), onMerged?.(id))}
						/>

						<!-- Gefahrenzone am unteren Ende des Stammdaten-Reiters, seit 16.09.2026 auch beim Kollegium -->
						{#if rechte.loeschen}
							<StudentDangerZone
								profile={st.profile}
								onDelete={() => (st.showDeleteConfirm = true)}
							/>
						{/if}
					{/if}
				</div>
			</div>
		</div>
	{:else}
		<StudentEditSheet
			student={st.profile}
			onClose={() => (st.showEditModal = false)}
			onSave={() => st.handleSaveEdit(st.profile.id)}
		/>
	{/if}
{:else}
	<LadeFehler
		onerneut={reloadProfile}
		titel="Akte nicht geladen"
		text="Die Daten dieses Lesers konnten nicht abgerufen werden."
	/>
{/if}

{#if st.showWebcam}
	<WebcamCapture
		studentId={st.profile.id}
		onCapture={() => st.handlePhotoCaptured(st.profile.id)}
		onClose={() => (st.showWebcam = false)}
	/>
{/if}

{#if st.profile}
	<!-- Die Abweichung liegt auf dem Profil, nicht in CardFace: So drucken Profil, Klassensatz
	     und Designer dieselbe Karte. -->
	<StudentPrintCard
		profile={{ ...st.profile, ausweis_gueltig_bis: gueltigBisEffektiv }}
		timestamp={st.timestamp}
	/>
{/if}

<StudentProfileDeleteModal
	open={st.showDeleteConfirm}
	profile={st.profile}
	onclose={() => (st.showDeleteConfirm = false)}
	onsuccess={() => st.handleDeleteSuccess(onDeselect)}
/>

{#if st.showDamageModal && st.damageBook}
	<DamageReportModal
		book={st.damageBook}
		isSubmitting={st.isSubmittingDamage}
		onCancel={() => (st.showDamageModal = false)}
		onSubmit={(r, a, art) => st.submitDamageReport(st.profile.id, r, a, art)}
	/>
{/if}

<StudentLockModal
	bind:open={st.showLockModal}
	profile={st.profile}
	onsuccess={st.handleLockSuccess}
/>

{#if st.profile}
	<StudentPrintReceipt profile={st.profile} />
{/if}
