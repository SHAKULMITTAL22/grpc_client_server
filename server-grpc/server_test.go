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
func Testcheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tests := []struct {
		name			string
		isDatabaseReady		*bool
		expectedStatus		grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedGRPCErrorCode	codes.Code
	}{{name: "Scenario 1: Server Returns SERVING When Database is Ready", isDatabaseReady: boolPtr(true), expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedGRPCErrorCode: codes.OK}, {name: "Scenario 2: Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: boolPtr(false), expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedGRPCErrorCode: codes.OK}, {name: "Scenario 3: Server Returns UNKNOWN for Unspecified Database State", isDatabaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedGRPCErrorCode: codes.OK}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDatabaseReady = tt.isDatabaseReady
			mockHealthServer := &Health{}
			grpcServer, lis := startMockGRPCServer(mockHealthServer)
			defer grpcServer.Stop()
			conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
			if err != nil {
				t.Fatalf("Failed to connect to gRPC server: %v", err)
			}
			defer conn.Close()
			client := grpc_health_v1.NewHealthClient(conn)
			req := &grpc_health_v1.HealthCheckRequest{}
			resp, err := client.Check(context.Background(), req)
			if err != nil {
				st, ok := status.FromError(err)
				if !ok || st.Code() != tt.expectedGRPCErrorCode {
					t.Errorf("Unexpected gRPC error code: got %v, want %v", st.Code(), tt.expectedGRPCErrorCode)
				}
			} else {
				if resp.Status != tt.expectedStatus {
					t.Errorf("Unexpected status: got %v, want %v", resp.Status, tt.expectedStatus)
				}
			}
			t.Logf("Test '%s': Passed", tt.name)
		})
	}
	t.Run("Scenario 4: Context Cancellation Handling", func(t *testing.T) {
		mockHealthServer := &Health{}
		grpcServer, lis := startMockGRPCServer(mockHealthServer)
		defer grpcServer.Stop()
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to connect to gRPC server: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err == nil || status.Code(err) != codes.Canceled {
			t.Errorf("Expected context canceled error, got: %v", err)
		}
		t.Log("Test 'Scenario 4: Context Cancellation Handling': Passed")
	})
	t.Run("Scenario 5: Handling of Nil HealthCheckRequest", func(t *testing.T) {
		mockHealthServer := &Health{}
		grpcServer, lis := startMockGRPCServer(mockHealthServer)
		defer grpcServer.Stop()
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to connect to gRPC server: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		_, err = client.Check(context.Background(), nil)
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Errorf("Expected invalid argument error, got: %v", err)
		}
		t.Log("Test 'Scenario 5: Handling of Nil HealthCheckRequest': Passed")
	})
}

func boolPtr(b bool) *bool {
	return &b
}

func startMockGRPCServer(mockHealthServer grpc_health_v1.HealthServer) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, mockHealthServer)
	go grpcServer.Serve(lis)
	return grpcServer, lis
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
	}{{name: "Valid Input", input: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, want: "HELLO😊"}, {name: "Empty Text", input: &api.InputRequest{ClientName: "TestClient", Text: ""}, want: "😊"}, {name: "Special Characters", input: &api.InputRequest{ClientName: "TestClient", Text: "!@#$%^&*()"}, want: "!@#$%^&*()😊"}, {name: "Long Text Input", input: &api.InputRequest{ClientName: "TestClient", Text: strings.Repeat("a", 10000)}, want: strings.ToUpper(strings.Repeat("a", 10000)) + "😊"}, {name: "Nil Input Request", input: nil, expectError: true}, {name: "Mixed Case Text", input: &api.InputRequest{ClientName: "TestClient", Text: "HeLLo WoRLd"}, want: "HELLO WORLD😊"}, {name: "Non-ASCII Characters", input: &api.InputRequest{ClientName: "TestClient", Text: "こんにちは世界"}, want: "こんにちは世界😊"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tt.name == "gRPC Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("Upper() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if err == nil && resp.Text != tt.want {
				t.Errorf("Upper() = %v, want %v", resp.Text, tt.want)
			}
			t.Logf("Test %v passed", tt.name)
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

