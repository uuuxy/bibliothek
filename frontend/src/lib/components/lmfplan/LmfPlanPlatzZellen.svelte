<!-- @component LmfPlanPlatzZellen — die drei gerechneten Zellen einer Planer-Zeile:
     Wochentag, Datum, Stunde. Sie sind selbst der kurze Weg zum festen Platz, ohne das
     Zeilenmenü: Ein Klick auf eine Zelle legt die Zeile fest, vorbelegt mit dem Platz, den
     sie gerade hat, damit der Klick nichts verschiebt, und setzt den Fokus in das
     Feld, das man angeklickt hat (Wochentag und Datum → Datumsfeld, Stunde → Auswahl).
     Der Rest der Reihenfolge fließt um den festen Platz herum; die Nummer der Zeile
     bleibt (die Reihenfolge ist die Wahrheit, der feste Platz die Ausnahme, wie im Excel).
     Die Stecknadel vor dem Wochentag zeigt „festgelegt" und
     löst den Platz mit einem Klick; „Festen Platz lösen" bleibt auch im Zeilenmenü.
     Ohne ersten Tag (kein Platz) gibt es nichts anzuklicken — die Zellen bleiben leer. -->
<script>
	import { tick } from 'svelte';
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';
	import Datumsfeld from '../ui/Datumsfeld.svelte';
	import Select from '../ui/Select.svelte';
	import { STUNDEN, datumKurz, stundeText, wochentag } from './lmfplanDienst.js';

	/** @type {{ zeile: import('./lmfplanDienst.js').PlanZeile, i: number, platz: { datum: string, stunde: number } | undefined, onfest: () => void }} */
	let { zeile = $bindable(), i, platz, onfest } = $props();

	/** Klick auf eine gerechnete Zelle: festlegen, dann das passende Feld fokussieren.
	 *  @param {'datum' | 'stunde'} feld */
	async function festlegen(feld) {
		onfest();
		await tick();
		document.getElementById(`lmf-zeile-fest-${feld}-${i}`)?.focus();
	}

	const ZELLE = '-mx-2 h-8 cursor-pointer rounded-md px-2 text-left whitespace-nowrap';
</script>

{#if zeile.fest}
	<td class="whitespace-nowrap">
		<span class="inline-flex items-center gap-1">
			<Button
				variant="symbol"
				size="sm"
				onclick={onfest}
				data-tip="Festen Platz lösen"
				aria-label="Festen Platz Zeile {i + 1} lösen"
				class="-ml-2.5"
			>
				<Icon name="keep" class="text-primary" />
			</Button>
			{zeile.fest.datum ? wochentag(zeile.fest.datum) : ''}
		</span>
	</td>
	<td>
		<Datumsfeld
			id="lmf-zeile-fest-datum-{i}"
			aria-label="Fester Tag Zeile {i + 1}"
			bind:value={zeile.fest.datum}
			ungueltig={!zeile.fest.datum}
			feld="w-40"
		/>
	</td>
	<td>
		<Select
			id="lmf-zeile-fest-stunde-{i}"
			aria-label="Feste Stunde Zeile {i + 1}"
			bind:value={zeile.fest.stunde}
			options={STUNDEN.map((st) => ({ value: st, label: `${st}. Std.` }))}
			class="w-28"
		/>
	</td>
{:else if platz}
	<td>
		<button
			type="button"
			class={ZELLE}
			data-tip="Datum und Stunde festlegen"
			aria-label="Wochentag Zeile {i + 1}: {wochentag(platz.datum)}, festlegen"
			onclick={() => festlegen('datum')}>{wochentag(platz.datum)}</button
		>
	</td>
	<td class="tabular-nums">
		<button
			type="button"
			class={ZELLE}
			data-tip="Datum und Stunde festlegen"
			aria-label="Datum Zeile {i + 1}: {datumKurz(platz.datum)}, festlegen"
			onclick={() => festlegen('datum')}>{datumKurz(platz.datum)}</button
		>
	</td>
	<td>
		<button
			type="button"
			class={ZELLE}
			data-tip="Datum und Stunde festlegen"
			aria-label="Stunde Zeile {i + 1}: {stundeText(platz.stunde)}, festlegen"
			onclick={() => festlegen('stunde')}>{stundeText(platz.stunde)}</button
		>
	</td>
{:else}
	<td></td>
	<td></td>
	<td></td>
{/if}
