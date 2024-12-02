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

func Testcheck(t *testing.T) {
	conn, cleanup := startMockGRPCServer(t)
	defer cleanup()
	client := grpc_health_v1.NewHealthClient(conn)
	tests := []struct {
		name		string
		isDatabaseReady	interface{}
		expectedStatus	grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedCode	codes.Code
	}{{name: "Server Returns SERVING When Database is Ready", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedCode: codes.OK}, {name: "Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedCode: codes.OK}, {name: "Server Returns UNKNOWN When Database State is Undefined", isDatabaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedCode: codes.OK}, {name: "Context Cancellation Handling", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedCode: codes.Canceled}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDatabaseReady = tt.isDatabaseReady
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tt.name == "Context Cancellation Handling" {
				cancel()
			}
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			st, _ := status.FromError(err)
			if st.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
			}
			if err == nil && resp.Status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
			t.Logf("Test %s passed with status %v and code %v", tt.name, resp.Status, st.Code())
		})
	}
}

func startMockGRPCServer(t *testing.T) (*grpc.ClientConn, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &mockHealthServer{})
	reflection.Register(s)
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	return conn, func() {
		s.Stop()
		conn.Close()
	}
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
	server, lis := startMockGRPCServer(t, ":50051")
	defer server.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperServiceClient(conn)
	tests := []struct {
		name		string
		req		*api.InputRequest
		wantErr		bool
		expectedMsg	string
	}{{name: "Valid Input", req: &api.InputRequest{ClientName: "TestClient", Text: "hello world"}, expectedMsg: "HELLO WORLD😊"}, {name: "Empty Text", req: &api.InputRequest{ClientName: "TestClient", Text: ""}, expectedMsg: "😊"}, {name: "Special Characters", req: &api.InputRequest{ClientName: "TestClient", Text: "!@#$%^&*()"}, expectedMsg: "!@#$%^&*()😊"}, {name: "Long Text Input", req: &api.InputRequest{ClientName: "TestClient", Text: "a long text input" + strings.Repeat("a", 1000)}, expectedMsg: strings.ToUpper("a long text input"+strings.Repeat("a", 1000)) + "😊"}, {name: "Numeric Characters", req: &api.InputRequest{ClientName: "TestClient", Text: "1234567890"}, expectedMsg: "1234567890😊"}, {name: "Nil Request", req: nil, wantErr: true}, {name: "Invalid Client Name", req: &api.InputRequest{ClientName: "", Text: "test"}, wantErr: true}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing scenario: %s", tc.name)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			resp, err := client.Upper(ctx, tc.req)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
				} else {
					t.Logf("Received expected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Upper function returned error: %v", err)
			}
			if resp.Text != tc.expectedMsg {
				t.Errorf("Unexpected response text: got %v, want %v", resp.Text, tc.expectedMsg)
			} else {
				t.Logf("Received expected response: %v", resp.Text)
			}
		})
	}
	t.Run("Context Cancellation", func(t *testing.T) {
		t.Log("Testing context cancellation")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.Upper(ctx, &api.InputRequest{ClientName: "TestClient", Text: "test"})
		if err == nil || status.Code(err) != codes.Canceled {
			t.Errorf("Expected context canceled error, got: %v", err)
		} else {
			t.Logf("Received expected context cancellation error: %v", err)
		}
	})
}

func (s *mockServer) Upper(ctx context.Context, req *api.InputRequest) (*api.OutputResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	if req.GetClientName() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid client name")
	}
	reqAck := fmt.Sprintf("➡️ Received message from client %v: %v", req.GetClientName(), req.GetText())
	log.Printf(reqAck)
	message := zb.Message{TopicName: zbutil.TopicName, Data: []byte(reqAck), HintPartition: ""}
	zbutil.SendMessageToZBIO([]zb.Message{message})
	x := happyUpper(req.GetText())
	return &api.OutputResponse{ServerName: "TestServer", Text: x}, nil
}

func startMockGRPCServer(t *testing.T, port string) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", port)
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

