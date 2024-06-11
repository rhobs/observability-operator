/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"crypto/sha256"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func HashOfTLSSecret(
	secretName string,
	secretKey string,
	namespace string,
	client client.Client,
) (string, error) {
	var secret v1.Secret
	err := client.Get(context.Background(), types.NamespacedName{
		Name:      secretName,
		Namespace: namespace,
	}, &secret)
	if err != nil {
		return "", fmt.Errorf("Couldn't get TLS secret %s: %s", secretName, err)
	}

	hash := sha256.Sum256(secret.Data[secretKey])
	return rand.SafeEncodeString(fmt.Sprint(hash)), nil
}
