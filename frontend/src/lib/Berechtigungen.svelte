<script>
	import PageShell from './components/layout/PageShell.svelte';
	import PermissionManager from './PermissionManager.svelte';
</script>

<PageShell>
	<!-- KEINE Ueberschrift (a3e4184): die Seitenleiste sagt, wo man ist. Der Satz
	     erklaert stattdessen die Tragweite — die Tabelle hier steuert Menue UND API,
	     und die Drift-Warnung der Betriebsbereitschaft zeigt genau hierher. -->
	<p class="max-w-2xl text-sm text-on-surface-variant">
		Konten und Rollen-Rechte. Was hier steht, steuert Menü und API — Abweichungen von der
		Code-Vorgabe meldet die Betriebsbereitschaft.
	</p>

	<div class="mt-6">
		<PermissionManager />
	</div>
</PageShell>
