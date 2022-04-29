// Copyright 2022 Micah Kemp
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package value provides the Value type, which enables explicitly setting values.
package value

import (
	"context"
	"sync"
)

// Value is a generic type that represents explicitly settable values.
type Value[T any] struct {
	value   T
	set     bool
	mu      sync.Mutex
	waiting chan bool
}

// initWaiting initializes the Explicit.waiting channel with a size of 0. This
// makes it blocking even on the first write.
//
// initWaiting acquires the lock from start to finish.
func (configValue *Value[T]) initWaiting() {
	configValue.mu.Lock()
	defer configValue.mu.Unlock()

	if configValue.waiting == nil {
		configValue.waiting = make(chan bool, 0)
	}
}

// Set sets the value explicitly.
//
// Set acquires the lock prior to:
//
// * setting the value
//
// * draining the waiting channel
func (configValue *Value[T]) Set(value T) {
	configValue.initWaiting()

	// lock so nothing can add to waiting until after it's drained
	configValue.mu.Lock()
	defer configValue.mu.Unlock()

	configValue.value = value
	configValue.set = true

	// drain waiting
	for {
		select {
		default:
			return
		case <-configValue.waiting:
		}
	}
}

// Value returns the stored value.
func (configValue *Value[T]) Value() T {
	return configValue.value
}

// ValueOk returns the stored value and a boolean indicating if the value
// was explicitly set.
func (configValue *Value[T]) ValueOk() (T, bool) {
	return configValue.value, configValue.set
}

// ValueWait returns the stored value, but blocks until the value is next
// explicitly set, or the Context is cancelled. If returning after Context
// cancellation, the last known stored value will be returned. This may be
// the zero value of the type, if the value was never set.
func (configValue *Value[T]) ValueWait(ctx context.Context) (T, error) {
	configValue.initWaiting()

	// acquiring the lock ensures the waiting channel isn't currently being drained.
	// it is immediately unlocked again.
	configValue.mu.Lock()
	configValue.mu.Unlock()

	select {
	case <-ctx.Done():
		return configValue.Value(), ctx.Err()
	case configValue.waiting <- true:
		return configValue.Value(), nil
	}
}

// New returns a new Explicit with its value explicitly set.
func New[T any](value T) *Value[T] {
	var newExplicit Value[T]

	newExplicit.Set(value)

	return &newExplicit
}
