package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"time"
	"github.com/golang/mock/gomock"
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
	tests := []struct {
		name			string
		databaseReady		bool
		expectedStatus		grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLog		string
		expectedGRPCCode	codes.Code
	}{{name: "Scenario 1: Server Returns SERVING When Database is Ready", databaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLog: "✅ Server's status is SERVING", expectedGRPCCode: codes.OK}, {name: "Scenario 2: Server Returns NOT_SERVING When Database is Not Ready", databaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLog: "🚫 Server's status is NOT_SERVING", expectedGRPCCode: codes.OK}, {name: "Scenario 3: Server Returns UNKNOWN When Database Readiness is Indeterminate", databaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLog: "🚫 Server's status is UNKNOWN", expectedGRPCCode: codes.OK}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDatabaseReady = tt.databaseReady
			ctx := context.Background()
			conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}
			defer conn.Close()
			client := grpc_health_v1.NewHealthClient(conn)
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() != tt.expectedGRPCCode {
					t.Errorf("Expected gRPC code %v, got %v", tt.expectedGRPCCode, st.Code())
				}
				t.Logf("Received expected error: %v", err)
				return
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
			t.Logf("Expected log: %s", tt.expectedLog)
		})
	}
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &Health{})
	reflection.Register(s)
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
func Testupper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	lis, s := startMockServer(t)
	defer s.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperServiceClient(conn)
	tests := []struct {
		name		string
		input		*api.InputRequest
		wantErr		bool
		wantOutput	string
	}{{name: "Valid Input", input: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, wantErr: false, wantOutput: "HELLO😊"}, {name: "Empty Text", input: &api.InputRequest{ClientName: "TestClient", Text: ""}, wantErr: false, wantOutput: "😊"}, {name: "Special Characters", input: &api.InputRequest{ClientName: "TestClient", Text: "!@#$%^&*()"}, wantErr: false, wantOutput: "!@#$%^&*()😊"}, {name: "Long Text Input", input: &api.InputRequest{ClientName: "TestClient", Text: strings.Repeat("a", 1000)}, wantErr: false, wantOutput: strings.ToUpper(strings.Repeat("a", 1000)) + "😊"}, {name: "Nil Input Request", input: nil, wantErr: true, wantOutput: ""}, {name: "gRPC Context Cancellation", input: &api.InputRequest{ClientName: "TestClient", Text: "hello"}, wantErr: true, wantOutput: ""}, {name: "Mixed Case Text", input: &api.InputRequest{ClientName: "TestClient", Text: "HeLLo WoRLd"}, wantErr: false, wantOutput: "HELLO WORLD😊"}, {name: "Non-ASCII Characters", input: &api.InputRequest{ClientName: "TestClient", Text: "こんにちは"}, wantErr: false, wantOutput: "こんにちは😊"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tt.name == "gRPC Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && resp.Text != tt.wantOutput {
				t.Errorf("Upper() got = %v, want %v", resp.Text, tt.wantOutput)
			}
			if err != nil {
				t.Logf("Expected error: %v", err)
			} else {
				t.Logf("Success: received response %v", resp.Text)
			}
		})
	}
}

func (m *mockServer) Upper(ctx context.Context, req *api.InputRequest) (*api.OutputResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	if ctx.Err() == context.Canceled {
		return nil, status.Error(codes.Canceled, "context canceled")
	}
	reqAck := fmt.Sprintf("➡️ Received message from client %v: %v ", req.GetClientName(), req.GetText())
	log.Printf(reqAck)
	message := zb.Message{TopicName: zbutil.TopicName, Data: []byte(reqAck), HintPartition: ""}
	zbutil.SendMessageToZBIO([]zb.Message{message})
	x := happyUpper(req.GetText())
	return &api.OutputResponse{ServerName: "mockServer", Text: x}, nil
}

func startMockServer(t *testing.T) (net.Listener, *grpc.Server) {
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
	return lis, s
}

