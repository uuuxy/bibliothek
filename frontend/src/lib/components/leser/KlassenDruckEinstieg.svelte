<script>
	/**
	 * @component KlassenDruckEinstieg
	 * Sichtbarer Einstieg für den klassenweisen Ausweisdruck: Der Weg über die Leserdatei
	 * (Klasse suchen, Kopf-Kästchen anhaken, Aktionsleiste) funktioniert, ist aber für neue
	 * Benutzer unauffindbar, weil die Leiste erst mit der ersten Markierung erscheint.
	 *
	 * Dieser Block druckt nicht selbst (keine zweite Tür zum selben Zustand): Er springt in
	 * die Leserdatei mit fertig markierter Klasse. Damit bleiben die Kontrollen des einen
	 * Druckwegs davor, die Warnung bei fehlendem Ablaufdatum und die Startposition des
	 * Etikettenbogens.
	 */
	import { onMount } from 'svelte';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Select from '../ui/Select.svelte';

	/** @type {string[]} */
	let klassen = $state([]);
	let klasse = $state('');
	// Der Grund, wenn die Klassenliste nicht kam. Die Klasse wird hier gewählt, nicht getippt:
	// Eine leere Auswahl ohne Wort sähe aus wie eine Schule ohne Klassen.
	let ladefehler = $state('');

	async function ladeKlassen() {
		ladefehler = '';
		try {
			const res = await apiFetch('/api/klassen');
			if (!res.ok) {
				ladefehler = await leseApiFehler(res);
				return;
			}
			const daten = await res.json();
			klassen = Array.isArray(daten) ? daten : [];
		} catch (e) {
			ladefehler = fehlertext(e);
		}
	}

	onMount(ladeKlassen);

	function markieren() {
		if (!klasse) return;
		uiStore.requestedKlassenDruck = klasse;
		uiStore.activeTab = 'students_dir';
	}
</script>

<div class="flex flex-col gap-4">
	<p class="max-w-2xl text-sm text-on-surface-variant">
		Öffnet die Leserdatei mit fertig markierter Klasse — dort startet der Druck über die
		Aktionsleiste (Ausweiskarten oder Etikettenbogen, je nach Betriebsart im Reiter
		„Schülerausweise“).
	</p>
	{#if ladefehler}
		<LadeFehler titel="Klassen nicht geladen" text={ladefehler} onerneut={ladeKlassen} />
	{:else}
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
	{/if}
</div>
