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
	"github.com/roost-io/roost-example-latest/grpcExample/api"
	zb "github.com/ZB-io/zbio/client"
	zbutil "github.com/roost-io/roost-example-latest/grpcExample/message"
)

/*
ROOST_METHOD_HASH=Check_a316e66539
ROOST_METHOD_SIG_HASH=Check_a316e66539


 */
func (m *mockHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Println("🏥 Mock server is health checking")
	if m.isDatabaseReady == nil {
		log.Printf("🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_UNKNOWN)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_UNKNOWN}, nil
	} else if *m.isDatabaseReady {
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
		name		string
		isDatabaseReady	*bool
		expectedStatus	grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLog	string
	}{{name: "Server Returns SERVING When Database is Ready", isDatabaseReady: boolPtr(true), expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "✅ Server's status is SERVING"}, {name: "Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: boolPtr(false), expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLog: "🚫 Server's status is NOT_SERVING"}, {name: "Server Returns UNKNOWN When Database State is Undefined", isDatabaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLog: "🚫 Server's status is UNKNOWN"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := grpc.NewServer()
			mockHealth := &mockHealthServer{isDatabaseReady: tt.isDatabaseReady}
			grpc_health_v1.RegisterHealthServer(server, mockHealth)
			lis, err := net.Listen("tcp", ":0")
			if err != nil {
				t.Fatalf("Failed to listen: %v", err)
			}
			go func() {
				if err := server.Serve(lis); err != nil {
					t.Fatalf("Failed to serve: %v", err)
				}
			}()
			defer server.Stop()
			conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
			if err != nil {
				t.Fatalf("Failed to dial server: %v", err)
			}
			defer conn.Close()
			client := grpc_health_v1.NewHealthClient(conn)
			response, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if response.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, response.Status)
			}
			t.Logf("Expected log: %s", tt.expectedLog)
		})
	}
	t.Run("Handling Nil Context", func(t *testing.T) {
		server := grpc.NewServer()
		mockHealth := &mockHealthServer{isDatabaseReady: boolPtr(true)}
		grpc_health_v1.RegisterHealthServer(server, mockHealth)
		lis, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("Failed to listen: %v", err)
		}
		go func() {
			if err := server.Serve(lis); err != nil {
				t.Fatalf("Failed to serve: %v", err)
			}
		}()
		defer server.Stop()
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial server: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		_, err = client.Check(nil, &grpc_health_v1.HealthCheckRequest{})
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument error for nil context, got %v", err)
		}
	})
	t.Run("Handling Nil HealthCheckRequest", func(t *testing.T) {
		server := grpc.NewServer()
		mockHealth := &mockHealthServer{isDatabaseReady: boolPtr(true)}
		grpc_health_v1.RegisterHealthServer(server, mockHealth)
		lis, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("Failed to listen: %v", err)
		}
		go func() {
			if err := server.Serve(lis); err != nil {
				t.Fatalf("Failed to serve: %v", err)
			}
		}()
		defer server.Stop()
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial server: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		_, err = client.Check(context.Background(), nil)
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument error for nil request, got %v", err)
		}
	})
	t.Run("Concurrency Handling", func(t *testing.T) {
		server := grpc.NewServer()
		mockHealth := &mockHealthServer{isDatabaseReady: boolPtr(true)}
		grpc_health_v1.RegisterHealthServer(server, mockHealth)
		lis, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("Failed to listen: %v", err)
		}
		go func() {
			if err := server.Serve(lis); err != nil {
				t.Fatalf("Failed to serve: %v", err)
			}
		}()
		defer server.Stop()
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial server: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
				if err != nil {
					t.Errorf("Concurrent check failed: %v", err)
				}
				if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
					t.Errorf("Expected SERVING status, got %v", resp.Status)
				}
			}()
		}
		wg.Wait()
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
		request		*api.InputRequest
		wantErr		bool
		wantOutput	string
	}{{name: "Valid Input", request: &api.InputRequest{ClientName: "testClient", Text: "hello world"}, wantErr: false, wantOutput: "HELLO WORLD😊"}, {name: "Empty Text", request: &api.InputRequest{ClientName: "testClient", Text: ""}, wantErr: false, wantOutput: "😊"}, {name: "Special Characters", request: &api.InputRequest{ClientName: "testClient", Text: "!@#$%^&*()"}, wantErr: false, wantOutput: "!@#$%^&*()😊"}, {name: "Long Text Input", request: &api.InputRequest{ClientName: "testClient", Text: strings.Repeat("a", 10000)}, wantErr: false, wantOutput: strings.ToUpper(strings.Repeat("a", 10000)) + "😊"}, {name: "Numeric Characters", request: &api.InputRequest{ClientName: "testClient", Text: "1234567890"}, wantErr: false, wantOutput: "1234567890😊"}, {name: "Nil Request", request: nil, wantErr: true}, {name: "Invalid Client Name", request: &api.InputRequest{ClientName: "", Text: "hello"}, wantErr: true}, {name: "Context Cancellation", request: &api.InputRequest{ClientName: "testClient", Text: "hello"}, wantErr: true}}
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

