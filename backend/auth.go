package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type KeycloakClaims struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	jwt.RegisteredClaims
}

type KeycloakJWKS struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

type KeycloakValidator struct {
	config *Config
	keys   map[string]*rsa.PublicKey
}

func NewKeycloakValidator(config *Config) *KeycloakValidator {
	return &KeycloakValidator{
		config: config,
		keys:   make(map[string]*rsa.PublicKey),
	}
}

func (kv *KeycloakValidator) fetchPublicKeys() error {
	// For development, we'll skip fetching keys and use a simple validation
	// In production, you should fetch keys from Keycloak's JWKS endpoint
	// URL: {KEYCLOAK_URL}/realms/{REALM}/protocol/openid-connect/certs

	log.Println("Using development JWT validation (skip signature verification)")
	return nil
}

func (kv *KeycloakValidator) validateToken(tokenString string) (*KeycloakClaims, error) {
	// For development, we'll parse the token without signature verification
	// In production, you should implement proper RSA signature verification

	// Parse token manually to bypass signature verification
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	// Parse the claims
	var claims KeycloakClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Validate token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	// Validate token issuer (optional but recommended)
	expectedIssuer := fmt.Sprintf("%s/realms/%s", kv.config.KeycloakURL, kv.config.KeycloakRealm)
	if claims.Issuer != expectedIssuer {
		log.Printf("Warning: token issuer mismatch. Expected: %s, Got: %s", expectedIssuer, claims.Issuer)
	}

	return &claims, nil
}

func AuthMiddleware(config *Config) gin.HandlerFunc {
	validator := NewKeycloakValidator(config)

	// Fetch public keys (in production)
	if err := validator.fetchPublicKeys(); err != nil {
		log.Printf("Warning: failed to fetch public keys: %v", err)
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Validate the token
		claims, err := validator.validateToken(tokenString)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		// Add user info to context
		c.Set("user_id", claims.Sub)
		c.Set("username", claims.PreferredUsername)
		c.Set("email", claims.Email)
		c.Set("roles", claims.RealmAccess.Roles)
		c.Next()
	}
}

// Helper function to convert JWK to RSA public key (for production use)
func jwkToRSAPublicKey(n, e string) (*rsa.PublicKey, error) {
	// Decode base64url encoded values
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	// Convert to big integers
	nBig := new(big.Int).SetBytes(nBytes)
	eBig := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	publicKey := &rsa.PublicKey{
		N: nBig,
		E: int(eBig.Int64()),
	}

	return publicKey, nil
}
