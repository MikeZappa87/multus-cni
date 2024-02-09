package libkni

import (
	"net"
	"os"

	"github.com/MikeZappa87/kni-api/pkg/apis/runtime/beta"
	"github.com/MikeZappa87/libkni/pkg/cni"
	"google.golang.org/grpc"
)

func NewKNIServer(sockAddr, protocol string, implementation beta.KNIServer) error {
	if _, err := os.Stat(sockAddr); !os.IsNotExist(err) {
		if err := os.RemoveAll(sockAddr); err != nil {
			return err
		}
	}

	listener, err := net.Listen(protocol, sockAddr)
	if err != nil {
		return err
	}

	server := grpc.NewServer()

	beta.RegisterKNIServer(server, implementation)

	return server.Serve(listener)
}

func NewDefaultKNIServer(sockAddr, protocol string, config cni.KNIConfig) error {
	if _, err := os.Stat(sockAddr); !os.IsNotExist(err) {
		if err := os.RemoveAll(sockAddr); err != nil {
			return err
		}
	}

	listener, err := net.Listen(protocol, sockAddr)
	if err != nil {
		return err
	}

	server := grpc.NewServer()

	srv, err := cni.NewKniService(&config)

	if err != nil {
		return err
	}
	
	beta.RegisterKNIServer(server, srv)

	return server.Serve(listener)
}
