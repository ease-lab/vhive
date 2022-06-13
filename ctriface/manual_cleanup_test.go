// MIT License
//
// Copyright (c) 2020 Dmitrii Ustiugov, Plamen Petrov and EASE lab
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package ctriface

import (
	"context"
	"fmt"
	"github.com/ease-lab/vhive/snapshotting"
	"os"
	"sync"
	"testing"
	"time"

	ctrdlog "github.com/containerd/containerd/log"
	"github.com/containerd/containerd/namespaces"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestSnapLoad(t *testing.T) {
	// Need to clean up manually after this test because StopVM does not
	// work for stopping machines which are loaded from snapshots yet
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: ctrdlog.RFC3339NanoFixed,
		FullTimestamp:   true,
	})
	//log.SetReportCaller(true) // FIXME: make sure it's false unless debugging

	log.SetOutput(os.Stdout)

	log.SetLevel(log.InfoLevel)

	testTimeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(namespaces.WithNamespace(context.Background(), NamespaceName), testTimeout)
	defer cancel()

	orch := NewOrchestrator(
		"devmapper",
		"",
		"",
		"",
		1,
		WithTestModeOn(true),
		WithUPF(*isUPFEnabled),
		WithLazyMode(*isLazyMode),
		WithFullLocal(*isFullLocal),
	)
	defer orch.Cleanup()

	vmID := "1"
	snapId := vmID
	if *isFullLocal {
		snapId = "myrev-1"
	}

	_, _, err := orch.StartVM(ctx, vmID, TestImageName, 256, 1, *isSparseSnaps)
	require.NoError(t, err, "Failed to start VM")

	err = orch.PauseVM(ctx, vmID)
	require.NoError(t, err, "Failed to pause VM")

	snap := snapshotting.NewSnapshot(snapId, "/fccd/snapshots", TestImageName, 256, 1, *isSparseSnaps)
	if *isFullLocal {
		_ = snap.CreateSnapDir()
	}

	err = orch.CreateSnapshot(ctx, vmID, snap)
	require.NoError(t, err, "Failed to create snapshot of VM")

	_, err = orch.ResumeVM(ctx, vmID)
	require.NoError(t, err, "Failed to resume VM")

	if *isFullLocal {
		err = orch.StopSingleVM(ctx, vmID)
		require.NoError(t, err, "Failed to stop VM")
	} else {
		err = orch.OffloadVM(ctx, vmID)
		require.NoError(t, err, "Failed to offload VM")
	}

	_, _, err = orch.LoadSnapshot(ctx, vmID, snap)
	require.NoError(t, err, "Failed to load snapshot of VM")

	_, err = orch.ResumeVM(ctx, vmID)
	require.NoError(t, err, "Failed to resume VM")

	if *isFullLocal {
		err = orch.StopSingleVM(ctx, vmID)
		require.NoError(t, err, "Failed to stop VM")
	}
}

func TestSnapLoadMultiple(t *testing.T) {
	// Needs to be cleaned up manually.
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: ctrdlog.RFC3339NanoFixed,
		FullTimestamp:   true,
	})
	//log.SetReportCaller(true) // FIXME: make sure it's false unless debugging

	log.SetOutput(os.Stdout)

	log.SetLevel(log.InfoLevel)

	testTimeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(namespaces.WithNamespace(context.Background(), NamespaceName), testTimeout)
	defer cancel()

	orch := NewOrchestrator(
		"devmapper",
		"",
		"",
		"",
		10,
		WithTestModeOn(true),
		WithUPF(*isUPFEnabled),
		WithLazyMode(*isLazyMode),
		WithFullLocal(*isFullLocal),
	)
	defer orch.Cleanup()

	vmID := "3"
	snapId := vmID
	if *isFullLocal {
		snapId = "myrev-3"
	}

	_, _, err := orch.StartVM(ctx, vmID, TestImageName, 256, 1, *isSparseSnaps)
	require.NoError(t, err, "Failed to start VM")

	err = orch.PauseVM(ctx, vmID)
	require.NoError(t, err, "Failed to pause VM")

	snap := snapshotting.NewSnapshot(snapId, "/fccd/snapshots", TestImageName, 256, 1, *isSparseSnaps)
	if *isFullLocal {
		err = snap.CreateSnapDir()
	}
	require.NoError(t, err, "Failed to create directory for snapshot")

	err = orch.CreateSnapshot(ctx, vmID, snap)
	require.NoError(t, err, "Failed to create snapshot of VM")

	if *isFullLocal {
		// TODO: stopVM does not work without resuming
		_, err = orch.ResumeVM(ctx, vmID)
		require.NoError(t, err, "Failed to resume VM")

		err = orch.StopSingleVM(ctx, vmID)
		require.NoError(t, err, "Failed to stop VM")
	} else {
		err = orch.OffloadVM(ctx, vmID)
		require.NoError(t, err, "Failed to offload VM")
	}

	_, _, err = orch.LoadSnapshot(ctx, vmID, snap)
	require.NoError(t, err, "Failed to load snapshot of VM")

	_, err = orch.ResumeVM(ctx, vmID)
	require.NoError(t, err, "Failed to resume VM")

	if *isFullLocal {
		err = orch.StopSingleVM(ctx, vmID)
		require.NoError(t, err, "Failed to stop VM")
	} else {
		err = orch.OffloadVM(ctx, vmID)
		require.NoError(t, err, "Failed to offload VM")
	}

	_, _, err = orch.LoadSnapshot(ctx, vmID, snap)
	require.NoError(t, err, "Failed to load snapshot of VM")

	_, err = orch.ResumeVM(ctx, vmID)
	require.NoError(t, err, "Failed to resume VM, ")

	if *isFullLocal {
		err = orch.StopSingleVM(ctx, vmID)
		require.NoError(t, err, "Failed to stop VM")
	}
}

