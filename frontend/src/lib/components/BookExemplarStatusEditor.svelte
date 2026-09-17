<script>
	import { showToast } from '../../inventur/lib/store.svelte.js';
	import { apiClient } from '../apiFetch.js';
	import Select from './ui/Select.svelte';
	import Feld from './ui/Feld.svelte';
	import Button from './ui/Button.svelte';
	import { formatEuro } from '../utils/format.js';
	import { ersatzwertBekannt } from './exemplarErsatzwert.js';

	const STATUS = ['Verfügbar', 'Gesperrt (Defekt/Reserviert)', 'Verloren'].map((s) => ({
		value: s,
		label: s
	}));

	/**
	 * Status-Editor eines Exemplars (Verfügbar / Gesperrt / Verloren).
	 * Initialisiert sich aus ex; speichert in-place und schließt via onDone.
	 * @type {{ ex: any, onDone: () => void }}
	 */
	let { ex, onDone } = $props();

	// Der Compiler schlaegt hier $derived vor ("captures the initial value of ex").
	// BEWUSST NICHT: Das sind Arbeitskopien im Editor. Als $derived wuerde jede
	// Aenderung an ex die Eingabe des Benutzers ueberschreiben — genau das, was ein
	// Formular nicht tun darf. Einmal aus dem Prop befuellen ist hier richtig.
	// svelte-ignore state_referenced_locally
	let editStatusType = $state(
		ex.ist_ausleihbar
			? 'Verfügbar'
			: ex.ist_ausgesondert ||
				  (ex.zustand_notiz && ex.zustand_notiz.toLowerCase().includes('verloren'))
				? 'Verloren'
				: 'Gesperrt (Defekt/Reserviert)'
	);
	// svelte-ignore state_referenced_locally
	let editStatusNote = $state(ex.zustand_notiz || '');
	// Der Beschädigungsgrad (Migration 127): Was das Buch durch seinen Zustand an Wert
	// verloren hat. Er steht in DIESEM Dialog, weil er zum Zustand gehört — und er
	// überlebt „Verfügbar", anders als die Notiz: Ein Band mit Wasserrand darf
	// ausleihbar sein und trägt seinen Abschlag weiter in jeden künftigen Ersatzbetrag.
	// svelte-ignore state_referenced_locally
	let editAbwertung = $state(ex.zustand_abwertung_prozent ?? 0);
	let statusError = $state('');

	async function saveStatus() {
		statusError = '';
		try {
			const isAusleihbar = editStatusType === 'Verfügbar';
			const isAusgesondert = editStatusType === 'Verloren' ? true : false;
			const notiz = isAusleihbar ? '' : editStatusNote.trim();
			/** @type {Record<string, any>} */
			const koerper = {
				ist_ausleihbar: isAusleihbar,
				ist_ausgesondert: isAusgesondert,
				zustand_notiz: notiz
			};
			// Der Grad reist nur mit, wenn wir wissen, was drinstand, oder wenn hier
			// etwas eingetragen ist. Hätte eine alte Antwort das Feld nicht geliefert,
			// wäre die 0 aus dem Startwert sonst eine stille Löschung eines erfassten
			// Schadens — der Server lässt ein FEHLENDES Feld ausdrücklich unangetastet.
			const grad = Number(editAbwertung);
			const bekannt = typeof ex.zustand_abwertung_prozent === 'number';
			if (Number.isFinite(grad) && (bekannt || grad !== 0)) {
				koerper.zustand_abwertung_prozent = grad;
			}
			const res = await apiClient.put(`/api/buecher/exemplare/${ex.id}/status`, koerper);
			if (res.ok) {
				ex.ist_ausleihbar = isAusleihbar;
				ex.ist_ausgesondert = isAusgesondert;
				ex.zustand_notiz = notiz;
				if ('zustand_abwertung_prozent' in koerper) {
					ex.zustand_abwertung_prozent = grad;
				}
				// Der neue Ersatzwert kommt aus der Antwort, nicht aus einer Rechnung
				// hier: Sonst stünde an der Karte eine zweite Zahl für dasselbe Buch —
				// und ohne ihn zeigte sie nach dem Speichern den alten Wert.
				const antwort = await res.json().catch(() => ({}));
				if (typeof antwort.ersatzwert === 'number') {
					ex.ersatzwert = antwort.ersatzwert;
					ex.ersatzwert_herleitung = antwort.ersatzwert_herleitung ?? '';
					// Die Auskunft „liegt ein Preis zugrunde?" muss MITgehen. Bliebe der alte
					// Wert stehen, zeigte die Karte nach dem Speichern eine Zeile, die zur neuen
					// Zahl nicht mehr passt — oder ließe sie weg, obwohl es jetzt eine gibt.
					ex.ersatzwert_bekannt = antwort.ersatzwert_bekannt === true;
				}
				onDone();
				showToast('Status erfolgreich gespeichert', 'success');
			} else {
				const errData = await res.json().catch(() => ({}));
				statusError = errData.error || 'Fehler beim Speichern';
			}
		} catch {
			statusError = 'Netzwerkfehler';
		}
	}
</script>

<div class="mt-2 rounded-xl border border-outline-variant bg-surface-container-low p-3">
	<div class="flex items-center gap-2 mb-2">
		<Select bind:value={editStatusType} options={STATUS} aria-label="Status des Exemplars" />
	</div>
	{#if editStatusType !== 'Verfügbar'}
		<Feld
			bind:value={editStatusNote}
			placeholder="Notiz (optional)"
			aria-label="Notiz zum Status"
			feld="mb-2"
			onkeydown={(e) => {
				if (e.key === 'Enter') saveStatus();
				if (e.key === 'Escape') onDone();
			}}
		/>
	{/if}
	<!-- Immer sichtbar, auch bei „Verfügbar": Der Abschlag ist eine Eigenschaft des
	     Buchs, kein Status. 0 heißt „kein Schaden erfasst".

	     SICHTBARE Beschriftung statt nachlaufendem Text im Feld: Der Zusatz stand am
	     rechten Rand des Feldkastens und damit weit weg von der Zahl — „20" links,
	     „% Wertverlust" 15 cm daneben. Eine Beschriftung über dem Feld ist die Bauform
	     des Hauses (Feld.svelte) und benennt das Feld auch für den Screenreader. -->
	<div class="mb-2 max-w-40">
		<Feld
			bind:value={editAbwertung}
			label="Wertverlust (%)"
			type="number"
			min="0"
			max="100"
			step="5"
			feld="w-24"
			onkeydown={(e) => {
				if (e.key === 'Enter') saveStatus();
				if (e.key === 'Escape') onDone();
			}}
		/>
	</div>
	{#if ersatzwertBekannt(ex)}
		<!-- Die Herleitung steht HIER und nicht auf der Karte: Nachgerechnet wird der
		     Betrag dort, wo man ihn beeinflusst. Nach dem Speichern trägt die Antwort des
		     Servers den neuen Wert — gerechnet wird nie in der Oberfläche. -->
		<p class="text-label-small mb-2 text-on-surface-variant">
			<span class="font-semibold text-on-surface"
				>Ersatzwert heute: {formatEuro(ex.ersatzwert ?? 0)}</span
			>
			<span class="block">{ex.ersatzwert_herleitung}</span>
		</p>
	{/if}
	<div class="flex items-center justify-between gap-2">
		<Button variant="ghost" size="sm" onclick={onDone}>Abbrechen</Button>
		<Button size="sm" onclick={saveStatus}>Speichern</Button>
	</div>
	{#if statusError}
		<p class="text-label-small text-error mt-1">{statusError}</p>
	{/if}
</div>
