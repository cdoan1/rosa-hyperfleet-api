/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oidc

import (
	"context"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	minKeyBits = 2048

	discoveryPath = ".well-known/openid-configuration"
)

// SecretName returns the deterministic AWS Secrets Manager path for
// configID's OIDC signing key. Shared by the writer (this package) and the
// ExternalSecret reader (render.oidcSigningKeySecret) to avoid path drift.
func SecretName(configID string) string {
	return "/hyperfleet/oidc/" + configID + "/signing-key"
}

// InfraClient abstracts the OIDC infrastructure operations needed by the
// OidcConfig controller.
type InfraClient interface {
	StorePrivateKey(ctx context.Context, configID string, privateKeyPEM []byte) error
	PrivateKeyExists(ctx context.Context, configID string) (bool, error)
	ReadCrossAccountSecret(ctx context.Context, secretARN, roleARN string) ([]byte, error)
	DeletePrivateKey(ctx context.Context, configID string) error
	VerifyIssuer(ctx context.Context, issuerURL string) (string, error)
}

// ValidateRSAPrivateKey checks that pemData is a valid PEM-encoded RSA private key with a modulus of at least minKeyBits
func ValidateRSAPrivateKey(pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return fmt.Errorf("no PEM block found")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return checkRSAKeySize(key)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("not a valid RSA private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("key is not RSA")
	}
	return checkRSAKeySize(rsaKey)
}

func checkRSAKeySize(key *rsa.PrivateKey) error {
	if bits := key.N.BitLen(); bits < minKeyBits {
		return fmt.Errorf("RSA key size %d bits is below the minimum of %d bits", bits, minKeyBits)
	}
	return nil
}

// AWSClient implements InfraClient using AWS Secrets Manager and STS.
type AWSClient struct {
	sm     *secretsmanager.Client
	sts    *sts.Client
	awsCfg aws.Config

	mu              sync.Mutex
	assumeRoleCache map[string]*aws.CredentialsCache
}

// NewAWSClient creates a new AWSClient.
func NewAWSClient(awsCfg aws.Config) *AWSClient {
	return &AWSClient{
		sm:              secretsmanager.NewFromConfig(awsCfg),
		sts:             sts.NewFromConfig(awsCfg),
		awsCfg:          awsCfg,
		assumeRoleCache: make(map[string]*aws.CredentialsCache),
	}
}

// assumeRoleCredentials returns a cached credentials provider for roleARN, creating and caching one on first use.
func (c *AWSClient) assumeRoleCredentials(roleARN string) *aws.CredentialsCache {
	c.mu.Lock()
	defer c.mu.Unlock()

	if creds, ok := c.assumeRoleCache[roleARN]; ok {
		return creds
	}
	creds := aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(c.sts, roleARN))
	c.assumeRoleCache[roleARN] = creds
	return creds
}

// StorePrivateKey creates the Secrets Manager secret holding the OIDC
// signing key, or overwrites it if one already exists.
func (c *AWSClient) StorePrivateKey(ctx context.Context, configID string, privateKeyPEM []byte) error {
	secretName := SecretName(configID)
	_, err := c.sm.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(string(privateKeyPEM)),
		Description:  aws.String("OIDC signing key for config " + configID),
	})
	if err != nil {
		var existsErr *smtypes.ResourceExistsException
		if !errors.As(err, &existsErr) {
			return fmt.Errorf("create secret: %w", err)
		}
		if _, err := c.sm.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(secretName),
			SecretString: aws.String(string(privateKeyPEM)),
		}); err != nil {
			return fmt.Errorf("update secret: %w", err)
		}
	}
	return nil
}

func (c *AWSClient) PrivateKeyExists(ctx context.Context, configID string) (bool, error) {
	secretName := SecretName(configID)
	_, err := c.sm.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		var notFoundErr *smtypes.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, fmt.Errorf("describe secret: %w", err)
	}
	return true, nil
}