func TestParallelSnapLoad(t *testing.T) {
	// Needs to be cleaned up manually.
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: ctrdlog.RFC3339NanoFixed,
		FullTimestamp:   true,
	})
	//log.SetReportCaller(true) // FIXME: make sure it's false unless debugging

	log.SetOutput(os.Stdout)

	log.SetLevel(log.InfoLevel)

	testTimeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(namespaces.WithNamespace(context.Background(), NamespaceName), testTimeout)
	defer cancel()

	vmNum := 5
	vmIDBase := 6

	orch := NewOrchestrator(
		"devmapper",
		"",
		"",
		"",
		10,
		WithTestModeOn(true),
		WithUPF(*isUPFEnabled),
		WithLazyMode(*isLazyMode),
		WithFullLocal(*isFullLocal),
	)
	defer orch.Cleanup()

	// Pull image
	_, err := orch.GetImage(ctx, TestImageName)
	require.NoError(t, err, "Failed to pull image "+TestImageName)

	var vmGroup sync.WaitGroup
	for i := 0; i < vmNum; i++ {
		vmGroup.Add(1)
		go func(i int) {
			defer vmGroup.Done()
			vmID := fmt.Sprintf("%d", i+vmIDBase)
			snapId := vmID
			if *isFullLocal {
				snapId = fmt.Sprintf("myrev-%d", i+vmIDBase)
			}

			_, _, err := orch.StartVM(ctx, vmID, TestImageName, 256, 1, *isSparseSnaps)
			require.NoError(t, err, "Failed to start VM, "+vmID)

			err = orch.PauseVM(ctx, vmID)
			require.NoError(t, err, "Failed to pause VM, "+vmID)

			snap := snapshotting.NewSnapshot(snapId, "/fccd/snapshots", TestImageName, 256, 1, *isSparseSnaps)
			if *isFullLocal {
				err = snap.CreateSnapDir()
			}
			require.NoError(t, err, "Failed to create directory for snapshot")

			err = orch.CreateSnapshot(ctx, vmID, snap)
			require.NoError(t, err, "Failed to create snapshot of VM, "+vmID)

			if *isFullLocal {
				// TODO: stopVM does not work without resuming
				_, err = orch.ResumeVM(ctx, vmID)
				require.NoError(t, err, "Failed to resume VM")

				err = orch.StopSingleVM(ctx, vmID)
				require.NoError(t, err, "Failed to stop VM, "+vmID)
			} else {
				err = orch.OffloadVM(ctx, vmID)
				require.NoError(t, err, "Failed to offload VM, "+vmID)
			}

			_, _, err = orch.LoadSnapshot(ctx, vmID, snap)
			require.NoError(t, err, "Failed to load snapshot of VM, "+vmID)

			_, err = orch.ResumeVM(ctx, vmID)
			require.NoError(t, err, "Failed to resume VM, "+vmID)

			if *isFullLocal {
				err = orch.StopSingleVM(ctx, vmID)
				require.NoError(t, err, "Failed to stop VM, "+vmID)
			}
		}(i)
	}
	vmGroup.Wait()
}

