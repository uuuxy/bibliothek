<!-- @component Eine Zeile der Lieferantentabelle — lesend oder im Bearbeiten-Zustand.

     Eigene Datei, seit die Maske eine zweite Kundennummer trägt (Schulträger, Migration
     109): SupplierManager mit beiden Zeilenformen lag über der 200-Zeilen-Marke.

     Drei Spalten mit je zwei Zeilen (M3-Listenzeile: Headline + Supporting) statt fünf
     einzeiligen: Seit dem Umzug in die Einstellungen (25.08.2026) hat die Tabelle bei
     1280 px nur 592 px — fünf Spalten brauchten gemessen 924. Name/E-Mail werden gekürzt
     (Block in der Zelle — max-width auf <td> ignoriert das Auto-Layout), der volle Text
     steht im title. -->
<script>
	import Switch from '../ui/Switch.svelte';
	import Feld from '../ui/Feld.svelte';
	import { untrack } from 'svelte';

	/**
	 * @type {{
	 *   s: { id: string, name: string, email: string, customerNumber: string, ist_hauptlieferant?: boolean, kundennummer_schultraeger?: string },
	 *   bearbeiten: boolean,
	 *   onEdit: (s: any) => void,
	 *   onRemove: (id: string) => void,
	 *   onSave: (werte: { name: string, email: string, customerNumber: string, istHauptlieferant: boolean, kundennummerSchultraeger: string }) => Promise<void>,
	 *   onCancel: () => void
	 * }}
	 */
	let { s, bearbeiten, onEdit, onRemove, onSave, onCancel } = $props();

	// Beim Bearbeiten IMMER den aktuellen Stand vorbelegen. Ohne den Hauptlieferanten-
	// Haken stünde beim Bearbeiten „aus" im Feld, und wer nur die E-Mail korrigiert,
	// degradierte den Hauptlieferanten still zum normalen Händler; dasselbe gilt für die
	// zweite Kundennummer. Bewusst der Anfangswert (untrack): SupplierManager hängt die
	// Zeile beim Wechsel in den Bearbeiten-Zustand neu ein ({#key}); ein Formular, das
	// dem Prop weiter folgt, verlöre die Eingabe beim nächsten Nachladen der Liste.
	let editName = $state(untrack(() => s.name));
	let editEmail = $state(untrack(() => s.email));
	let editCustNum = $state(untrack(() => s.customerNumber));
	let editCustNumSchultraeger = $state(untrack(() => s.kundennummer_schultraeger ?? ''));
	let editIstHaupt = $state(untrack(() => s.ist_hauptlieferant ?? false));
</script>

{#if bearbeiten}
	<tr aria-selected="true" class="align-top">
		<td class="space-y-2">
			<Feld aria-label="Name" bind:value={editName} />
			<Switch bind:checked={editIstHaupt} label="Hauptlieferant der Schule ({s.name})" />
		</td>
		<td class="space-y-2">
			<Feld aria-label="E-Mail" type="email" bind:value={editEmail} />
			<Feld aria-label="Kundennummer" bind:value={editCustNum} />
			<Feld
				aria-label="Kundennummer Schülerbücherei"
				placeholder="Kundennummer Schülerbücherei (falls abweichend)"
				bind:value={editCustNumSchultraeger}
			/>
		</td>
		<td class="text-right whitespace-nowrap">
			<button
				onclick={() =>
					onSave({
						name: editName,
						email: editEmail,
						customerNumber: editCustNum,
						istHauptlieferant: editIstHaupt,
						kundennummerSchultraeger: editCustNumSchultraeger
					})}
				aria-label="Änderungen für Lieferant {s.name} speichern"
				class="text-blue-600 hover:text-blue-800 font-bold cursor-pointer text-sm mr-3"
				>Speichern</button
			>
			<button
				onclick={onCancel}
				aria-label="Änderungen für Lieferant {s.name} abbrechen"
				class="text-slate-400 hover:text-slate-600 cursor-pointer text-sm">Abbrechen</button
			>
		</td>
	</tr>
{:else}
	<tr>
		<td>
			<span class="block max-w-52 truncate font-bold text-slate-800" title={s.name}>{s.name}</span>
			<!-- Nur die Abweichung wird benannt: „Bestellmail" in jeder Zeile wäre
			     Rauschen. Auffallen soll die eine Zeile, die anders ist. -->
			{#if s.ist_hauptlieferant}
				<span
					class="block text-xs font-semibold text-slate-700"
					data-tip="Vorausgewählt beim Bestellen, bekommt den Bestelllink (Etikettengröße + Bestätigung) und beklebt die Bücher selbst"
					>Hauptlieferant</span
				>
			{:else}
				<span class="block text-xs text-slate-400">nur Bestellmail</span>
			{/if}
		</td>
		<td>
			<span class="block max-w-60 truncate" title={s.email}>{s.email}</span>
			<span class="block text-xs text-slate-400 whitespace-nowrap">
				Kd.-Nr. {s.customerNumber || '–'}
				<!-- Die zweite Nummer nur, wenn es sie gibt — sonst gilt dieselbe. -->
				{#if s.kundennummer_schultraeger}
					· Schülerbücherei {s.kundennummer_schultraeger}
				{/if}
			</span>
		</td>
		<td class="text-right whitespace-nowrap">
			<button
				onclick={() => onEdit(s)}
				aria-label="Lieferant {s.name} bearbeiten"
				class="text-slate-500 hover:text-blue-600 cursor-pointer text-sm mr-3">Bearbeiten</button
			>
			<button
				onclick={() => onRemove(s.id)}
				aria-label="Lieferant {s.name} löschen"
				class="text-rose-600/80 hover:text-rose-700 cursor-pointer text-sm">Löschen</button
			>
		</td>
	</tr>
{/if}
