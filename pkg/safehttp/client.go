// Package safehttp liefert HTTP-Clients für Anfragen an FREMDE Ziele (Cover-Bilder,
// Buchmetadaten). Sie unterscheiden sich von http.DefaultClient in einem Punkt, der
// bei solchen Anfragen entscheidend ist: Verbindungen zu nicht-öffentlichen IP-Adressen
// werden abgelehnt.
//
// Warum das nötig ist: Eine Host-Allowlist prüft den Namen in der URL — und nur den.
// Sie sagt nichts darüber, wohin die Verbindung am Ende geht. Ein erlaubter Host darf
// per HTTP-Redirect auf ein beliebiges anderes Ziel verweisen, und der Standard-Client
// folgt dem bis zu zehnmal, ohne den neuen Host noch einmal zu prüfen. Zeigt ein solcher
// Hop auf 127.0.0.1, 169.254.169.254 oder eine Adresse im Docker-Netz, stellt der Server
// diese Anfrage aus dem Netzinneren heraus — genau die Ausgangslage einer SSRF.
//
// Die Prüfung sitzt deshalb im Dialer und nicht in der URL-Behandlung: Dort greift sie
// nach der DNS-Auflösung, also für jeden Redirect-Hop und auch dann, wenn ein Name beim
// zweiten Auflösen plötzlich auf eine interne Adresse zeigt (DNS-Rebinding).
package safehttp

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// VerbieteInterneZieladressen lehnt Verbindungen zu Loopback-, privaten, Link-Local-,
// Multicast- und unspezifizierten Adressen ab. Läuft als Dialer-Control nach der
// DNS-Auflösung, address ist also immer eine aufgelöste IP:Port-Kombination.
func VerbieteInterneZieladressen(_, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("safehttp: unerwartete Zieladresse %q: %w", address, err)
	}
	// Unmap: IPv4-in-IPv6 (::ffff:127.0.0.1) würde sonst an den Is*-Checks vorbeigehen.
	addr := addrPort.Addr().Unmap()
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return fmt.Errorf("safehttp: Ziel-IP %s ist nicht öffentlich", addr)
	}
	return nil
}

// NeuerClient liefert einen HTTP-Client mit hartem Gesamt-Timeout, der ausschließlich
// öffentliche Ziele erreicht — auch über Redirects hinweg.
func NeuerClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
				Control: VerbieteInterneZieladressen,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}
