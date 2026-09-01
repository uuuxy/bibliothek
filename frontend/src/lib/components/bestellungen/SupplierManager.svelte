<script>
	import Button from '../ui/Button.svelte';
	import Switch from '../ui/Switch.svelte';
	import Feld from '../ui/Feld.svelte';

	let { suppliers, onAddSupplier, onEditSupplier, onRemoveSupplier } = $props();

	let newName = $state('');
	let newEmail = $state('');
	let newCustNum = $state('');
	let newIstHaupt = $state(false);

	/** @type {string|null} */
	let editingId = $state(null);
	let editName = $state('');
	let editEmail = $state('');
	let editCustNum = $state('');
	let editIstHaupt = $state(false);

	/** @param {SubmitEvent} e */
	function handleSubmit(e) {
		e.preventDefault();
		onAddSupplier(newName, newEmail, newCustNum, newIstHaupt);
		newName = '';
		newEmail = '';
		newCustNum = '';
		newIstHaupt = false;
	}

	/** @param {{ id: string, name: string, email: string, customerNumber: string, ist_hauptlieferant?: boolean }} s */
	function startEdit(s) {
		editingId = s.id;
		editName = s.name;
		editEmail = s.email;
		editCustNum = s.customerNumber;
		// Ohne diese Zeile stünde beim Bearbeiten immer „aus" im Feld, und wer nur die
		// E-Mail korrigiert, degradierte den Hauptlieferanten still zum normalen Händler.
		editIstHaupt = s.ist_hauptlieferant ?? false;
	}

	function cancelEdit() {
		editingId = null;
	}

	async function saveEdit() {
		if (!editingId) return;
		await onEditSupplier(editingId, editName, editEmail, editCustNum, editIstHaupt);
		editingId = null;
	}
</script>

<!-- Formular und Tabelle UNTEREINANDER: Seit dem Umzug in die Einstellungen (25.08.2026)
     steht die Maske in der Detail-Spalte neben der Kategorienliste, max-w-4xl. Drei
     Spalten nebeneinander quetschten dort die Tabelle: Spaltenköpfe stießen zusammen,
     „Hauptlieferant" wurde abgeschnitten, ein Scrollbalken erschien. -->
