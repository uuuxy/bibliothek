# Plugin-Baukasten (Vorlage)

Dieser Ordner dient als Startpunkt für optionale Systemerweiterungen. Entwickler können hier neue Funktionalitäten hinzufügen, ohne den Kerncode der Schulbibliothek zu verändern.

> ### ⚠️ Stand 23.08.2026: geparkt, und das Anschließen hat eine Nebenwirkung
>
> Dieses Beispiel-Plugin **läuft nicht mehr mit**. Registriert schrieb es bei jeder
> Rückgabe am Tresen zwei Zeilen ins Produktions-Log; der Aufruf `vorlage.Init()` ist
> deshalb aus `main.go` entfernt (Begründung steht dort).
>
> **Wer ein echtes Plugin baut, muss zwei Dinge wissen:**
>
> 1. Ein `Init()` allein bewirkt nichts — es muss aus `main.go` aufgerufen werden.
> 2. Sobald das geschieht, wird `scripts/deadcode_gate.sh` **rot** mit der Meldung
>    „Baseline-Einträge, die es nicht mehr gibt". Das ist kein Fehler, sondern Absicht:
>    `RegisterHook` und `vorlage.Init` stehen als begründete Ausnahmen in
>    `scripts/deadcode_baseline.txt`, weil sie zurzeit keinen Aufrufer haben. Wer einen
>    schafft, trägt die beiden Zeilen dort aus. Ohne diesen Hinweis stünde man vor einem
>    roten Lauf ohne erkennbaren Grund.
>
> Ob der Erweiterungspunkt überhaupt bleibt, ist **nach dem Pilotbetrieb** zu entscheiden.
> Stand heute spricht viel fürs Löschen: genau ein Ereignistyp, in 15 Monaten kein echtes
> Plugin, und der Frontend-Teil (`Extension.svelte`) war nie angeschlossen. Die Begründung
> in voller Länge steht in [`docs/befunde.md`](../../docs/befunde.md).

## Backend-Hooks (Go)

Plugins können sich an Backend-Events anmelden, indem sie Callbacks registrieren.

### Registrierung eines Hooks
Importiere das `plugins`-Paket und rufe `RegisterHook` in einer `Init()` Funktion auf:

```go
package meinplugin

import (
	"context"
	"log"
	"bibliothek/plugins"
)

func Init() {
	plugins.RegisterHook(plugins.EventBookReturned, func(ctx context.Context, payload any) error {
		data, ok := payload.(plugins.BookReturnedPayload)
		if !ok {
			return nil
		}
		// Event verarbeiten
		log.Printf("Buch zurückgegeben: %s", data.Titel)
		return nil
	})
}
```

### Registrierung in `main.go`
Importiere dein Plugin in `main.go` und rufe die `Init()` Methode während des Anwendungsstarts auf.

---

## Frontend-Registry (Svelte 5)

Das Svelte-Frontend bietet feste Slots (Extension Points), an denen Komponenten dynamisch gerendert werden können.

### Verfügbare Extension Points
1. **Sidebar (`sidebar`)**: Wird unten im Navigationsmenü gerendert.
2. **Schüler-Tab (`studentTab`)**: Wird auf der Profilkarte des ausgewählten Schülers eingebunden.

### Registrierung einer Frontend-Komponente
Importiere die Registrierungs-Funktionen in deiner Plugin-Schnittstelle (z. B. in einer Initialisierungsdatei):

```javascript
import { registerSidebarExtension, registerStudentTabExtension } from "$lib/plugins.svelte.js";
import MeinSidebarComponent from "./MeinSidebarComponent.svelte";
import MeinStudentComponent from "./MeinStudentComponent.svelte";

// Registriere Sidebar-Erweiterung
registerSidebarExtension(MeinSidebarComponent, { customProp: "Beispiel" });

// Registriere Tab-Erweiterung
registerStudentTabExtension("Mein Plugin-Bereich", MeinStudentComponent);
```
