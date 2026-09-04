<!-- @component AbgaengerKopfzeile — oben die Suche der Seite, darunter links der
     Klassenfilter und rechts die zwei Wege für dasselbe Dokument: ausdrucken (Papier
     bleibt der Notweg, wenn keine Lehrer-Adresse hinterlegt ist) oder an die
     Klassenleitungen mailen. Der Druck folgt Suche UND Filter — was auf dem Bildschirm
     steht, steht auf dem Papier.

     Die Suche kam am 04.09.2026 dazu: Die Seite listete 62 Namen ohne jede Suche, als
     einzige Verwaltungsseite mit einer langen Liste. -->
<script>
	import { Mail, Printer } from '@lucide/svelte';
	import Button from './ui/Button.svelte';
	import Select from './ui/Select.svelte';
	import Suchpille from './ui/Suchpille.svelte';

	/**
	 * @type {{
	 *   suche: string,
	 *   klasse: string,
	 *   klassen: string[],
	 *   gesamt: number,
	 *   gefiltert: number,
	 *   laedt: boolean,
	 *   druckLaeuft: boolean,
	 *   onDrucken: () => void,
	 *   onMailen: () => void
	 * }}
	 */
	let {
		suche = $bindable(''),
		klasse = $bindable(),
		klassen,
		gesamt,
		gefiltert,
		laedt,
		druckLaeuft,
		onDrucken,
		onMailen
	} = $props();
</script>

<div class="flex flex-col gap-3 border-b border-slate-100 pb-5">
	<Suchpille
		id="abgaenger-suchfeld"
		bind:wert={suche}
		platzhalter="Name oder Klasse suchen …"
		etikett="Abgänger suchen"
	/>

	<div class="flex items-center justify-between gap-4">
		{#if !laedt && gesamt > 0}
			<div class="flex items-center gap-3 min-w-0">
				<label class="text-xs font-medium text-slate-500 shrink-0" for="grad-klasse">Klasse</label>
				<Select
					id="grad-klasse"
					bind:value={klasse}
					options={[
						{ value: '', label: `Alle Klassen (${gesamt})` },
						...klassen.map((/** @type {string} */ k) => ({ value: k, label: k }))
					]}
					class="w-48"
				/>
				<span class="text-xs text-slate-400 shrink-0">{gefiltert} Abgänger</span>
			</div>
		{:else}
			<div></div>
		{/if}

		<div class="flex items-center space-x-4 shrink-0">
			<Button
				variant="secondary"
				onclick={onDrucken}
				disabled={druckLaeuft || gesamt === 0}
				class="no-print"
			>
				{#if druckLaeuft}
					<div
						class="w-3.5 h-3.5 border-2 border-slate-400 border-t-transparent rounded-full animate-spin"
					></div>
					Lade Daten…
				{:else}
					<Printer class="h-4 w-4" aria-hidden="true" />
					{klasse ? `Kontoauszüge ${klasse}` : 'Kontoauszüge drucken'}
				{/if}
			</Button>
			<Button
				onclick={onMailen}
				disabled={gesamt === 0}
				class="no-print"
				title="Je Klasse eine Mail an die Klassenleitung, darin ein Kontoauszug je Abgänger"
			>
				<Mail class="h-4 w-4" />
				An Klassenleitungen mailen
			</Button>
			<div
				class="flex items-center gap-1.5 text-label-small font-semibold text-emerald-600 shrink-0"
				title="Änderungen an allen Arbeitsplätzen sofort sichtbar (Live-Synchronisation)"
			>
				<span class="h-2 w-2 rounded-full bg-emerald-500 animate-pulse shrink-0"></span>
				Live
			</div>
		</div>
	</div>
</div>
