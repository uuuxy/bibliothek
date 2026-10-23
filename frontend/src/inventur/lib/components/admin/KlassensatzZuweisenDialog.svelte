<script>
	// Kleiner, wiederverwendbarer Dialog: weist die übergebenen Bücher (bookIds) einer
	// Schulklasse zu. Genutzt an zwei Stellen — Mehrfachauswahl in der Bücher-Liste und
	// „Klasse zuweisen" in der Buch-Bearbeitungsmaske. Additiv über
	// POST /api/admin/class-books/add (klassenname wird serverseitig normalisiert).
	import { apiFetch, apiClient, leseApiFehler } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import Dialog from '../../../../lib/Dialog.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Select from '../../../../lib/components/ui/Select.svelte';
	import {
		erzeugeKlassenVorschlaege,
		klassenPlatzhalter,
		KLASSEN_LADEFEHLER
	} from '../../../../lib/components/leser/klassenVorschlaege.svelte.js';

	/** @type {{ titelIds: string[], onClose: () => void, onAssigned: () => void }} */
	let { titelIds, onClose, onAssigned } = $props();

	let klassenName = $state('');
	/** @type {string[]} */
	let existing = $state([]);
	let speichert = $state(false);
	let fehler = $state('');

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

	async function weiseZu() {
		const name = klassenName.trim();
		if (!name) {
			fehler = 'Bitte eine Klasse wählen.';
			return;
		}
		speichert = true;
		fehler = '';
		try {
			const res = await apiClient.post('/api/admin/class-books/add', {
				classNames: [name],
				bookIds: titelIds
			});
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Zuweisung fehlgeschlagen.'));
			onAssigned();
		} catch (err) {
			fehler = /** @type {any} */ (err).message;
		} finally {
			speichert = false;
		}
	}
</script>

<!-- Hintergrund, Fokusfalle und Escape stellt Dialog.svelte; der Name des Dialogs kommt als
     `beschriftung`. -->
<Dialog open={true} onclose={onClose} beschriftung="Zum Klassensatz hinzufügen">
	<div class="p-6 space-y-5">
		<h3 class="text-lg font-bold text-on-surface">Zum Klassensatz hinzufügen</h3>
		<p class="text-sm text-on-surface-variant">
			{titelIds.length}
			{titelIds.length === 1 ? 'Buch' : 'Bücher'} einer Schulklasse zuweisen.
		</p>

		<div class="grid gap-y-1.5">
			<label for="klasse-name" class="text-sm font-medium text-on-surface-variant">Klasse</label>
			<Select
				id="klasse-name"
				bind:value={klassenName}
				options={optionen}
				placeholder={klassenPlatzhalter(optionen.length, ladefehler)}
			/>
			{#if ladefehler}
				<span class="text-xs text-error">{KLASSEN_LADEFEHLER}</span>
			{/if}
		</div>

		{#if fehler}
			<div class="text-sm text-error">{fehler}</div>
		{/if}

		<div class="flex justify-end gap-3">
			<Button variant="ghost" onclick={onClose}>Abbrechen</Button>
			<Button onclick={weiseZu} disabled={speichert}>
				{speichert ? 'Speichern...' : 'Zuweisen'}
			</Button>
		</div>
	</div>
</Dialog>
