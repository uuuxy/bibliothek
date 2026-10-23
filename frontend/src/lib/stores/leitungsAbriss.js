// Reißt die Live-Leitung ab, fragt dieses Modul den Server, wie es um die Anmeldung steht. Der
// Server beendet die Leitung, sobald die Anmeldung abgemeldet, gesperrt oder über der Höchstdauer
// ist (sse/sse.go). Ohne die Frage stünde an einer gesperrten Theke nach 25 Sekunden
// „Keine Verbindung“ statt des Sperrbildschirms. Ist die Anmeldung offen oder antwortet der
// Server nicht, baut liveEvents.js die Leitung selbst neu auf.

import { registriereAbrissHandler } from '../liveEvents.js';
import { authStore } from './authStore.svelte.js';
import { leerlaufSperre } from './leerlaufSperre.svelte.js';
import { leseSperrzustand } from './sperreAmServer.js';

registriereAbrissHandler(async () => {
	const stand = await leseSperrzustand();
	if (stand.zustand === 'gesperrt') leerlaufSperre.vomServerGesperrt();
	else if (stand.zustand === 'beendet') authStore.sitzungAbgelaufen();
});
