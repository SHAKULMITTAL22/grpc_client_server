package main

import (
	"context"
	"log"
	"net"
	"sync"
	"testing"
	"time"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"fmt"
	"strings"
	"github.com/roost-io/roost-example-latest/grpcExample/api"
	zb "github.com/ZB-io/zbio/client"
	zbutil "github.com/roost-io/roost-example-latest/grpcExample/message"
)

/*
ROOST_METHOD_HASH=Check_a316e66539
ROOST_METHOD_SIG_HASH=Check_a316e66539


 */
func (m *mockHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if isDatabaseReady {
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	} else if !isDatabaseReady {
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}, nil
	}
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_UNKNOWN}, nil
}

func Testcheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, &mockHealthServer{})
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	defer grpcServer.Stop()
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	testCases := []struct {
		description	string
		databaseReady	bool
		expectedStatus	grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLog	string
	}{{description: "Server Returns SERVING When Database is Ready", databaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "✅ Server's status is SERVING"}, {description: "Server Returns NOT_SERVING When Database is Not Ready", databaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLog: "🚫 Server's status is NOT_SERVING"}, {description: "Server Returns UNKNOWN When Database Readiness is Indeterminate", databaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLog: "🚫 Server's status is UNKNOWN"}}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			isDatabaseReady = tc.databaseReady
			resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				t.Errorf("Check failed: %v", err)
			}
			if resp.Status != tc.expectedStatus {
				t.Errorf("Expected status %v, got %v", tc.expectedStatus, resp.Status)
			}
			t.Log(tc.expectedLog)
		})
	}
	t.Run("Handling of gRPC Context Cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if status.Code(err) != codes.Canceled {
			t.Errorf("Expected error code %v, got %v", codes.Canceled, status.Code(err))
		}
	})
	t.Run("Concurrent Health Check Requests", func(t *testing.T) {
		var wg sync.WaitGroup
		numRequests := 10
		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
				if err != nil {
					t.Errorf("Concurrent Check failed: %v", err)
				}
			}()
		}
		wg.Wait()
	})
	t.Run("Handling of Invalid Health Check Requests", func(t *testing.T) {
		_, err := client.Check(context.Background(), nil)
		if err == nil {
			t.Error("Expected error for invalid request, got nil")
		}
	})
	t.Run("Response Time for Health Check Requests", func(t *testing.T) {
		start := time.Now()
		_, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Errorf("Check failed: %v", err)
		}
		duration := time.Since(start)
		if duration > time.Second {
			t.Errorf("Health check response time exceeded 1 second: %v", duration)
		}
	})
}

/*
ROOST_METHOD_HASH=Watch_ee291f18f7
ROOST_METHOD_SIG_HASH=Watch_ee291f18f7


 */
func Testwatch(t *testing.T) {
	server := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(server, &MockHealthServer{})
	reflection.Register(server)
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	defer server.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	tests := []struct {
		name		string
		request		*grpc_health_v1.HealthCheckRequest
		wantCode	codes.Code
		wantErrMsg	string
	}{{name: "Unimplemented Method Response", request: &grpc_health_v1.HealthCheckRequest{}, wantCode: codes.Unimplemented, wantErrMsg: "Watching is not supported"}, {name: "Nil HealthCheckRequest", request: nil, wantCode: codes.Unimplemented, wantErrMsg: "Watching is not supported"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := client.Watch(context.Background(), tt.request)
			if err == nil {
				t.Fatal("Expected an error, but got none")
			}
			if st, ok := status.FromError(err); ok {
				if st.Code() != tt.wantCode {
					t.Errorf("Expected code %v, got %v", tt.wantCode, st.Code())
				}
				if st.Message() != tt.wantErrMsg {
					t.Errorf("Expected error message %q, got %q", tt.wantErrMsg, st.Message())
				}
			} else {
				t.Fatalf("Failed to parse error status: %v", err)
			}
			if stream != nil {
				t.Fatal("Expected no stream, but got one")
			}
		})
	}
	t.Run("Concurrency Handling", func(t *testing.T) {
		numRequests := 10
		errCh := make(chan error, numRequests)
		for i := 0; i < numRequests; i++ {
			go func() {
				_, err := client.Watch(context.Background(), &grpc_health_v1.HealthCheckRequest{})
				errCh <- err
			}()
		}
		for i := 0; i < numRequests; i++ {
			err := <-errCh
			if err == nil {
				t.Error("Expected an error, but got none")
			} else if st, ok := status.FromError(err); ok {
				if st.Code() != codes.Unimplemented {
					t.Errorf("Expected code %v, got %v", codes.Unimplemented, st.Code())
				}
			} else {
				t.Errorf("Failed to parse error status: %v", err)
			}
		}
	})
	t.Run("Response Time and Performance", func(t *testing.T) {
		start := time.Now()
		_, err := client.Watch(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		if err == nil {
			t.Fatal("Expected an error, but got none")
		}
		duration := time.Since(start)
		t.Logf("Response time: %v", duration)
		if duration > time.Second {
			t.Error("Response time exceeded 1 second")
		}
	})
}

