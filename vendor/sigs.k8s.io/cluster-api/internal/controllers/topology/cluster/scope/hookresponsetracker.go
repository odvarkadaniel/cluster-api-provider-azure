/*
Copyright 2021 The Kubernetes Authors.

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

package scope

import (
	"fmt"
	"strings"
	"time"

	runtimecatalog "sigs.k8s.io/cluster-api/exp/runtime/catalog"
	runtimehooksv1 "sigs.k8s.io/cluster-api/exp/runtime/hooks/api/v1alpha1"
)

// HookResponseTracker is a helper to capture the responses of the various lifecycle hooks.
type HookResponseTracker struct {
	responses map[string]runtimehooksv1.ResponseObject
}

// NewHookResponseTracker returns a new HookResponseTracker.
func NewHookResponseTracker() *HookResponseTracker {
	return &HookResponseTracker{
		responses: map[string]runtimehooksv1.ResponseObject{},
	}
}

// Add add the response of a hook to the tracker.
func (h *HookResponseTracker) Add(hook runtimecatalog.Hook, response runtimehooksv1.ResponseObject) {
	hookName := runtimecatalog.HookName(hook)
	h.responses[hookName] = response
}

// AggregateRetryAfter calculates the lowest non-zero retryAfterSeconds time from all the tracked responses.
func (h *HookResponseTracker) AggregateRetryAfter() time.Duration {
	res := int32(0)
	for _, resp := range h.responses {
		if retryResponse, ok := resp.(runtimehooksv1.RetryResponseObject); ok {
			res = lowestNonZeroRetryAfterSeconds(res, retryResponse.GetRetryAfterSeconds())
		}
	}
	return time.Duration(res) * time.Second
}

// AggregateMessage returns a human friendly message about the blocking status of hooks.
func (h *HookResponseTracker) AggregateMessage() string {
	blockingHooks := map[string]string{}
	for hook, resp := range h.responses {
		if retryResponse, ok := resp.(runtimehooksv1.RetryResponseObject); ok {
			if retryResponse.GetRetryAfterSeconds() != 0 {
				blockingHooks[hook] = resp.GetMessage()
			}
		}
	}
	if len(blockingHooks) == 0 {
		return ""
	}

	hookAndMessages := []string{}
	for hook, message := range blockingHooks {
		hookAndMessages = append(hookAndMessages, fmt.Sprintf("hook %q is blocking: %s", hook, message))
	}
	return strings.Join(hookAndMessages, "; ")
}

func lowestNonZeroRetryAfterSeconds(i, j int32) int32 {
	if i == 0 {
		return j
	}
	if j == 0 {
		return i
	}
	if i < j {
		return i
	}
	return j
}
