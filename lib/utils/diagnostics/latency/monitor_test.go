// Copyright 2023 Gravitational, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package latency

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gravitational/teleport/lib/utils"
)

func TestMain(m *testing.M) {
	utils.InitLoggerForTests()

	os.Exit(m.Run())
}

type fakePinger struct {
	clock   clockwork.FakeClock
	latency time.Duration
	pingC   chan struct{}
}

func (f fakePinger) Ping(ctx context.Context) error {
	f.clock.Sleep(f.latency)
	select {
	case f.pingC <- struct{}{}:
	default:
	}
	return nil
}

type fakeReporter struct {
	statsC chan Statistics
}

func (f fakeReporter) Report(ctx context.Context, stats Statistics) error {
	select {
	case f.statsC <- stats:
	default:
	}

	return nil
}

func TestMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	clock := clockwork.NewFakeClock()

	reporter := fakeReporter{
		statsC: make(chan Statistics, 20),
	}

	const pingLatency = 2 * time.Second
	clientPinger := fakePinger{clock: clock, latency: 2 * pingLatency, pingC: make(chan struct{}, 1)}
	serverPinger := fakePinger{clock: clock, latency: pingLatency, pingC: make(chan struct{}, 1)}

	monitor, err := NewMonitor(MonitorConfig{
		ClientPinger:          clientPinger,
		ServerPinger:          serverPinger,
		Reporter:              reporter,
		Clock:                 clock,
		PingInterval:          20 * time.Second,
		InitialPingInterval:   20 * time.Second,
		ReportInterval:        30 * time.Second,
		InitialReportInterval: 30 * time.Second,
	})
	require.NoError(t, err, "creating monitor")

	// Start the monitor in a goroutine since it's a blocking loop. The context
	// is terminated when the test ends which will terminate the monitor.
	go func() {
		monitor.Run(ctx)
	}()

	// Validate that stats are initially 0 for both legs.
	stats := monitor.GetStats()
	assert.Equal(t, Statistics{}, monitor.GetStats(), "expected initial latency stats to be zero got %v", stats)

	// Simulate a few ping loops to validate the pingers are activated appropriately.
	for i := 0; i < 10; i++ {
		// Wait for the ping timers and reporting timer to block.
		clock.BlockUntil(3)
		// Advance the clock enough to trigger a ping.
		clock.Advance(monitor.pingInterval)

		pingTimeout := time.After(15 * time.Second)
		// Wait for both pings to return a response.
		for i := 0; i < 2; i++ {
			// Wait for the fake pingers to sleep and the reporting timer to block.
			clock.BlockUntil(3)
			// Advance the clock in intervals of 5s to wake up the pingers one at a time.
			// This works because one fake pinger is configured to have double the latency of
			// the other.
			clock.Advance(pingLatency)

			select {
			case <-clientPinger.pingC:
			case <-serverPinger.pingC:
			case <-pingTimeout:
				t.Fatal("ping never processed")
			}
		}

		// Wait for the ping timers and reporting timer to block.
		clock.BlockUntil(3)
		// Advance the clock enough to trigger a report.
		clock.Advance(monitor.reportInterval - monitor.pingInterval - (pingLatency * 2))

		// Validate the stats reported
		reportTimeout := time.After(15 * time.Second)
		select {
		case reported := <-reporter.statsC:
			current := monitor.GetStats()
			assert.NotEqual(t, stats, reported, "expected reported stats not to be empty")
			assert.NotEqual(t, stats, current, "expected retrieved stats not to be empty")
			assert.Equal(t, reported, current, "expected reported and retrieved stats to match")
		case <-reportTimeout:
			t.Fatal("latency stats never received")
		}
	}
}
