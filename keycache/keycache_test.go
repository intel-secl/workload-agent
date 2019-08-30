/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package keycache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAndStore(t *testing.T) {
	assert := assert.New(t)
	cache := NewCache()
	key := Key{"keyid", []byte{0, 1, 2, 3}}
	cache.Store("foobar", key)
	actual, exists := cache.Get("foobar")
	assert.True(exists)
	assert.Equal(key, actual)
}

func TestGetNone(t *testing.T) {
	assert := assert.New(t)
	cache := NewCache()
	actual, exists := cache.Get("foobar")
	assert.False(exists)
	assert.Zero(actual)
}

func TestOverwrite(t *testing.T) {
	assert := assert.New(t)
	cache := NewCache()
	key1 := Key{"foo", []byte{0, 1, 2, 3}}
	key2 := Key{"bar", []byte{4, 5, 6, 7}}
	cache.Store("foobar", key1)
	cache.Store("foobar", key2)
	actual, exists := cache.Get("foobar")
	assert.True(exists)
	assert.Equal(key2, actual)
}
