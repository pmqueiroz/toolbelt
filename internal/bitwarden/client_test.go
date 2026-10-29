package bitwarden

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	sdkOrganizationID = "f4e44a7f-1190-432a-9d4a-af96013127cb"
	sdkPayload        = "2.E9fE8+M/VWMfhhim1KlCbQ==|eLsHR484S/tJbIkM6spnG/HP65tj9A6Tba7kAAvUp+rYuQmGLixiOCfMsqt5OvBctDfvvr/AesBu7cZimPLyOEhqEAjn52jF0eaI38XZfeOG2VJl0LOf60Wkfh3ryAMvfvLj3G4ZCNYU8sNgoC2+IQ==|lNApuCQ4Pyakfo/wwuuajWNaEX/2MW8/3rjXB/V7n+k="
	sdkOrgKey         = "k/6PcwG7Hm/eZfvvOvP6EqGx1JKgbzYrWwpOIHsxbJHQMpIMg5Ud94AQHduSR+XMMaFiSB+nszbO4JPXj04YwA=="
	projectID         = "11111111-1111-1111-1111-111111111111"
)

type fakeVault struct {
	t       *testing.T
	key     Key
	secrets map[string]secretModel
	order   []string
	next    int
}

func newFakeVault(t *testing.T) (*fakeVault, *httptest.Server) {
	raw, _ := base64.StdEncoding.DecodeString(sdkOrgKey)
	key, err := KeyFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	vault := &fakeVault{t: t, key: key, secrets: map[string]secretModel{}}
	server := httptest.NewServer(http.HandlerFunc(vault.serve))
	t.Cleanup(server.Close)
	return vault, server
}

func (f *fakeVault) enc(plaintext string) string {
	enc, err := Encrypt(f.key, []byte(plaintext))
	if err != nil {
		f.t.Fatal(err)
	}
	return enc
}

func (f *fakeVault) add(key, value, note string) string {
	f.next++
	id := strings.Repeat(string(rune('a'+f.next)), 8) + "-0000-0000-0000-000000000000"
	f.secrets[id] = secretModel{ID: id, Key: f.enc(key), Value: f.enc(value), Note: f.enc(note)}
	f.order = append(f.order, id)
	return id
}

func fakeJWT(claims string) string {
	encode := base64.RawURLEncoding.EncodeToString
	return encode([]byte(`{"alg":"none"}`)) + "." + encode([]byte(claims)) + ".sig"
}

func (f *fakeVault) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Device-Type") != deviceTypeSDK {
		http.Error(w, "missing device type", http.StatusBadRequest)
		return
	}
	if r.URL.Path == "/identity/connect/token" {
		r.ParseForm()
		if r.Form.Get("client_secret") != "C2IgxjjLF7qSshsbwe8JGcbM075YXw" || r.Form.Get("scope") != "api.secrets" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":      fakeJWT(`{"organization":"` + sdkOrganizationID + `"}`),
			"encrypted_payload": sdkPayload,
		})
		return
	}
	if r.Header.Get("Authorization") == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/organizations/"+sdkOrganizationID+"/projects":
		json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": projectID, "name": f.enc("personal")}}})
	case r.Method == http.MethodGet && r.URL.Path == "/api/projects/"+projectID+"/secrets":
		list := []map[string]string{}
		for _, id := range f.order {
			list = append(list, map[string]string{"id": id})
		}
		json.NewEncoder(w).Encode(map[string]any{"secrets": list})
	case r.Method == http.MethodPost && r.URL.Path == "/api/secrets/get-by-ids":
		var body struct{ IDs []string }
		json.NewDecoder(r.Body).Decode(&body)
		data := []secretModel{}
		for _, id := range body.IDs {
			data = append(data, f.secrets[id])
		}
		json.NewEncoder(w).Encode(map[string]any{"data": data})
	case r.Method == http.MethodPost && r.URL.Path == "/api/organizations/"+sdkOrganizationID+"/secrets":
		var body secretRequest
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.ProjectIDs) != 1 || body.ProjectIDs[0] != projectID {
			http.Error(w, "missing project", http.StatusBadRequest)
			return
		}
		id := "created0-0000-0000-0000-000000000000"
		model := secretModel{ID: id, Key: body.Key, Value: body.Value, Note: body.Note}
		f.secrets[id] = model
		f.order = append(f.order, id)
		json.NewEncoder(w).Encode(model)
	case strings.HasPrefix(r.URL.Path, "/api/secrets/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/secrets/")
		current, ok := f.secrets[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPut {
			var body secretRequest
			json.NewDecoder(r.Body).Decode(&body)
			if body.ValueChanged == nil || !*body.ValueChanged {
				http.Error(w, "valueChanged not set", http.StatusBadRequest)
				return
			}
			current = secretModel{ID: id, Key: body.Key, Value: body.Value, Note: body.Note}
			f.secrets[id] = current
		}
		json.NewEncoder(w).Encode(current)
	default:
		http.NotFound(w, r)
	}
}

