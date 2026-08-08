<script>
	import Button from '../ui/Button.svelte';
	import Switch from '../ui/Switch.svelte';

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

<div class="grid grid-cols-1 md:grid-cols-3 gap-8 items-start overflow-y-auto">
	<div class="space-y-4">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Neuer Lieferant
		</h2>
		<form onsubmit={handleSubmit} class="space-y-4 text-base">
			<div class="space-y-1.5">
				<label for="n" class="block font-medium text-slate-600 text-sm">Name</label><input
					id="n"
					type="text"
					bind:value={newName}
					required
					class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
				/>
			</div>
			<div class="space-y-1.5">
				<label for="e" class="block font-medium text-slate-600 text-sm">E-Mail</label><input
					id="e"
					type="email"
					bind:value={newEmail}
					required
					class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
				/>
			</div>
			<div class="space-y-1.5">
				<label for="c" class="block font-medium text-slate-600 text-sm">Kundennummer</label><input
					id="c"
					type="text"
					bind:value={newCustNum}
					required
					class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
				/>
			</div>
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
	<div class="md:col-span-2 space-y-4 min-w-0">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Aktive Lieferanten
		</h2>
		{#if !suppliers.length}
			<div class="py-12 text-center text-slate-400 text-base">Keine Lieferanten angelegt.</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left border-collapse text-base">
					<thead>
						<tr class="border-b border-slate-200 text-sm font-semibold text-slate-500">
							<th class="py-2.5">Name</th>
							<th class="py-2.5">E-Mail</th>
							<th class="py-2.5">Kundennummer</th>
							<th class="py-2.5">Rolle</th>
							<!-- Klebt am rechten Rand des Scrollbereichs: Die Tabelle kann breiter werden
							     als der Platz, den sie bekommt (lange Lieferantennamen). Ohne sticky steht
							     "Bearbeiten" hinter dem sichtbaren Rand — der Knopf ist dann zwar im DOM,
							     aber niemand findet ihn (gemessen: rechte Kante 1620 px bei 1280 px
							     Fenster). -->
							<th class="py-2.5 text-right sticky right-0 bg-white">Aktionen</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each suppliers as s (s.id)}
							{#if editingId === s.id}
								<tr class="bg-blue-50/60">
									<td class="py-2 pr-2"
										><input
											type="text"
											bind:value={editName}
											class="w-full px-2 py-1.5 rounded border border-blue-300 text-sm"
										/></td
									>
									<td class="py-2 pr-2"
										><input
											type="email"
											bind:value={editEmail}
											class="w-full px-2 py-1.5 rounded border border-blue-300 text-sm"
										/></td
									>
									<td class="py-2 pr-2"
										><input
											type="text"
											bind:value={editCustNum}
											class="w-full px-2 py-1.5 rounded border border-blue-300 text-sm"
										/></td
									>
									<td class="py-2 pr-2">
										<Switch
											bind:checked={editIstHaupt}
											label="Hauptlieferant der Schule ({s.name})"
										/>
									</td>
									<td class="py-2 text-right whitespace-nowrap sticky right-0 bg-blue-50">
										<button
											onclick={saveEdit}
											class="text-blue-600 hover:text-blue-800 font-bold cursor-pointer text-sm mr-3"
											>Speichern</button
										>
										<button
											onclick={cancelEdit}
											class="text-slate-400 hover:text-slate-600 cursor-pointer text-sm"
											>Abbrechen</button
										>
									</td>
								</tr>
							{:else}
								<tr class="hover:bg-slate-50/40">
									<td class="py-3 font-bold text-slate-800">{s.name}</td>
									<td class="py-3 text-slate-600">{s.email}</td>
									<td class="py-3 text-slate-600">{s.customerNumber}</td>
									<!-- Nur die Abweichung wird benannt. „Bestellmail" in jeder Zeile wäre
								     Rauschen: Es ließe das Auge jede Zeile lesen, um nichts zu erfahren.
								     Auffallen soll die eine Zeile, die anders ist. -->
									<td class="py-3">
										{#if s.ist_hauptlieferant}
											<span
												class="text-sm font-semibold text-slate-700"
												data-tip="Vorausgewählt beim Bestellen, bekommt den Bestelllink (Etikettengröße + Bestätigung) und beklebt die Bücher selbst"
											>
												Hauptlieferant
											</span>
										{:else}
											<span class="text-sm text-slate-400">nur Bestellmail</span>
										{/if}
									</td>
									<td class="py-3 text-right whitespace-nowrap sticky right-0 bg-white">
										<button
											onclick={() => startEdit(s)}
											class="text-slate-500 hover:text-blue-600 cursor-pointer text-sm mr-3"
											>Bearbeiten</button
										>
										<button
											onclick={() => onRemoveSupplier(s.id)}
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
