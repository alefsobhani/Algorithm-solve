package location

import (
	"io"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// Server implements the LocationServer interface.
type Server struct {
	observer *StreamObserver
}

// NewServer constructs a server.
func NewServer(observer *StreamObserver) *Server {
	return &Server{observer: observer}
}

// StreamLocation ingests driver locations and updates observer.
func (s *Server) StreamLocation(stream Location_StreamLocationServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&Ack{})
		}
		if err != nil {
			return err
		}
		driverID, err := uuid.Parse(msg.DriverId)
		if err != nil {
			continue
		}
		s.observer.Update(stream.Context(), driverID, domain.GeoPoint{Lat: msg.Lat, Lng: msg.Lng}, msg.Speed, msg.Accuracy)
	}
}
