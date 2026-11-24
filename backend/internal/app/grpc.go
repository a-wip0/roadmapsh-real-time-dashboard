package app

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"io"

	pb "github.com/a-wip0/roadmapsh-real-time-dashboard/gen/leaderboard/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const apiKeyHeader = "x-api-key"

// GRPCServer returns the trusted ingestion server. Every call must carry the
// configured IngestKey in the x-api-key metadata header; with no key
// configured, every call is rejected.
func (a *App) GRPCServer() *grpc.Server {
	s := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
			if err := a.checkKey(ctx); err != nil {
				return nil, err
			}
			return h(ctx, req)
		}),
		grpc.StreamInterceptor(func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, h grpc.StreamHandler) error {
			if err := a.checkKey(ss.Context()); err != nil {
				return err
			}
			return h(srv, ss)
		}),
	)
	pb.RegisterIngestServer(s, &ingest{app: a})
	return s
}

func (a *App) checkKey(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	keys := md.Get(apiKeyHeader)
	if a.cfg.IngestKey == "" || len(keys) != 1 ||
		subtle.ConstantTimeCompare([]byte(keys[0]), []byte(a.cfg.IngestKey)) != 1 {
		return status.Error(codes.Unauthenticated, "missing or invalid "+apiKeyHeader)
	}
	return nil
}

var errUnknownUser = errors.New("unknown player")

func (a *App) userExists(ctx context.Context, name string) (bool, error) {
	var one int
	err := a.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE username=?`, name).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// ingestScore is the shared path for both RPCs: reject unknown players, then
// use the same Submit as the HTTP API.
func (a *App) ingestScore(ctx context.Context, user, game string, score int64) (*SubmitResult, error) {
	ok, err := a.userExists(ctx, user)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errUnknownUser
	}
	return a.Submit(ctx, user, game, score)
}

func grpcErr(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errUnknownUser):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

type ingest struct {
	pb.UnimplementedIngestServer
	app *App
}

func (s *ingest) SubmitScore(ctx context.Context, req *pb.SubmitScoreRequest) (*pb.SubmitScoreResponse, error) {
	r, err := s.app.ingestScore(ctx, req.Username, req.Game, req.Score)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &pb.SubmitScoreResponse{
		Game: r.Game, Score: r.Score, PersonalBest: r.PersonalBest, NewPersonalBest: r.NewPersonalBest,
		GameRank: r.GameRank, GlobalRank: r.GlobalRank,
	}, nil
}

func (s *ingest) StreamScores(stream grpc.ClientStreamingServer[pb.SubmitScoreRequest, pb.StreamSummary]) error {
	var sum pb.StreamSummary
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&sum)
		}
		if err != nil {
			return err
		}
		if _, err := s.app.ingestScore(stream.Context(), req.Username, req.Game, req.Score); err != nil {
			sum.Rejected++
		} else {
			sum.Accepted++
		}
	}
}
