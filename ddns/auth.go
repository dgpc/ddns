/*
   Copyright 2020 Google LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       https://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package ddns

import (
	"bytes"
	"context"

	"cloud.google.com/go/datastore"
	"golang.org/x/crypto/scrypt"
)

const (
	DomainEntity = "Domain"
)

type Domain struct {
	Salt []byte `datastore:"salt,noindex"`
	Key  []byte `datastore:"key,noindex"`
}

func (srv *server) authorized(ctx context.Context, domain, token string) (bool, error) {
	var domainEntity Domain
	domainKey := datastore.NameKey(DomainEntity, domain, nil)
	err := srv.dsCli.Get(ctx, domainKey, &domainEntity)
	if err != nil {
		return false, err
	}

	key, err := scrypt.Key([]byte(token), domainEntity.Salt, 32768, 8, 1, 32)
	if err != nil {
		return false, err
	}

	if bytes.Equal(key, domainEntity.Key) {
		return true, nil
	}
	return false, nil
}
