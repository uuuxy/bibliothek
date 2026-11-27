<!-- @component KlassenleitungNeu — der neue Eintrag unter der Liste der Klassenleitungen.
     Eigene Datei, weil Zustand, Speichern und der Fehler am Feld nur diesem Formular gehören.
     Die Klasse ist eine Auswahl: Ohne sie bleibt „Hinzufügen" gesperrt. Eine fehlende Adresse
     nennt der Klick am Feld (ARCHITEKTUR, „Felder und Eingabe“). -->
<script>
	import { onMount, tick } from 'svelte';
	import { apiPost } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { meldeFormular } from '../../stores/verlassenSchutz.svelte.js';
	import Feld from '../ui/Feld.svelte';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';
	import Pflichthinweis from '../ui/Pflichthinweis.svelte';
	import {
		erzeugeKlassenVorschlaege,
		klassenPlatzhalter,
		klassenLadefehler
	} from '../leser/klassenVorschlaege.svelte.js';

	/** @type {{ ongespeichert: () => Promise<void> }} */
	let { ongespeichert } = $props();

	let klasse = $state('');
	let email = $state('');
	let speichert = $state(false);
	let verlangt = $state(false);
	const emailFehlt = $derived(verlangt && email.trim() === '');
	meldeFormular(() => klasse.trim() !== '' || email.trim() !== '');
	// Auswählen statt tippen: Ein Tippfehler legte sonst eine Klasse an, die es an der Schule
	// nicht gibt, und ihre Mahnliste ginge an niemanden.
	const klassenListe = erzeugeKlassenVorschlaege();
	onMount(() => {
		klassenListe.lade();
	});

	async function hinzufuegen() {
		if (!klasse.trim() || speichert) return;
		verlangt = true;
		if (email.trim() === '') {
			await tick();
			document.getElementById('routing-email')?.focus();
			return;
		}
		speichert = true;
		try {
			await apiPost('/api/klassen-mapping', { klasse: klasse.trim(), lehrer_email: email.trim() });
			klasse = '';
			email = '';
			verlangt = false;
			await ongespeichert();
			snackbars.addToast('Zuordnung gespeichert', 'success');
		} catch {
			// Die Meldung kam bereits aus apiPost.
		} finally {
			speichert = false;
		}
	}
</script>

<div class="space-y-4">
	<Pflichthinweis />
	<div class="flex flex-col md:flex-row items-end gap-4">
		<div class="w-full md:w-32">
			<div class="grid gap-y-1.5">
				<label for="routing-klasse" class="text-sm font-medium text-on-surface-variant"
					>Klasse</label
				>
				<Select
					id="routing-klasse"
					bind:value={klasse}
					options={klassenListe.liste.map((k) => ({ value: k, label: k }))}
					placeholder={klassenPlatzhalter(klassenListe.liste.length, klassenListe.ladefehler)}
				/>
			</div>
		</div>
		<div class="flex-1 w-full">
			<Feld
				id="routing-email"
				bind:value={email}
				etikett="E-Mail"
				type="email"
				placeholder="lehrkraft@schule.de"
				aria-required="true"
				ungueltig={emailFehlt}
				hint={emailFehlt ? 'Bitte die E-Mail-Adresse der Klassenleitung eintragen' : ''}
			/>
		</div>
		<Button
			onclick={hinzufuegen}
			disabled={speichert || !klasse.trim()}
			class="w-full md:w-auto whitespace-nowrap"
		>
			{speichert ? 'Wird gespeichert …' : 'Hinzufügen'}
		</Button>
	</div>
	<!-- Unter der Zeile, nicht unter der Auswahl: Auswahl, Feld und Knopf stehen bündig unten. -->
	{#if klassenListe.ladefehler}
		<p role="alert" class="text-xs text-error">{klassenLadefehler(klassenListe.ladefehler)}</p>
	{/if}
</div>
