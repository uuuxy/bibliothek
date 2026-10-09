package bestelllink

import (
	"strings"
	"testing"
)

// Unter einem Tag gilt die Vorgabe, jeder andere Wert gilt, wie er eingestellt ist.
func TestTage(t *testing.T) {
	faelle := map[int]int{-5: VorgabeTage, 0: VorgabeTage, 1: 1, 20: 20, 45: 45, 365: 365}
	for eingestellt, will := range faelle {
		if ist := Tage(eingestellt); ist != will {
			t.Errorf("Tage(%d) = %d, erwartet %d", eingestellt, ist, will)
		}
	}
	if VorgabeTage < 1 {
		t.Errorf("die Vorgabe ist %d Tage, ein Link ohne Frist darf nicht entstehen", VorgabeTage)
	}
}

// Die Adresse kommt aus einem Eingabefeld und kann alles Mögliche enthalten.
func TestAdresse_WirdNormalisiert(t *testing.T) {
	faelle := []struct {
		name, basis, token, erwartet string
	}{
		{"mit Schema", "https://bib.schule.de", "ABC", "https://bib.schule.de/bestellung/ABC"},
		{"Schrägstrich am Ende", "https://bib.schule.de/", "ABC", "https://bib.schule.de/bestellung/ABC"},
		{"ohne Schema", "bib.schule.de", "ABC", "https://bib.schule.de/bestellung/ABC"},
		{"http bleibt http", "http://intern.schule", "ABC", "http://intern.schule/bestellung/ABC"},
		{"Leerzeichen", "  https://bib.schule.de  ", "ABC", "https://bib.schule.de/bestellung/ABC"},
		// Leer heißt: kein Link. Ein "/bestellung/ABC" ohne Host wäre in einer Mail wertlos
		// und sähe trotzdem nach einem echten Link aus.
		{"keine Adresse", "", "ABC", ""},
		{"kein Token", "https://bib.schule.de", "", ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := Adresse(f.basis, f.token); got != f.erwartet {
				t.Errorf("Adresse(%q, %q) = %q, erwartet %q", f.basis, f.token, got, f.erwartet)
			}
		})
	}
}

// Zwei Bestellungen dürfen nie denselben Token bekommen, und der gespeicherte Hash darf
// den Token nicht preisgeben.
func TestNeuerToken_EinmaligUndNurAlsHashGespeichert(t *testing.T) {
	tokenA, hashA, err := NeuerToken()
	if err != nil {
		t.Fatalf("Token erzeugen: %v", err)
	}
	tokenB, hashB, err := NeuerToken()
	if err != nil {
		t.Fatalf("Token erzeugen: %v", err)
	}

	if tokenA == tokenB || hashA == hashB {
		t.Fatal("zwei Aufrufe lieferten denselben Token — der Zufall funktioniert nicht")
	}
	if strings.Contains(hashA, tokenA) || hashA == tokenA {
		t.Fatal("der gespeicherte Hash enthält den Token im Klartext")
	}
	if len(hashA) != 64 {
		t.Errorf("Hashlänge = %d, erwartet 64 (SHA-256 als Hex)", len(hashA))
	}
	if Hash(tokenA) != hashA {
		t.Error("Hash liefert für denselben Token einen anderen Wert — die Seite fände die Bestellung nie")
	}
}

// Der Token trägt 32 Byte Zufall und nur Zeichen, die in einer Adresse unverändert bleiben.
func TestNeuerToken_Form(t *testing.T) {
	token, _, err := NeuerToken()
	if err != nil {
		t.Fatalf("Token erzeugen: %v", err)
	}
	if len(token) != 43 {
		t.Errorf("Token hat %d Zeichen, erwartet 43 (32 Byte in base64url ohne Füllzeichen)", len(token))
	}
	const erlaubt = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	if rest := strings.Trim(token, erlaubt); rest != "" {
		t.Errorf("Token %q enthält Zeichen außerhalb von base64url: %q", token, rest)
	}
}

// Der Hash ist SHA-256 als Hex. Mit einem anderen Verfahren fände die Seite keinen der
// gespeicherten Links mehr.
func TestHash_IstSHA256AlsHex(t *testing.T) {
	const will = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if ist := Hash("abc"); ist != will {
		t.Errorf("Hash(\"abc\") = %s, erwartet %s", ist, will)
	}
}
