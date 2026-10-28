<script>
	import Tabelle from '../ui/Tabelle.svelte';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	// apiPut/apiDelete werfen im Fehlerfall und haben die Meldung des Servers dann schon gezeigt.
	// Ein zweiter Toast im catch verdeckte den Grund („Kürzel existiert bereits").
	import { apiPut, apiDelete } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';

	/**
	 * Die Sachgruppen als Tabelle; je Zeile ändern oder löschen. Nach jeder Änderung lädt
	 * onGeaendert die Liste neu.
	 * @type {{ liste: any[], onGeaendert: () => Promise<void> }}
	 */
	let { liste, onGeaendert } = $props();

	/** Zeile, die gerade bearbeitet wird (null = keine). */
	let bearbeiteId = $state(/** @type {string | null} */ (null));
	let bearbeiteKuerzel = $state('');
	let bearbeiteBezeichnung = $state('');
	// „Sichern" bleibt bei leerem Feld bedienbar: Der Klick nennt am Feld, was fehlt, und führt
	// dorthin (wie `verlangt` in BuchFormular).
	let verlangt = $state(false);
	const kuerzelFehlt = $derived(verlangt && !bearbeiteKuerzel.trim());
	const bezeichnungFehlt = $derived(verlangt && !bearbeiteBezeichnung.trim());
	// Die Sachgruppe, deren Löschen läuft; bis zur Antwort ist „Löschen" gesperrt, der Ladekreis
	// steht nur in ihrer Zeile. Ein zweiter Auftrag vor der Antwort schickte ein zweites DELETE,
	// und nach „Sachgruppe gelöscht." stand „Sachgruppe nicht gefunden".
	let loescht = $state(/** @type {string | null} */ (null));

	/** @param {any} eintrag */
	function starteBearbeiten(eintrag) {
		bearbeiteId = eintrag.id;
		bearbeiteKuerzel = eintrag.kuerzel;
		bearbeiteBezeichnung = eintrag.bezeichnung;
		verlangt = false;
	}

	function brichBearbeitenAb() {
		bearbeiteId = null;
		bearbeiteKuerzel = '';
		bearbeiteBezeichnung = '';
		verlangt = false;
	}

	async function speichereBearbeitung() {
		verlangt = true;
		if (!bearbeiteKuerzel.trim()) return document.getElementById('sys-aendern-kuerzel')?.focus();
		if (!bearbeiteBezeichnung.trim())
			return document.getElementById('sys-aendern-bezeichnung')?.focus();
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
		await onGeaendert();
		// Die Bezeichnung wird jetzt auf die Bücher mitgezogen (buecher_titel.subject).
		// Nur die Signatur am Buchrücken bleibt — die klebt physisch und folgt einem
		// Umlabeln, nicht einem Klick.
		if (daten?.titel_mitgezogen > 0) {
			snackbars.addToast(
				`Geändert. ${daten.titel_mitgezogen} Bücher auf das neue Fach umgestellt (Signatur am Buchrücken bleibt).`,
				'success'
			);
		} else {
			snackbars.addToast('Sachgruppe geändert.', 'success');
		}
	}

	/** @param {any} eintrag */
	async function loeschen(eintrag) {
		if (loescht) return;
		if (
			!(await loeschenBestaetigen(
				`Sachgruppe „${eintrag.kuerzel} – ${eintrag.bezeichnung}“ löschen?`
			))
		)
			return;
		loescht = eintrag.id;
		try {
			await apiDelete(`/api/systematics/${eintrag.id}`);
			await onGeaendert();
			snackbars.addToast('Sachgruppe gelöscht.', 'success');
		} catch {
			// Meldung kam bereits aus apiDelete (z. B. "hängt noch an Büchern").
		} finally {
			loescht = null;
		}
	}
</script>

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
						<!-- Oben ausgerichtet: Ein Fehlertext macht die Zeile höher, und mittig rückten das
						     andere Feld und die Knöpfe aus der Zeile der Eingaben. -->
						<td class="align-top">
							<Feld
								id="sys-aendern-kuerzel"
								bind:value={bearbeiteKuerzel}
								aria-label="Kürzel bearbeiten"
								feld="w-24"
								ungueltig={kuerzelFehlt}
								hint={kuerzelFehlt ? 'Bitte ein Kürzel eintragen.' : ''}
							/>
						</td>
						<td class="align-top">
							<Feld
								id="sys-aendern-bezeichnung"
								bind:value={bearbeiteBezeichnung}
								aria-label="Bezeichnung bearbeiten"
								ungueltig={bezeichnungFehlt}
								hint={bezeichnungFehlt ? 'Bitte eine Bezeichnung eintragen.' : ''}
							/>
						</td>
						<td class="align-top">
							<div class="flex h-9 items-center justify-end gap-1 whitespace-nowrap">
								<Button size="sm" onclick={speichereBearbeitung}>Sichern</Button>
								<Button size="sm" variant="ghost" onclick={brichBearbeitenAb}>Abbrechen</Button>
							</div>
						</td>
					{:else}
						<td class="font-mono">{eintrag.kuerzel}</td>
						<td>{eintrag.bezeichnung}</td>
						<td class="text-right whitespace-nowrap">
							<Button size="sm" variant="secondary" onclick={() => starteBearbeiten(eintrag)}>
								Ändern
							</Button>
							<Button
								size="sm"
								variant="danger"
								onclick={() => loeschen(eintrag)}
								disabled={loescht !== null}
							>
								{#if loescht === eintrag.id}
									<Ladekreis size="sm" farbe="aktuell" />
								{/if}
								Löschen
							</Button>
						</td>
					{/if}
				</tr>
			{/each}
		</tbody>
	</Tabelle>
</div>
