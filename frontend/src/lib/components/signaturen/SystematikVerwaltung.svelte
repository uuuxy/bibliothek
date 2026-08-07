<script>
	import { onMount } from 'svelte';
	// apiPost/apiPut/apiDelete liefern die geparste Antwort und WERFEN im Fehlerfall —
	// den Fehler-Toast haben sie dann schon gezeigt. Deshalb hier kein zweiter Toast im
	// catch: Ein generisches "Fehler beim Speichern" verdeckte sonst die Servermeldung,
	// die den Grund nennt (z. B. "Kürzel existiert bereits").
	import { apiGet, apiPost, apiPut, apiDelete } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';

	/** @type {{ onChanged?: () => void }} */
	let { onChanged } = $props();

	let liste = $state(/** @type {any[]} */ ([]));
	let laedt = $state(true);
	let kuerzel = $state('');
	let bezeichnung = $state('');
	let speichert = $state(false);
	/** Zeile, die gerade bearbeitet wird (null = keine). */
	let bearbeiteId = $state(/** @type {string | null} */ (null));
	let bearbeiteKuerzel = $state('');
	let bearbeiteBezeichnung = $state('');

	async function laden() {
		laedt = true;
		try {
			liste = (await apiGet('/api/systematics')) || [];
		} catch {
			// Meldung kam bereits aus apiGet.
		} finally {
			laedt = false;
		}
	}

	onMount(laden);

	async function anlegen() {
		if (!kuerzel.trim() || !bezeichnung.trim()) return;
		speichert = true;
		try {
			await apiPost('/api/systematics', {
				kuerzel: kuerzel.trim(),
				bezeichnung: bezeichnung.trim()
			});
			kuerzel = '';
			bezeichnung = '';
			await laden();
			onChanged?.();
			toastStore.addToast('Sachgruppe angelegt.', 'success');
		} catch {
			// Meldung kam bereits aus apiPost.
		} finally {
			speichert = false;
		}
	}

	/** @param {any} eintrag */
	function starteBearbeiten(eintrag) {
		bearbeiteId = eintrag.id;
		bearbeiteKuerzel = eintrag.kuerzel;
		bearbeiteBezeichnung = eintrag.bezeichnung;
	}

	function brichBearbeitenAb() {
		bearbeiteId = null;
		bearbeiteKuerzel = '';
		bearbeiteBezeichnung = '';
	}

	async function speichereBearbeitung() {
		if (!bearbeiteKuerzel.trim() || !bearbeiteBezeichnung.trim()) return;
		let daten;
		try {
			daten = await apiPut(`/api/systematics/${bearbeiteId}`, {
				kuerzel: bearbeiteKuerzel.trim(),
				bezeichnung: bearbeiteBezeichnung.trim()
			});
		} catch {
			return; // Meldung kam bereits aus apiPut.
		}
		brichBearbeitenAb();
		await laden();
		onChanged?.();
		// Ehrlich benennen, was NICHT mitgeändert wurde: Die Bücher tragen das Fach als
		// Text und die Signatur klebt am Buchrücken — beides zieht eine Umbenennung nicht nach.
		if (daten?.titel_mit_altfach > 0) {
			toastStore.addToast(
				`Geändert. ${daten.titel_mit_altfach} Bücher tragen noch das alte Fach — die müssen von Hand umgestellt werden.`,
				'info'
			);
		} else {
			toastStore.addToast('Sachgruppe geändert.', 'success');
		}
	}

	/** @param {any} eintrag */
	async function loeschen(eintrag) {
		if (!confirm(`Sachgruppe „${eintrag.kuerzel} – ${eintrag.bezeichnung}“ wirklich löschen?`))
			return;
		try {
			await apiDelete(`/api/systematics/${eintrag.id}`);
		} catch {
			return; // Meldung kam bereits aus apiDelete (z. B. "hängt noch an Büchern").
		}
		await laden();
		onChanged?.();
		toastStore.addToast('Sachgruppe gelöscht.', 'success');
	}
