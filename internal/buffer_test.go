package internal

import (
	"bytes"
	"strings"
	"testing"
)

func TestBufferPool_Basic(t *testing.T) {
	pool := NewBufferPool(1024)

	buf := pool.Get()
	if buf == nil {
		t.Fatal("Expected buffer from pool, got nil")
	}

	if buf.Cap() != 1024 {
		t.Errorf("Expected buffer capacity 1024, got %d", buf.Cap())
	}

	// Use the buffer
	buf.WriteString("test data")
	if buf.String() != "test data" {
		t.Errorf("Expected 'test data', got '%s'", buf.String())
	}

	// Return to pool
	pool.Put(buf)

	// Get another buffer (should be the same one, reset)
	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Errorf("Expected empty buffer after reset, got length %d", buf2.Len())
	}

	pool.Put(buf2)
}

func TestBufferPool_GetBytes(t *testing.T) {
	pool := NewBufferPool(512)

	result := pool.GetBytes(func(buf *bytes.Buffer) {
		buf.WriteString("hello")
		buf.WriteString(" ")
		buf.WriteString("world")
	})

	expected := "hello world"
	if string(result) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(result))
	}
}

func TestByteSlicePool_Basic(t *testing.T) {
	pool := NewByteSlicePool(4096)

	slice := pool.Get()
	if slice == nil {
		t.Fatal("Expected slice from pool, got nil")
	}

	if len(slice) != 4096 {
		t.Errorf("Expected slice length 4096, got %d", len(slice))
	}

	if cap(slice) < 4096 {
		t.Errorf("Expected slice capacity >= 4096, got %d", cap(slice))
	}

	// Use the slice
	copy(slice, []byte("test data"))

	// Return to pool
	pool.Put(slice)

	// Get another slice
	slice2 := pool.Get()
	if len(slice2) != 4096 {
		t.Errorf("Expected slice length 4096 after reuse, got %d", len(slice2))
	}

	pool.Put(slice2)
}

func TestRequestBufferManager_JSONBuffer(t *testing.T) {
	manager := NewRequestBufferManager()

	buf := manager.GetJSONBuffer()
	if buf == nil {
		t.Fatal("Expected JSON buffer, got nil")
	}

	// Test JSON construction
	buf.WriteString(`{"key": "value", "number": 123}`)

	jsonData := buf.String()
	if !strings.Contains(jsonData, "key") || !strings.Contains(jsonData, "value") {
		t.Errorf("Expected valid JSON data, got '%s'", jsonData)
	}

	manager.PutJSONBuffer(buf)
}

func TestRequestBufferManager_HeaderBuffer(t *testing.T) {
	manager := NewRequestBufferManager()

	buf := manager.GetHeaderBuffer()
	if buf == nil {
		t.Fatal("Expected header buffer, got nil")
	}

	// Test header construction
	buf.WriteString("Authorization: Bearer token123")

	headerData := buf.String()
	if !strings.Contains(headerData, "Authorization") {
		t.Errorf("Expected header data, got '%s'", headerData)
	}

	manager.PutHeaderBuffer(buf)
}

func TestRequestBufferManager_QueryBuffer(t *testing.T) {
	manager := NewRequestBufferManager()

	buf := manager.GetQueryBuffer()
	if buf == nil {
		t.Fatal("Expected query buffer, got nil")
	}

	// Test query parameter construction
	buf.WriteString("param1=value1&param2=value2")

	queryData := buf.String()
	if !strings.Contains(queryData, "param1=value1") {
		t.Errorf("Expected query data, got '%s'", queryData)
	}

	manager.PutQueryBuffer(buf)
}

func TestRequestBufferManager_ReadBuffer(t *testing.T) {
	manager := NewRequestBufferManager()

	buf := manager.GetReadBuffer()
	if buf == nil {
		t.Fatal("Expected read buffer, got nil")
	}

	if len(buf) != 8192 {
		t.Errorf("Expected read buffer length 8192, got %d", len(buf))
	}

	// Test using the buffer for reading
	testData := []byte("test data for reading")
	copy(buf, testData)

	if !bytes.Equal(buf[:len(testData)], testData) {
		t.Errorf("Expected copied data to match")
	}

	manager.PutReadBuffer(buf)
}

func TestRequestBufferManager_AllBuffers(t *testing.T) {
	manager := NewRequestBufferManager()

	// Test that we can get all buffer types
	jsonBuf := manager.GetJSONBuffer()
	headerBuf := manager.GetHeaderBuffer()
	queryBuf := manager.GetQueryBuffer()
	readBuf := manager.GetReadBuffer()

	if jsonBuf == nil || headerBuf == nil || queryBuf == nil || readBuf == nil {
		t.Fatal("One or more buffers were nil")
	}

	// Use all buffers
	jsonBuf.WriteString(`{"test": true}`)
	headerBuf.WriteString("Content-Type: application/json")
	queryBuf.WriteString("search=test")
	copy(readBuf, []byte("response data"))

	// Return all buffers
	manager.PutJSONBuffer(jsonBuf)
	manager.PutHeaderBuffer(headerBuf)
	manager.PutQueryBuffer(queryBuf)
	manager.PutReadBuffer(readBuf)

	// Get them again to test reuse
	jsonBuf2 := manager.GetJSONBuffer()
	headerBuf2 := manager.GetHeaderBuffer()
	queryBuf2 := manager.GetQueryBuffer()
	readBuf2 := manager.GetReadBuffer()

	// Should be reset/empty
	if jsonBuf2.Len() != 0 || headerBuf2.Len() != 0 || queryBuf2.Len() != 0 {
		t.Errorf("Expected buffers to be reset after reuse")
	}

	manager.PutJSONBuffer(jsonBuf2)
	manager.PutHeaderBuffer(headerBuf2)
	manager.PutQueryBuffer(queryBuf2)
	manager.PutReadBuffer(readBuf2)
}

func BenchmarkBufferPool_GetPut(b *testing.B) {
	pool := NewBufferPool(1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		buf.WriteString("benchmark test data")
		pool.Put(buf)
	}
}

func BenchmarkByteSlicePool_GetPut(b *testing.B) {
	pool := NewByteSlicePool(4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := pool.Get()
		copy(slice, []byte("benchmark test data"))
		pool.Put(slice)
	}
}

func BenchmarkRequestBufferManager_JSON(b *testing.B) {
	manager := NewRequestBufferManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := manager.GetJSONBuffer()
		buf.WriteString(`{"benchmark": true, "iteration": `)
		buf.WriteString("123")
		buf.WriteString(`}`)
		manager.PutJSONBuffer(buf)
	}
}

func BenchmarkDirectBufferAllocation(b *testing.B) {
	// Comparison benchmark - direct allocation without pooling
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 0, 2048))
		buf.WriteString(`{"benchmark": true, "iteration": `)
		buf.WriteString("123")
		buf.WriteString(`}`)
		// No put back to pool - just GC
	}
}