func login(t *testing.T, server *httptest.Server) *Session {
	token, err := ParseAccessToken(sdkTestToken)
	if err != nil {
		t.Fatal(err)
	}
	endpoints := Endpoints{Identity: server.URL + "/identity", API: server.URL + "/api"}
	session, err := Login(context.Background(), endpoints, server.Client(), token)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestLoginUnwrapsTheOrganizationKey(t *testing.T) {
	_, server := newFakeVault(t)
	session := login(t, server)
	if session.OrganizationID != sdkOrganizationID {
		t.Fatalf("organization = %s", session.OrganizationID)
	}
	if base64.StdEncoding.EncodeToString(session.orgKey[:]) != sdkOrgKey {
		t.Fatal("organization key was not unwrapped")
	}
}

func TestLoginReportsRejectedCredentials(t *testing.T) {
	_, server := newFakeVault(t)
	token, _ := ParseAccessToken(sdkTestToken)
	token.ClientSecret = "wrong"
	_, err := Login(context.Background(), Endpoints{Identity: server.URL + "/identity"}, server.Client(), token)
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("Login with a wrong secret = %v; want a 400 APIError", err)
	}
}

func TestProjectsDecryptsNames(t *testing.T) {
	_, server := newFakeVault(t)
	projects, err := login(t, server).Projects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0] != (Project{ID: projectID, Name: "personal"}) {
		t.Fatalf("projects = %+v", projects)
	}
}

func TestSecretsListsAndDecryptsAProject(t *testing.T) {
	vault, server := newFakeVault(t)
	vault.add("GITHUB_TOKEN", "ghp_123", "")
	vault.add("NPM_TOKEN", "npm_456", "")
	secrets, err := login(t, server).Secrets(context.Background(), projectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets) != 2 || secrets[0].Key != "GITHUB_TOKEN" || secrets[1].Value != "npm_456" {
		t.Fatalf("secrets = %+v", secrets)
	}
}

func TestSecretsOfAnEmptyProject(t *testing.T) {
	_, server := newFakeVault(t)
	secrets, err := login(t, server).Secrets(context.Background(), projectID)
	if err != nil || len(secrets) != 0 {
		t.Fatalf("secrets = %+v, %v", secrets, err)
	}
}

func TestCreateEncryptsForTheOrganization(t *testing.T) {
	vault, server := newFakeVault(t)
	created, err := login(t, server).Create(context.Background(), projectID, " API_KEY ", "v1")
	if err != nil {
		t.Fatal(err)
	}
	stored := vault.secrets[created.ID]
	if key, _ := DecryptString(vault.key, stored.Key); key != "API_KEY" {
		t.Fatalf("stored key = %q", key)
	}
	if created.Key != "API_KEY" || created.Value != "v1" {
		t.Fatalf("created = %+v", created)
	}
}

func TestUpdateKeepsTheNote(t *testing.T) {
	vault, server := newFakeVault(t)
	id := vault.add("API_KEY", "v1", "rotated monthly")
	updated, err := login(t, server).Update(context.Background(), projectID, Secret{ID: id, Key: "API_KEY", Value: "v2"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Value != "v2" {
		t.Fatalf("updated = %+v", updated)
	}
	if note, _ := DecryptString(vault.key, vault.secrets[id].Note); note != "rotated monthly" {
		t.Fatalf("note = %q", note)
	}
}

func TestRegion(t *testing.T) {
	if endpoints, err := Region("eu"); err != nil || endpoints.API != "https://api.bitwarden.eu" {
		t.Fatalf("Region(eu) = %+v, %v", endpoints, err)
	}
	if _, err := Region("mars"); err == nil {
		t.Fatal("Region(mars) accepted an unknown region")
	}
}
