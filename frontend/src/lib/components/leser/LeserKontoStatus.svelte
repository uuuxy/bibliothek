<!-- @component Der Konto-Status eines Schülers samt dem Knopf, der ihn umschaltet: Zustand
     und seine Umschaltung, nicht „noch ein Abschnitt in der Profilkarte".

     Die Sperre steht bei dem Zustand, den sie umschaltet, nicht zwischen Knöpfen, die nur
     ein PDF erzeugen: Dort hinge ihre Stelle an der Fensterbreite, und bei schmalem Fenster
     rutschte sie dorthin, wo ein Dialog seinen Bestätigen-Knopf hat.

     Nicht zu verwechseln mit der „Gefahrenzone" im Stammdaten-Reiter: Dort steht das
     unumkehrbare Löschen. Eine Sperre ist umkehrbar und gehört deshalb hierher. -->
<script>
	import { Lock, LockOpen } from '@lucide/svelte';
	import { ausleiheGesperrt } from './sperrStatus.js';
	import { istKollegium } from './leserArt.js';
	import Button from '../ui/Button.svelte';
	import { formatEuro } from '../../utils/format.js';

	/** @type {{ leser: any, onLock?: () => void, offen?: { anzahl: number, summe: number } }} */
	let { leser, onLock, offen = undefined } = $props();

	const gesperrt = $derived(ausleiheGesperrt(leser));
	const kollege = $derived(istKollegium(leser));
</script>

<div class="w-full space-y-3 border-t border-b border-outline-variant py-3">
	<div class="flex items-center justify-between">
		<span class="text-base text-on-surface-variant">Konto-Status</span>
		<!-- Gesperrt ist die Ausnahme und trägt Farbe; „Aktiv" ist der Normalfall und
		     bleibt still. Ein pulsierender grüner Punkt für „alles in Ordnung" zieht
		     Aufmerksamkeit auf die einzige Stelle, die keine braucht. -->
		{#if gesperrt}
			<span class="text-sm font-medium text-error">Gesperrt</span>
		{:else}
			<span class="text-sm text-on-surface-variant">Aktiv</span>
		{/if}
	</div>

	<!-- Der Grund der Sperre, erfasst beim Sperren (LeserSperreDialog, Pflicht per
	     DB-Constraint): Wer entsperren will, soll nicht raten müssen, warum gesperrt wurde.
	     Sichtbar nur im gesperrten Zustand; das Profil steht ohnehin hinter view_students
	     (PII-Matrix: Sperrgrund = Stufe 2 hinter genau diesem Recht). -->
	{#if gesperrt && leser.block_reason}
		<p class="text-sm text-on-surface-variant">
			<span class="font-medium text-on-surface">Grund:</span>
			{leser.block_reason}
		</p>
	{/if}

	<!-- Was der Leser schuldet, ist Kontozustand: Die Zeile steht bei jedem Leser, und „keine"
	     bleibt still wie „Aktiv". Die Liste dazu trägt der Reiter „Gebühren & Schäden". Die Theke
	     ohne Akte kennt die Forderungen nicht; „keine" wäre dort eine ungeprüfte Auskunft. -->
	{#if offen}
		<div class="flex items-center justify-between">
			<span class="text-base text-on-surface-variant">Offene Forderungen</span>
			{#if offen.anzahl > 0}
				<span class="text-sm font-medium text-error">{formatEuro(offen.summe)}</span>
			{:else}
				<span class="text-sm text-on-surface-variant">keine</span>
			{/if}
		</div>
	{/if}

	{#if onLock}
		<!-- Beschriftung nach dem einen Prädikat (sperrStatus.js): Der Knopf hebt jede Sperre am
		     Leser auf, auch die der Ehemaligen (api/leser/leser_sperre.go). Ein Kollege wird nie
		     gesperrt: Der Knopf bleibt an seiner Stelle, verschlossen, mit dem Satz warum (eine
		     Maske für jeden). -->
		<Button
			variant={gesperrt ? 'success' : 'danger'}
			class="w-full"
			onclick={onLock}
			disabled={kollege}
		>
			{#if gesperrt}
				<LockOpen class="w-4 h-4" aria-hidden="true" /> Sperre aufheben
			{:else}
				<Lock class="w-4 h-4" aria-hidden="true" />
				{kollege ? 'Kollegen sperren' : 'Schüler sperren'}
			{/if}
		</Button>
		{#if kollege}
			<p class="text-sm text-on-surface-variant">Kollegen werden nicht gesperrt.</p>
		{/if}
	{/if}
</div>
