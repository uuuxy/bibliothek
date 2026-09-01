<!-- @component Der Konto-Status eines Schülers samt dem Knopf, der ihn umschaltet.

     Eigene Datei, seit die Sperre hier eingezogen ist: Sonst wäre StudentProfileCard
     um dreissig Zeilen weiter über die 200-Zeilen-Marke gewachsen, die in diesem
     Projekt gilt. Der Block hat damit auch einen Namen für das, was er ist — Zustand
     UND seine Umschaltung, nicht „noch ein Abschnitt in der Profilkarte".

     Die Sperre stand bis zum 08.08.2026 im Kasten „Dokumente & Aktionen", zwischen
     vier Knöpfen, die nur ein PDF erzeugen. Ihre Position kam dort aus einem ml-auto
     in einem flex-wrap-Container — und das schiebt nur innerhalb der aktuellen ZEILE
     nach rechts. Bei breitem Fenster stand die folgenreichste Aktion des Bildschirms
     also unabgesetzt am Zeilenende, bei schmalem rutschte sie allein nach unten
     rechts, genau dorthin, wo ein Dialog seinen Bestätigen-Knopf hat. Ihre Prominenz
     hing an der Fensterbreite. Jetzt steht sie bei dem Zustand, den sie umschaltet.

     Nicht zu verwechseln mit der „Gefahrenzone" im Stammdaten-Reiter: Dort steht das
     unumkehrbare Löschen. Eine Sperre ist umkehrbar und gehört deshalb hierher. -->
<script>
	import { Lock, Unlock } from '@lucide/svelte';
	import { ausleiheGesperrt } from '../../sperrStatus.js';
	import Button from '../ui/Button.svelte';

	/** @type {{ profile: any, onLock?: () => void }} */
	let { profile, onLock } = $props();
</script>

<div class="w-full space-y-3 border-t border-b border-slate-200 py-3">
	<div class="flex items-center justify-between">
		<span class="text-base text-slate-600">Konto-Status</span>
		<!-- Gesperrt ist die Ausnahme und trägt Farbe; „Aktiv" ist der Normalfall und
		     bleibt still. Ein pulsierender grüner Punkt für „alles in Ordnung" zieht
		     Aufmerksamkeit auf die einzige Stelle, die keine braucht. -->
		{#if ausleiheGesperrt(profile)}
			<span class="text-sm font-medium text-rose-600">Gesperrt</span>
		{:else}
			<span class="text-sm text-slate-500">Aktiv</span>
		{/if}
	</div>

	<!-- Der GRUND der Sperre. Er wird beim Sperren erfasst (StudentLockModal, Pflicht
	     per DB-Constraint) und kam bis zum 01.09.2026 in jeder Profil-Antwort mit,
	     wurde aber NIRGENDS angezeigt — wer entsperren wollte, musste raten, warum
	     gesperrt wurde (nur die DSGVO-Auskunft druckte ihn). Sichtbar nur im
	     gesperrten Zustand; das Profil steht ohnehin hinter view_students
	     (PII-Matrix: Sperrgrund = Stufe 2 hinter genau diesem Recht). -->
	{#if ausleiheGesperrt(profile) && profile.block_reason}
		<p class="text-sm text-on-surface-variant">
			<span class="font-medium text-on-surface">Grund:</span>
			{profile.block_reason}
		</p>
	{/if}

	{#if onLock}
		<!-- Beschriftung nach dem MANUELLEN Schloss, nicht nach ist_gesperrt: Ein Schüler
		     kann wegen Überfälligkeit gesperrt sein, ohne dass ihn jemand von Hand
		     gesperrt hätte. Stünde dann hier „Sperre aufheben", verspräche der Knopf
		     etwas, das er nicht halten kann — er löst nur das Handschloss. -->
		<Button
			variant={profile.is_manually_blocked ? 'success' : 'danger'}
			class="w-full"
			onclick={onLock}
		>
			{#if profile.is_manually_blocked}
				<Unlock class="w-4 h-4" aria-hidden="true" /> Sperre aufheben
			{:else}
				<Lock class="w-4 h-4" aria-hidden="true" /> Schüler sperren
			{/if}
		</Button>
	{/if}
</div>
