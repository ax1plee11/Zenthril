package benchmarks

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"
)

// BenchmarkAES256GCMEncryption тестирует производительность AES-256-GCM шифрования
func BenchmarkAES256GCMEncryption(b *testing.B) {
	key := make([]byte, 32) // AES-256
	rand.Read(key)
	
	block, err := aes.NewCipher(key)
	if err != nil {
		b.Fatal(err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		b.Fatal(err)
	}
	
	// Тестируем разные размеры сообщений
	sizes := []int{
		100,    // Короткое сообщение
		1024,   // 1 KB
		10240,  // 10 KB
		102400, // 100 KB
	}
	
	for _, size := range sizes {
		plaintext := make([]byte, size)
		rand.Read(plaintext)
		
		b.Run(formatSize(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				nonce := make([]byte, gcm.NonceSize())
				rand.Read(nonce)
				
				_ = gcm.Seal(nil, nonce, plaintext, nil)
			}
		})
	}
}

// BenchmarkAES256GCMDecryption тестирует производительность AES-256-GCM дешифрования
func BenchmarkAES256GCMDecryption(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	
	block, err := aes.NewCipher(key)
	if err != nil {
		b.Fatal(err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		b.Fatal(err)
	}
	
	sizes := []int{100, 1024, 10240, 102400}
	
	for _, size := range sizes {
		plaintext := make([]byte, size)
		rand.Read(plaintext)
		
		nonce := make([]byte, gcm.NonceSize())
		rand.Read(nonce)
		
		ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
		
		b.Run(formatSize(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := gcm.Open(nil, nonce, ciphertext, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkKeyGeneration тестирует генерацию ключей
func BenchmarkKeyGeneration(b *testing.B) {
	b.Run("AES-256", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := make([]byte, 32)
			rand.Read(key)
		}
	})
}

func formatSize(size int) string {
	if size < 1024 {
		return string(append([]byte{}, []byte("bytes_")...))
	}
	if size < 1024*1024 {
		kb := size / 1024
		return string(append([]byte{}, []byte("KB_")...))
	}
	mb := size / (1024 * 1024)
	return string(append([]byte{}, []byte("MB_")...))
}
