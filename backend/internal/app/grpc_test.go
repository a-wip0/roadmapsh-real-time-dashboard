package app

import (
	"context"
	"net"
	"testing"

	pb "github.com/a-wip0/roadmapsh-real-time-dashboard/gen/leaderboard/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testKey = "ingest-test-key"

// newIngestClient starts the gRPC server on a free port and returns a client
// plus a context carrying the valid API key.
func newIngestClient(t *testing.T) (*App, pb.IngestClient, context.Context) {
	t.Helper()
	a := newTestApp(t)
	a.cfg.IngestKey = testKey
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := a.GRPCServer()
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	ctx := metadata.AppendToOutgoingContext(context.Background(), apiKeyHeader, testKey)
	return a, pb.NewIngestClient(conn), ctx
}

func addUser(t *testing.T, a *App, name string) {
	t.Helper()
	if _, err := a.db.Exec(`INSERT INTO users(username, password_hash) VALUES(?, 'x')`, name); err != nil {
		t.Fatal(err)
	}
}

func wantCode(t *testing.T, what string, err error, want codes.Code) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Errorf("%s: code = %v (%v), want %v", what, got, err, want)
	}
}

func TestIngestAuth(t *testing.T) {
	a, c, _ := newIngestClient(t)
	addUser(t, a, "alice")
	req := &pb.SubmitScoreRequest{Username: "alice", Game: "chess", Score: 10}

	_, err := c.SubmitScore(context.Background(), req)
	wantCode(t, "no key", err, codes.Unauthenticated)
	bad := metadata.AppendToOutgoingContext(context.Background(), apiKeyHeader, "wrong")
	_, err = c.SubmitScore(bad, req)
	wantCode(t, "wrong key", err, codes.Unauthenticated)

	st, err := c.StreamScores(context.Background())
	if err == nil {
		_, err = st.CloseAndRecv()
	}
	wantCode(t, "stream without key", err, codes.Unauthenticated)

	// a server with no configured key rejects everything, even an empty key
	a.cfg.IngestKey = ""
	empty := metadata.AppendToOutgoingContext(context.Background(), apiKeyHeader, "")
	_, err = c.SubmitScore(empty, req)
	wantCode(t, "key disabled", err, codes.Unauthenticated)
}

func TestIngestSubmitScore(t *testing.T) {
	a, c, ctx := newIngestClient(t)
	addUser(t, a, "alice")

	res, err := c.SubmitScore(ctx, &pb.SubmitScoreRequest{Username: "alice", Game: "chess", Score: 100})
	if err != nil || !res.NewPersonalBest || res.PersonalBest != 100 || res.GameRank != 1 || res.GlobalRank != 1 {
		t.Fatalf("submit = %+v err=%v", res, err)
	}
	res, err = c.SubmitScore(ctx, &pb.SubmitScoreRequest{Username: "alice", Game: "chess", Score: 40})
	if err != nil || res.NewPersonalBest || res.PersonalBest != 100 {
		t.Errorf("lower = %+v err=%v", res, err)
	}

	// goes through the same path as REST: board and SQLite history both updated
	if g, _ := a.top(context.Background(), globalKey, 10); len(g) != 1 || g[0].Score != 100 {
		t.Errorf("global = %+v", g)
	}
	var n int
	a.db.QueryRow(`SELECT COUNT(*) FROM scores WHERE username='alice'`).Scan(&n)
	if n != 2 {
		t.Errorf("history rows = %d, want 2", n)
	}

	_, err = c.SubmitScore(ctx, &pb.SubmitScoreRequest{Username: "ghost", Game: "chess", Score: 5})
	wantCode(t, "unknown player", err, codes.NotFound)
	if g, _ := a.top(context.Background(), globalKey, 10); len(g) != 1 {
		t.Errorf("unknown player must not reach the board: %+v", g)
	}
	_, err = c.SubmitScore(ctx, &pb.SubmitScoreRequest{Username: "alice", Game: "bad game", Score: 5})
	wantCode(t, "bad game", err, codes.InvalidArgument)
	_, err = c.SubmitScore(ctx, &pb.SubmitScoreRequest{Username: "alice", Game: "chess", Score: -1})
	wantCode(t, "negative score", err, codes.InvalidArgument)
}

func TestIngestStream(t *testing.T) {
	a, c, ctx := newIngestClient(t)
	addUser(t, a, "alice")
	addUser(t, a, "bob")

	st, err := c.StreamScores(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []*pb.SubmitScoreRequest{
		{Username: "alice", Game: "chess", Score: 100},
		{Username: "ghost", Game: "chess", Score: 999}, // rejected, stream continues
		{Username: "bob", Game: "chess", Score: 120},
		{Username: "bob", Game: "x y", Score: 1}, // rejected
		{Username: "alice", Game: "pong", Score: 30},
	} {
		if err := st.Send(r); err != nil {
			t.Fatal(err)
		}
	}
	sum, err := st.CloseAndRecv()
	if err != nil || sum.Accepted != 3 || sum.Rejected != 2 {
		t.Fatalf("summary = %+v err=%v", sum, err)
	}
	g, _ := a.top(context.Background(), globalKey, 10)
	if len(g) != 2 || g[0] != (Entry{1, "alice", 130}) || g[1] != (Entry{2, "bob", 120}) {
		t.Errorf("global = %+v", g)
	}
}
