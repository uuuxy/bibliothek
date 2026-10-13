package logger

import (
	"testing"
)

func TestOhneUmbruch(t *testing.T) {
	faelle := []struct {
		name    string
		eingabe string
		want    string
	}{
		{name: "leer", eingabe: "", want: ""},
		{name: "ohne Umbruch", eingabe: "hello world", want: "hello world"},
		{name: "Zeilenvorschub", eingabe: "hello\nworld", want: "helloworld"},
		{name: "Wagenrücklauf", eingabe: "hello\rworld", want: "helloworld"},
		{name: "beide", eingabe: "hello\r\nworld", want: "helloworld"},
		{name: "mehrere, auch am Rand", eingabe: "\r\nhello\n\rworld\n\r", want: "helloworld"},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := OhneUmbruch(f.eingabe); got != f.want {
				t.Errorf("OhneUmbruch(%q) = %q, erwartet %q", f.eingabe, got, f.want)
			}
		})
	}
}
