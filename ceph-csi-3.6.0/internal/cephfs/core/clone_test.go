/*
Copyright 2021 The Ceph-CSI Authors.

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

package core

import (
	"testing"

	cerrors "github.com/ceph/ceph-csi/internal/cephfs/errors"

	"github.com/stretchr/testify/assert"
)

func TestCloneStateToError(t *testing.T) {
	t.Parallel()
	errorState := make(map[cephFSCloneState]error)
	errorState[CephFSCloneComplete] = nil
	errorState[CephFSCloneError] = cerrors.ErrInvalidClone
	errorState[CephFSCloneInprogress] = cerrors.ErrCloneInProgress
	errorState[CephFSClonePending] = cerrors.ErrClonePending
	errorState[CephFSCloneFailed] = cerrors.ErrCloneFailed

	for state, err := range errorState {
		assert.Equal(t, state.toError(), err)
	}
}
