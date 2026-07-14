package idle

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/acidsailor/ts3afkmover/config"
	"github.com/acidsailor/ts3afkmover/internal/ts3"
)

const idleChannelID = 9

func TestMain(m *testing.M) {
	// Silence the global logger; the mover logs via slog.Default().
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

type moveCall struct {
	clid int
	cid  int
}

type fakeClient struct {
	listResp ts3.ClientList
	listOK   bool
	moveOK   bool
	sendOK   bool

	moves []moveCall
	gms   []string
}

func (f *fakeClient) ClientListWithTimes(
	context.Context,
) (ts3.ClientList, bool) {
	return f.listResp, f.listOK
}

func (f *fakeClient) MoveClient(_ context.Context, clid, cid int) bool {
	f.moves = append(f.moves, moveCall{clid: clid, cid: cid})
	return f.moveOK
}

func (f *fakeClient) SendGM(_ context.Context, message string) bool {
	f.gms = append(f.gms, message)
	return f.sendOK
}

func newMover(t *testing.T, f *fakeClient) *Mover {
	t.Helper()
	cfg := &config.Config{
		IdleTime:        60,
		IdleChannelId:   idleChannelID,
		MessageTemplate: "User %s was moved to Idle Channel",
	}
	return New(cfg, f)
}

func listWith(entries ...ts3.ClientEntry) ts3.ClientList {
	return ts3.ClientList{Body: entries}
}

func TestMoveIdleClients_MovesIdleRegularClient(t *testing.T) {
	f := &fakeClient{
		listOK: true,
		moveOK: true,
		sendOK: true,
		listResp: listWith(ts3.ClientEntry{
			ClientType:     "0",
			Clid:           "5",
			Cid:            "2",
			ClientIdleTime: "3700000", // > 60 min
			ClientNickname: "bob",
		}),
	}

	newMover(t, f).MoveIdleClients(context.Background())

	if len(f.moves) != 1 ||
		f.moves[0] != (moveCall{clid: 5, cid: idleChannelID}) {
		t.Fatalf(
			"moves = %+v, want one move of clid 5 -> cid %d",
			f.moves,
			idleChannelID,
		)
	}
	if len(f.gms) != 1 || f.gms[0] != "User bob was moved to Idle Channel" {
		t.Fatalf("gms = %+v, want one formatted notification", f.gms)
	}
}

func TestMoveIdleClients_ThresholdIsStrict(t *testing.T) {
	cases := map[string]struct {
		idleMillis string
		wantMoved  bool
	}{
		"below threshold": {"1000", false},
		"exactly threshold": {
			"3600000",
			false,
		}, // equal must NOT move (strict >)
		"above threshold": {"3600001", true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := &fakeClient{
				listOK: true, moveOK: true, sendOK: true,
				listResp: listWith(ts3.ClientEntry{
					ClientType: "0", Clid: "5", Cid: "2",
					ClientIdleTime: c.idleMillis, ClientNickname: "bob",
				}),
			}

			newMover(t, f).MoveIdleClients(context.Background())

			if got := len(f.moves) == 1; got != c.wantMoved {
				t.Fatalf(
					"moved = %v (moves=%+v), want %v",
					got,
					f.moves,
					c.wantMoved,
				)
			}
		})
	}
}

func TestMoveIdleClients_SkipsQueryClient(t *testing.T) {
	f := &fakeClient{
		listOK: true, moveOK: true, sendOK: true,
		listResp: listWith(ts3.ClientEntry{
			ClientType: "1", Clid: "5", Cid: "2",
			ClientIdleTime: "9999999", ClientNickname: "querybot",
		}),
	}

	newMover(t, f).MoveIdleClients(context.Background())

	if len(f.moves) != 0 {
		t.Fatalf("moves = %+v, want no moves for a query client", f.moves)
	}
}

func TestMoveIdleClients_SkipsClientAlreadyInIdleChannel(t *testing.T) {
	f := &fakeClient{
		listOK: true, moveOK: true, sendOK: true,
		listResp: listWith(ts3.ClientEntry{
			ClientType: "0", Clid: "5", Cid: "9", // already in idle channel
			ClientIdleTime: "9999999", ClientNickname: "bob",
		}),
	}

	newMover(t, f).MoveIdleClients(context.Background())

	if len(f.moves) != 0 {
		t.Fatalf(
			"moves = %+v, want no moves for a client already in the idle channel",
			f.moves,
		)
	}
}

func TestMoveIdleClients_ParseErrorSkipsClient(t *testing.T) {
	f := &fakeClient{
		listOK: true, moveOK: true, sendOK: true,
		listResp: listWith(ts3.ClientEntry{
			ClientType: "not-a-number", Clid: "5", Cid: "2",
			ClientIdleTime: "9999999", ClientNickname: "bob",
		}),
	}

	newMover(t, f).MoveIdleClients(context.Background())

	if len(f.moves) != 0 {
		t.Fatalf(
			"moves = %+v, want no moves when a field fails to parse",
			f.moves,
		)
	}
}

func TestMoveIdleClients_SkipsRunWhenFetchFails(t *testing.T) {
	f := &fakeClient{listOK: false}

	newMover(t, f).MoveIdleClients(context.Background())

	if len(f.moves) != 0 || len(f.gms) != 0 {
		t.Fatalf(
			"moves=%+v gms=%+v, want no action when the fetch fails",
			f.moves,
			f.gms,
		)
	}
}

func TestMoveIdleClients_NoNotificationWhenMoveFails(t *testing.T) {
	f := &fakeClient{
		listOK: true,
		moveOK: false, // move fails
		sendOK: true,
		listResp: listWith(ts3.ClientEntry{
			ClientType: "0", Clid: "5", Cid: "2",
			ClientIdleTime: "3700000", ClientNickname: "bob",
		}),
	}

	newMover(t, f).MoveIdleClients(context.Background())

	if len(f.moves) != 1 {
		t.Fatalf("moves = %+v, want the move to be attempted", f.moves)
	}
	if len(f.gms) != 0 {
		t.Fatalf("gms = %+v, want no notification when the move fails", f.gms)
	}
}
