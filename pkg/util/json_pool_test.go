package util

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type TestStruct struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Address       string    `json:"address"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Zip           string    `json:"zip"`
	Country       string    `json:"country"`
	Phone         string    `json:"phone"`
	Age           int       `json:"age"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Tags          []string  `json:"tags"`
	Roles         []string  `json:"roles"`
	Permissions   []string  `json:"permissions"`
	Groups        []string  `json:"groups"`
	Subscriptions []string  `json:"subscriptions"`
	Preferences   []string  `json:"preferences"`
	Settings      []string  `json:"settings"`
}

func TestJsonPool_Marshal(t *testing.T) {
	pool := NewJsonPool()

	testData := TestStruct{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	result, err := pool.Marshal(testData)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify the result is valid JSON
	var unmarshaled TestStruct
	if err := json.Unmarshal(result, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ID != testData.ID || unmarshaled.Name != testData.Name || unmarshaled.Email != testData.Email {
		t.Errorf("Expected %+v, got %+v", testData, unmarshaled)
	}
}

func TestJsonPool_Unmarshal(t *testing.T) {
	pool := NewJsonPool()

	jsonData := []byte(`{"id":1,"name":"John Doe","email":"john@example.com"}`)

	var result TestStruct
	if err := pool.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	expected := TestStruct{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	if result.ID != expected.ID || result.Name != expected.Name || result.Email != expected.Email {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestJsonPool_UnmarshalString(t *testing.T) {
	pool := NewJsonPool()

	jsonString := `{"id":1,"name":"John Doe","email":"john@example.com"}`

	var result TestStruct
	if err := pool.UnmarshalString(jsonString, &result); err != nil {
		t.Fatalf("UnmarshalString failed: %v", err)
	}

	expected := TestStruct{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	if result.ID != expected.ID || result.Name != expected.Name || result.Email != expected.Email {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestJsonPool_MarshalToString(t *testing.T) {
	pool := NewJsonPool()

	testData := TestStruct{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	result, err := pool.MarshalToString(testData)
	if err != nil {
		t.Fatalf("MarshalToString failed: %v", err)
	}

	// Verify the result is valid JSON
	var unmarshaled TestStruct
	if err := json.Unmarshal([]byte(result), &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ID != testData.ID || unmarshaled.Name != testData.Name || unmarshaled.Email != testData.Email {
		t.Errorf("Expected %+v, got %+v", testData, unmarshaled)
	}
}

func TestJsonPool_MarshalIndent(t *testing.T) {
	pool := NewJsonPool()

	testData := TestStruct{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	result, err := pool.MarshalIndent(testData, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}

	// Verify the result is valid JSON
	var unmarshaled TestStruct
	if err := json.Unmarshal(result, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ID != testData.ID || unmarshaled.Name != testData.Name || unmarshaled.Email != testData.Email {
		t.Errorf("Expected %+v, got %+v", testData, unmarshaled)
	}
}

func TestJsonPool_Concurrency(t *testing.T) {
	pool := NewJsonPool()

	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				testData := TestStruct{
					ID:    id*numOperations + j,
					Name:  "Test User",
					Email: "test@example.com",
				}

				// Test Marshal
				result, err := pool.Marshal(testData)
				if err != nil {
					t.Errorf("Marshal failed: %v", err)
					return
				}

				// Test Unmarshal
				var unmarshaled TestStruct
				if err := pool.Unmarshal(result, &unmarshaled); err != nil {
					t.Errorf("Unmarshal failed: %v", err)
					return
				}

				if unmarshaled.ID != testData.ID {
					t.Errorf("Expected ID %d, got %d", testData.ID, unmarshaled.ID)
				}
			}
		}(i)
	}

	wg.Wait()
}

// Benchmark comparing pooled vs standard JSON operations
func BenchmarkJsonPool_Marshal(b *testing.B) {
	pool := NewJsonPool()
	testData := TestStruct{
		ID:            1,
		Name:          "John Doe",
		Email:         "john@example.com",
		Address:       "123 Main St, Anytown, USA",
		City:          "Anytown",
		State:         "CA",
		Zip:           "12345",
		Country:       "USA",
		Phone:         "123-456-7890",
		Age:           30,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Tags:          []string{"tag1", "tag2", "tag3"},
		Roles:         []string{"role1", "role2", "role3"},
		Permissions:   []string{"perm1", "perm2", "perm3"},
		Groups:        []string{"group1", "group2", "group3"},
		Subscriptions: []string{"sub1", "sub2", "sub3"},
		Preferences:   []string{"pref1", "pref2", "pref3"},
		Settings:      []string{"setting1", "setting2", "setting3"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := pool.Marshal(testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkStandard_Marshal(b *testing.B) {
	testData := TestStruct{
		ID:            1,
		Name:          "John Doe",
		Email:         "john@example.com",
		Address:       "123 Main St, Anytown, USA",
		City:          "Anytown",
		State:         "CA",
		Zip:           "12345",
		Country:       "USA",
		Phone:         "123-456-7890",
		Age:           30,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Tags:          []string{"tag1", "tag2", "tag3"},
		Roles:         []string{"role1", "role2", "role3"},
		Permissions:   []string{"perm1", "perm2", "perm3"},
		Groups:        []string{"group1", "group2", "group3"},
		Subscriptions: []string{"sub1", "sub2", "sub3"},
		Preferences:   []string{"pref1", "pref2", "pref3"},
		Settings:      []string{"setting1", "setting2", "setting3"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkJsonPool_Unmarshal(b *testing.B) {
	pool := NewJsonPool()
	jsonData := []byte(`{"id":1,"name":"John Doe","email":"john@example.com"}`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result TestStruct
			err := pool.Unmarshal(jsonData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkStandard_Unmarshal(b *testing.B) {
	jsonData := []byte(`{"id":1,"name":"John Doe","email":"john@example.com"}`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result TestStruct
			err := json.Unmarshal(jsonData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Memory allocation benchmark
func BenchmarkJsonPool_MemoryAllocations(b *testing.B) {
	pool := NewJsonPool()
	testData := TestStruct{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := pool.Marshal(testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStandard_MemoryAllocations(b *testing.B) {
	testData := TestStruct{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}
