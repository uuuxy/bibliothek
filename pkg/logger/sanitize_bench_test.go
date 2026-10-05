package logger

import "testing"

func BenchmarkSanitizeLog_Clean(b *testing.B) {
	input := "hello world without any special characters for log sanitization check"
	for i := 0; i < b.N; i++ {
		SanitizeLog(input)
	}
}

func BenchmarkSanitizeLog_Dirty(b *testing.B) {
	input := "hello\r\nworld\n with multiple \r\n characters for \n log sanitization check\r\n"
	for i := 0; i < b.N; i++ {
		SanitizeLog(input)
	}
}
