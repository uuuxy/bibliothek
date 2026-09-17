<script>
	import { showToast } from '../../inventur/lib/store.svelte.js';
	import { apiClient } from '../apiFetch.js';
	import Select from './ui/Select.svelte';
	import Feld from './ui/Feld.svelte';

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

<div class="mt-2 bg-slate-50 p-3 rounded-lg border border-slate-200">
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
	     Buchs, kein Status. 0 heißt „kein Schaden erfasst". -->
	<Feld
		bind:value={editAbwertung}
		type="number"
		min="0"
		max="100"
		step="5"
		aria-label="Wertverlust durch Beschädigung in Prozent"
		feld="mb-2 w-24"
		onkeydown={(e) => {
			if (e.key === 'Enter') saveStatus();
			if (e.key === 'Escape') onDone();
		}}
	>
		{#snippet nachlaufend()}% Wertverlust{/snippet}
	</Feld>
	<div class="flex items-center justify-between">
		<button
			onclick={onDone}
			class="text-label-small text-slate-500 hover:text-slate-700 font-semibold cursor-pointer"
			>Abbrechen</button
		>
		<button
			onclick={saveStatus}
			class="text-label-small bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded font-semibold cursor-pointer"
			>Speichern</button
		>
	</div>
	{#if statusError}
		<p class="text-label-small text-rose-600 mt-1">{statusError}</p>
	{/if}
</div>