func (c *AWSClient) ReadCrossAccountSecret(ctx context.Context, secretARN, roleARN string) ([]byte, error) {
	crossSM := secretsmanager.NewFromConfig(c.awsCfg, func(o *secretsmanager.Options) {
		o.Credentials = c.assumeRoleCredentials(roleARN)
	})

	result, err := crossSM.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretARN),
	})
	if err != nil {
		return nil, fmt.Errorf("read cross-account secret: %w", err)
	}
	if result.SecretBinary != nil {
		return result.SecretBinary, nil
	}
	return []byte(aws.ToString(result.SecretString)), nil
}

func (c *AWSClient) DeletePrivateKey(ctx context.Context, configID string) error {
	secretName := SecretName(configID)
	_, err := c.sm.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(secretName),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	if err != nil {
		var notFoundErr *smtypes.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return nil
		}
		return fmt.Errorf("delete secret: %w", err)
	}
	return nil
}

// oidcDiscoveryResponse is the subset of the OIDC discovery document
type oidcDiscoveryResponse struct {
	Issuer string `json:"issuer"`
}

// VerifyIssuer confirms issuerURL is actually serving a valid OIDC discovery document
func (c *AWSClient) VerifyIssuer(ctx context.Context, issuerURL string) (string, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	hostname, dialAddr, err := resolveIssuerHost(dialCtx, issuerURL)
	if err != nil {
		return "", err
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, dialAddr)
		},
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: hostname,
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	return verifyIssuerDocument(dialCtx, client, issuerURL)
}

// verifyIssuerDocument performs the actual GET check against client and validates the response
func verifyIssuerDocument(ctx context.Context, client *http.Client, issuerURL string) (string, error) {
	normalizedIssuerURL := strings.TrimRight(issuerURL, "/")
	discoveryURL := normalizedIssuerURL + "/" + discoveryPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return "", fmt.Errorf("build discovery request for %s: %w", discoveryURL, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", discoveryURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: unexpected status %d", discoveryURL, resp.StatusCode)
	}

	var doc oidcDiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("decode discovery document from %s: %w", discoveryURL, err)
	}
	if doc.Issuer == "" {
		return "", fmt.Errorf("discovery document from %s has no issuer field", discoveryURL)
	}
	if doc.Issuer != normalizedIssuerURL {
		return "", fmt.Errorf("discovery document issuer %q from %s does not match expected issuer %q", doc.Issuer, discoveryURL, normalizedIssuerURL)
	}

	// Use the root CA certificate per AWS IAM OIDC provider convention
	if resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 || len(resp.TLS.VerifiedChains[0]) == 0 {
		return "", fmt.Errorf("no verified TLS certificate chain from %s", discoveryURL)
	}

	chain := resp.TLS.VerifiedChains[0]
	root := chain[len(chain)-1]
	// AWS IAM OIDC providers require the thumbprint to be a SHA-1 fingerprint of
	// the root CA certificate; this is an API contract, not a security choice.
	fingerprint := sha1.Sum(root.Raw) //nolint:gosec // SHA-1 required by AWS IAM OIDC provider API contract
	return fmt.Sprintf("%x", fingerprint[:]), nil
}

// resolveIssuerHost validates issuerURL against SSRF and resolves it to a concrete dial address
func resolveIssuerHost(ctx context.Context, issuerURL string) (hostname, dialAddr string, err error) {
	u, err := url.Parse(issuerURL)
	if err != nil {
		return "", "", fmt.Errorf("parse issuer URL: %w", err)
	}
	if u.Scheme != "https" {
		return "", "", fmt.Errorf("issuer URL must use https scheme, got %q", u.Scheme)
	}

	hostname = u.Hostname()
	if hostname == "" {
		return "", "", fmt.Errorf("issuer URL has no host")
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return "", "", fmt.Errorf("resolve issuer host %s: %w", hostname, err)
	}
	if len(addrs) == 0 {
		return "", "", fmt.Errorf("issuer host %s did not resolve to any address", hostname)
	}
	for _, addr := range addrs {
		if isDisallowedIssuerIP(addr.IP) {
			return "", "", fmt.Errorf("issuer host %s resolves to disallowed address %s", hostname, addr.IP)
		}
	}

	return hostname, net.JoinHostPort(addrs[0].IP.String(), port), nil
}

func isDisallowedIssuerIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsPrivate() ||
		ip.IsUnspecified()
}
