<script>
	/**
	 * @component AusweiseJeKlasse
	 * Der Reiter „Ausweise je Klasse" im Druck-Center: Klasse wählen, und die Ausweise all ihrer
	 * Schüler gehen in den Druck. Gedruckt wird über denselben Baustein wie in der Leserdatei
	 * (ausweisdruck.svelte.js) und mit derselben Steuerung (AusweisDruckSteuerung): Wer keine
	 * Ausweisnummer hat, fällt heraus und wird genannt; ob Karten oder Etiketten kommen, legt das
	 * Ausweis-Design fest. Einzelne Schüler druckt die Leserdatei.
	 */
	import { onMount } from 'svelte';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Select from '../ui/Select.svelte';
	import AusweisDruckSteuerung from './AusweisDruckSteuerung.svelte';
	import AusweisStapeldruck from './AusweisStapeldruck.svelte';
	import { erzeugeAusweisdruck, mitAusweisnummer, ohneAblaufjahr } from './ausweisdruck.svelte.js';

	/** @type {string[]} */
	let klassen = $state([]);
	let klasse = $state('');
	// Der Grund, wenn die Klassenliste nicht kam. Die Klasse wird hier gewählt, nicht getippt:
	// Eine leere Auswahl ohne Wort sähe aus wie eine Schule ohne Klassen.
	let ladefehler = $state('');
	/** @type {any[]} */
	let schueler = $state([]);
	let schuelerFehler = $state('');
	let laedt = $state(false);
	const druck = erzeugeAusweisdruck();

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

	// Nur die jüngste Anfrage schreibt: Wer schnell die Klasse wechselt, druckte sonst die
	// Schüler der vorigen.
	let ladeNr = 0;
	/** @param {string} gewaehlt */
	async function ladeSchueler(gewaehlt) {
		const nr = ++ladeNr;
		schueler = [];
		schuelerFehler = '';
		if (!gewaehlt) return;
		laedt = true;
		try {
			const res = await apiFetch(`/api/schueler?klasse=${encodeURIComponent(gewaehlt)}`);
			if (!res.ok) {
				const fehler = await leseApiFehler(res);
				if (nr === ladeNr) schuelerFehler = fehler;
				return;
			}
			const daten = await res.json();
			if (nr === ladeNr) schueler = Array.isArray(daten) ? daten : [];
		} catch (e) {
			if (nr === ladeNr) schuelerFehler = fehlertext(e);
		} finally {
			if (nr === ladeNr) laedt = false;
		}
	}

	$effect(() => {
		ladeSchueler(klasse);
	});

	const druckbar = $derived(mitAusweisnummer(schueler));
	const zeile = $derived(
		schueler.length === 0
			? 'In dieser Klasse steht kein Schüler.'
			: `${schueler.length} Schüler` +
					(druckbar.length < schueler.length
						? `, ${schueler.length - druckbar.length} ohne Ausweisnummer`
						: '')
	);
</script>

<div class="no-print flex flex-col gap-4">
	<p class="max-w-2xl text-sm text-on-surface-variant">
		Druckt die Ausweise aller Schüler einer Klasse, als Karte oder auf dem Etikettenbogen, wie unter
		„Ausweis-Design“ eingestellt. Einzelne Schüler druckt die Leserdatei.
	</p>
	{#if ladefehler}
		<LadeFehler titel="Klassen nicht geladen" text={ladefehler} onerneut={ladeKlassen} />
	{:else}
		<Select
			bind:value={klasse}
			options={klassen.map((k) => ({ value: k, label: k }))}
			placeholder="Klasse wählen"
			aria-label="Klasse"
			class="w-48"
		/>
		{#if schuelerFehler}
			<LadeFehler
				titel="Schüler nicht geladen"
				text={schuelerFehler}
				onerneut={() => ladeSchueler(klasse)}
			/>
		{:else if klasse && !laedt}
			<p class="text-sm text-on-surface">{zeile}</p>
			<div class="flex flex-wrap items-center gap-4">
				<AusweisDruckSteuerung
					ohneDatum={ohneAblaufjahr(schueler)}
					etikettModus={druck.etikettModus}
					maxPosition={druck.maxPosition}
					bind:startPosition={druck.startPosition}
					gesperrt={druckbar.length === 0}
					onDrucken={() => druck.drucke(schueler)}
				/>
			</div>
		{/if}
	{/if}
</div>

<!-- Die Druckfläche steht außerhalb von .no-print, sonst blendet die Druck-CSS sie mit aus und
     der Ausdruck bliebe leer (wie in der Leserdatei und im Ausweis-Design). Im Etikettenmodus
     gar nicht: Der Bogen kommt als PDF vom Server. -->
{#if druckbar.length > 0 && !druck.etikettModus}
	<AusweisStapeldruck leserListe={druckbar} />
{/if}
