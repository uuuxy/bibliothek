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

    let sendingStudentId = $state(/** @type {string|null} */ (null));
    let studentMessages = $state(/** @type {Record<string, {type: 'success'|'error', text: string} | null>} */ ({}));

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

    /**
     * Sends a reminder directly to a student's parents.
     * @param {string} schuelerId 
     */
    async function sendStudentMahnung(schuelerId) {
        sendingStudentId = schuelerId;
        studentMessages[schuelerId] = null;
        try {
            const res = await apiFetch(`/api/mail/send-overdue-notification/${schuelerId}`, { method: "POST" });
            const json = await res.json();
            if (res.ok) {
                studentMessages[schuelerId] = { type: 'success', text: json.message || "Gesendet" };
            } else {
                studentMessages[schuelerId] = { type: 'error', text: json.error || json.message || "Fehler" };
            }
        } catch (e) {
            studentMessages[schuelerId] = { type: 'error', text: String(e) };
        } finally {
            sendingStudentId = null;
        }
    }

    return {
        get modalOpen() { return modalOpen; },
        get modalKlasse() { return modalKlasse; },
        get modalEmail() { return modalEmail; },
        set modalEmail(v) { modalEmail = v; },
        get modalSending() { return modalSending; },
        get modalMsg() { return modalMsg; },
        get sendingStudentId() { return sendingStudentId; },
        get studentMessages() { return studentMessages; },
        openModal,
        closeModal,
        sendMahnliste,
        sendStudentMahnung
    };
}
