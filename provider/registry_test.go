package provider

import (
	"context"
	"testing"
)

func TestParseModel_ProviderSlashModel(t *testing.T) {
	prov, model, err := ParseModel("xai/grok-imagine-image")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "xai" {
		t.Errorf("provider = %q, want xai", prov)
	}
	if model != "grok-imagine-image" {
		t.Errorf("model = %q, want grok-imagine-image", model)
	}
}

func TestParseModel_ProviderSlashAlias(t *testing.T) {
	prov, model, err := ParseModel("google/flash")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "google" {
		t.Errorf("provider = %q, want google", prov)
	}
	if model != "gemini-3.1-flash-image-preview" {
		t.Errorf("model = %q", model)
	}
}

func TestParseModel_ProviderSlashUnknownPassthrough(t *testing.T) {
	prov, model, err := ParseModel("xai/some-future-model")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "xai" {
		t.Errorf("provider = %q, want xai", prov)
	}
	if model != "some-future-model" {
		t.Errorf("model = %q, want some-future-model", model)
	}
}

func TestParseModel_BareAlias_Flash(t *testing.T) {
	prov, model, err := ParseModel("flash")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "google" {
		t.Errorf("provider = %q, want google", prov)
	}
	if model != "gemini-3.1-flash-image-preview" {
		t.Errorf("model = %q", model)
	}
}

func TestParseModel_BareAlias_Grok(t *testing.T) {
	prov, model, err := ParseModel("grok")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "xai" {
		t.Errorf("provider = %q, want xai", prov)
	}
	if model != "grok-imagine-image" {
		t.Errorf("model = %q", model)
	}
}

func TestParseModel_BareAlias_Nb2(t *testing.T) {
	prov, model, err := ParseModel("nb2")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "google" {
		t.Errorf("provider = %q", prov)
	}
	if model != "gemini-3.1-flash-image-preview" {
		t.Errorf("model = %q", model)
	}
}

func TestParseModel_BareAlias_NbPro(t *testing.T) {
	prov, model, err := ParseModel("nb-pro")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "google" {
		t.Errorf("provider = %q", prov)
	}
	if model != "gemini-3-pro-image-preview" {
		t.Errorf("model = %q", model)
	}
}

func TestParseModel_BareAlias_GrokImagine(t *testing.T) {
	prov, model, err := ParseModel("grok-imagine")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "xai" {
		t.Errorf("provider = %q, want xai", prov)
	}
	if model != "grok-imagine-image" {
		t.Errorf("model = %q", model)
	}
}

func TestParseModel_UnknownBare_Error(t *testing.T) {
	_, _, err := ParseModel("nonexistent")
	if err == nil {
		t.Error("expected error for unknown bare alias")
	}
}

func TestParseModel_GooglePro(t *testing.T) {
	prov, model, err := ParseModel("google/pro")
	if err != nil {
		t.Fatal(err)
	}
	if prov != "google" {
		t.Errorf("provider = %q", prov)
	}
	if model != "gemini-3-pro-image-preview" {
		t.Errorf("model = %q", model)
	}
}

type stubProvider struct{}

func (s stubProvider) Name() string                     { return "stub" }
func (s stubProvider) Capabilities(_ string) Capability { return Capability{} }
func (s stubProvider) Generate(_ context.Context, _ GenerateRequest) (*GenerateResult, error) {
	return &GenerateResult{}, nil
}

func TestRegister_Get(t *testing.T) {
	Register("stub", func(apiKey string) Provider { return stubProvider{} })
	p, err := Get("stub", "key")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "stub" {
		t.Errorf("Name() = %q, want stub", p.Name())
	}
}

func TestGet_UnknownProvider(t *testing.T) {
	_, err := Get("doesnotexist", "key")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}
