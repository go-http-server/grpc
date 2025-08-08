package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
)

// Payload is the structure that holds the account information
type Payload struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

func NewPayload(username, role string, duration time.Duration) (*Payload, error) {
	payload := &Payload{
		Username:  username,
		Role:      role,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}

	return payload, nil
}

func (payload Payload) IsValid() error {
	if time.Now().After(payload.ExpiredAt) {
		return errors.New("token expired")
	}
	return nil
}

type TokenMaker interface {
	// CreateToken generates a new token for the given account with a specified duration.
	CreateToken(acc *Account, duration time.Duration) (string, error)

	// VerifyToken checks the validity of the token and returns the associated account if valid.
	VerifyToken(token string) (*Payload, error)
}

type PasetoMaker struct {
	PrivateKey paseto.V4AsymmetricSecretKey
	PublicKey  paseto.V4AsymmetricPublicKey
	Parser     paseto.Parser
}

func NewPasetoMaker(privateKey paseto.V4AsymmetricSecretKey, parser paseto.Parser) TokenMaker {
	publicKey := privateKey.Public()
	return &PasetoMaker{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Parser:     parser,
	}
}

func (maker PasetoMaker) CreateToken(acc *Account, duration time.Duration) (string, error) {
	payload, err := NewPayload(acc.Username, acc.Role, duration)
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("cannot marshal payload: %s", err)
	}

	token, err := paseto.NewTokenFromClaimsJSON(claims, nil)
	if err != nil {
		return "", fmt.Errorf("cannot create token from claims json: %s", err)
	}
	tokenSigned := token.V4Sign(maker.PrivateKey, nil)
	return tokenSigned, nil
}

func (maker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	payload := &Payload{}
	tokenParser, err := maker.Parser.ParseV4Public(maker.PublicKey, token, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot parse token: %s", err)
	}

	err = json.Unmarshal(tokenParser.ClaimsJSON(), payload)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal from claims json: %s", err)
	}
	err = payload.IsValid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
