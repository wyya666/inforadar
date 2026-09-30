package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"inforadar/internal/constants"
)

type AccessClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func NewRandomToken() (string, error) {
	b := make([]byte, constants.RandomTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func NewSessionID() (string, error) {
	b := make([]byte, constants.SessionIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func TokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func SignAccess(secret []byte, userID uint64, sessionID string, now time.Time) (string, error) {
	claims := AccessClaims{SessionID: sessionID, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: constants.AccessIssuer, Subject: strconv.FormatUint(userID, 10), Audience: jwt.ClaimStrings{constants.AccessAudience},
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(constants.AccessTTL)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func ParseAccess(secret []byte, raw string) (uint64, string, error) {
	claims := &AccessClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(constants.AccessIssuer), jwt.WithAudience(constants.AccessAudience), jwt.WithExpirationRequired(),
	)
	if err != nil || claims.SessionID == "" {
		return 0, "", fmt.Errorf("invalid access token")
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || id == 0 {
		return 0, "", fmt.Errorf("invalid access token subject")
	}
	return id, claims.SessionID, nil
}
