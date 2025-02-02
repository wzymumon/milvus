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

package grpcindexcoordclient

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/milvus-io/milvus-proto/go-api/commonpb"
	"github.com/milvus-io/milvus-proto/go-api/milvuspb"
	"github.com/milvus-io/milvus/internal/log"
	"github.com/milvus-io/milvus/internal/proto/indexpb"
	"github.com/milvus-io/milvus/internal/proto/internalpb"
	"github.com/milvus-io/milvus/internal/util/commonpbutil"
	"github.com/milvus-io/milvus/internal/util/funcutil"
	"github.com/milvus-io/milvus/internal/util/grpcclient"
	"github.com/milvus-io/milvus/internal/util/paramtable"
	"github.com/milvus-io/milvus/internal/util/sessionutil"
	"github.com/milvus-io/milvus/internal/util/typeutil"
)

var Params *paramtable.ComponentParam = paramtable.Get()

// Client is the grpc client of IndexCoord.
type Client struct {
	grpcClient grpcclient.GrpcClient[indexpb.IndexCoordClient]
	sess       *sessionutil.Session
}

// NewClient creates a new IndexCoord client.
func NewClient(ctx context.Context, metaRoot string, etcdCli *clientv3.Client) (*Client, error) {
	sess := sessionutil.NewSession(ctx, metaRoot, etcdCli)
	if sess == nil {
		err := fmt.Errorf("new session error, maybe can not connect to etcd")
		log.Debug("IndexCoordClient NewClient failed", zap.Error(err))
		return nil, err
	}
	ClientParams := &Params.IndexCoordGrpcClientCfg
	client := &Client{
		grpcClient: &grpcclient.ClientBase[indexpb.IndexCoordClient]{
			ClientMaxRecvSize:      ClientParams.ClientMaxRecvSize.GetAsInt(),
			ClientMaxSendSize:      ClientParams.ClientMaxSendSize.GetAsInt(),
			DialTimeout:            ClientParams.DialTimeout.GetAsDuration(time.Millisecond),
			KeepAliveTime:          ClientParams.KeepAliveTime.GetAsDuration(time.Millisecond),
			KeepAliveTimeout:       ClientParams.KeepAliveTimeout.GetAsDuration(time.Millisecond),
			RetryServiceNameConfig: "milvus.proto.index.IndexCoord",
			MaxAttempts:            ClientParams.MaxAttempts.GetAsInt(),
			InitialBackoff:         float32(ClientParams.InitialBackoff.GetAsFloat()),
			MaxBackoff:             float32(ClientParams.MaxBackoff.GetAsFloat()),
			BackoffMultiplier:      float32(ClientParams.BackoffMultiplier.GetAsFloat()),
		},
		sess: sess,
	}
	client.grpcClient.SetRole(typeutil.IndexCoordRole)
	client.grpcClient.SetGetAddrFunc(client.getIndexCoordAddr)
	client.grpcClient.SetNewGrpcClientFunc(client.newGrpcClient)

	return client, nil
}

// Init initializes IndexCoord's grpc client.
func (c *Client) Init() error {
	return nil
}

// Start starts IndexCoord's client service. But it does nothing here.
func (c *Client) Start() error {
	return nil
}

// Stop stops IndexCoord's grpc client.
func (c *Client) Stop() error {
	return c.grpcClient.Close()
}

// Register dummy
func (c *Client) Register() error {
	return nil
}

func (c *Client) getIndexCoordAddr() (string, error) {
	key := c.grpcClient.GetRole()
	msess, _, err := c.sess.GetSessions(key)
	if err != nil {
		log.Debug("IndexCoordClient GetSessions failed", zap.Any("key", key), zap.Error(err))
		return "", err
	}
	log.Debug("IndexCoordClient GetSessions success", zap.Any("key", key), zap.Any("msess", msess))
	ms, ok := msess[key]
	if !ok {
		log.Debug("IndexCoordClient msess key not existed", zap.Any("key", key), zap.Any("len of msess", len(msess)))
		return "", fmt.Errorf("find no available indexcoord, check indexcoord state")
	}
	return ms.Address, nil
}

// newGrpcClient create a new grpc client of IndexCoord.
func (c *Client) newGrpcClient(cc *grpc.ClientConn) indexpb.IndexCoordClient {
	return indexpb.NewIndexCoordClient(cc)
}

