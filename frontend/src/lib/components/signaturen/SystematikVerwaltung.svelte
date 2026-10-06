<script>
	import { onMount } from 'svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	// apiPost/apiPut/apiDelete werfen im Fehlerfall und haben die Meldung des Servers dann
	// schon gezeigt. Ein zweiter Toast im catch verdeckte den Grund („Kürzel existiert bereits").
	import { apiGet, apiPost, apiPut, apiDelete } from '../../apiFetch.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
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
		if (
			!(await loeschenBestaetigen(
				`Sachgruppe „${eintrag.kuerzel} – ${eintrag.bezeichnung}“ löschen?`
			))
		)
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

<!-- Nachgeordneter Abschnitt: Das Regal nachschlagen ist tägliche Arbeit, das Vokabular
     pflegen seltene. Die Trennung trägt deshalb eine Linie und Abstand — kein Kasten, der
     beides zu gleichrangigen Objekten machte. -->
<section class="space-y-4 border-t border-outline-variant pt-6">
	<div>
		<h2 class="font-bold text-on-surface">Sachgruppen</h2>
		<p class="text-sm text-on-surface-variant mt-0.5">
			Das Fach-Vokabular des Katalogs. Die Signatur am Regal schlägt es nicht mehr vor: Dort gelten
			die Adressen aus dem Bestand.
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
		<div class="flex justify-center py-6"><Ladekreis label="Sachgruppen laden" /></div>
	{:else if liste.length === 0}
		<p class="text-sm text-on-surface-variant">
			Noch keine Sachgruppen. Ohne sie schlägt das Buchformular nur „BIB“ bzw. „LMF“ ohne Fachkürzel
			vor.
		</p>
	{:else}
		<div class="overflow-x-auto">
			<Tabelle beschriftung="Systematik">
				<thead>
					<tr>
						<th>Kürzel</th>
						<th>Bezeichnung</th>
						<th class="text-right">Aktion</th>
					</tr>
				</thead>
				<tbody>
					{#each liste as eintrag (eintrag.id)}
						<tr>
							{#if bearbeiteId === eintrag.id}
								<td>
									<Feld bind:value={bearbeiteKuerzel} aria-label="Kürzel bearbeiten" feld="w-24" />
								</td>
								<td>
									<Feld bind:value={bearbeiteBezeichnung} aria-label="Bezeichnung bearbeiten" />
								</td>
								<td class="text-right whitespace-nowrap">
									<Button size="sm" onclick={speichereBearbeitung}>Sichern</Button>
									<Button size="sm" variant="ghost" onclick={brichBearbeitenAb}>Abbrechen</Button>
								</td>
							{:else}
								<td class="font-mono">{eintrag.kuerzel}</td>
								<td>{eintrag.bezeichnung}</td>
								<td class="text-right whitespace-nowrap">
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
			</Tabelle>
		</div>
	{/if}
</section>
