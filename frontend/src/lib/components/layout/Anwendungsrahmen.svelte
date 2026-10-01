<!-- @component Anwendungsrahmen — Seitenleiste und Arbeitsfläche der angemeldeten Anwendung.

     Eigenes Bauteil, weil App.svelte an der 200-Zeilen-Ratsche steht und der Rahmen eine
     zweite Aufgabe bekommen hat: Hinter dem Sperrbildschirm bleibt er stehen, damit
     Ungespeichertes die Sperre überlebt. `verdeckt` blendet ihn dann aus (hidden: nicht
     sichtbar, nicht in der Druckvorschau, nicht für Screenreader) und macht ihn träge
     (inert: kein Fokus, kein Klick). Tasten und Zeiger der ganzen Seite hält sperrSchild.js
     fern. -->
<script>
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menu.js';
	import Router from '../../Router.svelte';
	import BackupAlert from '../system/BackupAlert.svelte';
	import Hauptbereich from './Hauptbereich.svelte';
	import Sidebar from './Sidebar.svelte';
	import SkipLink from './SkipLink.svelte';

	/** @type {{ verdeckt: boolean }} */
	let { verdeckt } = $props();
</script>

<div
	class="h-screen flex w-full overflow-hidden"
	hidden={verdeckt}
	inert={verdeckt}
	data-anwendungsrahmen
>
	<SkipLink />
	<Sidebar />
	<!-- Arbeitsflaeche WEISS, nicht getoent. Am 07.08. hatte ich sie auf `surface`
	     gestellt, damit die weissen Karten sich abheben — und genau das war der
	     Fehler: Die Karten gab es laengst, sie lagen nur unsichtbar auf weissem
	     Grund. Die Toenung hat sie erst hervorgeholt und die Anwendung wirkte
	     "in Kacheln gezwaengt". Drei Commits (f2320e1, e81ce75, 95d5d33) hatten
	     das Floating-Card-Muster vorher ausdruecklich abgeschafft: edge-to-edge,
	     volle Breite, getrennt nur durch divide-y. -->
	<div
		class="bg-surface-container-lowest flex w-full min-w-0 flex-1 flex-col overflow-y-auto px-4 py-6 md:px-8"
	>
		<!-- Systemzustand, der eine Handlung braucht, steht über dem Inhalt —
		     nicht in der Navigation. Sichtbar für alle, die den Backup-Status lesen
		     dürfen (GET /api/admin/system/backup-status verlangt manage_settings). -->
		{#if hatRecht(authStore.currentUser, 'manage_settings')}
			<BackupAlert />
		{/if}
		<Hauptbereich><Router /></Hauptbereich>
	</div>
</div>