</script>

<section class="bg-white border border-slate-200 rounded-xl p-5 space-y-4">
	<div>
		<h2 class="font-bold text-slate-900">Sachgruppen</h2>
		<p class="text-sm text-slate-500 mt-0.5">
			Das Vokabular, aus dem das Buchformular die Signatur vorschlägt — aus „Deu“ wird „BIB Deu“
			bzw. „LMF Deu“.
		</p>
	</div>

	<form
		class="flex flex-wrap items-end gap-2"
		onsubmit={(e) => {
			e.preventDefault();
			anlegen();
		}}
	>
		<div class="flex flex-col gap-1">
			<label for="sys-kuerzel" class="text-xs font-medium text-slate-600">Kürzel</label>
			<input
				id="sys-kuerzel"
				bind:value={kuerzel}
				placeholder="Deu"
				class="h-9 w-28 rounded-lg border border-slate-300 bg-slate-50 px-3 text-sm text-slate-900 outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20"
			/>
		</div>
		<div class="flex flex-col gap-1 grow min-w-48">
			<label for="sys-bezeichnung" class="text-xs font-medium text-slate-600">Bezeichnung</label>
			<input
				id="sys-bezeichnung"
				bind:value={bezeichnung}
				placeholder="Deutsch"
				class="h-9 w-full rounded-lg border border-slate-300 bg-slate-50 px-3 text-sm text-slate-900 outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20"
			/>
		</div>
		<Button type="submit" disabled={speichert || !kuerzel.trim() || !bezeichnung.trim()}>
			Anlegen
		</Button>
	</form>

	{#if laedt}
		<p class="text-sm text-slate-500">Wird geladen …</p>
	{:else if liste.length === 0}
		<p class="text-sm text-slate-500">
			Noch keine Sachgruppen. Ohne sie schlägt das Buchformular nur „BIB“ bzw. „LMF“ ohne Fachkürzel
			vor.
		</p>
	{:else}
		<div class="overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="text-left text-xs uppercase tracking-wide text-slate-500">
						<th class="py-2 pr-3 font-medium">Kürzel</th>
						<th class="py-2 pr-3 font-medium">Bezeichnung</th>
						<th class="py-2 font-medium text-right">Aktion</th>
					</tr>
				</thead>
				<tbody>
					{#each liste as eintrag (eintrag.id)}
						<tr class="border-t border-slate-100">
							{#if bearbeiteId === eintrag.id}
								<td class="py-2 pr-3">
									<input
										bind:value={bearbeiteKuerzel}
										aria-label="Kürzel bearbeiten"
										class="h-9 w-24 rounded-lg border border-slate-300 bg-white px-2 text-sm outline-none focus:border-emerald-500"
									/>
								</td>
								<td class="py-2 pr-3">
									<input
										bind:value={bearbeiteBezeichnung}
										aria-label="Bezeichnung bearbeiten"
										class="h-9 w-full rounded-lg border border-slate-300 bg-white px-2 text-sm outline-none focus:border-emerald-500"
									/>
								</td>
								<td class="py-2 text-right whitespace-nowrap">
									<Button size="sm" onclick={speichereBearbeitung}>Sichern</Button>
									<Button size="sm" variant="ghost" onclick={brichBearbeitenAb}>Abbrechen</Button>
								</td>
							{:else}
								<td class="py-2 pr-3 font-mono text-slate-900">{eintrag.kuerzel}</td>
								<td class="py-2 pr-3 text-slate-700">{eintrag.bezeichnung}</td>
								<td class="py-2 text-right whitespace-nowrap">
									<Button size="sm" variant="secondary" onclick={() => starteBearbeiten(eintrag)}>
										Ändern
									</Button>
									<Button size="sm" variant="danger" onclick={() => loeschen(eintrag)}>
										Löschen
									</Button>
								</td>
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>
