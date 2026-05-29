<!-- 
  IsbnFeld.svelte
  Verwaltet die Eingabe einer ISBN. Beinhaltet zwei Buttons: 
  Einen für die API-Abfrage der Metadaten und einen für den Barcode-Scanner.
-->
<script>
    let { formular = $bindable(), wirdGescannt = $bindable() } = $props();

    async function aktualisiereMetadaten() {
        if (!formular.isbn) return;
        try {
            const antwort = await fetch(`/api/lookup/${formular.isbn}`);
            if (antwort.ok) {
                const json = await antwort.json();
                const daten = json.data;
                if (daten.title) formular.title = daten.title;
                if (daten.author) formular.author = daten.author;
                if (daten.coverUrl) formular.coverUrl = daten.coverUrl;
                if (daten.subject) formular.subject = daten.subject;
                if (daten.grade) {
                    const parsedGrade = parseInt(daten.grade);
                    if (!Number.isNaN(parsedGrade)) {
                        formular.gradeLevel = parsedGrade;
                    }
                }
            }
        } catch (fehler) {
            console.error("Fehler beim Nachschlagen der ISBN", fehler);
        }
    }

    async function beiVerlassen() {
        if (formular.isbn && !formular.title) {
            await aktualisiereMetadaten();
        }
    }
</script>

<div>
    <label for="buch-isbn" class="block text-sm font-medium text-gray-700 mb-1">
        ISBN
    </label>
    <div class="relative">
        <input
            id="buch-isbn"
            type="text"
            bind:value={formular.isbn}
            onblur={beiVerlassen}
            class="w-full rounded-lg border-gray-300 bg-gray-50 px-4 py-2.5 pr-20 text-gray-900 focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none transition"
        />
        <button
            type="button"
            onclick={aktualisiereMetadaten}
            class="absolute right-10 top-2 text-gray-400 hover:text-emerald-600 p-0.5 rounded-full hover:bg-gray-200 transition-colors"
            title="Daten aus dem Internet aktualisieren"
            aria-label="Daten aus dem Internet aktualisieren"
        >
            <svg
                class="w-5 h-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                />
            </svg>
        </button>
        <button
            type="button"
            onclick={() => (wirdGescannt = true)}
            aria-pressed={wirdGescannt}
            class="absolute right-2 top-2 text-gray-400 hover:text-emerald-600 p-0.5 rounded-full hover:bg-gray-200 transition-colors"
            title="Scan ISBN"
            aria-label="Scan ISBN"
        >
            <svg
                class="w-5 h-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
            >
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"
                />
                <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"
                />
            </svg>
        </button>
    </div>
</div>

