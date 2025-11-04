package location

import "google.golang.org/grpc"

// DriverLocation represents a streaming update.
type DriverLocation struct {
	DriverId string
	Lat      float64
	Lng      float64
	Speed    float64
	Accuracy float64
	Ts       int64
}

// Ack is returned by the stream.
type Ack struct{}

// LocationServer defines the gRPC contract.
type LocationServer interface {
	StreamLocation(Location_StreamLocationServer) error
}

// RegisterLocationServer registers service implementation.
func RegisterLocationServer(s *grpc.Server, srv LocationServer) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "location.Location",
		HandlerType: (*LocationServer)(nil),
		Streams: []grpc.StreamDesc{{
			StreamName:    "StreamLocation",
			Handler:       _Location_StreamLocation_Handler,
			ServerStreams: true,
			ClientStreams: true,
		}},
	}, srv)
}

// Location_StreamLocationServer defines bidi stream interface.
type Location_StreamLocationServer interface {
	grpc.ServerStream
	SendAndClose(*Ack) error
	Recv() (*DriverLocation, error)
}

func _Location_StreamLocation_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(LocationServer).StreamLocation(&locationStreamServer{ServerStream: stream})
}

type locationStreamServer struct {
	grpc.ServerStream
}

func (s *locationStreamServer) SendAndClose(*Ack) error { return nil }

func (s *locationStreamServer) Recv() (*DriverLocation, error) {
	msg := new(DriverLocation)
	if err := s.ServerStream.RecvMsg(msg); err != nil {
		return nil, err
	}
	return msg, nil
}
