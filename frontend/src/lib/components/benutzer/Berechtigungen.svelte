<script>
	import Seitenrahmen from '../layout/Seitenrahmen.svelte';
	import BenutzerUndRechte from './BenutzerUndRechte.svelte';
</script>

<!-- Keine Überschrift: Die Seitenleiste sagt, wo man ist. Der Satz zur Tragweite steht im
     Reiter der Rechte (BenutzerUndRechte), nicht über allem: Dort schöbe er die Suchpille des
     Benutzer-Reiters von der Startlinie, und „steuert Menü und API" gilt nur für die
     Rechte-Tabelle. -->
<Seitenrahmen>
	<BenutzerUndRechte />
</Seitenrahmen>
