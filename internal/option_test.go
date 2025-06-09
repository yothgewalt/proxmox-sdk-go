package internal

import (
	"testing"
)

// Test struct to use with the generic option pattern
type TestConfig struct {
	Name    string
	Value   int
	Enabled bool
}

// Test option implementations
type NameOption string

func (n NameOption) apply(config *TestConfig) {
	config.Name = string(n)
}

type ValueOption int

func (v ValueOption) apply(config *TestConfig) {
	config.Value = int(v)
}

type EnabledOption bool

func (e EnabledOption) apply(config *TestConfig) {
	config.Enabled = bool(e)
}

func TestOptionFunc_Apply(t *testing.T) {
	config := &TestConfig{}

	// Test OptionFunc with a simple function
	nameOption := OptionFunc[TestConfig](func(c *TestConfig) {
		c.Name = "test-name"
	})

	nameOption.apply(config)

	if config.Name != "test-name" {
		t.Errorf("Expected Name to be 'test-name', got '%s'", config.Name)
	}
}

func TestOptionFunc_MultipleFields(t *testing.T) {
	config := &TestConfig{}

	// Test OptionFunc that modifies multiple fields
	multiOption := OptionFunc[TestConfig](func(c *TestConfig) {
		c.Name = "multi-test"
		c.Value = 42
		c.Enabled = true
	})

	multiOption.apply(config)

	if config.Name != "multi-test" {
		t.Errorf("Expected Name to be 'multi-test', got '%s'", config.Name)
	}
	if config.Value != 42 {
		t.Errorf("Expected Value to be 42, got %d", config.Value)
	}
	if !config.Enabled {
		t.Error("Expected Enabled to be true, got false")
	}
}

func TestApplyOptions_EmptyOptions(t *testing.T) {
	config := &TestConfig{
		Name:    "initial",
		Value:   10,
		Enabled: false,
	}

	ApplyOptions(config)

	// Values should remain unchanged
	if config.Name != "initial" {
		t.Errorf("Expected Name to remain 'initial', got '%s'", config.Name)
	}
	if config.Value != 10 {
		t.Errorf("Expected Value to remain 10, got %d", config.Value)
	}
	if config.Enabled {
		t.Error("Expected Enabled to remain false, got true")
	}
}

func TestApplyOptions_SingleOption(t *testing.T) {
	config := &TestConfig{}

	nameOpt := NameOption("single-test")
	ApplyOptions(config, nameOpt)

	if config.Name != "single-test" {
		t.Errorf("Expected Name to be 'single-test', got '%s'", config.Name)
	}
	// Other fields should remain at zero values
	if config.Value != 0 {
		t.Errorf("Expected Value to be 0, got %d", config.Value)
	}
	if config.Enabled {
		t.Error("Expected Enabled to be false, got true")
	}
}

func TestApplyOptions_MultipleOptions(t *testing.T) {
	config := &TestConfig{}

	nameOpt := NameOption("multi-option-test")
	valueOpt := ValueOption(100)
	enabledOpt := EnabledOption(true)

	ApplyOptions(config, nameOpt, valueOpt, enabledOpt)

	if config.Name != "multi-option-test" {
		t.Errorf("Expected Name to be 'multi-option-test', got '%s'", config.Name)
	}
	if config.Value != 100 {
		t.Errorf("Expected Value to be 100, got %d", config.Value)
	}
	if !config.Enabled {
		t.Error("Expected Enabled to be true, got false")
	}
}

func TestApplyOptions_OverwriteValues(t *testing.T) {
	config := &TestConfig{
		Name:    "original",
		Value:   50,
		Enabled: true,
	}

	// Apply options that overwrite existing values
	nameOpt := NameOption("overwritten")
	valueOpt := ValueOption(200)
	enabledOpt := EnabledOption(false)

	ApplyOptions(config, nameOpt, valueOpt, enabledOpt)

	if config.Name != "overwritten" {
		t.Errorf("Expected Name to be 'overwritten', got '%s'", config.Name)
	}
	if config.Value != 200 {
		t.Errorf("Expected Value to be 200, got %d", config.Value)
	}
	if config.Enabled {
		t.Error("Expected Enabled to be false, got true")
	}
}

