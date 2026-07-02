package provider_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/provider"
)

// fakeCaller is a minimal client.Caller for unit tests. It returns a valid
// TrueNAS 25.10 version string from every Call so checkServerVersion passes.
type fakeCaller struct {
	mu        sync.Mutex
	callCount int
}

func (f *fakeCaller) Call(_ context.Context, _ string, _ any) (json.RawMessage, error) {
	f.mu.Lock()
	f.callCount++
	f.mu.Unlock()
	return json.RawMessage(`"TrueNAS-25.10.0-ALPHA"`), nil
}

func (f *fakeCaller) CallWithJob(_ context.Context, _ string, _ any) (json.RawMessage, error) {
	return nil, nil
}

// dialKey uniquely identifies a set of connection credentials.
type dialKey struct {
	host               string
	apiKey             string
	username           string
	password           string
	insecureSkipVerify bool
}

// cachedDial returns a DialFunc that reuses existing connections for identical
// credential tuples. This prevents repeated logins across many acceptance tests.
// Errors are never cached so transient failures are always retried.
func cachedDial() provider.DialFunc {
	var mu sync.Mutex
	cache := make(map[dialKey]client.Caller)
	return func(host, apiKey, username, password string, insecureSkipVerify bool) (client.Caller, error) {
		key := dialKey{host, apiKey, username, password, insecureSkipVerify}
		mu.Lock()
		if c, ok := cache[key]; ok {
			mu.Unlock()
			return c, nil
		}
		mu.Unlock()

		c, err := client.NewWebSocketClient(host, apiKey, username, password, insecureSkipVerify)
		if err != nil {
			return nil, err
		}

		mu.Lock()
		cache[key] = c
		mu.Unlock()
		return c, nil
	}
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"truenas": providerserver.NewProtocol6WithError(provider.NewWithDialer("test", cachedDial())()),
}

var testAccFreshConnProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"truenas": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TRUENAS_HOST") == "" {
		t.Skip("TRUENAS_HOST environment variable not set")
	}
}

// providerConfigureRequest builds a ConfigureRequest for the given attribute values.
// Pass nil for any attribute to leave it null.
func providerConfigureRequest(host, apiKey, username, password *string) tfprovider.ConfigureRequest {
	p := provider.New("test")()

	schemaResp := &tfprovider.SchemaResponse{}
	p.Schema(context.Background(), tfprovider.SchemaRequest{}, schemaResp)

	strVal := func(s *string) tftypes.Value {
		if s == nil {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, *s)
	}

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"host":                 tftypes.String,
			"api_key":              tftypes.String,
			"username":             tftypes.String,
			"password":             tftypes.String,
			"insecure_skip_verify": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"host":                 strVal(host),
		"api_key":              strVal(apiKey),
		"username":             strVal(username),
		"password":             strVal(password),
		"insecure_skip_verify": tftypes.NewValue(tftypes.Bool, nil),
	})

	return tfprovider.ConfigureRequest{
		TerraformVersion: "1.11.0",
		Config: tfsdk.Config{
			Raw:    configVal,
			Schema: schemaResp.Schema,
		},
	}
}

func ptr(s string) *string { return &s }

func TestProvider_Configure_MissingCredentials(t *testing.T) {
	p := provider.New("test")()
	req := providerConfigureRequest(ptr("truenas.example.com"), nil, nil, nil)
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error diagnostic for missing credentials, got none")
	}
}

func TestProvider_Configure_ConflictingCredentials_APIKeyAndBoth(t *testing.T) {
	p := provider.New("test")()
	req := providerConfigureRequest(ptr("truenas.example.com"), ptr("my-api-key"), ptr("admin"), ptr("secret"))
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error diagnostic for conflicting credentials (api_key + username + password), got none")
	}
}

func TestProvider_Configure_ConflictingCredentials_APIKeyAndUsername(t *testing.T) {
	p := provider.New("test")()
	req := providerConfigureRequest(ptr("truenas.example.com"), ptr("my-api-key"), ptr("admin"), nil)
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error diagnostic for conflicting credentials (api_key + username), got none")
	}
}

func TestProvider_Configure_ConflictingCredentials_APIKeyAndPassword(t *testing.T) {
	p := provider.New("test")()
	req := providerConfigureRequest(ptr("truenas.example.com"), ptr("my-api-key"), nil, ptr("secret"))
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error diagnostic for conflicting credentials (api_key + password), got none")
	}
}

func TestProvider_Configure_IncompleteCredentials_UsernameWithoutPassword(t *testing.T) {
	p := provider.New("test")()
	req := providerConfigureRequest(ptr("truenas.example.com"), nil, ptr("admin"), nil)
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error diagnostic for username without password, got none")
	}
}

func TestProvider_Configure_IncompleteCredentials_PasswordWithoutUsername(t *testing.T) {
	p := provider.New("test")()
	req := providerConfigureRequest(ptr("truenas.example.com"), nil, nil, ptr("secret"))
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error diagnostic for password without username, got none")
	}
}

func TestProvider_Configure_MissingHost(t *testing.T) {
	p := provider.New("test")()
	req := providerConfigureRequest(nil, ptr("my-api-key"), nil, nil)
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error diagnostic for missing host, got none")
	}
}

func TestProvider_Configure_UsesInjectedDialer(t *testing.T) {
	dialCalled := false
	fake := &fakeCaller{}
	dial := func(host, apiKey, username, password string, insecureSkipVerify bool) (client.Caller, error) {
		dialCalled = true
		return fake, nil
	}

	p := provider.NewWithDialer("test", dial)()
	req := providerConfigureRequest(ptr("truenas.example.com"), ptr("my-api-key"), nil, nil)
	resp := &tfprovider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if !dialCalled {
		t.Error("expected injected dial function to be called, but it was not")
	}
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}

func TestProvider_Configure_VersionCheckRunsOncePerInstance(t *testing.T) {
	fake := &fakeCaller{}
	dial := func(host, apiKey, username, password string, insecureSkipVerify bool) (client.Caller, error) {
		return fake, nil
	}

	p := provider.NewWithDialer("test", dial)()
	req := providerConfigureRequest(ptr("truenas.example.com"), ptr("my-api-key"), nil, nil)

	for range 3 {
		resp := &tfprovider.ConfigureResponse{}
		p.Configure(context.Background(), req, resp)
	}

	fake.mu.Lock()
	count := fake.callCount
	fake.mu.Unlock()

	if count != 1 {
		t.Errorf("expected system.version to be called exactly once per provider instance, got %d", count)
	}
}

// Acceptance tests — require TF_ACC=1 and a live TrueNAS instance.

func TestAccProvider_APIKeyAuth(t *testing.T) {
	if os.Getenv("TRUENAS_API_KEY") == "" {
		t.Skip("TRUENAS_API_KEY environment variable not set")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccFreshConnProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:   providerHCL(),
				PlanOnly: true,
			},
		},
	})
}

func TestAccProvider_UsernamePasswordAuth(t *testing.T) {
	if os.Getenv("TRUENAS_USERNAME") == "" || os.Getenv("TRUENAS_PASSWORD") == "" {
		t.Skip("TRUENAS_USERNAME and TRUENAS_PASSWORD environment variables not set")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccFreshConnProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:   providerHCL(),
				PlanOnly: true,
			},
		},
	})
}

func providerHCL() string {
	return `provider "truenas" {}`
}
