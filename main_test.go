package main

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	fakerpb "go.cadenya.com/faker-mcp/fakerpb"
	grpcmcpgatewayv1 "go.cadenya.com/mcp-grpc-gateway/gen/grpcmcpgateway/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestFakerServiceListsFiltersAndGeneratesValues(t *testing.T) {
	server := newFakerServer()

	all, err := server.GetFakerOptions(context.Background(), &fakerpb.GetFakerOptionsRequest{})
	require.NoError(t, err)
	require.Greater(t, len(all.GetOptions()), 100)
	require.Contains(t, optionNames(all.GetOptions()), "internet.email")
	require.Contains(t, optionNames(all.GetOptions()), "lorem.sentence")
	require.Contains(t, optionNames(all.GetOptions()), "faker.int_between")

	filtered, err := server.GetFakerOptions(context.Background(), &fakerpb.GetFakerOptionsRequest{Filter: "email"})
	require.NoError(t, err)
	require.Contains(t, optionNames(filtered.GetOptions()), "internet.email")

	generated, err := server.GenerateFake(context.Background(), &fakerpb.GenerateFakeRequest{Name: "internet.email"})
	require.NoError(t, err)
	require.Equal(t, "internet.email", generated.GetName())
	require.Contains(t, generated.GetValue(), "@")
}

func TestFakerServiceExposesArgumentsAndUsesDefaults(t *testing.T) {
	server := newFakerServer()

	all, err := server.GetFakerOptions(context.Background(), &fakerpb.GetFakerOptionsRequest{Filter: "lorem.sentence"})
	require.NoError(t, err)
	require.NotEmpty(t, all.GetOptions())
	option := findOption(t, all.GetOptions(), "lorem.sentence")
	require.Equal(t, "count", option.GetArguments()[0].GetName())
	require.Equal(t, "integer", option.GetArguments()[0].GetType())
	require.Equal(t, float64(8), option.GetArguments()[0].GetDefault().GetNumberValue())

	generated, err := server.GenerateFake(context.Background(), &fakerpb.GenerateFakeRequest{Name: "lorem.sentence"})
	require.NoError(t, err)
	require.Equal(t, "lorem.sentence", generated.GetName())
	require.NotEmpty(t, generated.GetValue())
}

func TestFakerServiceRejectsUnknownFakeName(t *testing.T) {
	_, err := newFakerServer().GenerateFake(context.Background(), &fakerpb.GenerateFakeRequest{Name: "missing.fake"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported faker name")
}

func TestFakerServiceGRPCRoundTrip(t *testing.T) {
	addr, stop := startTestFakerGRPC(t)
	defer stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := fakerpb.NewFakerServiceClient(conn)

	resp, err := client.GenerateFake(context.Background(), &fakerpb.GenerateFakeRequest{Name: "internet.email"})
	require.NoError(t, err)
	require.Contains(t, resp.GetValue(), "@")
}

func TestFakerServiceGeneratesPG13CurseWords(t *testing.T) {
	server := newFakerServer()
	allowed := map[string]struct{}{
		"crap":   {},
		"damn":   {},
		"heck":   {},
		"jerk":   {},
		"moron":  {},
		"stupid": {},
	}

	for range 20 {
		resp, err := server.GenerateCurseWord(context.Background(), &fakerpb.GenerateCurseWordRequest{})
		require.NoError(t, err)
		require.Contains(t, allowed, resp.GetValue())
	}
}

func TestGenerateCurseWordToolAnnotationHasDestructiveHint(t *testing.T) {
	method := fakerpb.File_faker_v1_faker_proto.Services().
		ByName("FakerService").
		Methods().
		ByName("GenerateCurseWord")
	require.NotNil(t, method)

	options := method.Options().(*descriptorpb.MethodOptions)
	require.True(t, proto.HasExtension(options, grpcmcpgatewayv1.E_Tool))
	tool := proto.GetExtension(options, grpcmcpgatewayv1.E_Tool).(*grpcmcpgatewayv1.Tool)

	require.Equal(t, "GenerateCurseWord", tool.GetName())
	require.True(t, tool.GetDestructiveHint())
}

func startTestFakerGRPC(t *testing.T) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	fakerpb.RegisterFakerServiceServer(server, newFakerServer())
	reflection.Register(server)
	go func() {
		_ = server.Serve(listener)
	}()

	return listener.Addr().String(), func() {
		server.Stop()
		_ = listener.Close()
	}
}

func optionNames(options []*fakerpb.FakerOption) []string {
	names := make([]string, 0, len(options))
	for _, option := range options {
		names = append(names, option.GetName())
	}
	return names
}

func findOption(t *testing.T, options []*fakerpb.FakerOption, name string) *fakerpb.FakerOption {
	t.Helper()
	for _, option := range options {
		if option.GetName() == name {
			return option
		}
	}
	t.Fatalf("option %q not found", name)
	return nil
}
