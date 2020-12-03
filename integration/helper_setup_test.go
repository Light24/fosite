/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 *
 */

package integration_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	goauth "golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	goauth_jwt "golang.org/x/oauth2/jwt"
	"gopkg.in/square/go-jose.v2"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/internal"
	"github.com/ory/fosite/storage"
	"github.com/ory/fosite/token/hmac"
	"github.com/ory/fosite/token/jwt"
)

var firstPrivateKey, _ = rsa.GenerateKey(rand.Reader, 2048)
var secondPrivateKey, _ = rsa.GenerateKey(rand.Reader, 2048)

const secondKeyID = "321"
const firstKeyID = "123"
const secondJWTBearerIssuer = "second@tinkoff.ru"
const firstJWTBearerIssuer = "first@tinkoff.ru"
const jwtBearerSubject = "service-client"

var fositeStore = &storage.MemoryStore{
	Clients: map[string]fosite.Client{
		"my-client": &fosite.DefaultClient{
			ID:            "my-client",
			Secret:        []byte(`$2a$10$IxMdI6d.LIRZPpSfEwNoeu4rY3FhDREsxFJXikcgdRRAStxUlsuEO`), // = "foobar"
			RedirectURIs:  []string{"http://localhost:3846/callback"},
			ResponseTypes: []string{"id_token", "code", "token", "token code", "id_token code", "token id_token", "token code id_token"},
			GrantTypes:    []string{"implicit", "refresh_token", "authorization_code", "password", "client_credentials"},
			Scopes:        []string{"fosite", "offline", "openid"},
			Audience:      []string{"https://www.ory.sh/api"},
		},
		"public-client": &fosite.DefaultClient{
			ID:            "public-client",
			Secret:        []byte{},
			Public:        true,
			RedirectURIs:  []string{"http://localhost:3846/callback"},
			ResponseTypes: []string{"id_token", "code", "code id_token"},
			GrantTypes:    []string{"refresh_token", "authorization_code"},
			Scopes:        []string{"fosite", "offline", "openid"},
			Audience:      []string{"https://www.ory.sh/api"},
		},
	},
	Users: map[string]storage.MemoryUserRelation{
		"peter": {
			Username: "peter",
			Password: "secret",
		},
	},
	IssuerPublicKeys: map[string]storage.IssuerPublicKeys{
		firstJWTBearerIssuer: {
			Issuer: firstJWTBearerIssuer,
			KeysBySub: map[string]storage.SubjectPublicKeys{
				jwtBearerSubject: {
					Subject: jwtBearerSubject,
					Keys: map[string]storage.PublicKeyScopes{
						firstKeyID: {
							Key: &jose.JSONWebKey{
								Key:       firstPrivateKey.Public(),
								Algorithm: string(jose.RS256),
								Use:       "sig",
								KeyID:     firstKeyID,
							},
							Scopes: []string{
								"fosite",
							},
						},
					},
				},
			},
		},
		secondJWTBearerIssuer: {
			Issuer: secondJWTBearerIssuer,
			KeysBySub: map[string]storage.SubjectPublicKeys{
				jwtBearerSubject: {
					Subject: jwtBearerSubject,
					Keys: map[string]storage.PublicKeyScopes{
						secondKeyID: {
							Key: &jose.JSONWebKey{
								Key:       secondPrivateKey.Public(),
								Algorithm: string(jose.RS256),
								Use:       "sig",
								KeyID:     secondKeyID,
							},
							Scopes: []string{
								"fosite",
							},
						},
					},
				},
			},
		},
	},
	BlacklistedJTIs:        map[string]time.Time{},
	AuthorizeCodes:         map[string]storage.StoreAuthorizeCode{},
	PKCES:                  map[string]fosite.Requester{},
	AccessTokens:           map[string]fosite.Requester{},
	RefreshTokens:          map[string]fosite.Requester{},
	IDSessions:             map[string]fosite.Requester{},
	AccessTokenRequestIDs:  map[string]string{},
	RefreshTokenRequestIDs: map[string]string{},
}

type defaultSession struct {
	*openid.DefaultSession
}

var accessTokenLifespan = time.Hour

var authCodeLifespan = time.Minute

func newOAuth2Client(ts *httptest.Server) *goauth.Config {
	return &goauth.Config{
		ClientID:     "my-client",
		ClientSecret: "foobar",
		RedirectURL:  ts.URL + "/callback",
		Scopes:       []string{"fosite"},
		Endpoint: goauth.Endpoint{
			TokenURL:  ts.URL + "/token",
			AuthURL:   ts.URL + "/auth",
			AuthStyle: goauth.AuthStyleInHeader,
		},
	}
}

func newOAuth2AppClient(ts *httptest.Server) *clientcredentials.Config {
	return &clientcredentials.Config{
		ClientID:     "my-client",
		ClientSecret: "foobar",
		Scopes:       []string{"fosite"},
		TokenURL:     ts.URL + "/token",
	}
}

func newOAuth2JWTBearerAppClient(ts *httptest.Server) *goauth_jwt.Config {
	return &goauth_jwt.Config{
		Email:        firstJWTBearerIssuer,
		Subject:      jwtBearerSubject,
		Scopes:       []string{"fosite"},
		Audience:     "https://www.ory.sh/api",
		TokenURL:     ts.URL + "/token",
		PrivateKey:   x509.MarshalPKCS1PrivateKey(firstPrivateKey),
		PrivateKeyID: firstKeyID,
		Expires:      24 * time.Hour,
	}
}

var hmacStrategy = &oauth2.HMACSHAStrategy{
	Enigma: &hmac.HMACStrategy{
		GlobalSecret: []byte("some-super-cool-secret-that-nobody-knows"),
	},
	AccessTokenLifespan:   accessTokenLifespan,
	AuthorizeCodeLifespan: authCodeLifespan,
}

var jwtStrategy = &oauth2.DefaultJWTStrategy{
	JWTStrategy: &jwt.RS256JWTStrategy{
		PrivateKey: internal.MustRSAKey(),
	},
	HMACSHAStrategy: hmacStrategy,
}

func mockServer(t *testing.T, f fosite.OAuth2Provider, session fosite.Session) *httptest.Server {
	router := mux.NewRouter()
	router.HandleFunc("/auth", authEndpointHandler(t, f, session))
	router.HandleFunc("/token", tokenEndpointHandler(t, f))
	router.HandleFunc("/callback", authCallbackHandler(t))
	router.HandleFunc("/info", tokenInfoHandler(t, f, session))
	router.HandleFunc("/introspect", tokenIntrospectionHandler(t, f, session))
	router.HandleFunc("/revoke", tokenRevocationHandler(t, f, session))

	ts := httptest.NewServer(router)
	return ts
}
