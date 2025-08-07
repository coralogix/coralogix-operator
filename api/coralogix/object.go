// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coralogix

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Object is an interface that extends the client.Object interface with Coralogix-specific methods.
type Object interface {
	client.Object
	HasIDInStatus() bool
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
	GetPrintableStatus() string
	SetPrintableStatus(printableStatus string)
}
