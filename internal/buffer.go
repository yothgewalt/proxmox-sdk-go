package internal

import (
	"bytes"
	"sync"
)

// BufferPool manages a pool of reusable byte buffers for memory efficiency
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool with specified initial capacity
func NewBufferPool(initialCapacity int) *BufferPool {
	if initialCapacity <= 0 {
		initialCapacity = 1024 // Default 1KB buffers
	}

	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, initialCapacity))
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.pool.Get().(*bytes.Buffer)
}

// Put returns a buffer to the pool after resetting it
func (bp *BufferPool) Put(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bp.pool.Put(buf)
	}
}

// GetBytes retrieves a buffer, executes the function, and returns the bytes
func (bp *BufferPool) GetBytes(fn func(*bytes.Buffer)) []byte {
	buf := bp.Get()
	defer bp.Put(buf)

	fn(buf)

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())

	return result
}

// ByteSlicePool manages a pool of reusable byte slices
type ByteSlicePool struct {
	pool sync.Pool
	size int
}

// NewByteSlicePool creates a new byte slice pool with specified slice size
func NewByteSlicePool(size int) *ByteSlicePool {
	if size <= 0 {
		size = 4096 // Default 4KB slices
	}

	return &ByteSlicePool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get retrieves a byte slice from the pool
func (bsp *ByteSlicePool) Get() []byte {
	return bsp.pool.Get().([]byte)
}

// Put returns a byte slice to the pool
func (bsp *ByteSlicePool) Put(slice []byte) {
	if slice != nil && cap(slice) >= bsp.size {
		slice = slice[:bsp.size]
		bsp.pool.Put(slice)
	}
}

// RequestBufferManager handles buffer management specifically for HTTP requests
type RequestBufferManager struct {
	// Buffer pools for different use cases
	jsonBufferPool   *BufferPool    // For JSON request/response bodies
	headerBufferPool *BufferPool    // For header construction
	queryBufferPool  *BufferPool    // For query parameter building
	readBufferPool   *ByteSlicePool // For reading response bodies
}

// NewRequestBufferManager creates a new request buffer manager optimized for Proxmox API
func NewRequestBufferManager() *RequestBufferManager {
	return &RequestBufferManager{
		// Proxmox API typically has small to medium JSON payloads
		jsonBufferPool:   NewBufferPool(2048),    // 2KB for JSON
		headerBufferPool: NewBufferPool(512),     // 512B for headers
		queryBufferPool:  NewBufferPool(1024),    // 1KB for query params
		readBufferPool:   NewByteSlicePool(8192), // 8KB read buffer
	}
}

// GetJSONBuffer retrieves a buffer for JSON operations
func (rbm *RequestBufferManager) GetJSONBuffer() *bytes.Buffer {
	return rbm.jsonBufferPool.Get()
}

// PutJSONBuffer returns a JSON buffer to the pool
func (rbm *RequestBufferManager) PutJSONBuffer(buf *bytes.Buffer) {
	rbm.jsonBufferPool.Put(buf)
}

// GetHeaderBuffer retrieves a buffer for header construction
func (rbm *RequestBufferManager) GetHeaderBuffer() *bytes.Buffer {
	return rbm.headerBufferPool.Get()
}

// PutHeaderBuffer returns a header buffer to the pool
func (rbm *RequestBufferManager) PutHeaderBuffer(buf *bytes.Buffer) {
	rbm.headerBufferPool.Put(buf)
}

// GetQueryBuffer retrieves a buffer for query parameter building
func (rbm *RequestBufferManager) GetQueryBuffer() *bytes.Buffer {
	return rbm.queryBufferPool.Get()
}

// PutQueryBuffer returns a query buffer to the pool
func (rbm *RequestBufferManager) PutQueryBuffer(buf *bytes.Buffer) {
	rbm.queryBufferPool.Put(buf)
}

// GetReadBuffer retrieves a byte slice for reading operations
func (rbm *RequestBufferManager) GetReadBuffer() []byte {
	return rbm.readBufferPool.Get()
}

// PutReadBuffer returns a read buffer to the pool
func (rbm *RequestBufferManager) PutReadBuffer(buf []byte) {
	rbm.readBufferPool.Put(buf)
}