func TestApplyOptions_MixedOptionTypes(t *testing.T) {
	config := &TestConfig{}

	// Mix of custom option types and OptionFunc
	nameOpt := NameOption("mixed-test")
	funcOpt := OptionFunc[TestConfig](func(c *TestConfig) {
		c.Value = 999
	})
	enabledOpt := EnabledOption(true)

	ApplyOptions(config, nameOpt, funcOpt, enabledOpt)

	if config.Name != "mixed-test" {
		t.Errorf("Expected Name to be 'mixed-test', got '%s'", config.Name)
	}
	if config.Value != 999 {
		t.Errorf("Expected Value to be 999, got %d", config.Value)
	}
	if !config.Enabled {
		t.Error("Expected Enabled to be true, got false")
	}
}

func TestApplyOptions_OrderMatters(t *testing.T) {
	config := &TestConfig{}

	// Apply options that modify the same field in sequence
	firstValue := ValueOption(10)
	secondValue := ValueOption(20)
	thirdValue := ValueOption(30)

	ApplyOptions(config, firstValue, secondValue, thirdValue)

	// The last option should win
	if config.Value != 30 {
		t.Errorf("Expected Value to be 30 (last applied), got %d", config.Value)
	}
}

func TestApplyOptions_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option[TestConfig]
		expected TestConfig
	}{
		{
			name:     "no options",
			options:  []Option[TestConfig]{},
			expected: TestConfig{},
		},
		{
			name: "name only",
			options: []Option[TestConfig]{
				NameOption("test-name"),
			},
			expected: TestConfig{
				Name:    "test-name",
				Value:   0,
				Enabled: false,
			},
		},
		{
			name: "all fields",
			options: []Option[TestConfig]{
				NameOption("complete-test"),
				ValueOption(42),
				EnabledOption(true),
			},
			expected: TestConfig{
				Name:    "complete-test",
				Value:   42,
				Enabled: true,
			},
		},
		{
			name: "with function option",
			options: []Option[TestConfig]{
				NameOption("func-test"),
				OptionFunc[TestConfig](func(c *TestConfig) {
					c.Value = 123
					c.Enabled = true
				}),
			},
			expected: TestConfig{
				Name:    "func-test",
				Value:   123,
				Enabled: true,
			},
		},
		{
			name: "override values",
			options: []Option[TestConfig]{
				ValueOption(10),
				ValueOption(20), // This should override the previous value
				NameOption("override-test"),
			},
			expected: TestConfig{
				Name:    "override-test",
				Value:   20,
				Enabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TestConfig{}
			ApplyOptions(config, tt.options...)

			if config.Name != tt.expected.Name {
				t.Errorf("Expected Name to be '%s', got '%s'", tt.expected.Name, config.Name)
			}
			if config.Value != tt.expected.Value {
				t.Errorf("Expected Value to be %d, got %d", tt.expected.Value, config.Value)
			}
			if config.Enabled != tt.expected.Enabled {
				t.Errorf("Expected Enabled to be %t, got %t", tt.expected.Enabled, config.Enabled)
			}
		})
	}
}

// Test with different types to ensure generics work properly
type NumberConfig struct {
	Integer int
	Float   float64
}

type NumberIntegerOption int

func (n NumberIntegerOption) apply(config *NumberConfig) {
	config.Integer = int(n)
}

func TestApplyOptions_DifferentTypes(t *testing.T) {
	config := &NumberConfig{}

	intOpt := NumberIntegerOption(42)
	floatOpt := OptionFunc[NumberConfig](func(c *NumberConfig) {
		c.Float = 3.14
	})

	ApplyOptions(config, intOpt, floatOpt)

	if config.Integer != 42 {
		t.Errorf("Expected Integer to be 42, got %d", config.Integer)
	}
	if config.Float != 3.14 {
		t.Errorf("Expected Float to be 3.14, got %f", config.Float)
	}
}

// Benchmark tests
func BenchmarkApplyOptions_Single(b *testing.B) {
	config := &TestConfig{}
	option := NameOption("benchmark-test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyOptions(config, option)
	}
}

func BenchmarkApplyOptions_Multiple(b *testing.B) {
	config := &TestConfig{}
	options := []Option[TestConfig]{
		NameOption("benchmark-test"),
		ValueOption(42),
		EnabledOption(true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyOptions(config, options...)
	}
}

func BenchmarkOptionFunc_Apply(b *testing.B) {
	config := &TestConfig{}
	option := OptionFunc[TestConfig](func(c *TestConfig) {
		c.Name = "benchmark"
		c.Value = 100
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		option.apply(config)
	}
}
