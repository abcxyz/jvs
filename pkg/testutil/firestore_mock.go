// Copyright 2022 Google LLC
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

// Package testutil provides utilities that are intended to enable easier
// and more concise writing of unit test code.
package testutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/golang/protobuf/ptypes"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/api/option"
	firestorepb "google.golang.org/genproto/googleapis/firestore/v1"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

var (
	_ = io.EOF
	_ = ptypes.MarshalAny
	_ status.Status
)

type MockFirestoreServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	firestorepb.UnimplementedFirestoreServer

	Reqs []proto.Message

	// If set, all calls return this error.
	Error error

	// responses to return if err == nil
	Resps []proto.Message
}

func (s *MockFirestoreServer) GetDocument(ctx context.Context, req *firestorepb.GetDocumentRequest) (*firestorepb.Document, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	return s.Resps[0].(*firestorepb.Document), nil //nolint, only for testing
}

func (s *MockFirestoreServer) ListDocuments(ctx context.Context, req *firestorepb.ListDocumentsRequest) (*firestorepb.ListDocumentsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*firestorepb.ListDocumentsResponse), nil
}

func (s *MockFirestoreServer) CreateDocument(ctx context.Context, req *firestorepb.CreateDocumentRequest) (*firestorepb.Document, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*firestorepb.Document), nil
}

func (s *MockFirestoreServer) UpdateDocument(ctx context.Context, req *firestorepb.UpdateDocumentRequest) (*firestorepb.Document, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*firestorepb.Document), nil
}

func (s *MockFirestoreServer) DeleteDocument(ctx context.Context, req *firestorepb.DeleteDocumentRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*emptypb.Empty), nil
}

func (s *MockFirestoreServer) BatchGetDocuments(req *firestorepb.BatchGetDocumentsRequest, stream firestorepb.Firestore_BatchGetDocumentsServer) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return s.Error
	}
	for _, v := range s.Resps {
		//nolint
		if err := stream.Send(v.(*firestorepb.BatchGetDocumentsResponse)); err != nil {
			return err
		}
	}
	return nil
}

func (s *MockFirestoreServer) BeginTransaction(ctx context.Context, req *firestorepb.BeginTransactionRequest) (*firestorepb.BeginTransactionResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*firestorepb.BeginTransactionResponse), nil
}

func (s *MockFirestoreServer) Commit(ctx context.Context, req *firestorepb.CommitRequest) (*firestorepb.CommitResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*firestorepb.CommitResponse), nil
}

func (s *MockFirestoreServer) Rollback(ctx context.Context, req *firestorepb.RollbackRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*emptypb.Empty), nil
}

func (s *MockFirestoreServer) RunQuery(req *firestorepb.RunQueryRequest, stream firestorepb.Firestore_RunQueryServer) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return s.Error
	}
	for _, v := range s.Resps {
		//nolint
		if err := stream.Send(v.(*firestorepb.RunQueryResponse)); err != nil {
			return err
		}
	}
	return nil
}

func (s *MockFirestoreServer) Write(stream firestorepb.Firestore_WriteServer) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	for {
		if req, err := stream.Recv(); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		} else {
			s.Reqs = append(s.Reqs, req)
		}
	}
	if s.Error != nil {
		return s.Error
	}
	for _, v := range s.Resps {
		//nolint
		if err := stream.Send(v.(*firestorepb.WriteResponse)); err != nil {
			return err
		}
	}
	return nil
}

func (s *MockFirestoreServer) Listen(stream firestorepb.Firestore_ListenServer) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	for {
		if req, err := stream.Recv(); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		} else {
			s.Reqs = append(s.Reqs, req)
		}
	}
	if s.Error != nil {
		return s.Error
	}
	for _, v := range s.Resps {
		//nolint
		if err := stream.Send(v.(*firestorepb.ListenResponse)); err != nil {
			return err
		}
	}
	return nil
}

func (s *MockFirestoreServer) ListCollectionIds(ctx context.Context, req *firestorepb.ListCollectionIdsRequest) (*firestorepb.ListCollectionIdsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.Reqs = append(s.Reqs, req)
	if s.Error != nil {
		return nil, s.Error
	}
	//nolint
	return s.Resps[0].(*firestorepb.ListCollectionIdsResponse), nil
}

func NewMockFS(projectID string) (*firestore.Client, *MockFirestoreServer, func(), error) {
	serv := grpc.NewServer()
	mockFireStoreServer := &MockFirestoreServer{
		UnimplementedFirestoreServer: firestorepb.UnimplementedFirestoreServer{},
		Reqs:                         make([]proto.Message, 1),
		Error:                        nil,
		Resps:                        make([]proto.Message, 1),
	}

	firestorepb.RegisterFirestoreServer(serv, mockFireStoreServer)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, func() {}, err
	}
	// not checked, but makes linter happy
	errs := make(chan error, 1)
	go func() {
		errs <- serv.Serve(lis)
		close(errs)
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, func() {}, fmt.Errorf("failed to create Firestore connection: %w", err)
	}
	client, err := firestore.NewClient(context.Background(), projectID, option.WithGRPCConn(conn))
	if err != nil {
		return nil, nil, func() {}, fmt.Errorf("failed to create Firestore client: %w", err)
	}
	return client, mockFireStoreServer, func() {
		client.Close()
		serv.Stop()
	}, nil
}
