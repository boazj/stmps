package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetResponse(t *testing.T) {
	testCases := []struct {
		name         string
		serverStatus int
		serverBody   string
		expectError  bool
		caller       string
	}{
		{
			name:         "Success",
			serverStatus: http.StatusOK,
			serverBody:   `{"Response": {"Success": true}}`,
			expectError:  false,
			caller:       "TestCaller",
		},
		{
			name:         "Non-200 Status Code",
			serverStatus: http.StatusBadRequest,
			serverBody:   `{"Response": {"Success": false}}`,
			expectError:  true,
			caller:       "TestCaller",
		},
		{
			name:         "Invalid JSON Response",
			serverStatus: http.StatusOK,
			serverBody:   `{"Response": {"Success": `,
			expectError:  true,
			caller:       "TestCaller",
		},
		{
			name:         "Empty Caller",
			serverStatus: http.StatusOK,
			serverBody:   `{"Response": {"Success": true}}`,
			expectError:  false,
			caller:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server to simulate the HTTP response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverStatus)
				if _, err := w.Write([]byte(tc.serverBody)); err != nil {
					t.Fatalf("failed to write server response: %v", err)
				}
			}))
			defer server.Close()

			// Create an instance of SubsonicConnection
			connection := &SubsonicConnection{}

			// Call the function
			response, err := connection.getResponse(server.URL)

			// Validate the results
			if tc.expectError {
				if err == nil {
					t.Errorf("expected an error but got none")
				} else if !containsCallerInError(err, tc.caller) {
					t.Errorf("expected error to contain caller [%s], but got: %v", tc.caller, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
				if response == nil {
					t.Errorf("expected a response but got nil")
				}
			}
		})
	}
}

// Helper function to check if the error contains the caller
func containsCallerInError(err error, caller string) bool {
	return err != nil && (caller == "" || strings.Contains(err.Error(), "["+caller+"]"))
}
