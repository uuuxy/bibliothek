<script>
	/**
	 * @component KlassenDruckEinstieg
	 * Sichtbarer Einstieg für den klassenweisen Ausweisdruck (24.08.2026, auf Peters
	 * Ansage): Der Weg über die Schülerdatei — Klasse suchen, Kopf-Kästchen anhaken,
	 * Aktionsleiste — funktioniert, ist aber für neue Benutzer unauffindbar, weil die
	 * Leiste erst mit der ersten Markierung erscheint.
	 *
	 * Dieser Block druckt bewusst NICHT selbst (keine zweite Tür zum selben Zustand):
	 * Er springt in die Schülerdatei mit fertig markierter Klasse. Damit bleiben die
	 * Kontrollen des EINEN Druckwegs davor — die Warnung bei fehlendem Ablaufdatum
	 * und die Startposition des Etikettenbogens.
	 */
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';

	/** @type {string[]} */
	let klassen = $state([]);
	let klasse = $state('');

	onMount(async () => {
		try {
			const res = await apiFetch('/api/klassen');
			if (res.ok) {
				const data = await res.json();
				klassen = Array.isArray(data) ? data : [];
			}
		} catch {
			/* Ohne Klassenliste bleibt der Weg über die Schülerdatei offen. */
		}
	});

	function markieren() {
		if (!klasse) return;
		uiStore.requestedKlassenDruck = klasse;
		uiStore.activeTab = 'students_dir';
	}
</script>

<div class="flex flex-col gap-4">
	<p class="max-w-2xl text-body-medium text-on-surface-variant">
		Öffnet die Schülerdatei mit fertig markierter Klasse — dort startet der Druck über die
		Aktionsleiste (Ausweiskarten oder Etikettenbogen, je nach Betriebsart im Reiter
		„Schülerausweise“).
	</p>
	<div class="flex flex-wrap items-end gap-4">
		<Select
			bind:value={klasse}
			options={klassen.map((k) => ({ value: k, label: k }))}
			placeholder="Klasse wählen"
			aria-label="Klasse"
			class="w-48"
		/>
		<Button onclick={markieren} disabled={!klasse}>Klasse zum Druck markieren</Button>
	</div>
</div>