// GetComponentStates gets the component states of IndexCoord.
func (c *Client) GetComponentStates(ctx context.Context) (*milvuspb.ComponentStates, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.GetComponentStates(ctx, &milvuspb.GetComponentStatesRequest{})
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*milvuspb.ComponentStates), err
}

// GetStatisticsChannel gets the statistics channel of IndexCoord.
func (c *Client) GetStatisticsChannel(ctx context.Context) (*milvuspb.StringResponse, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.GetStatisticsChannel(ctx, &internalpb.GetStatisticsChannelRequest{})
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*milvuspb.StringResponse), err
}

// CreateIndex sends the build index request to IndexCoord.
func (c *Client) CreateIndex(ctx context.Context, req *indexpb.CreateIndexRequest) (*commonpb.Status, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.CreateIndex(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*commonpb.Status), err
}

// GetIndexState gets the index states from IndexCoord.
func (c *Client) GetIndexState(ctx context.Context, req *indexpb.GetIndexStateRequest) (*indexpb.GetIndexStateResponse, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.GetIndexState(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*indexpb.GetIndexStateResponse), err
}

// GetSegmentIndexState gets the index states from IndexCoord.
func (c *Client) GetSegmentIndexState(ctx context.Context, req *indexpb.GetSegmentIndexStateRequest) (*indexpb.GetSegmentIndexStateResponse, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.GetSegmentIndexState(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*indexpb.GetSegmentIndexStateResponse), err
}

// GetIndexInfos gets the index file paths from IndexCoord.
func (c *Client) GetIndexInfos(ctx context.Context, req *indexpb.GetIndexInfoRequest) (*indexpb.GetIndexInfoResponse, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.GetIndexInfos(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*indexpb.GetIndexInfoResponse), err
}

// DescribeIndex describe the index info of the collection.
func (c *Client) DescribeIndex(ctx context.Context, req *indexpb.DescribeIndexRequest) (*indexpb.DescribeIndexResponse, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.DescribeIndex(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*indexpb.DescribeIndexResponse), err
}

// GetIndexBuildProgress describe the progress of the index.
func (c *Client) GetIndexBuildProgress(ctx context.Context, req *indexpb.GetIndexBuildProgressRequest) (*indexpb.GetIndexBuildProgressResponse, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.GetIndexBuildProgress(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*indexpb.GetIndexBuildProgressResponse), err
}

// DropIndex sends the drop index request to IndexCoord.
func (c *Client) DropIndex(ctx context.Context, req *indexpb.DropIndexRequest) (*commonpb.Status, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.DropIndex(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*commonpb.Status), err
}

// ShowConfigurations gets specified configurations para of IndexCoord
func (c *Client) ShowConfigurations(ctx context.Context, req *internalpb.ShowConfigurationsRequest) (*internalpb.ShowConfigurationsResponse, error) {
	req = typeutil.Clone(req)
	commonpbutil.UpdateMsgBase(
		req.GetBase(),
		commonpbutil.FillMsgBaseFromClient(paramtable.GetNodeID(), commonpbutil.WithTargetID(c.sess.ServerID)),
	)
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.ShowConfigurations(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}

	return ret.(*internalpb.ShowConfigurationsResponse), err
}

// GetMetrics gets the metrics info of IndexCoord.
func (c *Client) GetMetrics(ctx context.Context, req *milvuspb.GetMetricsRequest) (*milvuspb.GetMetricsResponse, error) {
	req = typeutil.Clone(req)
	commonpbutil.UpdateMsgBase(
		req.GetBase(),
		commonpbutil.FillMsgBaseFromClient(paramtable.GetNodeID(), commonpbutil.WithTargetID(c.sess.ServerID)),
	)
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.GetMetrics(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*milvuspb.GetMetricsResponse), err
}

func (c *Client) CheckHealth(ctx context.Context, req *milvuspb.CheckHealthRequest) (*milvuspb.CheckHealthResponse, error) {
	ret, err := c.grpcClient.ReCall(ctx, func(client indexpb.IndexCoordClient) (any, error) {
		if !funcutil.CheckCtxValid(ctx) {
			return nil, ctx.Err()
		}
		return client.CheckHealth(ctx, req)
	})
	if err != nil || ret == nil {
		return nil, err
	}
	return ret.(*milvuspb.CheckHealthResponse), err
}
