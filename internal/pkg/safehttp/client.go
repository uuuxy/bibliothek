// Package safehttp liefert HTTP-Clients für Anfragen an fremde Ziele (Cover, Metadaten zu
// Titeln). Anders als http.DefaultClient lehnen sie Verbindungen zu Adressen ab, die nicht
// öffentlich sind.
//
// Eine Liste erlaubter Hosts prüft nur den Namen in der URL und sagt nichts darüber, wohin die
// Verbindung am Ende geht. Ein erlaubter Host kann auf ein beliebiges anderes Ziel
// weiterleiten, und der Standard-Client folgt dem bis zu zehnmal, ohne den neuen Host zu
// prüfen. Zeigt eine solche Weiterleitung auf 127.0.0.1, 169.254.169.254 oder eine Adresse im
// Docker-Netz, stellt der Server die Anfrage aus dem Inneren des Netzes (SSRF).
//
// Die Prüfung sitzt deshalb im Dialer und nicht in der Behandlung der URL: Dort greift sie
// nach der Auflösung des Namens, also bei jeder Weiterleitung und auch dann, wenn ein Name
// beim zweiten Auflösen auf eine interne Adresse zeigt (DNS-Rebinding).
package safehttp

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// verbieteInterneZieladressen lehnt Verbindungen zu Loopback-, privaten, Link-Local-,
// Multicast- und unspezifizierten Adressen ab. Sie läuft als Control des Dialers nach der
// Auflösung des Namens; adresse ist deshalb immer eine IP mit Port.
func verbieteInterneZieladressen(_, adresse string, _ syscall.RawConn) error {
	ipPort, err := netip.ParseAddrPort(adresse)
	if err != nil {
		return fmt.Errorf("safehttp: unerwartete Zieladresse %q: %w", adresse, err)
	}
	// Ohne Unmap ginge eine IPv4-Adresse in IPv6-Schreibweise (::ffff:127.0.0.1) an den
	// Prüfungen vorbei.
	ip := ipPort.Addr().Unmap()
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("safehttp: Ziel-IP %s ist nicht öffentlich", ip)
	}
	return nil
}

// NewClient liefert einen HTTP-Client, der nur öffentliche Ziele erreicht, auch über
// Weiterleitungen hinweg; frist begrenzt die ganze Anfrage.
func NewClient(frist time.Duration) *http.Client {
	return &http.Client{
		Timeout: frist,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
				Control: verbieteInterneZieladressen,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}