<div class="flex flex-col gap-10">
	<div class="space-y-4 max-w-md">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Neuer Lieferant
		</h2>
		<form onsubmit={handleSubmit} class="space-y-4 text-sm">
			<Feld id="n" label="Name" bind:value={newName} required />
			<Feld id="e" label="E-Mail" type="email" bind:value={newEmail} required />
			<Feld id="c" label="Kundennummer" bind:value={newCustNum} required />
			<!-- EIN Schalter statt drei. Vorher standen hier „beklebt die Bücher",
			     „voreingestellt beim Bestellen" und „bekommt den Bestelllink" einzeln — drei
			     Haken für eine einzige Tatsache aus dem Schulalltag, und eine Kombination davon
			     war eine stille Falle: Bestelllink ohne „beklebt" hiess, der Händler klebt und
			     die Bibliothek druckt trotzdem noch einmal. Siehe Migration 066. -->
			<div class="flex items-start justify-between gap-4 border-t border-slate-100 pt-4">
				<label for="ist-hauptlieferant" class="cursor-pointer text-sm">
					<span class="block font-semibold text-slate-700">Hauptlieferant der Schule</span>
					<span class="mt-0.5 block text-xs text-slate-500">
						Beim Bestellen vorausgewählt. Bekommt statt der reinen Bestellmail einen Link: wählt
						darüber große oder kleine Etiketten, beklebt die Bücher selbst und bestätigt damit die
						Bestellung — die Bestätigung erscheint automatisch in der Bestellhistorie. Seine Bücher
						stehen deshalb nicht auf der Nachdruck-Liste. Es kann immer nur einer sein; der
						bisherige wird zum normalen Händler.
					</span>
				</label>
				<Switch
					id="ist-hauptlieferant"
					bind:checked={newIstHaupt}
					label="Hauptlieferant der Schule"
				/>
			</div>
			<p class="text-xs text-slate-500">
				Alle anderen Lieferanten bekommen einfach nur die Bestellmail.
			</p>
			<Button type="submit" size="lg" class="w-full">Lieferanten speichern</Button>
		</form>
	</div>

	<!-- min-w-0 + eigener Scrollbereich: Ohne beides schiebt ein langer Lieferantenname die
	     Tabelle über ihre Rasterzelle hinaus (Raster-Kinder haben min-width:auto und
	     schrumpfen nicht unter ihren Inhalt). Gemessen bei 1700 px Fensterbreite: Tabelle
	     1072 px in einer 909-px-Zelle, "Bearbeiten" landete bei 1760 px — ausserhalb des
	     Fensters und damit unerreichbar. Die Spalte war da, nur nicht anklickbar. -->
	<div class="space-y-4 min-w-0">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Aktive Lieferanten
		</h2>
		{#if !suppliers.length}
			<div class="py-12 text-center text-slate-400 text-base">Keine Lieferanten angelegt.</div>
		{:else}
			<!-- Drei Spalten mit je zwei Zeilen (M3-Listenzeile: Headline + Supporting) statt
			     fünf einzeiligen: Seit dem Umzug in die Einstellungen (25.08.2026) hat die Tabelle
			     bei 1280 px nur 592 px — fünf Spalten brauchten gemessen 924. Zuerst klebte die
			     Aktionsspalte und legte sich über „Rolle", dann half auch Kürzen nicht. Zwei
			     Zeilen je Zelle passen ohne Scrollbalken und lesen sich wie die Kategorienliste
			     daneben. Name/E-Mail werden gekürzt (Block in der Zelle — max-width auf <td>
			     ignoriert das Auto-Layout), der volle Text steht im title. -->
			<div class="overflow-x-auto">
				<table class="w-full text-left border-collapse text-sm">
					<thead>
						<tr class="border-b border-slate-200 text-xs font-medium text-slate-500">
							<th class="py-2.5 pr-4">Lieferant</th>
							<th class="py-2.5 pr-4">Kontakt</th>
							<th class="py-2.5 text-right">Aktionen</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each suppliers as s (s.id)}
							{#if editingId === s.id}
								<tr class="bg-blue-50/60 align-top">
									<td class="py-2 pr-4 space-y-2">
										<Feld aria-label="Name" bind:value={editName} />
										<Switch
											bind:checked={editIstHaupt}
											label="Hauptlieferant der Schule ({s.name})"
										/>
									</td>
									<td class="py-2 pr-4 space-y-2">
										<Feld aria-label="E-Mail" type="email" bind:value={editEmail} />
										<Feld aria-label="Kundennummer" bind:value={editCustNum} />
									</td>
									<td class="py-2 text-right whitespace-nowrap">
										<button
											onclick={saveEdit}
											aria-label="Änderungen für Lieferant {s.name} speichern"
											class="text-blue-600 hover:text-blue-800 font-bold cursor-pointer text-sm mr-3"
											>Speichern</button
										>
										<button
											onclick={cancelEdit}
											aria-label="Änderungen für Lieferant {s.name} abbrechen"
											class="text-slate-400 hover:text-slate-600 cursor-pointer text-sm"
											>Abbrechen</button
										>
									</td>
								</tr>
							{:else}
								<tr class="hover:bg-slate-50/40">
									<td class="py-3 pr-4">
										<span class="block max-w-52 truncate font-bold text-slate-800" title={s.name}
											>{s.name}</span
										>
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
									<td class="py-3 pr-4 text-slate-600">
										<span class="block max-w-60 truncate" title={s.email}>{s.email}</span>
										<span class="block text-xs text-slate-400 whitespace-nowrap"
											>Kd.-Nr. {s.customerNumber || '–'}</span
										>
									</td>
									<td class="py-3 text-right whitespace-nowrap">
										<button
											onclick={() => startEdit(s)}
											aria-label="Lieferant {s.name} bearbeiten"
											class="text-slate-500 hover:text-blue-600 cursor-pointer text-sm mr-3"
											>Bearbeiten</button
										>
										<button
											onclick={() => onRemoveSupplier(s.id)}
											aria-label="Lieferant {s.name} löschen"
											class="text-rose-600/80 hover:text-rose-700 cursor-pointer text-sm"
											>Löschen</button
										>
									</td>
								</tr>
							{/if}
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>
