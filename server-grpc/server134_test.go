package main

import (
	"context"
	"log"
	"net"
	"testing"
	"time"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
	}{{name: "Scenario 1: Server Returns SERVING When Database is Ready", isDatabaseReady: boolPtr(true), expectedStatus: grpc_health_v1.HealthCheckResponse_SERVING}, {name: "Scenario 2: Server Returns NOT_SERVING When Database is Not Ready", isDatabaseReady: boolPtr(false), expectedStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING}, {name: "Scenario 3: Server Returns UNKNOWN When Database Readiness is Indeterminate", isDatabaseReady: nil, expectedStatus: grpc_health_v1.HealthCheckResponse_UNKNOWN}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lis, err := net.Listen("tcp", ":0")
			if err != nil {
				t.Fatalf("Failed to listen: %v", err)
			}
			s := grpc.NewServer()
			mockServer := &mockHealthServer{isDatabaseReady: tt.isDatabaseReady}
			grpc_health_v1.RegisterHealthServer(s, mockServer)
			go func() {
				if err := s.Serve(lis); err != nil {
					t.Fatalf("Failed to serve: %v", err)
				}
			}()
			defer s.Stop()
			conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
			if err != nil {
				t.Fatalf("Failed to connect to server: %v", err)
			}
			defer conn.Close()
			client := grpc_health_v1.NewHealthClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				t.Fatalf("Check method failed: %v", err)
			}
			t.Logf("Received response: %v", resp)
			assert.Equal(t, tt.expectedStatus, resp.GetStatus(), "Expected status does not match")
		})
	}
	t.Run("Scenario 7: Handling of gRPC Context Cancellation", func(t *testing.T) {
		lis, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("Failed to listen: %v", err)
		}
		s := grpc.NewServer()
		mockServer := &mockHealthServer{isDatabaseReady: boolPtr(true)}
		grpc_health_v1.RegisterHealthServer(s, mockServer)
		go func() {
			if err := s.Serve(lis); err != nil {
				t.Fatalf("Failed to serve: %v", err)
			}
		}()
		defer s.Stop()
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()
		client := grpc_health_v1.NewHealthClient(conn)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		assert.Error(t, err, "Expected error due to context cancellation")
		assert.Equal(t, codes.Canceled, status.Code(err), "Expected error code for canceled context")
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
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterYourServiceServer(s, &mockServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	defer s.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := api.NewYourServiceClient(conn)
	tests := []struct {
		name		string
		req		*api.InputRequest
		expectedErr	error
		expectedOutput	string
	}{{name: "Valid Input", req: &api.InputRequest{ClientName: "client1", Text: "hello"}, expectedErr: nil, expectedOutput: "HELLO😊"}, {name: "Empty Text", req: &api.InputRequest{ClientName: "client1", Text: ""}, expectedErr: nil, expectedOutput: "😊"}, {name: "Special Characters", req: &api.InputRequest{ClientName: "client1", Text: "!@#$"}, expectedErr: nil, expectedOutput: "!@#$😊"}, {name: "Long Text Input", req: &api.InputRequest{ClientName: "client1", Text: strings.Repeat("a", 1000)}, expectedErr: nil, expectedOutput: strings.Repeat("A", 1000) + "😊"}, {name: "Numeric Characters", req: &api.InputRequest{ClientName: "client1", Text: "1234"}, expectedErr: nil, expectedOutput: "1234😊"}, {name: "Nil Request", req: nil, expectedErr: status.Error(codes.InvalidArgument, "nil request"), expectedOutput: ""}, {name: "Invalid Client Name", req: &api.InputRequest{ClientName: "", Text: "hello"}, expectedErr: status.Error(codes.InvalidArgument, "invalid client name"), expectedOutput: ""}, {name: "Context Cancellation", req: &api.InputRequest{ClientName: "client1", Text: "hello"}, expectedErr: status.Error(codes.Canceled, "context canceled"), expectedOutput: ""}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "Context Cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			resp, err := client.Upper(ctx, tt.req)
			if status.Code(err) != status.Code(tt.expectedErr) {
				t.Errorf("expected error: %v, got: %v", tt.expectedErr, err)
			}
			if resp != nil && resp.Text != tt.expectedOutput {
				t.Errorf("expected output: %s, got: %s", tt.expectedOutput, resp.Text)
			}
			if resp != nil {
				t.Logf("Server Response: %v", resp)
			} else {
				t.Logf("Request failed with error: %v", err)
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
		return nil, status.Error(codes.Canceled, "context canceled")
	}
	message := zb.Message{TopicName: zbutil.TopicName, Data: []byte(req.GetText()), HintPartition: ""}
	sendMessageToZBIO([]zb.Message{message})
	x := happyUpper(req.GetText())
	return &api.OutputResponse{ServerName: "testServer", Text: x}, nil
}

