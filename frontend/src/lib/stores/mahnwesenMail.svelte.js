import { apiFetch } from "../apiFetch.js";

/**
 * Handles mailing logic for Mahnwesen.
 */
export function useMahnwesenMail() {
    let modalOpen = $state(false);
    let modalKlasse = $state("");
    let modalEmail = $state("");
    let modalSending = $state(false);
    let modalMsg = $state(/** @type {{ type: 'success'|'error', text: string }|null} */ (null));

    /**
     * @param {string} klasse 
     * @param {string|null} [email] 
     */
    function openModal(klasse, email) {
        modalKlasse = klasse;
        modalEmail = email ?? "";
        modalMsg = null;
        modalOpen = true;
    }

    /**
     * Closes the mail modal.
     */
    function closeModal() {
        modalOpen = false;
        modalKlasse = "";
        modalEmail = "";
        modalMsg = null;
    }

    /**
     * Sends the Mahnliste to the specified class email.
     */
    async function sendMahnliste() {
        if (!modalEmail.trim()) { modalMsg = { type: 'error', text: 'E-Mail-Adresse angeben.' }; return; }
        modalSending = true;
        modalMsg = null;
        try {
            const res = await apiFetch("/api/mahnwesen/senden", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ klasse: modalKlasse, email: modalEmail }),
            });
            const json = await res.json();
            if (res.ok) {
                modalMsg = { type: 'success', text: json.message ?? "Gesendet." };
            } else {
                modalMsg = { type: 'error', text: json.error ?? json.message ?? "Fehler." };
            }
        } catch (e) {
            modalMsg = { type: 'error', text: String(e) };
        } finally {
            modalSending = false;
        }
    }

    return {
        get modalOpen() { return modalOpen; },
        get modalKlasse() { return modalKlasse; },
        get modalEmail() { return modalEmail; },
        set modalEmail(v) { modalEmail = v; },
        get modalSending() { return modalSending; },
        get modalMsg() { return modalMsg; },
        openModal,
        closeModal,
        sendMahnliste
    };
}
