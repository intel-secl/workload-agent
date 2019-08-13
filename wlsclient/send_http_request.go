package wlsclient

import (
	"io/ioutil"
	"net/http"

	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"

	"intel/isecl/lib/clients"
	"intel/isecl/lib/clients/aas"
)

var aasClient = aas.NewJWTClient(config.Configuration.Aas.BaseURL)

func addJWTToken(req *http.Request) error {

	if aasClient.HTTPClient == nil {
		c, err := clients.HTTPClientWithCADir(consts.TrustedCaCertsDir)
		if err != nil {
			return err
		}
		aasClient.HTTPClient = c
		aasClient.AddUser(config.Configuration.Wls.APIUsername, config.Configuration.Wls.APIPassword)
		aasClient.FetchAllTokens()
	}

	jwtToken, err := aasClient.GetUserToken(config.Configuration.Wls.APIUsername)
	if err != nil {
		return err
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
		aasClient.FetchAllTokens()
		err = addJWTToken(req)
		if err != nil {
			return nil, err
		}
		response, err = client.Do(req)
		if err != nil {
			return nil, err
		}
	}
	//create byte array of HTTP response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
