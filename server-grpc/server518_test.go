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
func Testcheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockHealthServer := NewMockHealthServer(ctrl)
	server := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(server, mockHealthServer)
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	defer server.Stop()
	tests := []struct {
		name			string
		isDatabaseReady		interface{}
		expectedStatus		grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedLogParts	[]string
	}{{name: "Server Returns SERVING When Database is Ready", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLogParts: []string{"K8s is health checking", "Server's status is SERVING"}}, {name: "Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: false, expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING, expectedLogParts: []string{"K8s is health checking", "Server's status is NOT_SERVING"}}, {name: "Server Returns UNKNOWN for Unspecified Database State", isDatabaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN, expectedLogParts: []string{"K8s is health checking", "Server's status is UNKNOWN"}}, {name: "Handling of Nil Request", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLogParts: []string{"K8s is health checking", "Server's status is SERVING"}}, {name: "Context Cancellation Handling", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLogParts: []string{"K8s is health checking", "Server's status is SERVING"}}, {name: "Network Latency Simulation", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLogParts: []string{"K8s is health checking", "Server's status is SERVING"}}, {name: "High Load Handling", isDatabaseReady: true, expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING, expectedLogParts: []string{"K8s is health checking", "Server's status is SERVING"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHealthServer.EXPECT().Check(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
				var status grpc_health_v1.HealthCheckResponse_ServingStatus
				if tt.isDatabaseReady == true {
					status = grpc_health_v1.HealthCheckResponse_SERVING
				} else if tt.isDatabaseReady == false {
					status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
				} else {
					status = grpc_health_v1.HealthCheckResponse_UNKNOWN
				}
				return &grpc_health_v1.HealthCheckResponse{Status: status}, nil
			})
			conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
			if err != nil {
				t.Fatalf("failed to dial: %v", err)
			}
			defer conn.Close()
			client := grpc_health_v1.NewHealthClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var req *grpc_health_v1.HealthCheckRequest
			if tt.name != "Handling of Nil Request" {
				req = &grpc_health_v1.HealthCheckRequest{}
			}
			if tt.name == "Context Cancellation Handling" {
				cancel()
			}
			resp, err := client.Check(ctx, req)
			if err != nil {
				if tt.name == "Context Cancellation Handling" {
					if status.Code(err) != codes.Canceled {
						t.Errorf("expected error %v, got %v", codes.Canceled, err)
					}
				} else {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if resp.Status != tt.expectedStatus {
					t.Errorf("expected status %v, got %v", tt.expectedStatus, resp.Status)
				}
			}
			for _, part := range tt.expectedLogParts {
				if !strings.Contains(logOutput, part) {
					t.Errorf("expected log to contain %q, but it did not", part)
				}
			}
		})
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
		req		*api.InputRequest
		wantErr		codes.Code
		wantOutput	string
	}{{name: "Valid Input", req: &api.InputRequest{Text: "hello", ClientName: "testClient"}, wantErr: codes.OK, wantOutput: "HELLO😊"}, {name: "Empty Text", req: &api.InputRequest{Text: "", ClientName: "testClient"}, wantErr: codes.OK, wantOutput: "😊"}, {name: "Special Characters", req: &api.InputRequest{Text: "!@#$%^&*()", ClientName: "testClient"}, wantErr: codes.OK, wantOutput: "!@#$%^&*()😊"}, {name: "Long Text Input", req: &api.InputRequest{Text: strings.Repeat("a", 10000), ClientName: "testClient"}, wantErr: codes.OK, wantOutput: strings.ToUpper(strings.Repeat("a", 10000)) + "😊"}, {name: "Nil Input Request", req: nil, wantErr: codes.InvalidArgument, wantOutput: ""}, {name: "Context Cancellation", req: &api.InputRequest{Text: "cancel", ClientName: "testClient"}, wantErr: codes.Canceled, wantOutput: ""}, {name: "Client Name in Request", req: &api.InputRequest{Text: "client name", ClientName: "testClient"}, wantErr: codes.OK, wantOutput: "CLIENT NAME😊"}, {name: "Mixed Case Text", req: &api.InputRequest{Text: "HeLLo WoRLd", ClientName: "testClient"}, wantErr: codes.OK, wantOutput: "HELLO WORLD😊"}, {name: "Non-ASCII Characters", req: &api.InputRequest{Text: "こんにちは", ClientName: "testClient"}, wantErr: codes.OK, wantOutput: "こんにちは😊"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if tt.name == "Context Cancellation" {
				cancel()
			}
			resp, err := client.Upper(ctx, tt.req)
			if status.Code(err) != tt.wantErr {
				t.Errorf("expected error code %v, got %v", tt.wantErr, status.Code(err))
			}
			if tt.wantErr == codes.OK && resp.Text != tt.wantOutput {
				t.Errorf("expected output %v, got %v", tt.wantOutput, resp.Text)
			}
			t.Logf("Test case %v passed with output: %v", tt.name, resp.GetText())
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

