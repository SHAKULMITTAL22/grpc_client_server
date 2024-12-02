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
	log.Println("Mock: K8s is health checking")
	if m.isDatabaseReady == true {
		log.Printf("Mock: ✅ Server's status is %s", grpc_health_v1.HealthCheckResponse_SERVING)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	} else if m.isDatabaseReady == false {
		log.Printf("Mock: 🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}, nil
	} else {
		log.Printf("Mock: 🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_UNKNOWN)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_UNKNOWN}, nil
	}
}

func Testcheck(t *testing.T) {
	tests := []struct {
		name		string
		isDatabaseReady	interface{}
		expectedStatus	grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLog	string
	}{{name: "Scenario 1: Server Returns SERVING When Database is Ready", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "Mock: ✅ Server's status is SERVING"}, {name: "Scenario 2: Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLog: "Mock: 🚫 Server's status is NOT_SERVING"}, {name: "Scenario 3: Server Returns UNKNOWN When Database State is Undefined", isDatabaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLog: "Mock: 🚫 Server's status is UNKNOWN"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, client := startMockServer(t, tt.isDatabaseReady)
			defer conn.Close()
			req := &grpc_health_v1.HealthCheckRequest{}
			resp, err := client.Check(context.Background(), req)
			if err != nil {
				t.Errorf("Check() error = %v", err)
				return
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
		})
	}
	t.Run("Scenario 7: HealthCheckRequest Handling", func(t *testing.T) {
		conn, client := startMockServer(t, true)
		defer conn.Close()
		req := &grpc_health_v1.HealthCheckRequest{}
		if _, err := client.Check(context.Background(), req); err != nil {
			t.Errorf("HealthCheckRequest handling failed: %v", err)
		}
	})
	t.Run("Scenario 8: Context Handling in Check Function", func(t *testing.T) {
		conn, client := startMockServer(t, true)
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		_, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if status.Code(err) != codes.DeadlineExceeded {
			t.Errorf("Expected deadline exceeded error, got %v", err)
		}
	})
}

func startMockServer(t *testing.T, isDatabaseReady interface{}) (*grpc.ClientConn, grpc_health_v1.HealthClient) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	server := grpc.NewServer()
	mockServer := &mockHealthServer{isDatabaseReady: isDatabaseReady}
	grpc_health_v1.RegisterHealthServer(server, mockServer)
	go server.Serve(listener)
	t.Cleanup(func() {
		server.Stop()
		listener.Close()
	})
	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	client := grpc_health_v1.NewHealthClient(conn)
	return conn, client
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
	s, lis := startTestServer()
	defer s.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperClient(conn)
	tests := []struct {
		name		string
		input		*api.InputRequest
		want		string
		expectError	bool
	}{{name: "Valid Input", input: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, want: "HELLO😊"}, {name: "Empty Text", input: &api.InputRequest{ClientName: "TestClient", Text: ""}, want: "😊"}, {name: "Special Characters", input: &api.InputRequest{ClientName: "TestClient", Text: "!@#$%^&*()"}, want: "!@#$%^&*()😊"}, {name: "Long Text Input", input: &api.InputRequest{ClientName: "TestClient", Text: strings.Repeat("a", 10000)}, want: strings.ToUpper(strings.Repeat("a", 10000)) + "😊"}, {name: "Nil Input Request", input: nil, expectError: true}, {name: "Mixed Case Text", input: &api.InputRequest{ClientName: "TestClient", Text: "HeLLo WorLD"}, want: "HELLO WORLD😊"}, {name: "Non-ASCII Characters", input: &api.InputRequest{ClientName: "TestClient", Text: "こんにちは"}, want: "こんにちは😊"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			resp, err := client.Upper(ctx, tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got none")
				} else {
					t.Logf("received expected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Upper() error = %v", err)
				return
			}
			if resp.Text != tt.want {
				t.Errorf("Upper() = %v, want %v", resp.Text, tt.want)
			} else {
				t.Logf("Upper() success: received %v", resp.Text)
			}
		})
	}
}

func (s *mockServer) Upper(ctx context.Context, req *api.InputRequest) (*api.OutputResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	reqAck := fmt.Sprintf("➡️ Received message from client %v: %v ", req.GetClientName(), req.GetText())
	log.Printf(reqAck)
	err := s.SendMessageToZBIO([]zb.Message{{TopicName: zbutil.TopicName, Data: []byte(reqAck), HintPartition: ""}})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to send message to ZBIO")
	}
	x := happyUpper(req.GetText())
	return &api.OutputResponse{ServerName: "TestServer", Text: x}, nil
}

func startTestServer() (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterUpperServer(s, &mockServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	return s, lis
}