func (m *MockHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watching is not supported")
}

/*
ROOST_METHOD_HASH=Upper_6c4de803cd
ROOST_METHOD_SIG_HASH=Upper_6c4de803cd


 */
func (m *MockZBIOClient) SendMessageToZBIO(messages []zb.Message) error {
	return nil
}

func Testupper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	s := grpc.NewServer()
	api.RegisterUpperServer(s, &mockServer{})
	go s.Serve(lis)
	defer s.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperClient(conn)
	tests := []struct {
		name		string
		request		*api.InputRequest
		expected	*api.OutputResponse
		errCode		codes.Code
	}{{name: "Valid Input", request: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, expected: &api.OutputResponse{ServerName: "TestServer", Text: "HELLO😊"}, errCode: codes.OK}, {name: "Empty Text", request: &api.InputRequest{ClientName: "TestClient", Text: ""}, expected: &api.OutputResponse{ServerName: "TestServer", Text: "😊"}, errCode: codes.OK}, {name: "Special Characters", request: &api.InputRequest{ClientName: "TestClient", Text: "!@#$%^&*()"}, expected: &api.OutputResponse{ServerName: "TestServer", Text: "!@#$%^&*()😊"}, errCode: codes.OK}, {name: "Long Text Input", request: &api.InputRequest{ClientName: "TestClient", Text: strings.Repeat("a", 1000)}, expected: &api.OutputResponse{ServerName: "TestServer", Text: strings.Repeat("A", 1000) + "😊"}, errCode: codes.OK}, {name: "Nil Input Request", request: nil, expected: nil, errCode: codes.InvalidArgument}, {name: "Context Cancellation", request: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, expected: nil, errCode: codes.Canceled}, {name: "Mixed Case Text", request: &api.InputRequest{ClientName: "TestClient", Text: "HeLLo WoRLd"}, expected: &api.OutputResponse{ServerName: "TestServer", Text: "HELLO WORLD😊"}, errCode: codes.OK}, {name: "Non-ASCII Characters", request: &api.InputRequest{ClientName: "TestClient", Text: "こんにちは"}, expected: &api.OutputResponse{ServerName: "TestServer", Text: "こんにちは😊"}, errCode: codes.OK}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tt.name == "Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.request)
			if tt.errCode != codes.OK {
				if err == nil || status.Code(err) != tt.errCode {
					t.Errorf("expected error code %v, got %v", tt.errCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Upper failed: %v", err)
			}
			if resp.ServerName != tt.expected.ServerName || resp.Text != tt.expected.Text {
				t.Errorf("expected response %v, got %v", tt.expected, resp)
			}
			t.Logf("Test %s passed.", tt.name)
		})
	}
}

func (s *mockServer) Upper(ctx context.Context, req *api.InputRequest) (*api.OutputResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	reqAck := fmt.Sprintf("➡️ Received message from client %v: %v ", req.GetClientName(), req.GetText())
	log.Printf(reqAck)
	message := zb.Message{TopicName: zbutil.TopicName, Data: []byte(reqAck), HintPartition: ""}
	err := s.SendMessageToZBIO([]zb.Message{message})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to send message to ZBIO")
	}
	x := happyUpper(req.GetText())
	return &api.OutputResponse{ServerName: "TestServer", Text: x}, nil
}

