package benchmarks

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BenchmarkDatabaseInsert тестирует производительность вставки в БД
func BenchmarkDatabaseInsert(b *testing.B) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		b.Skip("DB_URL not set, skipping database benchmarks")
	}
	
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()
	
	// Создаём тестовую таблицу
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS bench_test_messages (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Exec(context.Background(), "DROP TABLE IF EXISTS bench_test_messages")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pool.Exec(context.Background(),
			"INSERT INTO bench_test_messages (content) VALUES ($1)",
			fmt.Sprintf("test message %d", i),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDatabaseSelect тестирует производительность SELECT запросов
func BenchmarkDatabaseSelect(b *testing.B) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		b.Skip("DB_URL not set, skipping database benchmarks")
	}
	
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()
	
	// Создаём тестовую таблицу и заполняем данными
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS bench_test_select (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Exec(context.Background(), "DROP TABLE IF EXISTS bench_test_select")
	
	// Вставляем тестовые данные
	for i := 0; i < 1000; i++ {
		pool.Exec(context.Background(),
			"INSERT INTO bench_test_select (content) VALUES ($1)",
			fmt.Sprintf("test message %d", i),
		)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := pool.Query(context.Background(),
			"SELECT id, content, created_at FROM bench_test_select LIMIT 50",
		)
		if err != nil {
			b.Fatal(err)
		}
		
		for rows.Next() {
			var id int
			var content string
			var createdAt time.Time
			rows.Scan(&id, &content, &createdAt)
		}
		rows.Close()
	}
}

// BenchmarkDatabaseUpdate тестирует производительность UPDATE запросов
func BenchmarkDatabaseUpdate(b *testing.B) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		b.Skip("DB_URL not set, skipping database benchmarks")
	}
	
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()
	
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS bench_test_update (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Exec(context.Background(), "DROP TABLE IF EXISTS bench_test_update")
	
	// Вставляем тестовые данные
	for i := 0; i < 100; i++ {
		pool.Exec(context.Background(),
			"INSERT INTO bench_test_update (content) VALUES ($1)",
			fmt.Sprintf("test message %d", i),
		)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pool.Exec(context.Background(),
			"UPDATE bench_test_update SET content = $1, updated_at = NOW() WHERE id = $2",
			fmt.Sprintf("updated message %d", i),
			(i%100)+1,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDatabaseTransaction тестирует производительность транзакций
func BenchmarkDatabaseTransaction(b *testing.B) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		b.Skip("DB_URL not set, skipping database benchmarks")
	}
	
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()
	
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS bench_test_tx (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL
		)
	`)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Exec(context.Background(), "DROP TABLE IF EXISTS bench_test_tx")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			b.Fatal(err)
		}
		
		for j := 0; j < 10; j++ {
			_, err := tx.Exec(context.Background(),
				"INSERT INTO bench_test_tx (content) VALUES ($1)",
				fmt.Sprintf("message %d-%d", i, j),
			)
			if err != nil {
				tx.Rollback(context.Background())
				b.Fatal(err)
			}
		}
		
		if err := tx.Commit(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDatabaseConcurrentQueries тестирует параллельные запросы
func BenchmarkDatabaseConcurrentQueries(b *testing.B) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		b.Skip("DB_URL not set, skipping database benchmarks")
	}
	
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()
	
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS bench_test_concurrent (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL
		)
	`)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Exec(context.Background(), "DROP TABLE IF EXISTS bench_test_concurrent")
	
	// Заполняем данными
	for i := 0; i < 1000; i++ {
		pool.Exec(context.Background(),
			"INSERT INTO bench_test_concurrent (content) VALUES ($1)",
			fmt.Sprintf("test %d", i),
		)
	}
	
	concurrencyLevels := []int{1, 10, 50, 100}
	
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrent_%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					rows, err := pool.Query(context.Background(),
						"SELECT id, content FROM bench_test_concurrent LIMIT 10",
					)
					if err != nil {
						b.Fatal(err)
					}
					
					for rows.Next() {
						var id int
						var content string
						rows.Scan(&id, &content)
					}
					rows.Close()
				}
			})
		})
	}
}

// BenchmarkDatabaseIndexedVsNonIndexed сравнивает запросы с индексом и без
func BenchmarkDatabaseIndexedVsNonIndexed(b *testing.B) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		b.Skip("DB_URL not set, skipping database benchmarks")
	}
	
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()
	
	// Таблица без индекса
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS bench_test_no_index (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Exec(context.Background(), "DROP TABLE IF EXISTS bench_test_no_index")
	
	// Таблица с индексом
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS bench_test_with_index (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_username ON bench_test_with_index(username);
	`)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Exec(context.Background(), "DROP TABLE IF EXISTS bench_test_with_index")
	
	// Заполняем обе таблицы
	for i := 0; i < 10000; i++ {
		username := fmt.Sprintf("user%d", i)
		email := fmt.Sprintf("user%d@example.com", i)
		
		pool.Exec(context.Background(),
			"INSERT INTO bench_test_no_index (username, email) VALUES ($1, $2)",
			username, email,
		)
		pool.Exec(context.Background(),
			"INSERT INTO bench_test_with_index (username, email) VALUES ($1, $2)",
			username, email,
		)
	}
	
	b.Run("without_index", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rows, _ := pool.Query(context.Background(),
				"SELECT id, email FROM bench_test_no_index WHERE username = $1",
				fmt.Sprintf("user%d", i%10000),
			)
			if rows != nil {
				rows.Close()
			}
		}
	})
	
	b.Run("with_index", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rows, _ := pool.Query(context.Background(),
				"SELECT id, email FROM bench_test_with_index WHERE username = $1",
				fmt.Sprintf("user%d", i%10000),
			)
			if rows != nil {
				rows.Close()
			}
		}
	})
}
