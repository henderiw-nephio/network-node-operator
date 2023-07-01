/*
Copyright 2022 Nokia.

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

package config

import (
	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	Name            string            `json:"name"`
	License         string            `json:"license,omitempty"`
	Image           string            `json:"image,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Sysctls         map[string]string `json:"sysctls,omitempty"`
	User            string            `json:"user,omitempty"`
	Entrypoint      string            `json:"entrypoint,omitempty"`
	Cmd             string            `json:"cmd,omitempty"`
	// Exec is a list of commands to execute inside the container backing the node.
	Exec []string          `json:"exec,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
	// Bind mounts strings (src:dest:options).
	Binds []string `json:"binds,omitempty"`
}
