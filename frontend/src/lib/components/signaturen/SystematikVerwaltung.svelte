<script>
	import { onMount } from 'svelte';
	import SystematikTabelle from './SystematikTabelle.svelte';
	// apiPost wirft im Fehlerfall und hat die Meldung des Servers dann schon gezeigt. Ein
	// zweiter Toast im catch verdeckte den Grund („Kürzel existiert bereits").
	import { apiGet, apiPost } from '../../apiFetch.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ onChanged?: () => void }} */
	let { onChanged } = $props();

	let liste = $state(/** @type {any[]} */ ([]));
	let laedt = $state(true);
	let kuerzel = $state('');
	let bezeichnung = $state('');
	let speichert = $state(false);
	// „Anlegen" bleibt bei leerem Feld bedienbar: Der Klick nennt am Feld, was fehlt, und führt
	// dorthin (wie titelVerlangt in BuchFormular).
	let verlangt = $state(false);
	const kuerzelFehlt = $derived(verlangt && !kuerzel.trim());
	const bezeichnungFehlt = $derived(verlangt && !bezeichnung.trim());

	// Der Ladekreis steht nur beim ersten Laden. Ersetzte er die Tabelle auch beim Nachladen nach
	// einer Änderung, ginge eine offene Zeile samt Eingabe mit ihr.
	async function laden() {
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
		verlangt = true;
		if (!kuerzel.trim()) return document.getElementById('sys-kuerzel')?.focus();
		if (!bezeichnung.trim()) return document.getElementById('sys-bezeichnung')?.focus();
		speichert = true;
		try {
			await apiPost('/api/systematics', {
				kuerzel: kuerzel.trim(),
				bezeichnung: bezeichnung.trim()
			});
			verlangt = false;
			kuerzel = '';
			bezeichnung = '';
			await laden();
			onChanged?.();
			snackbars.addToast('Sachgruppe angelegt.', 'success');
		} catch {
			// Meldung kam bereits aus apiPost.
		} finally {
			speichert = false;
		}
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

	<!-- Ein Raster statt einer Flex-Reihe: Die Felder legen Beschriftung, Eingabe und Fehlertext
	     in seine drei Zeilen (Subgrid, ui/Feld), und der Knopf steht in der Zeile der Eingaben,
	     auch wenn unter einem Feld ein Fehlertext steht. Der Zeilenabstand ist der des Felds;
	     ein anderer verschöbe die Eingaben um die halbe Differenz gegen den Knopf. -->
	<form
		class="grid grid-cols-[7rem_minmax(0,1fr)_auto] gap-x-2 gap-y-1.5"
		onsubmit={(e) => {
			e.preventDefault();
			anlegen();
		}}
	>
		<Feld
			id="sys-kuerzel"
			etikett="Kürzel"
			bind:value={kuerzel}
			placeholder="Deu"
			ungueltig={kuerzelFehlt}
			hint={kuerzelFehlt ? 'Bitte ein Kürzel eintragen.' : ''}
		/>
		<Feld
			id="sys-bezeichnung"
			etikett="Bezeichnung"
			bind:value={bezeichnung}
			placeholder="Deutsch"
			ungueltig={bezeichnungFehlt}
			hint={bezeichnungFehlt ? 'Bitte eine Bezeichnung eintragen.' : ''}
		/>
		<Button type="submit" disabled={speichert} class="col-start-3 row-start-2">Anlegen</Button>
	</form>

	{#if laedt}
		<div class="flex justify-center py-6"><Ladekreis etikett="Sachgruppen laden" /></div>
	{:else if liste.length === 0}
		<p class="text-sm text-on-surface-variant">
			Noch keine Sachgruppen. Ohne sie schlägt das Buchformular nur „BIB“ bzw. „LMF“ ohne Fachkürzel
			vor.
		</p>
	{:else}
		<SystematikTabelle
			{liste}
			onGeaendert={async () => {
				await laden();
				onChanged?.();
			}}
		/>
	{/if}
</section>
