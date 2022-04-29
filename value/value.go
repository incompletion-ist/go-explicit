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
	stored  T
	set     bool
	mu      sync.Mutex
	waiting chan bool
}

// initWaiting initializes the Explicit.waiting channel with a size of 0. This
// makes it blocking even on the first write.
//
// initWaiting acquires the lock from start to finish.
func (v *Value[T]) initWaiting() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.waiting == nil {
		v.waiting = make(chan bool, 0)
	}
}

// Set sets the value explicitly.
//
// Set acquires the lock prior to:
//
// * setting the value
//
// * draining the waiting channel
func (v *Value[T]) Set(storeValue T) {
	v.initWaiting()

	// lock so nothing can add to waiting until after it's drained
	v.mu.Lock()
	defer v.mu.Unlock()

	v.stored = storeValue
	v.set = true

	// drain waiting
	for {
		select {
		default:
			return
		case <-v.waiting:
		}
	}
}

// Get returns the stored value.
func (v *Value[T]) Get() T {
	return v.stored
}

// GetOk returns the stored value and a boolean indicating if the value
// was explicitly set.
func (value *Value[T]) GetOk() (T, bool) {
	return value.stored, value.set
}

// GetWait returns the stored value, but blocks until the value is next
// explicitly set, or the Context is cancelled. If returning after Context
// cancellation, the last known stored value will be returned. This may be
// the zero value of the type, if the value was never set.
func (v *Value[T]) GetWait(ctx context.Context) (T, error) {
	v.initWaiting()

	// acquiring the lock ensures the waiting channel isn't currently being drained.
	// it is immediately unlocked again.
	v.mu.Lock()
	v.mu.Unlock()

	select {
	case <-ctx.Done():
		return v.Get(), ctx.Err()
	case v.waiting <- true:
		return v.Get(), nil
	}
}

// New returns a new Explicit with its value explicitly set.
func New[T any](storeValue T) *Value[T] {
	var newValue Value[T]

	newValue.Set(storeValue)

	return &newValue
}
