<script>
	import { showToast } from '../../inventur/lib/store.svelte.js';
	import { apiClient } from '../apiFetch.js';
	import Select from './ui/Select.svelte';

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
	let editStatusType = $state(
		ex.ist_ausleihbar
			? 'Verfügbar'
			: ex.ist_ausgesondert ||
				  (ex.zustand_notiz && ex.zustand_notiz.toLowerCase().includes('verloren'))
				? 'Verloren'
				: 'Gesperrt (Defekt/Reserviert)'
	);
	let editStatusNote = $state(ex.zustand_notiz || '');
	let statusError = $state('');

	async function saveStatus() {
		statusError = '';
		try {
			const isAusleihbar = editStatusType === 'Verfügbar';
			const isAusgesondert = editStatusType === 'Verloren' ? true : false;
			const notiz = isAusleihbar ? '' : editStatusNote.trim();
			const res = await apiClient.put(`/api/buecher/exemplare/${ex.id}/status`, {
				ist_ausleihbar: isAusleihbar,
				ist_ausgesondert: isAusgesondert,
				zustand_notiz: notiz
			});
			if (res.ok) {
				ex.ist_ausleihbar = isAusleihbar;
				ex.ist_ausgesondert = isAusgesondert;
				ex.zustand_notiz = notiz;
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
		<div class="mb-2">
			<input
				type="text"
				bind:value={editStatusNote}
				placeholder="Notiz (optional)"
				class="w-full text-xs border border-slate-300 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
				onkeydown={(e) => {
					if (e.key === 'Enter') saveStatus();
					if (e.key === 'Escape') onDone();
				}}
			/>
		</div>
	{/if}
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
