package app

import (
	"github.com/stretchr/testify/assert"
	"os"
	"os/user"
	"path"
	"strings"
	"testing"
)

func TestParseKubeConfigFailures(t *testing.T) {
	tests := []struct {
		filename string
		complain string
	}{
		{
			filename: "test_data/kubeconfig_missing",
			complain: "not read",
		},
		{
			filename: "test_data/kubeconfig_invalid",
			complain: "invalid",
		},
	}

	for _, test := range tests {
		_, err := parseKubeConfig(test.filename)
		if err == nil {
			t.Errorf("Expected an error for file %s", test.filename)
			continue
		}

		if !strings.Contains(err.Error(), test.complain) {
			t.Errorf("Error [%s] does not contain [%s]", err, test.complain)
		}
	}
}

func TestParseKubeConfig(t *testing.T) {
	actual, err := parseKubeConfig("test_data/kubeconfig_valid")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}

	expected := Config{
		CurrentContext: "prod",
		Contexts: []ContextWrap{
			{"dev", Context{"cluster_2", "red", "user_2"}},
			{"prod", Context{"cluster_1", "blue", "user_1"}},
		},
		Clusters: []ClusterWrap{
			{"cluster_1", Cluster{Server: "https://foo.com", CertificateAuthority: "ca.pem"}},
			{"cluster_2", Cluster{Server: "https://bar.com", CertificateAuthority: "ca.pem", SkipVerify: true}},
		},
		Users: []UserWrap{
			{"user_1", User{"cert.pem", "key.pem"}},
			{"user_2", User{"cert.pem", "key.pem"}},
		},
	}

	assert.Equal(t, expected, actual)
}

func TestConfigMakeFilter(t *testing.T) {
	conf := Config{
		CurrentContext: "prod",
		Contexts: []ContextWrap{
			{"dev", Context{Cluster: "cluster_2", Namespace: "red"}},
			{"prod", Context{Cluster: "cluster_1", Namespace: "blue"}},
		},
		Clusters: []ClusterWrap{
			{"cluster_1", Cluster{Server: "https://foo.com:8443"}},
			{"cluster_2", Cluster{Server: "https://bar.com"}},
		},
	}

	expected := MrrFilter{Server: "https://foo.com:8443", Namespace: "blue"}
	actual := conf.makeFilter()
	assert.Equal(t, expected, actual)
}

func TestConfigMakeTLSConfig(t *testing.T) {
	cfg := Config{
		CurrentContext: "x",
		Contexts:       []ContextWrap{{"x", Context{Cluster: "cluster", User: "user"}}},
		Clusters:       []ClusterWrap{{"cluster", Cluster{CertificateAuthority: "test_data/ca.pem", SkipVerify: true}}},
		Users:          []UserWrap{{"user", User{"test_data/cert.pem", "test_data/key.pem"}}},
	}

	tls, err := cfg.GenerateTLSConfig()

	if assert.NoError(t, err) {
		assert.Equal(t, 1, len(tls.RootCAs.Subjects()), "must have parsed Certificate Authority")
	}
	assert.Equal(t, true, tls.InsecureSkipVerify)
}

//Copyright 2014 The Kubernetes Authors.
func TestSubstituteUserHome(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Logf("SKIPPING TEST: unexpected error: %v", err)
		return
	}
	tests := []struct {
		input     string
		expected  string
		expectErr bool
	}{
		{input: "~/foo", expected: path.Join(os.Getenv("HOME"), "foo")},
		{input: "~" + usr.Username + "/bar", expected: usr.HomeDir + "/bar"},
		{input: "/foo/bar", expected: "/foo/bar"},
		{input: "~doesntexit/bar", expectErr: true},
	}
	for _, test := range tests {
		output, err := substituteUserHome(test.input)
		if test.expectErr {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, test.expected, output)
	}
}
