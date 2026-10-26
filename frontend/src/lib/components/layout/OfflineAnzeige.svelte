<!-- @component OfflineAnzeige — ein Band, kein Vollbild: An der Theke darf der
     Verbindungsverlust das Scannen nicht anhalten, und ein großer Block ließe nichts mehr
     buchen.

     Drei Lautstärken statt einer, weil die drei Lagen nicht gleich schlimm sind:
       1. Warteschlange nicht lesbar: Offline-Scans gehen verloren. Laut (Fehlerrolle).
       2. Vorgänge liegen auf diesem Rechner: sichtbar, mit Sicherung. Mittel.
       3. Nur keine Verbindung, nichts offen: eine Zeile, leise, ohne Bitte um eine
          Sicherung, denn es gibt nichts zu sichern.

     Farben aus den M3-Rollen statt aus der Palette (Farb-Ratsche zählt die Summe). -->
<script>
	import { offlineSync } from '../../stores/offlineSync.svelte.js';
	import { nachbuchMeldungen } from '../../stores/nachbuchMeldungen.svelte.js';
	import { CloudOff, Download, ListChecks, TriangleAlert, Upload } from '@lucide/svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { hatRecht } from '../../menue.js';
	import { authStore } from '../../stores/authStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import NachbuchMeldungen from './NachbuchMeldungen.svelte';
	import { fehlertext } from '../../utils/fehlertext.js';

	// Der Herzschlag kommt von außen (App.svelte), und sein Ausfall steht in derselben Zeile
	// wie jeder andere Verbindungsverlust: dieselbe Lage, eine Anzeige.
	/** @type {{ verbindungVerloren?: boolean }} */
	let { verbindungVerloren = false } = $props();

	let ohneNetz = $derived(offlineSync.ohneNetz || verbindungVerloren);
	// Die Liste der Meldungen darf `view_students` sehen; die Zahl gehört jeder Theken-Rolle.
	// Ein Helfer sieht also, dass etwas offen ist, und holt jemanden; das ist besser als eine
	// Meldung, die niemandem auffällt.
	let darfMeldungenSehen = $derived(hatRecht(authStore.currentUser, 'view_students'));
	let sichtbar = $derived(
		offlineSync.wartendeAnzahl > 0 ||
			ohneNetz ||
			offlineSync.warteschlangeFehler ||
			offlineSync.abgelehnt !== null ||
			nachbuchMeldungen.offen > 0 ||
			nachbuchMeldungen.geoeffnet
	);
	// Rot wie der Warteschlangen-Fehler: Beide heißen „hier muss jemand etwas tun",
	// während der Zähler allein nur „es läuft noch" heißt.
	let haengt = $derived(offlineSync.warteschlangeFehler || offlineSync.abgelehnt !== null);

	async function sichere() {
		try {
			await offlineSync.exportiereWarteschlange();
		} catch (err) {
			console.error('Sicherung nicht möglich:', err);
			snackbars.addToast(
				'Sicherung nicht möglich: Die Warteschlange auf diesem Rechner ist nicht lesbar.',
				'error'
			);
		}
	}

	/** @type {HTMLInputElement | null} */
	let fileInput = $state(null);

	// Mehrere Dateien auf einmal: Bei zehn Kiosk-Rechnern liegen im Sicherungsordner
	// zehn Dateien. Sie einzeln auszuwählen lädt dazu ein, eine zu übersehen.
	async function waehleDatei(e) {
		const input = /** @type {HTMLInputElement} */ (e.target);
		const files = [...(input.files ?? [])];
		if (files.length === 0) return;

		let gesamt = 0;
		const fehler = [];
		for (const file of files) {
			try {
				gesamt += await offlineSync.importiereWarteschlange(file);
			} catch (err) {
				fehler.push(`${file.name}: ${fehlertext(err)}`);
			}
		}

		if (gesamt > 0) {
			snackbars.addToast(
				`${gesamt} Vorgang/Vorgänge aus ${files.length - fehler.length} Datei(en) übernommen — werden jetzt übertragen.`,
				'success'
			);
		}
		if (fehler.length > 0) {
			snackbars.addToast(fehler.join(' · '), 'error');
		}
		input.value = ''; // reset
	}
</script>

