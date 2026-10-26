package session

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/workos/workos-go/v10"
)

const (
	testClientID       = "client_test"
	testCookiePassword = "0123456789abcdef0123456789abcdef"
	testKeyID          = "key_1"
	testIssuer         = "https://api.workos.com/"
)

var (
	keyOnce  sync.Once
	testKey  *rsa.PrivateKey
	otherKey *rsa.PrivateKey
)

func keys(t *testing.T) (*rsa.PrivateKey, *rsa.PrivateKey) {
	t.Helper()
	keyOnce.Do(func() {
		var err error
		testKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		otherKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
	})
	return testKey, otherKey
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func signToken(t *testing.T, key *rsa.PrivateKey, sessionID string, expiresIn time.Duration) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": testKeyID})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": testIssuer,
		"aud": testClientID,
		"sid": sessionID,
		"exp": time.Now().Add(expiresIn).Unix(),
		"iat": time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	signingInput := b64(header) + "." + b64(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + b64(signature)
}

func seal(t *testing.T, data *workos.SessionData) string {
	t.Helper()
	sealed, err := workos.SealSession(data, testCookiePassword)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func validSession(t *testing.T, userID string) string {
	t.Helper()
	key, _ := keys(t)
	return seal(t, &workos.SessionData{
		AccessToken:  signToken(t, key, "session_1", 5*time.Minute),
		RefreshToken: "refresh_1",
		User:         &workos.User{ID: userID, Email: userID + "@example.com"},
	})
}

func expiredSession(t *testing.T, userID string) string {
	t.Helper()
	key, _ := keys(t)
	return seal(t, &workos.SessionData{
		AccessToken:  signToken(t, key, "session_1", -time.Hour),
		RefreshToken: "refresh_1",
		User:         &workos.User{ID: userID, Email: userID + "@example.com"},
	})
}

type workosServer struct {
	*httptest.Server
	mu           sync.Mutex
	jwksStatus   int
	authStatus   int
	authBody     any
	authRequests []map[string]any
}

func newWorkOSServer(t *testing.T) *workosServer {
	t.Helper()
	key, _ := keys(t)
	ws := &workosServer{jwksStatus: http.StatusOK, authStatus: http.StatusOK}

	mux := http.NewServeMux()
	mux.HandleFunc("/sso/jwks/"+testClientID, func(w http.ResponseWriter, r *http.Request) {
		ws.mu.Lock()
		status := ws.jwksStatus
		ws.mu.Unlock()
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		e := big.NewInt(int64(key.PublicKey.E)).Bytes()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"kid": testKeyID,
				"alg": "RS256",
				"use": "sig",
				"n":   b64(key.PublicKey.N.Bytes()),
				"e":   b64(e),
			}},
		})
	})
	mux.HandleFunc("/user_management/authenticate", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ws.mu.Lock()
		ws.authRequests = append(ws.authRequests, body)
		status := ws.authStatus
		resp := ws.authBody
		ws.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	})

	ws.Server = httptest.NewServer(mux)
	t.Cleanup(ws.Close)
	return ws
}

func (ws *workosServer) setJWKSStatus(status int) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.jwksStatus = status
}

func (ws *workosServer) setAuthResponse(status int, body any) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.authStatus = status
	ws.authBody = body
}

func (ws *workosServer) requests() []map[string]any {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return append([]map[string]any(nil), ws.authRequests...)
}

func (ws *workosServer) client() *workos.Client {
	return workos.NewClient(
		"sk_test",
		workos.WithBaseURL(ws.URL),
		workos.WithClientID(testClientID),
		workos.WithMaxRetries(0),
	)
}

func authResponse(accessToken, refreshToken, userID string) map[string]any {
	return map[string]any{
		"user": map[string]any{
			"id":    userID,
			"email": userID + "@example.com",
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}
}
