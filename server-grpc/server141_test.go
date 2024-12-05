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
	"strings"
	"fmt"
	"github.com/roost-io/roost-example-latest/grpcExample/api"
	zb "github.com/ZB-io/zbio/client"
	zbutil "github.com/roost-io/roost-example-latest/grpcExample/message"
)

/*
ROOST_METHOD_HASH=Check_a316e66539
ROOST_METHOD_SIG_HASH=Check_a316e66539


 */
func (s *MockServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Println("🏥 K8s is health checking")
	if s.isDatabaseReady {
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
		isDatabaseReady	bool
		expectedStatus	grpc_health_v1.HealthCheckResponse_ServingStatus
		expectedError	error
	}{{"Database is Ready", true, grpc_health_v1.HealthCheckResponse_SERVING, nil}, {"Database is Not Ready", false, grpc_health_v1.HealthCheckResponse_NOT_SERVING, nil}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server, lis := startMockServer(tt.isDatabaseReady)
			defer server.Stop()
			defer lis.Close()
			resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
			if tt.expectedError != nil {
				if err == nil || status.Code(err) != codes.Unavailable {
					t.Fatalf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if resp.Status != tt.expectedStatus {
					t.Fatalf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
				}
			}
		})
	}
	t.Run("Context Cancellation", func(t *testing.T) {
		client, server, lis := startMockServer(true)
		defer server.Stop()
		defer lis.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err == nil || status.Code(err) != codes.Canceled {
			t.Fatalf("Expected context canceled error, got %v", err)
		}
	})
	t.Run("Invalid HealthCheckRequest", func(t *testing.T) {
		client, server, lis := startMockServer(true)
		defer server.Stop()
		defer lis.Close()
		resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Fatalf("Expected status SERVING, got %v", resp.Status)
		}
	})
	t.Run("Concurrent Health Checks", func(t *testing.T) {
		client, server, lis := startMockServer(true)
		defer server.Stop()
		defer lis.Close()
		var wg sync.WaitGroup
		numConcurrentRequests := 10
		for i := 0; i < numConcurrentRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
					t.Errorf("Expected status SERVING, got %v", resp.Status)
				}
			}()
		}
		wg.Wait()
	})
}

func startMockServer(isDatabaseReady bool) (grpc_health_v1.HealthClient, *grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	mockServer := &MockServer{isDatabaseReady: isDatabaseReady}
	grpc_health_v1.RegisterHealthServer(s, mockServer)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial server: %v", err)
	}
	client := grpc_health_v1.NewHealthClient(conn)
	return client, s, lis
}

/*
ROOST_METHOD_HASH=Watch_ee291f18f7
ROOST_METHOD_SIG_HASH=Watch_ee291f18f7


 */
func TestWatch(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &MockHealthServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	defer s.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	tests := []struct {
		name	string
		request	*grpc_health_v1.HealthCheckRequest
	}{{"Verify Unimplemented Watch Method Error", &grpc_health_v1.HealthCheckRequest{Service: "test"}}, {"Handle Nil HealthCheckRequest", nil}, {"Test WatchServer Stream Behavior", &grpc_health_v1.HealthCheckRequest{Service: "test"}}, {"Simulate Concurrent Watch Requests", &grpc_health_v1.HealthCheckRequest{Service: "test"}}, {"Evaluate Watch Method with Invalid Context", &grpc_health_v1.HealthCheckRequest{Service: "test"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "Evaluate Watch Method with Invalid Context" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			stream, err := client.Watch(ctx, tt.request)
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Expected grpc status error, got %v", err)
			}
			if st.Code() != codes.Unimplemented {
				t.Errorf("Expected code %v, got %v", codes.Unimplemented, st.Code())
			}
			if !strings.Contains(st.Message(), "Watching is not supported") {
				t.Errorf("Expected error message to contain 'Watching is not supported', got %v", st.Message())
			}
			if tt.name == "Simulate Concurrent Watch Requests" {
				var wg sync.WaitGroup
				numRequests := 10
				for i := 0; i < numRequests; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						stream, err := client.Watch(ctx, tt.request)
						if err == nil {
							t.Errorf("Expected error, got nil")
						}
						st, ok := status.FromError(err)
						if !ok {
							t.Errorf("Expected grpc status error, got %v", err)
						}
						if st.Code() != codes.Unimplemented {
							t.Errorf("Expected code %v, got %v", codes.Unimplemented, st.Code())
						}
					}()
				}
				wg.Wait()
			}
		})
	}
	t.Log("TestWatch completed successfully")
}

