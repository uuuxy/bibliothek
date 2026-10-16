<!-- @component AusleiheRueckgabe — die Zelle „Rückgabe" einer Ausleihe: Datum, das Zeichen für
     überfällig und das Ändern des Datums.

     Eine eigene Datei, damit die Liste unter 200 Zeilen bleibt; der Bearbeiten-Zustand
     gehört zur Zeile.

     Überfällig trägt Farbe und Zeichen; „in Frist" ist der Normalfall und braucht keins. Das
     Ausleihdatum steht beim Zeigen auf dem Datum. Eine Dauerleihe hat keine Frist. -->
<script>
	import { apiFetch } from './apiFetch.js';
	import { melde } from '../inventur/lib/store.svelte.js';
	import { Check, Pencil, X } from '@lucide/svelte';
	import Feld from './components/ui/Feld.svelte';
	import UeberfaelligZeichen from './components/UeberfaelligZeichen.svelte';

	/** @type {{ ausleihe: any, ueberfaellig: boolean }} */
	let { ausleihe, ueberfaellig } = $props();

	let bearbeitet = $state(false);
	let neuesDatum = $state('');
	let speichert = $state(false);

	/** @param {string} wert */
	const datum = (wert) => new Date(wert).toLocaleDateString('de-DE');
	const geliehenAm = $derived(datum(ausleihe.ausgeliehen_am));

	async function speichere() {
		const id = ausleihe.ausleihe_id || ausleihe.id;
		if (!id || !neuesDatum) return;
		speichert = true;
		try {
			const response = await apiFetch(`/api/admin/ausleihen/${id}/faelligkeit`, {
				method: 'PATCH',
				body: JSON.stringify({ faellig_am: neuesDatum })
			});
			if (response.ok) {
				const daten = await response.json();
				ausleihe.rueckgabe_frist = daten.faellig_am;
				bearbeitet = false;
				melde(`Rückgabedatum auf ${datum(daten.faellig_am)} gesetzt.`, 'success');
			} else {
				const fehler = await response.json().catch(() => ({}));
				melde(fehler.error ?? 'Datum konnte nicht gespeichert werden.', 'error');
			}
		} catch (e) {
			console.error(e);
			melde('Netzwerkfehler beim Speichern des Datums.', 'error');
		} finally {
			speichert = false;
		}
	}
</script>

{#if bearbeitet}
	<div class="flex items-center gap-1.5">
		<Feld
			type="date"
			bind:value={neuesDatum}
			aria-label="Rückgabedatum"
			disabled={speichert}
			feld="w-40"
		/>
		<!-- Ohne Datum ist der Knopf gesperrt, statt beim Klick nichts zu tun. -->
		<button
			type="button"
			class="icon-btn text-success disabled:cursor-not-allowed disabled:text-on-surface/38"
			onclick={speichere}
			disabled={speichert || !neuesDatum}
			aria-label="Rückgabedatum speichern"
			data-tip="Speichern"
		>
			<Check class="h-4 w-4" aria-hidden="true" />
		</button>
		<button
			type="button"
			class="icon-btn text-on-surface-variant disabled:cursor-not-allowed disabled:text-on-surface/38"
			onclick={() => (bearbeitet = false)}
			disabled={speichert}
			aria-label="Bearbeiten abbrechen"
			data-tip="Abbrechen"
		>
			<X class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>
{:else}
	<div class="flex items-center gap-1">
		{#if ausleihe.ist_dauerleihe}
			<span data-tip="Geliehen am {geliehenAm}">Dauerleihe</span>
			<span class="sr-only">, ohne Frist, geliehen am {geliehenAm}</span>
		{:else}
			<span
				class={ueberfaellig ? 'font-semibold text-error' : ''}
				data-tip={ueberfaellig
					? `Überfällig, geliehen am ${geliehenAm}`
					: `Geliehen am ${geliehenAm}`}
			>
				{datum(ausleihe.rueckgabe_frist)}
			</span>
			<span class="sr-only">, geliehen am {geliehenAm}</span>
			{#if ueberfaellig}
				<UeberfaelligZeichen />
			{/if}
		{/if}
		<button
			type="button"
			class="icon-btn text-on-surface-variant"
			onclick={() => {
				neuesDatum = (ausleihe.rueckgabe_frist ?? '').split('T')[0];
				bearbeitet = true;
			}}
			aria-label="Rückgabedatum bearbeiten"
			data-tip="Rückgabedatum ändern"
		>
			<Pencil class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>
{/if}
