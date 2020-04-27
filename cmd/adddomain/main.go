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

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/ssh/terminal"

	"ddns/ddns"
	"ddns/third_party/netutil"
)

func main() {
	domain, token := input()

	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		log.Fatalf("failed to generate salt: %v\n", err)
	}

	key, err := scrypt.Key([]byte(token), salt, 32768, 8, 1, 32)
	if err != nil {
		log.Fatalf("failed to derive scrypt key from token: %v\n", err)
	}

	domainEntity := &ddns.Domain{
		Salt: salt,
		Key:  key,
    }
    domainKey := datastore.NameKey(ddns.DomainEntity, domain, nil)

	dsCli, err := datastore.NewClient(context.Background(), ddns.Project)
	if err != nil {
		log.Fatalf("failed to create datastore client: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = dsCli.Put(ctx, domainKey, domainEntity)
	if err != nil {
		log.Fatalf("failed to write to datastore: %v\n", err)
	}

	fmt.Printf("successfully stored token for domain %q\n", domain)
}

func input() (string, string) {
	fmt.Print("Enter Domain: ")
	reader := bufio.NewReader(os.Stdin)
	domain, _ := reader.ReadString('\n')
	domain = netutil.AbsDomainName([]byte(strings.TrimSpace(domain)))
	if !netutil.IsDomainName(domain) {
		log.Fatalf("%q is not a valid domain name\n", domain)
	}

	fmt.Print("Enter Token: ")
	bytePassword, err := terminal.ReadPassword(0)
	if err != nil {
		log.Fatalf("failed to read token: %v\n", err)
	}
	fmt.Printf("\n")
	password := string(bytePassword)

	return domain, strings.TrimSpace(password)
}