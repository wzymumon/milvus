// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package querycoordv2

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotHealthy = errors.New("NotHealthy")

	// Node Availability
	ErrLackSegment           = errors.New("LackSegment")
	ErrNodeOffline           = errors.New("NodeOffline")
	ErrNodeHeartbeatOutdated = errors.New("NodeHeartbeatOutdated")
)

func WrapErrLackSegment(segmentID int64) error {
	return fmt.Errorf("%w(segmentID=%v)", ErrLackSegment, segmentID)
}

func WrapErrNodeOffline(nodeID int64) error {
	return fmt.Errorf("%w(nodeID=%v)", ErrNodeOffline, nodeID)
}

func WrapErrNodeHeartbeatOutdated(nodeID int64, lastHeartbeat time.Time) error {
	return fmt.Errorf("%w(nodeID=%v, lastHeartbeat=%v)",
		ErrNodeHeartbeatOutdated,
		nodeID,
		lastHeartbeat,
	)
}
