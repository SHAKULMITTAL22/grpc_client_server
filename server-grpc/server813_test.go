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
func (m *mockServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Println("🏥 K8s is health checking")
	if isDatabaseReady == nil {
		log.Printf("🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_UNKNOWN)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_UNKNOWN}, nil
	}
	if *isDatabaseReady {
		log.Printf("✅ Server's status is %s", grpc_health_v1.HealthCheckResponse_SERVING)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	}
	log.Printf("🚫 Server's status is %s", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}, nil
}

func Testcheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	server, lis := startMockServer(t, "50051")
	defer server.Stop()
	defer lis.Close()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	tests := []struct {
		name			string
		databaseReady		*bool
		expectedStatus		grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLog		string
		contextCancelled	bool
	}{{name: "Server Returns SERVING When Database is Ready", databaseReady: boolPtr(true), expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "✅ Server's status is SERVING"}, {name: "Server Returns NOT_SERVING When Database is Not Ready", databaseReady: boolPtr(false), expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLog: "🚫 Server's status is NOT_SERVING"}, {name: "Server Returns UNKNOWN for Unspecified Database State", databaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLog: "🚫 Server's status is UNKNOWN"}, {name: "Context Cancellation Handling", databaseReady: boolPtr(true), expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "✅ Server's status is SERVING", contextCancelled: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDatabaseReady = tt.databaseReady
			ctx := context.Background()
			if tt.contextCancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if tt.contextCancelled {
				if status.Code(err) != codes.Canceled {
					t.Errorf("Expected error code %v, got %v", codes.Canceled, status.Code(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func startMockServer(t *testing.T, port string) (*grpc.Server, net.Listener) {
	t.Helper()
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, &mockServer{})
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
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
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterUpperServer(s, &mockServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	defer s.Stop()
	conn, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperClient(conn)
	tests := []struct {
		name		string
		req		*api.InputRequest
		wantErr		bool
		wantStatus	codes.Code
		wantText	string
	}{{name: "Valid Input", req: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, wantErr: false, wantStatus: codes.OK, wantText: "HELLO😊"}, {name: "Empty Text", req: &api.InputRequest{ClientName: "TestClient", Text: ""}, wantErr: false, wantStatus: codes.OK, wantText: "😊"}, {name: "Special Characters", req: &api.InputRequest{ClientName: "TestClient", Text: "!@#"}, wantErr: false, wantStatus: codes.OK, wantText: "!@#😊"}, {name: "Long Text Input", req: &api.InputRequest{ClientName: "TestClient", Text: strings.Repeat("a", 10000)}, wantErr: false, wantStatus: codes.OK, wantText: strings.ToUpper(strings.Repeat("a", 10000)) + "😊"}, {name: "Nil Input Request", req: nil, wantErr: true, wantStatus: codes.InvalidArgument, wantText: ""}, {name: "Context Cancellation", req: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, wantErr: true, wantStatus: codes.Canceled, wantText: ""}, {name: "Mixed Case Text", req: &api.InputRequest{ClientName: "TestClient", Text: "HeLLo"}, wantErr: false, wantStatus: codes.OK, wantText: "HELLO😊"}, {name: "Non-ASCII Characters", req: &api.InputRequest{ClientName: "TestClient", Text: "こんにちは"}, wantErr: false, wantStatus: codes.OK, wantText: "こんにちは😊"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if tt.name == "Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
				} else {
					st, _ := status.FromError(err)
					if st.Code() != tt.wantStatus {
						t.Errorf("Expected status %v, got %v", tt.wantStatus, st.Code())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if resp.Text != tt.wantText {
						t.Errorf("Expected text %v, got %v", tt.wantText, resp.Text)
					}
				}
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

