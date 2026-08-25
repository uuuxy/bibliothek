<script>
	// Kleiner, wiederverwendbarer Dialog: weist die übergebenen Bücher (bookIds) einer
	// Schulklasse zu. Genutzt an zwei Stellen — Mehrfachauswahl in der Bücher-Liste und
	// „Klasse zuweisen" in der Buch-Bearbeitungsmaske. Additiv über
	// POST /api/admin/class-books/add (klassenname wird serverseitig normalisiert).
	import { apiFetch, apiClient } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';

	/** @type {{ bookIds: string[], onClose: () => void, onAssigned: () => void }} */
	let { bookIds, onClose, onAssigned } = $props();

	let className = $state('');
	/** @type {string[]} */
	let existing = $state([]);
	let saving = $state(false);
	let error = $state('');

	onMount(async () => {
		// Bestehende Klassennamen als Auswahlvorschläge (Datalist). Rein optional —
		// scheitert der Abruf, tippt man den Namen einfach frei.
		try {
			const res = await apiFetch('/api/admin/class-books', { credentials: 'include' });
			if (res.ok) {
				const json = await res.json();
				existing = (json.data || []).map((/** @type {any} */ g) => g.className);
			}
		} catch {
			/* Vorschläge sind optional */
		}
	});

	async function assign() {
		const name = className.trim();
		if (!name) {
			error = 'Bitte einen Klassennamen angeben.';
			return;
		}
		saving = true;
		error = '';
		try {
			const res = await apiClient.post('/api/admin/class-books/add', {
				classNames: [name],
				bookIds
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				throw new Error(body.error || 'Zuweisung fehlgeschlagen.');
			}
			onAssigned();
		} catch (err) {
			error = /** @type {any} */ (err).message;
		} finally {
			saving = false;
		}
	}
</script>

<!-- Escape schliesst; vorher fuehrte nur der Klick auf den Hintergrund heraus. -->
<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape') onClose();
	}}
/>

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
	role="presentation"
	onclick={(e) => {
		// Der Treffer-Vergleich ersetzt das stopPropagation, das vorher am Dialog hing.
		// Dort machte es das Dialog-Element zu einem Klickziel und verlangte damit
		// tabindex und einen Tastaturweg — fuer einen Handler, der nichts tut, ausser
		// ein Ereignis aufzuhalten. Hier gefragt ist ohnehin nur: Wurde der Hintergrund
		// selbst getroffen?
		if (e.target === e.currentTarget) onClose();
	}}
>
	<div
		class="bg-white rounded-3xl shadow-xl w-full max-w-md p-6 space-y-5"
		role="dialog"
		aria-modal="true"
		aria-label="Klasse zuweisen"
	>
		<h3 class="text-lg font-bold text-slate-900">Klasse zuweisen</h3>
		<p class="text-sm text-slate-500">
			{bookIds.length}
			{bookIds.length === 1 ? 'Buch' : 'Bücher'} einer Schulklasse zuweisen.
		</p>

		<Feld
			id="klasse-name"
			label="Klasse"
			list="klassen-vorschlaege"
			bind:value={className}
			placeholder="z. B. 5a"
			maxlength={20}
		/>
		<datalist id="klassen-vorschlaege">
			{#each existing as name (name)}
				<option value={name}></option>
			{/each}
		</datalist>

		{#if error}
			<div class="text-sm text-rose-600">{error}</div>
		{/if}

		<div class="flex justify-end gap-3">
			<Button variant="ghost" onclick={onClose}>Abbrechen</Button>
			<Button onclick={assign} disabled={saving}>
				{saving ? 'Speichern...' : 'Zuweisen'}
			</Button>
		</div>
	</div>
</div>
