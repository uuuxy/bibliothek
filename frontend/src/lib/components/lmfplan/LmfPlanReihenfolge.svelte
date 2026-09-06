<!-- @component LmfPlanReihenfolge — Abschnitt „Reihenfolge": der Plan als Tabelle in
     der Form, die das Kollegium kennt (Wochentag, Datum, Stunde, Klassen,
     Besonderheiten), nur dass sie hier bearbeitet wird: Zeilen ziehen oder mit den
     Pfeilen schieben, zwei Zeilen zu einer Stunde zusammenlegen („10R1/10R2"), eine
     Zeile ohne Klasse davor einfügen („Bücher setzen"), festlegen, Klasse aus dem Plan
     nehmen. Die Zeile selbst ist LmfPlanZeile, ihre Aktionen LmfPlanZeileAktionen.

     Nach M3 Lists ohne Trennlinie je Zeile („Limit dividers to uncontained or complex
     lists, only when a stronger visual separation is necessary"): Zeilenhöhe 48 px und
     die Hover-Fläche trennen, die Linie steht nur unter dem Kopf. Wochentag, Datum und
     Stunde kommen vom Server (Vorschau) — die zwei Spalten, die im Excel von Hand
     falsch waren. -->
<script>
	import Button from '../ui/Button.svelte';
	import LmfPlanZeile from './LmfPlanZeile.svelte';

	/** @type {{ zeilen: import('../../lmfplanDienst.js').PlanZeile[], plaetze: { datum: string, stunde: number }[], marker: ReturnType<typeof import('../../lmfplanDienst.js').klassenMarker>, bereit: boolean, onklasseraus: (klasse: string) => void }} */
	let { zeilen = $bindable(), plaetze, marker, bereit, onklasseraus } = $props();

	/** @type {number | null} */
	let gezogen = $state(null);

	// Klassen im Plan ohne Schüler (Vorjahr, vor dem Import getippt) — der Satz oben
	// zählt sie, damit man nach dem LUSD-Import sieht, was übrig geblieben ist.
	const ohneSchueler = $derived(
		[...new Set(zeilen.flatMap((z) => z.klassen))].filter((k) => marker.ohneSchueler(k))
	);

	/** @param {number} von @param {number} nach */
	function verschiebe(von, nach) {
		if (von === nach || nach < 0 || nach >= zeilen.length) return;
		const kopie = [...zeilen];
		const [z] = kopie.splice(von, 1);
		kopie.splice(nach, 0, z);
		zeilen = kopie;
	}

	/** Zeile i mit der davor zusammenlegen: beide Klassen in einer Stunde. */
	function zusammenlegen(i) {
		if (i === 0) return;
		const oben = zeilen[i - 1];
		const unten = zeilen[i];
		const vermerk = [oben.vermerk, unten.vermerk].filter(Boolean).join(' · ');
		zeilen = [
			...zeilen.slice(0, i - 1),
			{ klassen: [...oben.klassen, ...unten.klassen], vermerk, fest: oben.fest ?? null },
			...zeilen.slice(i + 1)
		];
	}

	/** Eine Zeile mit mehreren Klassen wieder in einzelne Stunden trennen. */
	function trennen(i) {
		const z = zeilen[i];
		if (z.klassen.length < 2) return;
		const einzeln = z.klassen.map((k, n) => ({
			klassen: [k],
			vermerk: n === 0 ? z.vermerk : '',
			fest: n === 0 ? (z.fest ?? null) : null
		}));
		zeilen = [...zeilen.slice(0, i), ...einzeln, ...zeilen.slice(i + 1)];
	}

	function einfuegen(i) {
		zeilen = [
			...zeilen.slice(0, i),
			{ klassen: [], vermerk: 'Bücher setzen', fest: null },
			...zeilen.slice(i)
		];
	}

	function entfernen(i) {
		for (const k of zeilen[i].klassen) onklasseraus(k);
		zeilen = zeilen.filter((_, n) => n !== i);
	}

	function klasseRaus(i, k) {
		onklasseraus(k);
		const rest = zeilen[i].klassen.filter((x) => x !== k);
		if (rest.length === 0 && !zeilen[i].vermerk.trim()) {
			zeilen = zeilen.filter((_, n) => n !== i);
		} else {
			zeilen = zeilen.map((z, n) => (n === i ? { ...z, klassen: rest } : z));
		}
	}

	/** Festlegen: Die Zeile nimmt ihren Vorschau-Platz als Vorgabe mit, damit „festlegen"
	 *  zunächst nichts verschiebt. Lösen: sie fließt wieder mit. */
	function festWechseln(i) {
		zeilen = zeilen.map((z, n) => {
			if (n !== i) return z;
			if (z.fest) return { ...z, fest: null };
			return { ...z, fest: { datum: plaetze[i]?.datum ?? '', stunde: plaetze[i]?.stunde ?? 1 } };
		});
	}
</script>

<section aria-labelledby="lmf-reihenfolge-titel">
	<h2 id="lmf-reihenfolge-titel" class="text-title-medium font-medium text-on-surface">
		Reihenfolge
	</h2>
	<p class="mt-1 max-w-3xl text-sm text-on-surface-variant" data-testid="lmf-reihenfolge-hinweis">
		{#if bereit}
			Zeilen ziehen oder mit den Pfeilen schieben; Wochentag, Datum und Stunde rechnet der Plan.
		{:else}
			Ersten Tag wählen — dann rechnet der Plan Wochentag, Datum und Stunde jeder Zeile.
		{/if}
		{#if ohneSchueler.length > 0}
			<span data-testid="lmf-ohne-schueler">
				{ohneSchueler.length === 1 ? 'Eine Klasse' : `${ohneSchueler.length} Klassen`} im Plan
				{ohneSchueler.length === 1 ? 'hat' : 'haben'} noch keine Schüler ({ohneSchueler.join(', ')})
				— sie kommen mit dem LUSD-Import oder gehören aus dem Plan.
			</span>
		{/if}
	</p>
	<div class="mt-4 overflow-x-auto">
		<table class="w-full border-collapse text-left text-sm" data-testid="lmf-reihenfolge">
			<thead>
				<tr class="border-b border-outline-variant text-on-surface-variant">
					<th class="w-10 px-2 py-2 text-right font-medium">#</th>
					<th class="px-4 py-2 font-medium">Wochentag</th>
					<th class="px-4 py-2 font-medium">Datum</th>
					<th class="px-4 py-2 font-medium">Stunde</th>
					<th class="px-4 py-2 font-medium">Klassen</th>
					<th class="px-4 py-2 font-medium">Besonderheiten</th>
					<th class="px-4 py-2 text-right font-medium">Aktionen</th>
				</tr>
			</thead>
			<tbody>
				{#each zeilen, i (i)}
					<LmfPlanZeile
						bind:zeile={zeilen[i]}
						{i}
						anzahl={zeilen.length}
						platz={plaetze[i]}
						{marker}
						gezogen={gezogen === i}
						onziehstart={() => (gezogen = i)}
						onablegen={() => {
							if (gezogen !== null) verschiebe(gezogen, i);
							gezogen = null;
						}}
						onklasseraus={(k) => klasseRaus(i, k)}
						onhoch={() => verschiebe(i, i - 1)}
						onrunter={() => verschiebe(i, i + 1)}
						onzusammen={() => zusammenlegen(i)}
						ontrennen={() => trennen(i)}
						oneinfuegen={() => einfuegen(i)}
						onfest={() => festWechseln(i)}
						onentfernen={() => entfernen(i)}
					/>
				{/each}
			</tbody>
		</table>
		{#if zeilen.length === 0}
			<p class="px-4 py-6 text-sm text-on-surface-variant">
				Noch keine Zeile — Klassen aus „Nicht im Plan" holen.
			</p>
		{/if}
		<div class="px-2 pt-2">
			<Button variant="ghost" size="sm" onclick={() => einfuegen(zeilen.length)}>
				Zeile ohne Klasse anhängen
			</Button>
		</div>
	</div>
</section>
