package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test structs for generic functionality
type TestRequest struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type TestResponse struct {
	Status string `json:"status"`
	Data   string `json:"data"`
	ID     int    `json:"id"`
}

func TestJSONRequestEncoder(t *testing.T) {
	encoder := JSONRequestEncoder[TestRequest]{}

	testData := TestRequest{
		Message: "hello world",
		Count:   42,
	}

	data, err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Expected no error encoding, got %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected encoded data, got empty bytes")
	}
}

func TestJSONResponseDecoder(t *testing.T) {
	decoder := JSONResponseDecoder[TestResponse]{}

	jsonData := `{"status": "success", "data": "test data", "id": 123}`

	result, err := decoder.Decode([]byte(jsonData))
	if err != nil {
		t.Fatalf("Expected no error decoding, got %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
	}

	if result.Data != "test data" {
		t.Errorf("Expected data 'test data', got '%s'", result.Data)
	}

	if result.ID != 123 {
		t.Errorf("Expected ID 123, got %d", result.ID)
	}
}

func TestMarshalToJSON(t *testing.T) {
	testData := TestRequest{
		Message: "marshal test",
		Count:   42,
	}

	data, err := MarshalToJSON(testData)
	if err != nil {
		t.Fatalf("Expected no error marshaling, got %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected marshaled data, got empty bytes")
	}
}

func TestUnmarshalFromJSON(t *testing.T) {
	jsonData := `{"message": "unmarshal test", "count": 84}`

	result, err := UnmarshalFromJSON[TestRequest]([]byte(jsonData))
	if err != nil {
		t.Fatalf("Expected no error unmarshaling, got %v", err)
	}

	if result.Message != "unmarshal test" {
		t.Errorf("Expected message 'unmarshal test', got '%s'", result.Message)
	}

	if result.Count != 84 {
		t.Errorf("Expected count 84, got %d", result.Count)
	}
}

func TestMakeJSONRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "data": "json response", "id": 789}`))
	}))
	defer server.Close()

	instance := NewHttpInstance()

	requestData := TestRequest{
		Message: "json test",
		Count:   200,
	}

	response := MakeJSONRequest[TestRequest, TestResponse](
		instance,
		POST,
		server.URL+"/api2/json/test",
		requestData,
		nil,
	)

	if response.Error != nil {
		t.Fatalf("Expected no error, got %v", response.Error)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}

	if response.Data.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response.Data.Status)
	}

	if response.Data.ID != 789 {
		t.Errorf("Expected ID 789, got %d", response.Data.ID)
	}
}

func TestMakeTypedRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "data": "response data", "id": 456}`))
	}))
	defer server.Close()

	instance := NewHttpInstance()

	config := TypedRequest[TestRequest]{
		Method: POST,
		URL:    server.URL + "/api2/json/test",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		Data: TestRequest{
			Message: "test request",
			Count:   100,
		},
		Timeout: 5 * time.Second,
	}

	encoder := JSONRequestEncoder[TestRequest]{}
	decoder := JSONResponseDecoder[TestResponse]{}

	response := MakeTypedRequest[TestRequest, TestResponse](instance, config, encoder, decoder)

	if response.Error != nil {
		t.Fatalf("Expected no error, got %v", response.Error)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
	}

	if response.Data.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Data.Status)
	}

	if response.Data.Data != "response data" {
		t.Errorf("Expected data 'response data', got '%s'", response.Data.Data)
	}

	if response.Data.ID != 456 {
		t.Errorf("Expected ID 456, got %d", response.Data.ID)
	}
}

func TestConvertJSONResponse(t *testing.T) {
	// Create a regular response
	jsonData := `{"status": "converted", "data": "converted data", "id": 777}`

	regularResponse := &Response{
		StatusCode: http.StatusOK,
		Body:       []byte(jsonData),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Error:      nil,
	}

	typedResponse := ConvertJSONResponse[TestResponse](regularResponse)

	if typedResponse.Error != nil {
		t.Fatalf("Expected no error converting response, got %v", typedResponse.Error)
	}

	if typedResponse.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", typedResponse.StatusCode)
	}

	if typedResponse.Data.Status != "converted" {
		t.Errorf("Expected status 'converted', got '%s'", typedResponse.Data.Status)
	}

	if typedResponse.Data.ID != 777 {
		t.Errorf("Expected ID 777, got %d", typedResponse.Data.ID)
	}

	if len(typedResponse.RawBody) == 0 {
		t.Error("Expected raw body to be preserved")
	}
}
