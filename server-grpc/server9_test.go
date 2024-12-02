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
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"fmt"
	"github.com/roost-io/roost-example-latest/grpcExample/api"
	zb "github.com/ZB-io/zbio/client"
	zbutil "github.com/roost-io/roost-example-latest/grpcExample/message"
)

/*
ROOST_METHOD_HASH=Check_a316e66539
ROOST_METHOD_SIG_HASH=Check_a316e66539


 */
func Testcheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &Health{})
	defer s.Stop()
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	clientConn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer clientConn.Close()
	client := grpc_health_v1.NewHealthClient(clientConn)
	tests := []struct {
		name		string
		databaseReady	interface{}
		expectedStatus	grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLog	string
	}{{name: "Server Returns SERVING When Database is Ready", databaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "✅ Server's status is SERVING"}, {name: "Server Returns NOT_SERVING When Database is Not Ready", databaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLog: "🚫 Server's status is NOT_SERVING"}, {name: "Server Handles Unknown Database State Gracefully", databaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLog: "🚫 Server's status is UNKNOWN"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDatabaseReady = tt.databaseReady
			var logMu sync.Mutex
			log.SetFlags(0)
			log.SetOutput(&testWriter{t, &logMu})
			resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				t.Errorf("Check() error = %v", err)
				return
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
			if !strings.Contains(logOutput, tt.expectedLog) {
				t.Errorf("Expected log %v, got %v", tt.expectedLog, logOutput)
			}
		})
	}
	t.Run("Health Check Request Handling", func(t *testing.T) {
		isDatabaseReady = true
		resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Errorf("Expected SERVING status, got %v", resp.Status)
		}
	})
	t.Run("Error Handling in Check Function", func(t *testing.T) {
		isDatabaseReady = "unexpected"
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function panicked with: %v", r)
			}
		}()
		_, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		if err == nil {
			t.Error("Expected an error, got none")
		}
	})
	t.Run("Concurrent Health Check Requests", func(t *testing.T) {
		isDatabaseReady = true
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
					t.Errorf("Expected SERVING status, got %v", resp.Status)
				}
			}()
		}
		wg.Wait()
	})
	t.Run("Integration with gRPC Health Protocol", func(t *testing.T) {
		isDatabaseReady = true
		resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Errorf("Expected SERVING status, got %v", resp.Status)
		}
	})
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	logOutput += string(p)
	return len(p), nil
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
func Testupper(t *testing.T) {
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
		request		*api.InputRequest
		wantErr		bool
		wantOutput	string
	}{{name: "Valid Input", request: &api.InputRequest{ClientName: "testClient", Text: "hello world"}, wantErr: false, wantOutput: "HELLO WORLD😊"}, {name: "Empty Text", request: &api.InputRequest{ClientName: "testClient", Text: ""}, wantErr: false, wantOutput: "😊"}, {name: "Special Characters", request: &api.InputRequest{ClientName: "testClient", Text: "!@#$%^&*()"}, wantErr: false, wantOutput: "!@#$%^&*()😊"}, {name: "Long Text Input", request: &api.InputRequest{ClientName: "testClient", Text: string(make([]byte, 1024*1024))}, wantErr: false, wantOutput: string(make([]byte, 1024*1024)) + "😊"}, {name: "Numeric Characters", request: &api.InputRequest{ClientName: "testClient", Text: "1234567890"}, wantErr: false, wantOutput: "1234567890😊"}, {name: "Nil Request", request: nil, wantErr: true}, {name: "Invalid Client Name", request: &api.InputRequest{ClientName: "", Text: "test"}, wantErr: true}, {name: "Context Cancellation", request: &api.InputRequest{ClientName: "testClient", Text: "test"}, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tt.name == "Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && resp.GetText() != tt.wantOutput {
				t.Errorf("Upper() got = %v, want %v", resp.GetText(), tt.wantOutput)
			}
			if err == nil {
				t.Logf("Success: Received expected response: %v", resp.GetText())
			} else {
				t.Logf("Failure: Encountered error: %v", err)
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

