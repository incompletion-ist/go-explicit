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

package value_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.incompletion.ist/explicit/value"
)

type monitoredValues struct {
	temperature value.Value[float32]
	humidity    value.Value[float32]
}

func repeatUntilCancel(ctx context.Context, fn func()) {
	backgroundWG := sync.WaitGroup{}
	defer backgroundWG.Wait()

	backgroundWG.Add(1)
	go func() {
		backgroundWG.Done()
		for {
			select {
			default:
				fn()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func ExampleValue() {
	mainWG := sync.WaitGroup{}
	defer mainWG.Wait()

	myConfig := monitoredValues{}

	ctx, ctxCancel := context.WithCancel(context.Background())
	repeatUntilCancel(ctx, func() {
		if value, err := myConfig.temperature.GetWait(ctx); err == nil {
			fmt.Printf("new temperature value: %v\n", value)
		}
	})

	time.Sleep(50 * time.Millisecond)
	myConfig.temperature.Set(10)
	time.Sleep(50 * time.Millisecond)
	myConfig.temperature.Set(11)
	time.Sleep(50 * time.Millisecond)

	if v, err := myConfig.humidity.GetWaitTrigger(ctx, func() {
		myConfig.humidity.Set(12)
	}); err == nil {
		fmt.Printf("new triggered humidity value: %v\n", v)
	}
	time.Sleep(50 * time.Millisecond)

	ctxCancel()
	mainWG.Wait()

	// Output: new temperature value: 10
	// new temperature value: 11
	// new triggered humidity value: 12
}
