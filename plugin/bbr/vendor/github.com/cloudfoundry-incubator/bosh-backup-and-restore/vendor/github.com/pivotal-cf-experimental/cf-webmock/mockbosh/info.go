package mockbosh

import "github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

type infoMock struct {
	*mockhttp.MockHttp
}

func Info() *infoMock {
	return &infoMock{
		MockHttp: mockhttp.NewMockedHttpRequest("GET", "/info").SkipAuthentication(),
	}
}

func (m *infoMock) RespondsWithSufficientAPIVersion() *mockhttp.MockHttp {
	return m.RespondsWith(`{"version":"1.3262.0.0 (00000000)"}`)
}

func (m *infoMock) WithAuthTypeBasic() *mockhttp.MockHttp {
	return m.RespondsWithJson(map[string]interface{}{"user_authentication": map[string]string{"type": "basic"}})
}

func (m *infoMock) WithAuthTypeUAA(url string) *mockhttp.MockHttp {
	return m.RespondsWithJson(map[string]interface{}{"user_authentication": map[string]interface{}{
		"type": "uaa", "options": map[string]string{"url": url},
	}})
}
