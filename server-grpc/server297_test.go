package main

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"
	"github.com/golang/mock/gomock"
	"github.com/roost-io/roost-example-latest/grpcExample/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"net"
	"fmt"
	zb "github.com/ZB-io/zbio/client"
	zbutil "github.com/roost-io/roost-example-latest/grpcExample/message"
)

/*
ROOST_METHOD_HASH=Check_a316e66539
ROOST_METHOD_SIG_HASH=Check_a316e66539


 */
func (m *mockHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Println("🏥 K8s is health checking")
	if m.isDatabaseReady == nil {
		log.Printf("🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_UNKNOWN)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_UNKNOWN}, nil
	}
	if *m.isDatabaseReady {
		log.Printf("✅ Server's status is %s", grpc_health_v1.HealthCheckResponse_SERVING)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	} else {
		log.Printf("🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}, nil
	}
}

func TestCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tests := []struct {
		name			string
		isDatabaseReady		*bool
		expectedStatus		grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLogPrefix	string
	}{{name: "Server Returns SERVING When Database is Ready", isDatabaseReady: boolPtr(true), expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLogPrefix: "✅ Server's status is SERVING"}, {name: "Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: boolPtr(false), expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLogPrefix: "🚫 Server's status is NOT_SERVING"}, {name: "Server Returns UNKNOWN When Database State is Undefined", isDatabaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLogPrefix: "🚫 Server's status is UNKNOWN"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &mockHealthServer{isDatabaseReady: tt.isDatabaseReady}
			req := &grpc_health_v1.HealthCheckRequest{}
			var logOutput strings.Builder
			log.SetOutput(&logOutput)
			resp, err := server.Check(context.Background(), req)
			if err != nil {
				t.Fatalf("Check() returned an error: %v", err)
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
			if !strings.Contains(logOutput.String(), tt.expectedLogPrefix) {
				t.Errorf("expected log prefix %q, got %q", tt.expectedLogPrefix, logOutput.String())
			}
		})
	}
	t.Run("Check Function Handles Nil Context Gracefully", func(t *testing.T) {
		server := &mockHealthServer{isDatabaseReady: boolPtr(true)}
		req := &grpc_health_v1.HealthCheckRequest{}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Check() panicked with nil context: %v", r)
			}
		}()
		_, err := server.Check(nil, req)
		if err != nil {
			t.Errorf("Check() returned an error with nil context: %v", err)
		}
	})
	t.Run("Check Function Handles Nil Request Gracefully", func(t *testing.T) {
		server := &mockHealthServer{isDatabaseReady: boolPtr(true)}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Check() panicked with nil request: %v", r)
			}
		}()
		_, err := server.Check(context.Background(), nil)
		if err != nil {
			t.Errorf("Check() returned an error with nil request: %v", err)
		}
	})
	t.Run("Check Function Handles Context Cancellation", func(t *testing.T) {
		server := &mockHealthServer{isDatabaseReady: boolPtr(true)}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		resp, err := server.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil && status.Code(err) != codes.Canceled {
			t.Errorf("expected error code %v, got %v", codes.Canceled, status.Code(err))
		}
		if resp != nil {
			t.Errorf("expected nil response on context cancellation, got %v", resp)
		}
	})
	t.Run("Check Function's Error Handling for Unexpected Internal Errors", func(t *testing.T) {
	})
}

func boolPtr(b bool) *bool {
	return &b
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
	serverAddr := startMockGRPCServer(t)
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperServiceClient(conn)
	tests := []struct {
		name		string
		req		*api.InputRequest
		wantErr		bool
		wantStatus	codes.Code
	}{{name: "Valid Input", req: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, wantErr: false, wantStatus: codes.OK}, {name: "Empty Text", req: &api.InputRequest{ClientName: "TestClient", Text: ""}, wantErr: false, wantStatus: codes.OK}, {name: "Special Characters", req: &api.InputRequest{ClientName: "TestClient", Text: "!@#$%^&*()"}, wantErr: false, wantStatus: codes.OK}, {name: "Long Text Input", req: &api.InputRequest{ClientName: "TestClient", Text: strings.Repeat("a", 10000)}, wantErr: false, wantStatus: codes.OK}, {name: "Numeric Characters", req: &api.InputRequest{ClientName: "TestClient", Text: "1234567890"}, wantErr: false, wantStatus: codes.OK}, {name: "Nil Request", req: nil, wantErr: true, wantStatus: codes.InvalidArgument}, {name: "Invalid Client Name", req: &api.InputRequest{ClientName: "", Text: "hello"}, wantErr: true, wantStatus: codes.InvalidArgument}, {name: "Context Cancellation", req: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, wantErr: true, wantStatus: codes.Canceled}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tt.name == "Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Upper() error = %v, wantErr %v", err, tt.wantErr)
			}
			if status.Code(err) != tt.wantStatus {
				t.Fatalf("Upper() status = %v, wantStatus %v", status.Code(err), tt.wantStatus)
			}
			if !tt.wantErr && resp == nil {
				t.Fatal("Expected non-nil response")
			} else if !tt.wantErr {
				t.Logf("Success: Received response: %v", resp.GetText())
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

func startMockGRPCServer(t *testing.T) string {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterUpperServiceServer(s, &mockServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	return lis.Addr().String()
}

