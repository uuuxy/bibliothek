<script>
	// Kleiner, wiederverwendbarer Dialog: weist die übergebenen Bücher (bookIds) einer
	// Schulklasse zu. Genutzt an zwei Stellen — Mehrfachauswahl in der Bücher-Liste und
	// „Klasse zuweisen" in der Buch-Bearbeitungsmaske. Additiv über
	// POST /api/admin/class-books/add (klassenname wird serverseitig normalisiert).
	import { apiFetch, apiClient } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import Modal from '../../../../lib/Modal.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import {
		erzeugeKlassenVorschlaege,
		klassenPlatzhalter,
		KLASSEN_LADEFEHLER
	} from '../../../../lib/components/students/klassenVorschlaege.svelte.js';

	/** @type {{ bookIds: string[], onClose: () => void, onAssigned: () => void }} */
	let { bookIds, onClose, onAssigned } = $props();

	let className = $state('');
	/** @type {string[]} */
	let existing = $state([]);
	let saving = $state(false);
	let error = $state('');

	// Auswählen statt tippen: Ein Tippfehler legte sonst eine Klasse an, die es an der Schule
	// nicht gibt. Zur Wahl stehen die Klassen der Schüler und die Klassen, die schon eine
	// Buchliste haben — die 05F1 bleibt so auch im Sommer wählbar, bevor der LUSD-Abgleich die
	// neuen Fünftklässler bringt.
	const klassenListe = erzeugeKlassenVorschlaege();
	const optionen = $derived(
		[...new Set([...klassenListe.liste, ...existing])].sort().map((k) => ({ value: k, label: k }))
	);

	// Scheitert eine der beiden Quellen, fehlt womöglich genau die gesuchte Klasse; der Dialog
	// sagt es dann unter dem Auswahlfeld.
	let buchlistenFehler = $state(false);
	const ladefehler = $derived(klassenListe.ladefehler || buchlistenFehler);

	onMount(async () => {
		klassenListe.lade();
		try {
			const res = await apiFetch('/api/admin/class-books', { credentials: 'include' });
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			existing = (json.data || []).map((/** @type {any} */ g) => g.className);
		} catch {
			buchlistenFehler = true;
		}
	});

	async function assign() {
		const name = className.trim();
		if (!name) {
			error = 'Bitte eine Klasse wählen.';
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

<!-- Hintergrund, Fokusfalle und Escape stellt Modal.svelte; der Name des Dialogs kommt als
     `beschriftung`. -->
<Modal open={true} onclose={onClose} beschriftung="Zum Klassensatz hinzufügen">
	<div class="p-6 space-y-5">
		<h3 class="text-lg font-bold text-on-surface">Zum Klassensatz hinzufügen</h3>
		<p class="text-sm text-on-surface-variant">
			{bookIds.length}
			{bookIds.length === 1 ? 'Buch' : 'Bücher'} einer Schulklasse zuweisen.
		</p>

		<div class="grid gap-y-1.5">
			<label for="klasse-name" class="text-sm font-medium text-on-surface-variant">Klasse</label>
			<Select
				id="klasse-name"
				bind:value={className}
				options={optionen}
				placeholder={klassenPlatzhalter(optionen.length, ladefehler)}
			/>
			{#if ladefehler}
				<span class="text-xs text-error">{KLASSEN_LADEFEHLER}</span>
			{/if}
		</div>

		{#if error}
			<div class="text-sm text-error">{error}</div>
		{/if}

		<div class="flex justify-end gap-3">
			<Button variant="ghost" onclick={onClose}>Abbrechen</Button>
			<Button onclick={assign} disabled={saving}>
				{saving ? 'Speichern...' : 'Zuweisen'}
			</Button>
		</div>
	</div>
</Modal>