func TestParallelPhasedSnapLoad(t *testing.T) {
	// Needs to be cleaned up manually.
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: ctrdlog.RFC3339NanoFixed,
		FullTimestamp:   true,
	})
	//log.SetReportCaller(true) // FIXME: make sure it's false unless debugging

	log.SetOutput(os.Stdout)

	log.SetLevel(log.InfoLevel)

	testTimeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(namespaces.WithNamespace(context.Background(), NamespaceName), testTimeout)
	defer cancel()

	vmNum := 10
	vmIDBase := 11

	orch := NewOrchestrator(
		"devmapper",
		"",
		"",
		"",
		10,
		WithTestModeOn(true),
		WithUPF(*isUPFEnabled),
		WithLazyMode(*isLazyMode),
		WithFullLocal(*isFullLocal),
	)
	defer orch.Cleanup()

	// Pull image
	_, err := orch.GetImage(ctx, TestImageName)
	require.NoError(t, err, "Failed to pull image "+TestImageName)

	{
		var vmGroup sync.WaitGroup
		for i := 0; i < vmNum; i++ {
			vmGroup.Add(1)
			go func(i int) {
				defer vmGroup.Done()
				vmID := fmt.Sprintf("%d", i+vmIDBase)
				_, _, err := orch.StartVM(ctx, vmID, TestImageName, 256, 1, *isSparseSnaps)
				require.NoError(t, err, "Failed to start VM, "+vmID)
			}(i)
		}
		vmGroup.Wait()
	}

	{
		var vmGroup sync.WaitGroup
		for i := 0; i < vmNum; i++ {
			vmGroup.Add(1)
			go func(i int) {
				defer vmGroup.Done()
				vmID := fmt.Sprintf("%d", i+vmIDBase)
				err := orch.PauseVM(ctx, vmID)
				require.NoError(t, err, "Failed to pause VM, "+vmID)
			}(i)
		}
		vmGroup.Wait()
	}

	{
		var vmGroup sync.WaitGroup
		for i := 0; i < vmNum; i++ {
			vmGroup.Add(1)
			go func(i int) {
				defer vmGroup.Done()
				vmID := fmt.Sprintf("%d", i+vmIDBase)
				snapId := vmID
				if *isFullLocal {
					snapId = fmt.Sprintf("myrev-%d", i+vmIDBase)
				}
				snap := snapshotting.NewSnapshot(snapId, "/fccd/snapshots", TestImageName, 256, 1, *isSparseSnaps)
				if *isFullLocal {
					err = snap.CreateSnapDir()
				}
				require.NoError(t, err, "Failed to create directory for snapshot")

				err = orch.CreateSnapshot(ctx, vmID, snap)
				require.NoError(t, err, "Failed to create snapshot of VM, "+vmID)
			}(i)
		}
		vmGroup.Wait()
	}

	{
		var vmGroup sync.WaitGroup
		for i := 0; i < vmNum; i++ {
			vmGroup.Add(1)
			go func(i int) {
				defer vmGroup.Done()
				vmID := fmt.Sprintf("%d", i+vmIDBase)
				if *isFullLocal {
					// TODO: stopVM does not work without resuming
					_, err = orch.ResumeVM(ctx, vmID)
					require.NoError(t, err, "Failed to resume VM")

					err = orch.StopSingleVM(ctx, vmID)
					require.NoError(t, err, "Failed to stop VM, "+vmID)
				} else {
					err = orch.OffloadVM(ctx, vmID)
					require.NoError(t, err, "Failed to offload VM, "+vmID)
				}
			}(i)
		}
		vmGroup.Wait()
	}

	{
		var vmGroup sync.WaitGroup
		for i := 0; i < vmNum; i++ {
			vmGroup.Add(1)
			go func(i int) {
				defer vmGroup.Done()
				vmID := fmt.Sprintf("%d", i+vmIDBase)
				snapId := vmID
				if *isFullLocal {
					snapId = fmt.Sprintf("myrev-%d", i+vmIDBase)
				}
				snap := snapshotting.NewSnapshot(snapId, "/fccd/snapshots", TestImageName, 256, 1, *isSparseSnaps)
				_, _, err := orch.LoadSnapshot(ctx, vmID, snap)
				require.NoError(t, err, "Failed to load snapshot of VM, "+vmID)
			}(i)
		}
		vmGroup.Wait()
	}

	{
		var vmGroup sync.WaitGroup
		for i := 0; i < vmNum; i++ {
			vmGroup.Add(1)
			go func(i int) {
				defer vmGroup.Done()
				vmID := fmt.Sprintf("%d", i+vmIDBase)
				_, err := orch.ResumeVM(ctx, vmID)
				require.NoError(t, err, "Failed to resume VM, "+vmID)
			}(i)
		}
		vmGroup.Wait()
	}

	if *isFullLocal {
		{
			var vmGroup sync.WaitGroup
			for i := 0; i < vmNum; i++ {
				vmGroup.Add(1)
				go func(i int) {
					defer vmGroup.Done()
					vmID := fmt.Sprintf("%d", i+vmIDBase)
					err := orch.StopSingleVM(ctx, vmID)
					require.NoError(t, err, "Failed to stop VM, "+vmID)
				}(i)
			}
			vmGroup.Wait()
		}
	}
}
