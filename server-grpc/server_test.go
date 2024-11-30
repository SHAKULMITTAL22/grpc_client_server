package main

import (
	"context"
	"log"
	"net"
	"strings"
	"testing"
	"time"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/grpc/codes"
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
func Testcheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tests := []struct {
		name			string
		isDatabaseReady		bool
		expectedStatus		grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLogMessage	string
	}{{name: "Server Returns SERVING When Database is Ready", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLogMessage: "Server's status is SERVING"}, {name: "Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLogMessage: "Server's status is NOT_SERVING"}, {name: "Server Returns UNKNOWN Status for Undefined Database State", isDatabaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLogMessage: "Server's status is UNKNOWN"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDatabaseReady = tt.isDatabaseReady
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}
			defer conn.Close()
			client := grpc_health_v1.NewHealthClient(conn)
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
			logOutput := captureLogOutput(func() {
				client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			})
			if !strings.Contains(logOutput, tt.expectedLogMessage) {
				t.Errorf("Expected log message '%s', but got '%s'", tt.expectedLogMessage, logOutput)
			}
		})
	}
	t.Run("Simulate Context Cancellation Before Response", func(t *testing.T) {
		isDatabaseReady = true
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial bufnet: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err == nil {
			t.Fatalf("Expected error due to context cancellation, but got none")
		}
	})
	t.Run("Evaluate Response Time Under Normal Conditions", func(t *testing.T) {
		isDatabaseReady = true
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial bufnet: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		start := time.Now()
		_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			t.Errorf("Response time too slow: %v", duration)
		}
	})
	t.Run("Evaluate Response Time When Database is Not Ready", func(t *testing.T) {
		isDatabaseReady = false
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial bufnet: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		start := time.Now()
		_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			t.Errorf("Response time too slow: %v", duration)
		}
	})
	t.Run("Concurrent Check Requests Under Load", func(t *testing.T) {
		isDatabaseReady = true
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial bufnet: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		const numRequests = 100
		errChan := make(chan error, numRequests)
		for i := 0; i < numRequests; i++ {
			go func() {
				_, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
				errChan <- err
			}()
		}
		for i := 0; i < numRequests; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("Concurrent Check failed: %v", err)
			}
		}
	})
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func captureLogOutput(f func()) string {
	var buf strings.Builder
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(nil)
	}()
	f()
	return buf.String()
}

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &Health{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
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
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperClient(conn)
	tests := []struct {
		name		string
		request		*api.InputRequest
		wantText	string
		wantErr		codes.Code
	}{{name: "Test Upper Function with Valid Input", request: &api.InputRequest{ClientName: "TestClient", Text: "hello world"}, wantText: "HELLO WORLD😊", wantErr: codes.OK}, {name: "Test Upper Function with Empty Text", request: &api.InputRequest{ClientName: "TestClient", Text: ""}, wantText: "😊", wantErr: codes.OK}, {name: "Test Upper Function with Special Characters", request: &api.InputRequest{ClientName: "TestClient", Text: "!@#$%^&*()"}, wantText: "!@#$%^&*()😊", wantErr: codes.OK}, {name: "Test Upper Function with Long Text Input", request: &api.InputRequest{ClientName: "TestClient", Text: strings.Repeat("a", 10000)}, wantText: strings.Repeat("A", 10000) + "😊", wantErr: codes.OK}, {name: "Test Upper Function with Nil Input Request", request: nil, wantText: "", wantErr: codes.InvalidArgument}, {name: "Test Upper Function Handling of gRPC Context Cancellation", request: &api.InputRequest{ClientName: "TestClient", Text: "cancel"}, wantText: "", wantErr: codes.Canceled}, {name: "Test Upper Function with Client Name in Request", request: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, wantText: "HELLO😊", wantErr: codes.OK}, {name: "Test Upper Function with Mixed Case Text", request: &api.InputRequest{ClientName: "TestClient", Text: "HeLLo WoRLd"}, wantText: "HELLO WORLD😊", wantErr: codes.OK}, {name: "Test Upper Function with Non-ASCII Characters", request: &api.InputRequest{ClientName: "TestClient", Text: "こんにちは世界"}, wantText: "こんにちは世界😊", wantErr: codes.OK}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.name == "Test Upper Function Handling of gRPC Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.request)
			if err != nil {
				if status.Code(err) != tt.wantErr {
					t.Errorf("Expected error code %v, got %v", tt.wantErr, status.Code(err))
				}
				t.Logf("Test %s failed with error: %v", tt.name, err)
				return
			}
			if resp.Text != tt.wantText {
				t.Errorf("Expected response text %q, got %q", tt.wantText, resp.Text)
			} else {
				t.Logf("Test %s succeeded with response: %v", tt.name, resp.Text)
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

