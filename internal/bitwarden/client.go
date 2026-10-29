package bitwarden

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	deviceTypeSDK = "21"
	userAgent     = "toolbelt"
)

type Endpoints struct {
	Identity string
	API      string
}

var regions = map[string]Endpoints{
	"us": {Identity: "https://identity.bitwarden.com", API: "https://api.bitwarden.com"},
	"eu": {Identity: "https://identity.bitwarden.eu", API: "https://api.bitwarden.eu"},
}

func Region(name string) (Endpoints, error) {
	endpoints, ok := regions[name]
	if !ok {
		return Endpoints{}, fmt.Errorf("unknown region %q, expected us or eu", name)
	}
	return endpoints, nil
}

type Project struct {
	ID   string
	Name string
}

type Secret struct {
	ID    string
	Key   string
	Value string
}

type Session struct {
	OrganizationID string
	endpoints      Endpoints
	http           *http.Client
	bearer         string
	orgKey         Key
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bitwarden answered %d: %s", e.Status, e.Body)
}

func Login(ctx context.Context, endpoints Endpoints, client *http.Client, token AccessToken) (*Session, error) {
	form := url.Values{
		"scope":         {"api.secrets"},
		"client_id":     {token.ID},
		"client_secret": {token.ClientSecret},
		"grant_type":    {"client_credentials"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoints.Identity+"/connect/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	var res struct {
		AccessToken      string `json:"access_token"`
		EncryptedPayload string `json:"encrypted_payload"`
	}
	if err := do(client, req, &res); err != nil {
		return nil, err
	}
	payload, err := Decrypt(token.Key, res.EncryptedPayload)
	if err != nil {
		return nil, fmt.Errorf("decrypt login payload: %w", err)
	}
	var keyPayload struct {
		EncryptionKey string `json:"encryptionKey"`
	}
	if err := json.Unmarshal(payload, &keyPayload); err != nil {
		return nil, fmt.Errorf("decode login payload: %w", err)
	}
	rawKey, err := base64.StdEncoding.DecodeString(keyPayload.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode organization key: %w", err)
	}
	orgKey, err := KeyFromBytes(rawKey)
	if err != nil {
		return nil, err
	}
	orgID, err := organizationClaim(res.AccessToken)
	if err != nil {
		return nil, err
	}
	return &Session{OrganizationID: orgID, endpoints: endpoints, http: client, bearer: res.AccessToken, orgKey: orgKey}, nil
}

func organizationClaim(jwt string) (string, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("unexpected access token format")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return "", fmt.Errorf("decode access token claims: %w", err)
	}
	var claims struct {
		Organization string `json:"organization"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Organization == "" {
		return "", fmt.Errorf("access token has no organization claim")
	}
	return claims.Organization, nil
}

func (s *Session) Projects(ctx context.Context) ([]Project, error) {
	var res struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := s.call(ctx, http.MethodGet, "/organizations/"+s.OrganizationID+"/projects", nil, &res); err != nil {
		return nil, err
	}
	projects := make([]Project, 0, len(res.Data))
	for _, p := range res.Data {
		name, err := DecryptString(s.orgKey, p.Name)
		if err != nil {
			return nil, fmt.Errorf("decrypt project %s: %w", p.ID, err)
		}
		projects = append(projects, Project{ID: p.ID, Name: name})
	}
	return projects, nil
}

type secretModel struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Note  string `json:"note"`
}

type secretRequest struct {
	Key          string   `json:"key"`
	Value        string   `json:"value"`
	Note         string   `json:"note"`
	ProjectIDs   []string `json:"projectIds"`
	ValueChanged *bool    `json:"valueChanged,omitempty"`
}

func (s *Session) Secrets(ctx context.Context, projectID string) ([]Secret, error) {
	var list struct {
		Secrets []struct {
			ID string `json:"id"`
		} `json:"secrets"`
	}
	if err := s.call(ctx, http.MethodGet, "/projects/"+projectID+"/secrets", nil, &list); err != nil {
		return nil, err
	}
	if len(list.Secrets) == 0 {
		return nil, nil
	}
	ids := make([]string, len(list.Secrets))
	for i, secret := range list.Secrets {
		ids[i] = secret.ID
	}
	var full struct {
		Data []secretModel `json:"data"`
	}
	if err := s.call(ctx, http.MethodPost, "/secrets/get-by-ids", map[string][]string{"ids": ids}, &full); err != nil {
		return nil, err
	}
	secrets := make([]Secret, 0, len(full.Data))
	for _, model := range full.Data {
		secret, err := s.decryptSecret(model)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

func (s *Session) Create(ctx context.Context, projectID, key, value string) (Secret, error) {
	body, err := s.encryptRequest(projectID, key, value, "")
	if err != nil {
		return Secret{}, err
	}
	var res secretModel
	if err := s.call(ctx, http.MethodPost, "/organizations/"+s.OrganizationID+"/secrets", body, &res); err != nil {
		return Secret{}, err
	}
	return s.decryptSecret(res)
}

func (s *Session) Update(ctx context.Context, projectID string, secret Secret) (Secret, error) {
	var current secretModel
	if err := s.call(ctx, http.MethodGet, "/secrets/"+secret.ID, nil, &current); err != nil {
		return Secret{}, err
	}
	note, err := DecryptString(s.orgKey, current.Note)
	if err != nil {
		return Secret{}, fmt.Errorf("decrypt note of %s: %w", secret.ID, err)
	}
	body, err := s.encryptRequest(projectID, secret.Key, secret.Value, note)
	if err != nil {
		return Secret{}, err
	}
	changed := true
	body.ValueChanged = &changed
	var res secretModel
	if err := s.call(ctx, http.MethodPut, "/secrets/"+secret.ID, body, &res); err != nil {
		return Secret{}, err
	}
	return s.decryptSecret(res)
}

func (s *Session) encryptRequest(projectID, key, value, note string) (secretRequest, error) {
	fields := []string{strings.TrimSpace(key), value, strings.TrimSpace(note)}
	encrypted := make([]string, len(fields))
	for i, field := range fields {
		enc, err := Encrypt(s.orgKey, []byte(field))
		if err != nil {
			return secretRequest{}, err
		}
		encrypted[i] = enc
	}
	return secretRequest{Key: encrypted[0], Value: encrypted[1], Note: encrypted[2], ProjectIDs: []string{projectID}}, nil
}

func (s *Session) decryptSecret(model secretModel) (Secret, error) {
	key, err := DecryptString(s.orgKey, model.Key)
	if err != nil {
		return Secret{}, fmt.Errorf("decrypt key of %s: %w", model.ID, err)
	}
	value, err := DecryptString(s.orgKey, model.Value)
	if err != nil {
		return Secret{}, fmt.Errorf("decrypt value of %s: %w", model.ID, err)
	}
	return Secret{ID: model.ID, Key: key, Value: value}, nil
}

func (s *Session) call(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.endpoints.API+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.bearer)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return do(s.http, req, out)
}

func do(client *http.Client, req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Device-Type", deviceTypeSDK)
	req.Header.Set("User-Agent", userAgent)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return &APIError{Status: res.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	return json.Unmarshal(raw, out)
}
