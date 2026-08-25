<script>
	import { onMount } from 'svelte';
	// apiPost/apiPut/apiDelete liefern die geparste Antwort und WERFEN im Fehlerfall —
	// den Fehler-Toast haben sie dann schon gezeigt. Deshalb hier kein zweiter Toast im
	// catch: Ein generisches "Fehler beim Speichern" verdeckte sonst die Servermeldung,
	// die den Grund nennt (z. B. "Kürzel existiert bereits").
	import { apiGet, apiPost, apiPut, apiDelete } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

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
		// Die Bezeichnung wird jetzt auf die Bücher mitgezogen (buecher_titel.subject).
		// Nur die Signatur am Buchrücken bleibt — die klebt physisch und folgt einem
		// Umlabeln, nicht einem Klick.
		if (daten?.titel_mitgezogen > 0) {
			toastStore.addToast(
				`Geändert. ${daten.titel_mitgezogen} Bücher auf das neue Fach umgestellt (Signatur am Buchrücken bleibt).`,
				'success'
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

<!-- Nachgeordneter Abschnitt: Das Regal nachschlagen macht das Sekretariat taeglich,
     das Vokabular pflegen dreimal im Jahr. Die Trennung traegt deshalb eine Linie und
     Abstand — kein Kasten, der beides zu gleichrangigen Objekten machen wuerde. -->
<section class="space-y-4 border-t border-slate-200 pt-6">
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
		<Feld id="sys-kuerzel" label="Kürzel" bind:value={kuerzel} placeholder="Deu" class="w-28" />
		<Feld
			id="sys-bezeichnung"
			label="Bezeichnung"
			bind:value={bezeichnung}
			placeholder="Deutsch"
			class="grow min-w-48"
		/>
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
									<Feld bind:value={bearbeiteKuerzel} aria-label="Kürzel bearbeiten" feld="w-24" />
								</td>
								<td class="py-2 pr-3">
									<Feld bind:value={bearbeiteBezeichnung} aria-label="Bezeichnung bearbeiten" />
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
