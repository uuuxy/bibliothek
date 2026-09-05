<!-- @component LmfPlanReihenfolge — die Reihenfolge des Plans als Tabelle in der Form,
     die das Kollegium kennt (Wochentag, Datum, Stunde, Klassen, Besonderheiten), nur
     dass sie hier bearbeitet wird: Zeilen ziehen oder mit den Pfeilen schieben, zwei
     Zeilen zu einer Stunde zusammenlegen („10R1/10R2"), eine Zeile ohne Klasse davor
     einfügen („Bücher setzen"), Klasse aus dem Plan nehmen. Wochentag, Datum und Stunde
     kommen vom Server (Vorschau) und sind nicht editierbar — genau die zwei Spalten, die
     im Excel von Hand falsch waren. Die EINE Ausnahme: eine FESTGELEGTE Zeile (die
     Klasse mit dem Ausflug) bekommt Datum und Stunde von Hand, vorbelegt mit ihrem
     Platz aus der Vorschau; die übrigen Zeilen fließen um sie herum (Server). -->
<script>
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';
	import LmfPlanZeileAktionen from './LmfPlanZeileAktionen.svelte';
	import { STUNDEN, datumKurz, stundeText, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ zeilen: import('../../lmfplanDienst.js').PlanZeile[], plaetze: { datum: string, stunde: number }[], onklasseraus: (klasse: string) => void }} */
	let { zeilen = $bindable(), plaetze, onklasseraus } = $props();

	/** @type {number | null} */
	let gezogen = $state(null);

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
			{ klassen: [...oben.klassen, ...unten.klassen], vermerk },
			...zeilen.slice(i + 1)
		];
	}

	/** Eine Zeile mit mehreren Klassen wieder in einzelne Stunden trennen. */
	function trennen(i) {
		const z = zeilen[i];
		if (z.klassen.length < 2) return;
		const einzeln = z.klassen.map((k, n) => ({ klassen: [k], vermerk: n === 0 ? z.vermerk : '' }));
		zeilen = [...zeilen.slice(0, i), ...einzeln, ...zeilen.slice(i + 1)];
	}

	function einfuegen(i) {
		zeilen = [...zeilen.slice(0, i), { klassen: [], vermerk: 'Bücher setzen' }, ...zeilen.slice(i)];
	}

	function entfernen(i) {
		for (const k of zeilen[i].klassen) onklasseraus(k);
		zeilen = zeilen.filter((_, n) => n !== i);
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

	function klasseRaus(i, k) {
		onklasseraus(k);
		const rest = zeilen[i].klassen.filter((x) => x !== k);
		if (rest.length === 0 && !zeilen[i].vermerk.trim()) {
			zeilen = zeilen.filter((_, n) => n !== i);
		} else {
			zeilen = zeilen.map((z, n) => (n === i ? { ...z, klassen: rest } : z));
		}
	}
</script>

<div class="overflow-x-auto">
	<table class="w-full border-collapse text-left text-base" data-testid="lmf-reihenfolge">
		<thead>
			<tr class="border-b border-outline-variant text-sm text-on-surface-variant">
				<th class="w-10 px-2 py-2 text-right">#</th>
				<th class="px-4 py-2">Wochentag</th>
				<th class="px-4 py-2">Datum</th>
				<th class="px-4 py-2">Stunde</th>
				<th class="px-4 py-2">Klassen</th>
				<th class="px-4 py-2">Besonderheiten</th>
				<th class="px-4 py-2 text-right">Reihenfolge</th>
			</tr>
		</thead>
		<tbody class="divide-y divide-outline-variant">
			{#each zeilen as z, i (i)}
				{@const p = plaetze[i]}
				<tr
					draggable="true"
					ondragstart={() => (gezogen = i)}
					ondragover={(e) => e.preventDefault()}
					ondrop={() => {
						if (gezogen !== null) verschiebe(gezogen, i);
						gezogen = null;
					}}
					class="transition-colors hover:bg-surface-container-low {gezogen === i
						? 'opacity-50'
						: ''}"
				>
					<td class="px-2 py-1 text-right text-sm tabular-nums text-on-surface-variant">{i + 1}</td>
					{#if z.fest}
						<td class="px-4 py-1 text-on-surface-variant" title="Fester Platz">
							{z.fest.datum ? wochentag(z.fest.datum) : '…'}
						</td>
						<td class="px-4 py-1">
							<Feld
								id="lmf-zeile-fest-datum-{i}"
								aria-label="Fester Tag Zeile {i + 1}"
								type="date"
								bind:value={z.fest.datum}
								ungueltig={!z.fest.datum}
								feld="w-40"
							/>
						</td>
						<td class="px-4 py-1">
							<Select
								id="lmf-zeile-fest-stunde-{i}"
								aria-label="Feste Stunde Zeile {i + 1}"
								bind:value={z.fest.stunde}
								options={STUNDEN.map((st) => ({ value: st, label: `${st}. Std.` }))}
								class="w-28"
							/>
						</td>
					{:else}
						<td class="px-4 py-1 text-on-surface-variant">{p ? wochentag(p.datum) : '…'}</td>
						<td class="px-4 py-1 tabular-nums text-on-surface">{p ? datumKurz(p.datum) : '…'}</td>
						<td class="px-4 py-1 text-on-surface-variant">{p ? stundeText(p.stunde) : '…'}</td>
					{/if}
					<td class="px-4 py-1">
						<div class="flex flex-wrap gap-1">
							{#each z.klassen as k (k)}
								<LmfKlasseChip name={k} onentfernen={() => klasseRaus(i, k)} />
							{/each}
						</div>
					</td>
					<td class="px-4 py-1">
						<Feld
							id="lmf-zeile-vermerk-{i}"
							aria-label="Besonderheiten Zeile {i + 1}"
							bind:value={z.vermerk}
							placeholder={z.klassen.length === 0 ? 'Pflicht ohne Klasse' : ''}
							ungueltig={z.klassen.length === 0 && !z.vermerk.trim()}
						/>
					</td>
					<td class="px-4 py-1 text-right whitespace-nowrap">
						<LmfPlanZeileAktionen
							nummer={i + 1}
							anzahl={zeilen.length}
							mehrere={z.klassen.length > 1}
							fest={Boolean(z.fest)}
							onhoch={() => verschiebe(i, i - 1)}
							onrunter={() => verschiebe(i, i + 1)}
							onzusammen={() => zusammenlegen(i)}
							ontrennen={() => trennen(i)}
							oneinfuegen={() => einfuegen(i)}
							onfest={() => festWechseln(i)}
							onentfernen={() => entfernen(i)}
						/>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
	{#if zeilen.length === 0}
		<p class="px-4 py-6 text-sm text-on-surface-variant">
			Noch keine Zeile — Klassen aus „Nicht im Plan" holen.
		</p>
	{/if}
	<div class="px-4 pt-2">
		<Button variant="ghost" size="sm" onclick={() => einfuegen(zeilen.length)}>
			Zeile ohne Klasse anhängen
		</Button>
	</div>
</div>
