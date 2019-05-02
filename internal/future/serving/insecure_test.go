/*
Copyright 2019 Google LLC
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serving

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"

	"context"

	shellTesting "github.com/GoogleCloudPlatform/open-match/internal/future/testing"
	netlistenerTesting "github.com/GoogleCloudPlatform/open-match/internal/util/netlistener/testing"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestInsecureStartStop(t *testing.T) {
	assert := assert.New(t)
	grpcLh := netlistenerTesting.MustListen()
	httpLh := netlistenerTesting.MustListen()
	ff := shellTesting.NewFakeFrontend()

	params := NewParamsFromListeners(grpcLh, httpLh)
	params.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServer(s, ff)
	}, pb.RegisterFrontendHandlerFromEndpoint)
	s := newInsecureServer(grpcLh, httpLh)
	defer s.stop()
	waitForStart, err := s.start(params)
	assert.Nil(err)
	waitForStart()

	conn, err := grpc.Dial(fmt.Sprintf(":%d", grpcLh.Number()), grpc.WithInsecure())
	assert.Nil(err)
	ctx := context.Background()
	feClient := pb.NewFrontendClient(conn)
	grpcResp, err := feClient.CreatePlayer(ctx, &pb.CreatePlayerRequest{})
	assert.Nil(err)
	assert.NotNil(grpcResp)

	endpoint := fmt.Sprintf("http://localhost:%d", httpLh.Number())
	httpClient := &http.Client{
		Timeout: time.Second,
	}
	httpReq, err := http.NewRequest(http.MethodPut, endpoint+"/v1/frontend/players", strings.NewReader("{}"))
	assert.Nil(err)
	assert.NotNil(httpReq)
	httpResp, err := httpClient.Do(httpReq)
	assert.Nil(err)
	assert.NotNil(httpResp)
	defer func() {
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(httpResp.Body)
	assert.Nil(err)
	assert.Equal(200, httpResp.StatusCode)
	assert.Equal("{}", string(body))

	httpReq, err = http.NewRequest(http.MethodGet, endpoint+"/healthz", nil)
	assert.Nil(err)

	httpResp, err = httpClient.Do(httpReq)
	assert.Nil(err)
	assert.NotNil(httpResp)
	body, err = ioutil.ReadAll(httpResp.Body)
	assert.Nil(err)
	assert.Equal(200, httpResp.StatusCode)
	assert.Equal("ok", string(body))

	s.stop()
}