func (h *MockHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watching is not supported")
}

/*
ROOST_METHOD_HASH=Upper_6c4de803cd
ROOST_METHOD_SIG_HASH=Upper_6c4de803cd


 */
func (m *mockZBIOClient) SendMessageToZBIO(messages []zb.Message) error {
	for _, msg := range messages {
		if strings.Contains(string(msg.Data), "fail") {
			return fmt.Errorf("failed to send message")
		}
	}
	return nil
}

func TestUpper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockZBIO := &mockZBIOClient{}
	zbutil.SendMessageToZBIO = mockZBIO.SendMessageToZBIO
	lis, s := startTestServer(t)
	defer s.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
	client := api.NewUpperServiceClient(conn)
	tests := []struct {
		name		string
		input		*api.InputRequest
		expectedText	string
		expectedError	codes.Code
	}{{name: "Valid Input", input: &api.InputRequest{ClientName: "Client1", Text: "hello"}, expectedText: "HELLO😊", expectedError: codes.OK}, {name: "Empty Text Input", input: &api.InputRequest{ClientName: "Client2", Text: ""}, expectedText: "😊", expectedError: codes.OK}, {name: "Long Text Input", input: &api.InputRequest{ClientName: "Client3", Text: strings.Repeat("a", 1000)}, expectedText: strings.ToUpper(strings.Repeat("a", 1000)) + "😊", expectedError: codes.OK}, {name: "Empty Client Name", input: &api.InputRequest{ClientName: "", Text: "hello"}, expectedText: "HELLO😊", expectedError: codes.OK}, {name: "Special Characters", input: &api.InputRequest{ClientName: "Client4", Text: "!@#$%"}, expectedText: "!@#$%😊", expectedError: codes.OK}, {name: "Mixed Case Input", input: &api.InputRequest{ClientName: "Client5", Text: "HeLLo WorLD"}, expectedText: "HELLO WORLD😊", expectedError: codes.OK}, {name: "ZBIO Failure", input: &api.InputRequest{ClientName: "fail", Text: "hello"}, expectedText: "", expectedError: codes.Internal}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Upper(context.Background(), tt.input)
			if status.Code(err) != tt.expectedError {
				t.Errorf("expected error code %v, got %v", tt.expectedError, status.Code(err))
			}
			if tt.expectedError == codes.OK && resp.Text != tt.expectedText {
				t.Errorf("expected text %s, got %s", tt.expectedText, resp.Text)
			}
			if resp != nil && resp.ServerName != serverName {
				t.Errorf("expected server name %s, got %s", serverName, resp.ServerName)
			}
		})
	}
	t.Run("Simulate Network Latency", func(t *testing.T) {
		start := time.Now()
		_, err := client.Upper(context.Background(), &api.InputRequest{ClientName: "ClientLatency", Text: "latency test"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		duration := time.Since(start)
		t.Logf("Response time: %v", duration)
		if duration > 2*time.Second {
			t.Errorf("response time exceeded acceptable limits")
		}
	})
}

func (s *server) Upper(ctx context.Context, req *api.InputRequest) (*api.OutputResponse, error) {
	reqAck := fmt.Sprintf("➡️ Received message from client %v: %v ", req.GetClientName(), req.GetText())
	log.Printf(reqAck)
	message := zb.Message{TopicName: zbutil.TopicName, Data: []byte(reqAck), HintPartition: ""}
	zbutil.SendMessageToZBIO([]zb.Message{message})
	x := happyUpper(req.GetText())
	return &api.OutputResponse{ServerName: serverName, Text: x}, nil
}

func startTestServer(t *testing.T) (net.Listener, *grpc.Server) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterUpperServiceServer(s, &server{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	return lis, s
}

