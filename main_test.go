package main

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	fakerpb "go.cadenya.com/faker-mcp/fakerpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
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
