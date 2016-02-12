/*
Package oidc is an OpenID Connect Relying Party used to test TNaaS's support for the OpenID Connect Protocol
(i.e. TNaaS's ability to create a policy that an RP accesses via OpenID Connect).
All policies use the same TNaaS OpenID Connect Authn, Token and User Info endpoints.
Each policy defines a unique OpenID Connect Client ID and Secret that a single
Client then uses to access it.

This RP is configured at startup to access a single policy.

It is assumed that a browser will be used to issue a /login GET request to this RP.
This RP supports concurrent processing of /login requests; however, a particular browser
can only have one request in process at a time because the state for a request is
stored in an a single encrypted cookie.

If a /login request provides both a clientid and secret query parameters these override the default command flag values.

On receipt of a /login request, the following steps occur:

(1) The RP issues an OpenID Connect Authn Request as a redirect to the
TNaaS OP Authn Endpoint. This redirect sets an encrypted cookie containing state required
by the following steps.

(2) The RP waits to receive its Authn response via a redirect to its /authn-token endpoint.
This response contains an Authorization Code query parameter and provides the encrypted cookie
set in step 1.

(3) A Token Form POST request is issued to the TNaaS OP Token Endpoint.
This includes several Form parameters including the Authorization Code and a JWT encoded client assertion
used to identify this RP to the TNaaS OP.

(4) The Token response contains JSON with a UserInfo Access Token, ID Token and other properties.

(5) A GET request with the Authentication header set to the UserInfo Access Token is issued to the TNaaS OP User Info endpoint.

(6) The response is JSON encoded User Info for the authenticated subject.

(7) The ID Token JWT is decoded and the JSON encoded ID Token content and UserInfo content is returned in the /login response.

The service accepts the following command flags in either '-' or '--' form:
	-exthost   	- the public hostname of this RP
	-ophost		- the host name of this RP's OpenID Connect Authentication Server
	-clientid	- the default OpenID Connect client ID of this RP
	-secret		- the default client ID's secret this RP shares with its OP
	-scope		- the list of optional, space delimited Authn Request scope values; the full list is "profile email address phone"
	-log       	- The log file name
	-logprefix 	- The logging prefix
	-logflag   	- The logging flag

See the log package for descriptions of the logging prefix and logging flag.
*/
package main

import (
	"crypto/cipher"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"bitbucket.org/mark_hapner/tn-go/certbndl"

	"github.com/develrns/resilient/aead"
	"github.com/develrns/resilient/log"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
)

type (
	//TokenRspBody is the JSON body of a response to an OP Token Endpoint request
	TokenRspBody struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}

	//AuthnReqState is the content of an Authn Request cookie set by this RP
	AuthnReqState struct {
		ClientID string
		Secret   string
		State    string
		Nonce    string
	}
)

var (
	//The RP logger
	logger = log.Logger()

	//Command flags
	exthost               string
	ophost                string
	defaultClientID       string
	defaultOpSharedSecret string
	scope                 string

	//The HTTPS client used to issue OP requests
	opClient *http.Client

	//The OP Endpoints
	opAuthnEndpoint    string
	opTokenEndpoint    string
	opUserInfoEndpoint string

	//The AEAD cipher used to encrypt/decrypt all subscriber identifiers in the hidden fields of TBD 2nd Factor Selection Forms
	aeadCipher cipher.AEAD
)

/*
init reads the command line flags and initializes this executable's shared log instance
*/
func init() {
	var (
		logFileName string
		logPrefix   string
		logFlag     int
	)

	flag.StringVar(&exthost, "exthost", "", "the public hostname of this RP")
	flag.StringVar(&ophost, "ophost", "", "the host name of this RP's OpenID Connect Authentication Server")
	flag.StringVar(&defaultClientID, "clientid", "", "the default OpenID Connect client ID of this RP")
	flag.StringVar(&defaultOpSharedSecret, "secret", "", "the default client ID's secret this RP shares with its OP")
	flag.StringVar(&scope, "scope", "", `the list of optional, space delimited Authn Request scope values; the full list is "profile email address phone"`)
	flag.StringVar(&logFileName, "log", "", "log file name (default stdout)")
	flag.StringVar(&logPrefix, "logprefix", "", "logging prefix")
	flag.IntVar(&logFlag, "logflag", 0, "logging flag")
	flag.Parse()
	log.Config(logFileName, logPrefix, logFlag)

	//Initialize the OP Endpoints
	opAuthnEndpoint = "https://" + ophost + "/openId/authenticate"
	opTokenEndpoint = "https://" + ophost + "/openId/token"
	opUserInfoEndpoint = "https://" + ophost + "/openId/userinfo"
}

