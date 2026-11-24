<script module>
	// Zeile und Spaltenkopf (Warenkorb.svelte) teilen ihre Maße. Schmal stehen Topf, Menge und Preis
	// in einer zweiten Zeile unter dem Titel (basis-full, eingerückt um Cover und Abstand); ab
	// 768 px Breite der Liste in einer Zeile als Spalten. Gemessen wird die Liste (@container in
	// BestellEntwurf), nicht das Fenster: Navigation und Zusammenfassung nehmen ihr verschieden viel
	// Platz. `order` stellt das Kreuz schmal neben den Titel; die Tab-Reihenfolge folgt der breiten
	// Zeile.
	export const STELLEN =
		'order-2 flex basis-full flex-wrap items-center gap-3 pl-10 @3xl:order-1 @3xl:basis-auto @3xl:flex-nowrap @3xl:pl-0';
	export const SPALTEN = { topf: 'w-48', menge: 'w-28', preis: 'w-36' };
</script>

<!-- @component Eine Position im Warenkorb: Cover, Titel, Topf, Menge, Preis, Entfernen.

     Eigene Datei, weil der Warenkorb nach Topf gruppiert: Warenkorb trägt die Gruppen, diese
     Datei die Zeile. M3 Lists: „List items can contain multiple actions at once, like selection,
     icon buttons, overflow menus, and more." -->
<script>
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import Feld from '../ui/Feld.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import { MITTEL, MITTEL_REIHENFOLGE } from './mittel.js';
	import { bedarfZu } from './bedarfFilter.js';
	import { anzahlText } from '../../utils/format.js';
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';

	/** @type {{ position: import('../../stores/bestellStore.svelte.js').CartItem }} */
	let { position } = $props();

	const toepfe = MITTEL_REIHENFOLGE.map((m) => ({
		value: m,
		label: MITTEL[m].label,
		zusatz: MITTEL[m].traeger
	}));

	// Steht das Buch im Bestellbedarf, nennt die Marke, wie viele fehlen; die Menge daneben darf
	// davon abweichen.
	const bedarf = $derived(bedarfZu(bestellStore.recommendations, position));
</script>

<li class="flex flex-wrap items-center gap-x-3 gap-y-2 py-3 @3xl:flex-nowrap">
	<span class="w-7 shrink-0 self-start @3xl:self-center">
		<BuchCover
			coverUrl={position.cover_url}
			isbn={position.isbn}
			titel={position.titel}
			dekorativ
		/>
	</span>
	<div class="min-w-0 flex-1">
		<h4 class="text-title-small leading-snug text-on-surface">{position.titel}</h4>
		<p class="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-on-surface-variant">
			<span class="font-mono">{position.isbn || '–'}</span>
			{#if position.generate_barcodes}
				<StatusChip
					icon="label"
					text={position.menge === 1 ? '1 Barcode' : `${position.menge} Barcodes`}
				/>
			{/if}
			{#if bedarf}
				<StatusChip
					text="Bestellbedarf {bedarf.fehlen}"
					tip="{anzahlText(
						bedarf.fehlen,
						'Exemplar fehlt',
						'Exemplare fehlen'
					)} für das nächste Schuljahr"
				/>
			{/if}
		</p>
	</div>

	<div class={STELLEN}>
		<!-- Der Topf ist eine Entscheidung je Bestellung; der Titel schlägt ihn nur vor. Das Feld
		     schiebt die Position in einen anderen: ein falsch gekennzeichnetes Schulbuch in die
		     Lernmittel-Bestellung, eine Spende in die Bestellung ohne Topf. -->
		<div class={SPALTEN.topf}>
			<Select
				value={position.mittel}
				options={toepfe}
				aria-label="Topf von {position.titel}"
				onchange={(wert) => bestellStore.verschiebe(position, wert)}
			/>
		</div>
		<div class={SPALTEN.menge}>
			<div
				class="inline-flex items-center rounded-sm border border-outline-variant bg-surface-container-lowest"
			>
				<!-- Unter 1 geht die Menge nicht; entfernt wird die Position über das Kreuz. Der Knopf
				     ist dann gesperrt, statt ohne Wirkung zu bleiben. -->
				<Button
					variant="symbol"
					size="sm"
					aria-label="Menge verringern"
					data-tip="Menge verringern"
					disabled={position.menge <= 1}
					onclick={() => (position.menge = Math.max(1, position.menge - 1))}
				>
					<Icon name="remove" />
				</Button>
				<span
					class="min-w-6 px-2 text-center text-body-medium-emphasized text-on-surface tabular-nums"
					>{position.menge}</span
				>
				<Button
					variant="symbol"
					size="sm"
					aria-label="Menge erhöhen"
					data-tip="Menge erhöhen"
					onclick={() => (position.menge += 1)}
				>
					<Icon name="add" />
				</Button>
			</div>
		</div>
		{#if bestellStore.preiseErfassen}
			<div class="{SPALTEN.preis} flex items-center justify-end gap-1.5">
				<!-- Der Vorschlag bleibt als solcher erkennbar, solange er unverändert ist: Er ist
				     der DNB-Ladenpreis bei Erscheinen, nicht der Preis, den die Schule zahlt. Wer
				     ihn überschreibt, verliert das Abzeichen. -->
				{#if position.preis_vorschlag > 0 && Number(position.preis) === position.preis_vorschlag}
					<StatusChip
						ton="warten"
						text="DNB"
						tip="Ladenpreis aus dem DNB-Datensatz, bitte gegen den Schulpreis prüfen"
					/>
				{/if}
				<Feld
					type="number"
					step="0.01"
					bind:value={position.preis}
					aria-label="Preis"
					feld="w-20 text-right"
				/>
				<span class="text-body-medium-emphasized text-on-surface-variant">€</span>
			</div>
		{/if}
	</div>

	<Button
		variant="symbol"
		size="sm"
		loeschen
		onclick={() => bestellStore.removeFromCart(position)}
		aria-label="Entfernen"
		data-tip="Entfernen"
		class="order-1 shrink-0 @3xl:order-2"
	>
		<Icon name="close" />
	</Button>
</li>
