# Plugin-Baukasten (Vorlage)

Dieser Ordner dient als Startpunkt für optionale Systemerweiterungen. Entwickler können hier neue Funktionalitäten hinzufügen, ohne den Kerncode der Schulbibliothek zu verändern.

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