/*
writeError responds with 400 Bad Request and an error msg body
*/
func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
}

/*
handleLogin implements an RP login request. This is expected to be a GET issued by a browser user agent.

It initiates an OpenID Connect Authentication Request contained in the query string of a redirect to an OP Authentication
URL. This redirection is completed on return of the user agent via a redirect to the /authn-token
*/
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var (
		authnReqURL        string
		oidState           = uuid.NewRandom().String()
		oidNonce           = uuid.NewRandom().String()
		authnReqState      AuthnReqState
		authnReqStateBytes []byte
		authnCookie        http.Cookie
		authnCookieValue   string
		clientID           = defaultClientID
		secret             = defaultOpSharedSecret
		values             url.Values
		err                error
	)

	if r.Method != "GET" {
		writeError(w, fmt.Errorf("Bad HTTP Method: %v", r.Method))
		return
	}

	//If the request provides clientid and secret parameters these override the default values
	values = r.URL.Query()
	clientIDs, okClientIDs := values["clientid"]
	if okClientIDs {
		clientID = clientIDs[0]
	}
	secrets, okSecrets := values["secret"]
	if okSecrets {
		secret = secrets[0]
	}
	if okClientIDs != okSecrets {
		writeError(w, fmt.Errorf("Both clientid and secret query parameters must be provided"))
		return
	}

	//The Authn Request
	authnReqURL = opAuthnEndpoint + "?response_type=code&scope=openid%20" + scope + "&client_id=" + clientID + "&state=" + oidState + "&nonce=" + oidNonce + "&redirect_uri=https://" + exthost + "/authn-token"
	fmt.Println(authnReqURL)

	//The authnReqState is aead encrypted to produce a value stored as an authn cookie. This value transmits the oidState to the Authn Response while maintaining its privacy and integrity
	//from any prying eyes that may exist in the browser.
	authnReqState = AuthnReqState{ClientID: clientID, Secret: secret, State: oidState, Nonce: oidNonce}
	authnReqStateBytes, _ = json.Marshal(&authnReqState)
	authnCookieValue, err = aead.Encrypt(aeadCipher, "AuthnReqState", string(authnReqStateBytes))
	if err != nil {
		writeError(w, err)
		return
	}
	authnCookie = http.Cookie{Name: "authnCookie", Value: authnCookieValue, Path: "/authn-token", Domain: exthost, HttpOnly: true, Secure: true, MaxAge: 300}

	//Issue the Authn Request via a redirect to the OP Authn Reqest endpoint.
	w.Header().Set("Location", authnReqURL)
	http.SetCookie(w, &authnCookie)
	w.WriteHeader(http.StatusSeeOther)
}

