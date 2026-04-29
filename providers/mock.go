package providers

// MockProvider is a test double that returns a fixed response for every call.
type MockProvider struct {
	Response string
	Err      error
}

func (m *MockProvider) Complete(_, _ string) (string, error) {
	return m.Response, m.Err
}

func (m *MockProvider) Embed(_ string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, m.Err
}
