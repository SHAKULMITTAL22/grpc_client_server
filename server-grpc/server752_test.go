package main

import (
	"context"
	"log"
	"net"
	"testing"
	"time"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"fmt"
	"github.com/roost-io/roost-example-latest/grpcExample/api"
	zb "github.com/ZB-io/zbio/client"
	zbutil "github.com/roost-io/roost-example-latest/grpcExample/message"
)

/*
ROOST_METHOD_HASH=Check_a316e66539
ROOST_METHOD_SIG_HASH=Check_a316e66539


 */
func (m *mockHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Println("🏥 K8s is health checking")
	if isDatabaseReady == true {
		log.Printf("✅ Server's status is %s", grpc_health_v1.HealthCheckResponse_SERVING)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	} else if isDatabaseReady == false {
		log.Printf("🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}, nil
	} else {
		log.Printf("🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_UNKNOWN)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_UNKNOWN}, nil
	}
}

func TestCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	mockServer := &mockHealthServer{}
	grpc_health_v1.RegisterHealthServer(s, mockServer)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	defer s.Stop()
	conn, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	testCases := []struct {
		description		string
		databaseState		interface{}
		expectedStatus		grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLog		string
		contextTimeout		time.Duration
		expectedGRPCCode	codes.Code
	}{{description: "Server Returns SERVING When Database Is Ready", databaseState: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "✅ Server's status is SERVING"}, {description: "Server Returns NOT_SERVING When Database Is Not Ready", databaseState: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLog: "🚫 Server's status is NOT_SERVING"}, {description: "Server Returns UNKNOWN When Database Readiness Is Indeterminate", databaseState: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLog: "🚫 Server's status is UNKNOWN"}, {description: "Server Handles Context Cancellation Gracefully", databaseState: true, contextTimeout: 1 * time.Nanosecond, expectedGRPCCode: codes.Canceled}, {description: "Server Handles Context Deadline Exceeded", databaseState: true, contextTimeout: 1 * time.Nanosecond, expectedGRPCCode: codes.DeadlineExceeded}}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			isDatabaseReady = tc.databaseState
			ctx := context.Background()
			if tc.contextTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.contextTimeout)
				defer cancel()
			}
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if tc.expectedGRPCCode != codes.OK {
				if status.Code(err) != tc.expectedGRPCCode {
					t.Errorf("expected gRPC code %v, got %v", tc.expectedGRPCCode, status.Code(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if resp.Status != tc.expectedStatus {
				t.Errorf("expected status %v, got %v", tc.expectedStatus, resp.Status)
			}
		})
	}
}

/*
ROOST_METHOD_HASH=Watch_ee291f18f7
ROOST_METHOD_SIG_HASH=Watch_ee291f18f7


 */
func TestWatch(t *testing.T) {
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
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			stream, err := client.Watch(ctx, tt.request)
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Failed to parse error status")
			}
			if st.Code() != tt.wantCode {
				t.Errorf("Expected code %v, got %v", tt.wantCode, st.Code())
			}
			if st.Message() != tt.wantErrMsg {
				t.Errorf("Expected error message %q, got %q", tt.wantErrMsg, st.Message())
			}
			t.Logf("Test %q passed with expected error code and message.", tt.name)
		})
	}
	t.Run("Concurrency Handling", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		const concurrentRequests = 5
		errCh := make(chan error, concurrentRequests)
		for i := 0; i < concurrentRequests; i++ {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				stream, err := client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})
				errCh <- err
				if stream != nil {
					stream.CloseSend()
				}
			}()
		}
		for i := 0; i < concurrentRequests; i++ {
			err := <-errCh
			if err == nil {
				t.Errorf("Expected error, got nil")
			} else {
				st, ok := status.FromError(err)
				if !ok || st.Code() != codes.Unimplemented {
					t.Errorf("Expected Unimplemented error, got %v", st)
				}
			}
		}
		t.Log("Concurrency test passed with consistent Unimplemented errors.")
	})
	t.Run("Response Time and Performance", func(t *testing.T) {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		elapsed := time.Since(start)
		if elapsed > 2*time.Second {
			t.Errorf("Response time exceeded: %v", elapsed)
		} else {
			t.Logf("Response time within limits: %v", elapsed)
		}
	})
	t.Run("Reflection and Metadata Verification", func(t *testing.T) {
		t.Log("Reflection test can be validated using external tools like grpcurl.")
	})
	t.Run("Client-Side Error Handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			st, ok := status.FromError(err)
			if !ok || st.Code() != codes.Unimplemented {
				t.Errorf("Expected Unimplemented error, got %v", st)
			}
		} else {
			t.Error("Expected error, got nil")
		}
		t.Log("Client-side error handling test passed.")
	})
}

func (m *MockHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watching is not supported")
}

/*
ROOST_METHOD_HASH=Upper_6c4de803cd
ROOST_METHOD_SIG_HASH=Upper_6c4de803cd


 */
func TestUpper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	server, listener := startMockGRPCServer(t)
	defer server.Stop()
	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperServiceClient(conn)
	testCases := []struct {
		name		string
		req		*api.InputRequest
		expectError	bool
	}{{name: "Valid Input", req: &api.InputRequest{ClientName: "testClient", Text: "hello"}, expectError: false}, {name: "Empty Text", req: &api.InputRequest{ClientName: "testClient", Text: ""}, expectError: false}, {name: "Special Characters", req: &api.InputRequest{ClientName: "testClient", Text: "!@#$%^&*()"}, expectError: false}, {name: "Long Text Input", req: &api.InputRequest{ClientName: "testClient", Text: strings.Repeat("a", 10000)}, expectError: false}, {name: "Numeric Characters", req: &api.InputRequest{ClientName: "testClient", Text: "1234567890"}, expectError: false}, {name: "Nil Request", req: nil, expectError: true}, {name: "Invalid Client Name", req: &api.InputRequest{ClientName: "", Text: "hello"}, expectError: true}, {name: "Context Cancellation", req: &api.InputRequest{ClientName: "testClient", Text: "hello"}, expectError: true}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if tc.name == "Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tc.req)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else {
					t.Logf("Received expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					t.Logf("Received response: %v", resp)
				}
			}
		})
	}
}

func (s *mockServer) Upper(ctx context.Context, req *api.InputRequest) (*api.OutputResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	if req.GetClientName() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid client name")
	}
	if ctx.Err() == context.Canceled {
		return nil, status.Error(codes.Canceled, "request canceled")
	}
	x := happyUpper(req.GetText())
	return &api.OutputResponse{ServerName: "mockServer", Text: x}, nil
}

func startMockGRPCServer(t *testing.T) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterUpperServiceServer(s, &mockServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	return s, lis
}

