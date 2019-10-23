/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package clients

import (
	"io/ioutil"
	"net/http"
	"sync"

	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"

	"intel/isecl/lib/clients"
	"intel/isecl/lib/clients/aas"

	"github.com/pkg/errors"
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
	log.Trace("clients/send_http_request:addJWTToken() Entering")
	defer log.Trace("clients/send_http_request:addJWTToken() Leaving")
	if aasClient.BaseURL == "" {
		aasClient = aas.NewJWTClient(config.Configuration.Aas.BaseURL)
	}
	aasRWLock.RLock()
	jwtToken, err := aasClient.GetUserToken(config.Configuration.Wla.APIUsername)
	aasRWLock.RUnlock()
	// something wrong
	if err != nil {
		// lock aas with w lock
		aasRWLock.Lock()
		// check if other thread fix it already
		jwtToken, err = aasClient.GetUserToken(config.Configuration.Wla.APIUsername)
		// it is not fixed
		if err != nil {
			// these operation cannot be done in init() because it is not sure
			// if config.Configuration is loaded at that time
			aasClient.AddUser(config.Configuration.Wla.APIUsername, config.Configuration.Wla.APIPassword)
			err = aasClient.FetchAllTokens()
			if err != nil {
				return errors.Wrap(err, "clients/send_http_request.go:addJWTToken() Could not fetch token")
			}
		}
		aasRWLock.Unlock()
	}
	log.Debug("clients/send_http_request:addJWTToken() successfully added jwt bearer token")
	req.Header.Set("Authorization", "Bearer "+string(jwtToken))
	return nil
}

//SendRequest method is used to create an http client object and send the request to the server
func SendRequest(req *http.Request, insecureConnection bool) ([]byte, error) {
	log.Trace("clients/send_http_request:SendRequest() Entering")
	defer log.Trace("clients/send_http_request:SendRequest() Leaving")

	client, err := clients.HTTPClientWithCADir(consts.TrustedCaCertsDir)
	if err != nil {
		return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Failed to create http client")
	}
	err = addJWTToken(req)
	if err != nil {
		return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Failed to add JWT token")
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Error from response")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		// fetch token and try again
		aasRWLock.Lock()
		aasClient.FetchAllTokens()
		aasRWLock.Unlock()
		err = addJWTToken(req)
		if err != nil {
			return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Failed to add JWT token")
		}
		response, err = client.Do(req)
		if err != nil {
			return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Error from response")
		}
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Error from response")
	}

	//create byte array of HTTP response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Error from response")
	}
	log.Info("clients/send_http_request.go:SendRequest() Recieved the response successfully")
	return body, nil
}
