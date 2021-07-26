/*
 * Copyright (C) 2021 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package keyprovider_grpc

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"github.com/containers/ocicrypt/keywrap/keyprovider"
	keyproviderpb "github.com/containers/ocicrypt/utils/keyprovider"
	ocicryptKeyprovider "github.com/intel-secl/intel-secl/v4/pkg/model/ocicrypt"
	"github.com/pkg/errors"
	cLog "intel/isecl/lib/common/v4/log"
	"intel/isecl/wlagent/v4/flavor"
	"intel/isecl/wlagent/v4/util"
	"sync"
)

type GRPCServer struct {
	keyproviderpb.UnimplementedKeyProviderServiceServer
}

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()
var mtx sync.Mutex

func (*GRPCServer) UnWrapKey(ctx context.Context, request *keyproviderpb.KeyProviderKeyWrapProtocolInput) (*keyproviderpb.KeyProviderKeyWrapProtocolOutput, error) {
	log.Trace("keyprovider-grpc/server:UnWrapKey() Entering")
	defer log.Trace("keyprovider-grpc/server:UnWrapKey() Leaving")

	var keyP keyprovider.KeyProviderKeyWrapProtocolInput
	err := json.Unmarshal(request.KeyProviderKeyWrapProtocolInput, &keyP)
	if err != nil {
		return nil, errors.Wrap(err, "Error while unmarshalling KeyProviderKeyWrapProtocolInput")
	}

	apkt := ocicryptKeyprovider.AnnotationPacket{}
	err = json.Unmarshal(keyP.KeyUnwrapParams.Annotation, &apkt)
	if err != nil {
		return nil, errors.Wrap(err, "Error while unmarshalling annotation packet")
	}

	wrappedKey, returnCode := flavor.RetrieveKeyWithURL(apkt.KeyUrl)
	if !returnCode {
		return nil, errors.New("Error while retrieving wrapped kek")
	}
	symKey, err := util.UnwrapKey(wrappedKey)
	if err != nil {
		return nil, errors.Wrap(err, "Error while unwrapping kek")
	}

	unwrappedKey, err := aesDecrypt(symKey, apkt.WrappedKey)
	if err != nil {
		return nil, errors.Wrap(err, "Error while decrypting key")
	}

	keyProviderOutput := keyprovider.KeyProviderKeyWrapProtocolOutput{
		KeyUnwrapResults: keyprovider.KeyUnwrapResults{OptsData: unwrappedKey},
	}
	serializedKeyProviderOutput, err := json.Marshal(keyProviderOutput)
	if err != nil {
		return nil, errors.Wrap(err, "Error while serializing KeyProviderKeyWrapProtocolOutput")
	}
	k := keyproviderpb.KeyProviderKeyWrapProtocolOutput{}
	k.KeyProviderKeyWrapProtocolOutput = serializedKeyProviderOutput

	return &k, nil
}

func aesDecrypt(kek []byte, symKey []byte) ([]byte, error) {
	log.Trace("keyprovider-grpc/server:aesDecrypt() Entering")
	defer log.Trace("keyprovider-grpc/server:aesDecrypt() Leaving")

	if len(kek) != 32 {
		return nil, errors.New("Expected 256 bit key")
	}

	var aesp ocicryptKeyprovider.AesPacket
	err := json.Unmarshal(symKey, &aesp)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal aes packet")
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	key, err := aesgcm.Open(nil, aesp.Nonce, aesp.Ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// No operation
func (*GRPCServer) WrapKey(ctx context.Context, request *keyproviderpb.KeyProviderKeyWrapProtocolInput) (*keyproviderpb.KeyProviderKeyWrapProtocolOutput, error) {

	return nil, nil
}
