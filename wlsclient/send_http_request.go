package wlsclient

import (
	"io/ioutil"
	"net/http"
	"sync"

	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"

	"intel/isecl/lib/clients"
	"intel/isecl/lib/clients/aas"
)

var aasClient = aas.NewJWTClient(config.Configuration.Aas.BaseURL)
var aasRWLock = sync.RWMutex{}

func init() {
	aasRWLock.Lock()
	if aasClient.HTTPClient == nil {
		c, err := clients.HTTPClientWithCADir(consts.TrustedCaCertsDir)
		if err != nil {
			return
		}
		aasClient.HTTPClient = c
	}
	aasRWLock.Unlock()
}

func addJWTToken(req *http.Request) error {

	aasRWLock.RLock()
	jwtToken, err := aasClient.GetUserToken(config.Configuration.Wls.APIUsername)
	aasRWLock.RUnlock()
	// something wrong
	if err != nil {
		// lock aas with w lock
		aasRWLock.Lock()
		// check if other thread fix it already
		jwtToken, err = aasClient.GetUserToken(config.Configuration.Wls.APIUsername)
		// it is not fixed
		if err != nil {
			// these operation cannot be done in init() because it is not sure
			// if config.Configuration is loaded at that time
			aasClient.AddUser(config.Configuration.Wls.APIUsername, config.Configuration.Wls.APIPassword)
			aasClient.FetchAllTokens()
		}
		aasRWLock.Unlock()
	}
	req.Header.Set("Authorization", "Bearer "+string(jwtToken))
	return nil
}

//SendRequest method is used to create an http client object and send the request to the server
func SendRequest(req *http.Request, insecureConnection bool) ([]byte, error) {

	client, err := clients.HTTPClientWithCADir(consts.TrustedCaCertsDir)
	if err != nil {
		return nil, err
	}
	err = addJWTToken(req)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		// fetch token and try again
		aasRWLock.Lock()
		aasClient.FetchAllTokens()
		aasRWLock.Unlock()
		err = addJWTToken(req)
		if err != nil {
			return nil, err
		}
		response, err = client.Do(req)
		if err != nil {
			return nil, err
		}
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, err
	}
	//create byte array of HTTP response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
