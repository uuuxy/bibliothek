export function leseCsrfToken() {
    if (typeof document === "undefined") {
        return "";
    }

    const cookieEintraege = document.cookie ? document.cookie.split("; ") : [];
    let inventurToken = "";
    let mainToken = "";
    
    for (const eintrag of cookieEintraege) {
        if (eintrag.startsWith("inventur_csrf=")) {
            inventurToken = decodeURIComponent(eintrag.slice("inventur_csrf=".length));
        }
        if (eintrag.startsWith("csrf_token=")) {
            mainToken = decodeURIComponent(eintrag.slice("csrf_token=".length));
        }
    }

    return inventurToken || mainToken;
}

/**
 * @returns {Record<string, string>}
 */
export function csrfHeader() {
    const token = leseCsrfToken();
    if (!token) {
        return {};
    }
    return { "X-CSRF-Token": token };
}