/*
handleAuthnToken receives an Authentication Token as a query parameter of a redirect issued by the OP
(the Authn Response) and uses it to retrieve an Access Token and ID Token from the OP Token Endpoint.
The Access Token is used to retrieve the subject's User Info (as specified in the Authn Request Scope) from the OP
User Info Endpoint

To prevent a XSS attack from substituting a rogue Authentication Token as this redirect passes through the user agent,
the state parameter returned by the OP must be the same as the state parameter in the originating Authn Request.
In addition, the Authn Request nonce must match the ID Token nonce.

The content of the ID Token and User Info in JSON format is returned in the body of the Login response.
*/
func handleAuthnToken(w http.ResponseWriter, r *http.Request) {
	var (
		authnReqState       AuthnReqState
		authnReqStateString string
		authnRespParams     = r.URL.Query()
		authnCookie         *http.Cookie
		clientAssertion     = jwt.New(jwt.SigningMethodHS256)
		tokenRspBody        TokenRspBody
		idToken             *jwt.Token
		userInfoReq         *http.Request
		userInfoRsp         *http.Response
		ok                  bool
		err                 error
	)

	fmt.Println("https://" + exthost + "/authn-token/?" + r.URL.RawQuery)

	if r.Method != "GET" {
		writeError(w, fmt.Errorf("Bad HTTP Method: %v\n", r.Method))
		return
	}

	//The authnCookie contains the aead encrypted AuthnReqState
	authnCookie, err = r.Cookie("authnCookie")
	if err != nil {
		writeError(w, fmt.Errorf("Missing authnCookie\n"))
		return
	}
	_, authnReqStateString, err = aead.Decrypt(aeadCipher, authnCookie.Value)
	if err != nil {
		writeError(w, err)
		return
	}
	json.Unmarshal([]byte(authnReqStateString), &authnReqState)
	fmt.Println("AuthnReqState: ", authnReqState)

	//Validate that the oidState values match
	authnRespStates, ok := authnRespParams["state"]
	if !ok {
		writeError(w, fmt.Errorf("Missing Authn Response State\n"))
		return
	}
	switch len(authnRespStates) {
	case 1:
		if authnReqState.State != authnRespStates[0] {
			writeError(w, fmt.Errorf("State match failed\nexpected state: %v\nprovided state: %v\n", authnReqState.State, authnRespStates[0]))
			return
		}
	default:
		writeError(w, fmt.Errorf("Authn Response State has %v values", len(authnRespStates)))
		return
	}
	if authnReqState.State != authnRespParams["state"][0] {
		writeError(w, fmt.Errorf("State match failed\nexpected state: %v\nprovided state: %v\n", authnReqState.State, authnRespParams["state"]))
		return
	}

	//If the OP returned an Authn Request error, report it.
	_, ok = authnRespParams["error"]
	if ok {
		writeError(w, fmt.Errorf("OP Authn Request Error: %v\n %v\n %v\n", authnRespParams["error"], authnRespParams["error_description"], authnRespParams["error_uri"]))
		return
	}

	//One Authorization Code must be provided
	authnRespCodes, ok := authnRespParams["code"]
	if !ok {
		writeError(w, fmt.Errorf("Missing Authn Response Authorization Code"))
		return
	}
	if len(authnRespCodes) != 1 {
		writeError(w, fmt.Errorf("Authn Response Authorization Code has %v values\n", len(authnRespStates)))
		return
	}

	//Issue the Token Request to the OP Token Endpoint. TNaaS OPs always use client_secret_jwt client authentication.
	requestTime := time.Now().UTC()
	clientAssertion.Claims = map[string]interface{}{"iss": authnReqState.ClientID, "sub": authnReqState.ClientID, "aud": opTokenEndpoint, "jti": uuid.NewRandom().String(), "exp": requestTime.Add(time.Minute * 10).String(), "iat": requestTime.String()}
	fmt.Println("Client Assertion Claims: ", clientAssertion.Claims)
	clientAssertionString, err := clientAssertion.SignedString([]byte(authnReqState.Secret))
	if err != nil {
		writeError(w, fmt.Errorf("Client Assertion Signing Error: %v", err))
		return
	}
	tokenRequestForm := url.Values{"grant_type": {"authorization_code"}, "code": {authnRespParams["code"][0]}, "client_id": {authnReqState.ClientID}, "client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"}, "client_assertion": {clientAssertionString}, "redirect_uri": {"https://" + exthost + "/authn-token"}}
	tokenRsp, err := opClient.PostForm(opTokenEndpoint, tokenRequestForm)
	if err != nil {
		writeError(w, fmt.Errorf("Token Endpoint Form Post Error: %v", err))
		return
	}

	fmt.Println(opTokenEndpoint, " form: ", tokenRequestForm)

	//Read the Token Response Body
	tokenRspBodyBytes, err := ioutil.ReadAll(tokenRsp.Body)
	fmt.Println("Token Endpoint Response Body: ", string(tokenRspBodyBytes))

	//Validate the response is good and unmarshal it's JSON body
	if tokenRsp.StatusCode != http.StatusOK {
		writeError(w, fmt.Errorf("OP Token Request Status Error: %v\n%v", tokenRsp.Status, string(tokenRspBodyBytes)))
		return
	}
	if tokenRsp.Header.Get("Content-Type") != "application/json" {
		writeError(w, fmt.Errorf("OP Token Request Bad Content-Type: %v", tokenRsp.Header.Get("Content-Type")))
		return
	}
	err = json.Unmarshal(tokenRspBodyBytes, &tokenRspBody)
	if err != nil {
		writeError(w, fmt.Errorf("Error Decoding Token Response Body: %v", err))
		return
	}
	fmt.Println("Parsed Token Endpoint Response Body: ", tokenRspBody)

	//The ID Token provided by the OP is parsed
	if tokenRspBody.IDToken == "" {
		writeError(w, fmt.Errorf("Missing Token Response ID Token"))
		return
	}
	idToken, err = jwt.Parse(tokenRspBody.IDToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(authnReqState.Secret), nil
	})
	if err != nil {
		writeError(w, fmt.Errorf("ID Token Parsing Failed with Error: %v", err))
		return
	}

	//The Authn Request nonce  must match the ID Token nonce
	if authnReqState.Nonce != idToken.Claims["nonce"].(string) {
		writeError(w, fmt.Errorf("Authn Request Nonce does not match ID Token Nonce: %v  %v", authnReqState.Nonce, idToken.Claims["nonce"].(string)))
		return
	}

	//Use the Access Token to retrieve the subject's userinfo from the OP userinfo endpoint.
	if tokenRspBody.AccessToken == "" {
		writeError(w, fmt.Errorf("Missing Token Response Access Token"))
		return
	}
	userInfoReq, err = http.NewRequest("GET", opUserInfoEndpoint, nil)
	userInfoReq.Header.Set("Authorization", "Bearer "+tokenRspBody.AccessToken)
	fmt.Println("User Info Request: ", userInfoReq)
	userInfoRsp, err = opClient.Do(userInfoReq)
	if err != nil {
		writeError(w, fmt.Errorf("User Info Request Failed: %v", err))
		return
	}
	userInfoRspBodyBytes, err := ioutil.ReadAll(userInfoRsp.Body)
	if err != nil {
		writeError(w, fmt.Errorf("Reading User Info Request Body Failed: %v", err))
		return
	}
	if userInfoRsp.StatusCode != http.StatusOK {
		writeError(w, fmt.Errorf("User Info Request Failed: %v\n%v", userInfoRsp.Status, string(userInfoRspBodyBytes)))
		return
	}

	//The content of the ID Token Header and Claims is transformed to JSON
	headerJSON := "{"
	for key, val := range idToken.Header {
		headerJSON = headerJSON + `"` + key + `": "` + val.(string) + `",`
	}
	headerJSON = headerJSON[:len(headerJSON)-2] + "}"

	claimsJSON := "{"
	for key, val := range idToken.Claims {
		claimsJSON = claimsJSON + `"` + key + `": "` + fmt.Sprint(val) + `",`
	}
	claimsJSON = claimsJSON[:len(headerJSON)-2] + "}"

	idTokenJSON := `{"header": ` + headerJSON + `, "claims": ` + claimsJSON + "}"
	resultJSON := `{"idtoken": ` + idTokenJSON + `, "userinfo": ` + string(userInfoRspBodyBytes) + "}"

	w.Header().Set("Content-Type", "application/JSON")
	w.Write([]byte(resultJSON))
}

/*
main registers this RP's HTTP request handlers; creates the HTTPS client for issuing OP ID Token requests and starts its HTTP server.
*/
func main() {
	var (
		certPool *x509.CertPool
		server   http.Server
		err      error
	)

	//This aeadCipher is used to encrypt/decrypt the Authn Request Cookie that is used to pass the Authn Request State value
	//from the Authn Request to the Authn Response.
	aeadCipher, err = aead.NewAEADCipher()
	if err != nil {
		return
	}

	//Initialize an HTTPS capable client and replace the default aws HTTP client that doesn't support HTTPS
	certPool = x509.NewCertPool()
	certPool.AppendCertsFromPEM([]byte(certbndl.PemCerts))
	opClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}

	//Start the service
	server = http.Server{Addr: ":443", ReadTimeout: 10 * time.Minute, WriteTimeout: 10 * time.Minute, ErrorLog: logger.Logger()}
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/authn-token", handleAuthnToken)
	logger.Println("Starting oidc on " + exthost + ":443")
	err = server.ListenAndServeTLS("resilient-networks.crt", "resilient-networks.key")
	if err != nil {
		logger.Fatal(err)
	}

}