{#if sichtbar}
	<div
		class="fixed top-0 right-0 left-0 z-9999 border-b {haengt
			? 'bg-error text-on-error border-error'
			: offlineSync.wartendeAnzahl > 0
				? 'bg-error-container text-on-error-container border-outline-variant'
				: 'bg-surface-container-low text-on-surface-variant border-outline-variant'}"
		role="status"
		aria-live="polite"
	>
		<div class="mx-auto flex max-w-7xl items-center gap-3 px-4 py-1.5">
			{#if haengt}
				<TriangleAlert size={18} strokeWidth={2.5} class="shrink-0" />
			{:else}
				<CloudOff size={18} strokeWidth={2.5} class="shrink-0" />
			{/if}

			<p class="min-w-0 flex-1 truncate text-sm font-semibold">
				{#if offlineSync.warteschlangeFehler}
					Warteschlange nicht lesbar — Offline-Scans werden auf diesem Rechner NICHT gespeichert.
					Bitte an einem anderen Rechner weiterarbeiten.
				{:else if offlineSync.abgelehnt?.status === 401 || offlineSync.abgelehnt?.status === 403}
					<!-- Warten hilft hier nicht: Die Vorgänge sind gespeichert, aber diese Anmeldung
					     darf sie nicht buchen. Ohne diesen Satz lief der Versuch jede Minute ins
					     Leere, und das Band zeigte weiter nur „noch nicht im System". -->
					{offlineSync.wartendeAnzahl}
					{offlineSync.wartendeAnzahl === 1 ? 'Vorgang wartet' : 'Vorgänge warten'} — diese Anmeldung
					darf an der Theke nicht buchen. Bitte mit dem Bibliotheks-Konto anmelden; gespeichert bleibt
					alles.
				{:else if offlineSync.abgelehnt !== null}
					<!-- Der Grund des Servers steht da: Wer die Bibliothek verständigt, kann ihn nennen. -->
					{offlineSync.wartendeAnzahl}
					{offlineSync.wartendeAnzahl === 1 ? 'Vorgang lässt' : 'Vorgänge lassen'} sich nicht buchen:
					{offlineSync.abgelehnt.grund.replace(/\.$/, '')}. Gespeichert bleibt alles; bitte die
					Bibliothek verständigen.
				{:else if offlineSync.wartendeAnzahl > 0}
					<!-- Die Mehrzahl ausgeschrieben: Ein angehängtes „e" ergäbe „Vorgange". -->
					{offlineSync.wartendeAnzahl}
					{offlineSync.wartendeAnzahl === 1 ? 'Vorgang' : 'Vorgänge'} nur auf diesem Rechner — noch nicht
					im System.
					{#if offlineSync.gleichtAb}
						Wird übertragen …
					{:else if ohneNetz}
						Sobald die Verbindung zurück ist, geschieht das von selbst. Vorher bitte sichern.
					{/if}
				{:else if nachbuchMeldungen.offen > 0}
					{nachbuchMeldungen.offen}
					{nachbuchMeldungen.offen === 1 ? 'Buchung' : 'Buchungen'} aus dem Nachbuchen
					{nachbuchMeldungen.offen === 1 ? 'braucht' : 'brauchen'} einen Blick.
					{#if !darfMeldungenSehen}
						Bitte die Bibliothek ansprechen.
					{/if}
				{:else}
					Keine Verbindung — Scannen geht weiter, die Buchungen folgen von selbst.
				{/if}
			</p>

			{#if nachbuchMeldungen.offen > 0 && darfMeldungenSehen}
				<Button
					variant="secondary"
					size="sm"
					onclick={() => nachbuchMeldungen.oeffne()}
					class="shrink-0"
				>
					<ListChecks size={16} strokeWidth={2.5} />
					Meldungen ({nachbuchMeldungen.offen})
				</Button>
			{/if}

			{#if offlineSync.wartendeAnzahl > 0}
				<Button variant="secondary" size="sm" onclick={sichere} class="shrink-0">
					<Download size={16} strokeWidth={2.5} />
					Sicherung speichern
				</Button>
			{/if}

			<!-- Verstecktes File Input für Import -->
			<input
				type="file"
				accept=".json"
				multiple
				bind:this={fileInput}
				onchange={waehleDatei}
				class="hidden"
			/>

			<Button
				variant="ghost"
				size="sm"
				onclick={() => fileInput?.click()}
				class="shrink-0"
				title="Sicherung eines anderen Rechners übernehmen"
			>
				<Upload size={16} strokeWidth={2.5} />
				Sicherung einspielen
			</Button>
		</div>
	</div>
{/if}

<NachbuchMeldungen />
