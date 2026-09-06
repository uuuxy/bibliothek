<!-- @component LmfPlanPlatzZellen — die drei gerechneten Zellen einer Planer-Zeile:
     Wochentag, Datum, Stunde. Seit dem 06.09.2026 (Peter: „einfach anklicken, um es zu
     ändern … statt immer über die drei Punkte rechts") sind sie selbst der Weg zum
     festen Platz: Ein Klick auf eine Zelle legt die Zeile fest — vorbelegt mit dem Platz,
     den sie gerade hat, damit der Klick nichts verschiebt — und setzt den Fokus in das
     Feld, das man angeklickt hat (Wochentag und Datum → Datumsfeld, Stunde → Auswahl).
     Der Rest der Reihenfolge fließt um den festen Platz herum; die Nummer der Zeile
     bleibt (Entscheidung 06.09.: die Reihenfolge ist die Wahrheit, der feste Platz die
     Ausnahme wie im Excel). Die Stecknadel vor dem Wochentag zeigt „festgelegt" und
     löst den Platz mit einem Klick; „Festen Platz lösen" bleibt auch im Zeilenmenü.
     Ohne ersten Tag (kein Platz) gibt es nichts anzuklicken — die Zellen bleiben leer. -->
<script>
	import { tick } from 'svelte';
	import { Pin } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import { STUNDEN, datumKurz, stundeText, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ zeile: import('../../lmfplanDienst.js').PlanZeile, i: number, platz: { datum: string, stunde: number } | undefined, onfest: () => void }} */
	let { zeile = $bindable(), i, platz, onfest } = $props();

	/** Klick auf eine gerechnete Zelle: festlegen, dann das passende Feld fokussieren.
	 *  @param {'datum' | 'stunde'} feld */
	async function festlegen(feld) {
		onfest();
		await tick();
		document.getElementById(`lmf-zeile-fest-${feld}-${i}`)?.focus();
	}

	const ZELLE =
		'-mx-2 h-8 cursor-pointer rounded-md px-2 text-left whitespace-nowrap hover:bg-surface-container';
</script>

{#if zeile.fest}
	<td class="px-4 py-1 whitespace-nowrap text-on-surface-variant">
		<span class="inline-flex items-center gap-1">
			<Button
				variant="ghost"
				size="sm"
				onclick={onfest}
				title="Festen Platz lösen"
				aria-label="Festen Platz Zeile {i + 1} lösen"
				class="-ml-2.5"
			>
				<Pin class="h-4 w-4 text-primary" aria-hidden="true" />
			</Button>
			{zeile.fest.datum ? wochentag(zeile.fest.datum) : ''}
		</span>
	</td>
	<td class="px-4 py-1">
		<Feld
			id="lmf-zeile-fest-datum-{i}"
			aria-label="Fester Tag Zeile {i + 1}"
			type="date"
			bind:value={zeile.fest.datum}
			ungueltig={!zeile.fest.datum}
			feld="w-40"
		/>
	</td>
	<td class="px-4 py-1">
		<Select
			id="lmf-zeile-fest-stunde-{i}"
			aria-label="Feste Stunde Zeile {i + 1}"
			bind:value={zeile.fest.stunde}
			options={STUNDEN.map((st) => ({ value: st, label: `${st}. Std.` }))}
			class="w-28"
		/>
	</td>
{:else if platz}
	<td class="px-4 py-1 text-on-surface-variant">
		<button
			type="button"
			class={ZELLE}
			title="Datum und Stunde festlegen"
			aria-label="Wochentag Zeile {i + 1}: {wochentag(platz.datum)} — festlegen"
			onclick={() => festlegen('datum')}>{wochentag(platz.datum)}</button
		>
	</td>
	<td class="px-4 py-1 tabular-nums text-on-surface">
		<button
			type="button"
			class={ZELLE}
			title="Datum und Stunde festlegen"
			aria-label="Datum Zeile {i + 1}: {datumKurz(platz.datum)} — festlegen"
			onclick={() => festlegen('datum')}>{datumKurz(platz.datum)}</button
		>
	</td>
	<td class="px-4 py-1 text-on-surface-variant">
		<button
			type="button"
			class={ZELLE}
			title="Datum und Stunde festlegen"
			aria-label="Stunde Zeile {i + 1}: {stundeText(platz.stunde)} — festlegen"
			onclick={() => festlegen('stunde')}>{stundeText(platz.stunde)}</button
		>
	</td>
{:else}
	<td class="px-4 py-1"></td>
	<td class="px-4 py-1"></td>
	<td class="px-4 py-1"></td>
{/if}
