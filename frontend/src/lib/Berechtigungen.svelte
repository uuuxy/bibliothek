<script>
	import PageShell from './components/layout/PageShell.svelte';
	import PermissionManager from './PermissionManager.svelte';
</script>

<!-- KEINE Ueberschrift (a3e4184): die Seitenleiste sagt, wo man ist.

     Der Erklaersatz zur Tragweite stand bis zum 04.09.2026 HIER, ueber allem — und damit
     ueber der Suchpille des Benutzer-Reiters, die dadurch als einzige im Haus nicht an der
     Startlinie begann. Er stimmte an dieser Stelle ohnehin nur halb: „steuert Menue und
     API" gilt fuer die Rechte-Tabelle, nicht fuer die Benutzerliste. Er steht jetzt in
     ihrem eigenen Reiter, siehe PermissionManager. -->
<PageShell>
	<PermissionManager />
</PageShell>
