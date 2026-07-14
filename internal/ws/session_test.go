package ws

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goserveractorfsm/internal/logx"
	"goserveractorfsm/internal/protocol"
	kickpb "goserveractorfsm/internal/protocol/pb"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

func TestSessionHandlesKickRoundTrip(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, joined, closeRoom := openRunningRoom(t, url)
	defer closeRoom()
	cycle := joined.GetSnapshot().GetActiveCycles()[0]
	waitUntilStand(t, joined.GetSnapshot(), cycle)
	writeEnvelope(t, conn, 42, protocol.KickReqID, &kickpb.KickRequest{
		AttackEpoch: joined.GetAttackEpoch(),
		HammerType:  1,
		KickShrew:   true,
		NumOfShrew:  1,
		ComboId:     5,
		Shrews: []*kickpb.KickShrew{
			{ShrewIndex: cycle.GetHoleIndex(), ProtectType: cycle.GetProtectType(), SpawnSeq: cycle.GetSpawnSeq()},
		},
	})

	envelope := readEnvelope(t, conn)
	if envelope.GetSeqId() != 42 {
		t.Fatalf("Envelope.SeqId = %d, want 42", envelope.GetSeqId())
	}
	if envelope.GetMsgId() != protocol.KickRespID.Uint32() {
		t.Fatalf("Envelope.MsgId = %d, want %d", envelope.GetMsgId(), protocol.KickRespID)
	}

	response := &kickpb.KickResponse{}
	if err := proto.Unmarshal(envelope.GetPayload(), response); err != nil {
		t.Fatalf("Unmarshal KickResponse error = %v", err)
	}
	if response.GetMoney() != 10 {
		t.Fatalf("KickResponse.Money = %d, want 10", response.GetMoney())
	}
	if response.GetComboId() != 5 {
		t.Fatalf("KickResponse.ComboId = %d, want 5", response.GetComboId())
	}
}

func TestSessionHandlesJoinRoomRequest(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}
	defer conn.Close()

	response := joinRoom(t, conn, 7)
	if response.GetPlayerId() == 0 || response.GetAttackId() == 0 || response.GetAttackEpoch() == 0 {
		t.Fatalf("JoinRoomResponse ids = player %d attack %d epoch %d, want non-zero", response.GetPlayerId(), response.GetAttackId(), response.GetAttackEpoch())
	}
	if len(response.GetSnapshot().GetActiveCycles()) != 0 {
		t.Fatalf("snapshot active cycles = %d, want 0 while filling", len(response.GetSnapshot().GetActiveCycles()))
	}
	if response.GetSnapshot().GetMapTimeline().GetNextSwitchMs() != 0 {
		t.Fatalf("filling next_switch_ms = %d, want 0", response.GetSnapshot().GetMapTimeline().GetNextSwitchMs())
	}
}

func TestSessionUsesEnvelopeSeqIDAsOnlyRequestReplyID(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, joined, closeRoom := openRunningRoom(t, url)
	defer closeRoom()
	cycle := joined.GetSnapshot().GetActiveCycles()[0]
	waitUntilStand(t, joined.GetSnapshot(), cycle)
	writeEnvelope(t, conn, 88, protocol.KickReqID, &kickpb.KickRequest{
		AttackEpoch: joined.GetAttackEpoch(),
		HammerType:  1,
		KickShrew:   true,
		NumOfShrew:  1,
		ComboId:     5,
		Shrews: []*kickpb.KickShrew{
			{ShrewIndex: cycle.GetHoleIndex(), ProtectType: cycle.GetProtectType(), SpawnSeq: cycle.GetSpawnSeq()},
		},
	})

	envelope := readEnvelope(t, conn)
	if envelope.GetSeqId() != 88 {
		t.Fatalf("Envelope.SeqId = %d, want 88", envelope.GetSeqId())
	}

	response := &kickpb.KickResponse{}
	if err := proto.Unmarshal(envelope.GetPayload(), response); err != nil {
		t.Fatalf("Unmarshal KickResponse error = %v", err)
	}
	if response.GetMoney() != 10 {
		t.Fatalf("KickResponse.Money = %d, want 10", response.GetMoney())
	}
}

func TestSessionLogsClientOperationAndResult(t *testing.T) {
	var logs bytes.Buffer
	server := httptest.NewServer(NewHandlerWithLogger(logx.New(&logs)))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, joined, closeRoom := openRunningRoom(t, url)
	defer closeRoom()
	cycle := joined.GetSnapshot().GetActiveCycles()[0]
	waitUntilStand(t, joined.GetSnapshot(), cycle)
	writeEnvelope(t, conn, 77, protocol.KickReqID, &kickpb.KickRequest{
		AttackEpoch: joined.GetAttackEpoch(),
		HammerType:  1,
		KickShrew:   true,
		NumOfShrew:  1,
		ComboId:     9,
		Shrews: []*kickpb.KickShrew{
			{ShrewIndex: cycle.GetHoleIndex(), ProtectType: cycle.GetProtectType(), SpawnSeq: cycle.GetSpawnSeq()},
		},
	})
	_ = readEnvelope(t, conn)

	got := logs.String()
	for _, want := range []string{
		"msg=client_connected",
		"msg=message_received",
		"msg_id=2001",
		"msg_name=kick_req",
		"seq_id=77",
		"msg=kick_request",
		"hammer_type=1",
		"shrew_count=1",
		"combo_id=9",
		"msg=kick_response",
		"money=10",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("logs = %q, want %q", got, want)
		}
	}
}

func joinRoom(t *testing.T, conn *websocket.Conn, seqID uint32) *kickpb.JoinRoomResponse {
	t.Helper()
	writeEnvelope(t, conn, seqID, protocol.JoinRoomReqID, &kickpb.JoinRoomRequest{})
	for {
		envelope := readEnvelope(t, conn)
		if envelope.GetSeqId() != seqID || envelope.GetMsgId() != protocol.JoinRoomRespID.Uint32() {
			continue
		}
		response := &kickpb.JoinRoomResponse{}
		if err := proto.Unmarshal(envelope.GetPayload(), response); err != nil {
			t.Fatalf("Unmarshal JoinRoomResponse error = %v", err)
		}
		return response
	}
}

func openRunningRoom(t *testing.T, url string) (*websocket.Conn, *kickpb.JoinRoomResponse, func()) {
	t.Helper()
	connections := make([]*websocket.Conn, 0, 3)
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			for _, opened := range connections {
				_ = opened.Close()
			}
			t.Fatalf("Dial error = %v", err)
		}
		connections = append(connections, conn)
	}

	joined := joinRoom(t, connections[0], 1)
	for index := 1; index < len(connections); index++ {
		joinRoom(t, connections[index], uint32(index+1))
	}
	return connections[0], joined, func() {
		for _, conn := range connections {
			_ = conn.Close()
		}
	}
}

func waitUntilStand(t *testing.T, snapshot *kickpb.GameSnapshot, cycle *kickpb.ShrewCycle) {
	t.Helper()
	delayMS := cycle.GetStandStartMs() - snapshot.GetServerTimeMs() + 20
	if delayMS <= 0 {
		return
	}
	time.Sleep(time.Duration(delayMS) * time.Millisecond)
}

func writeEnvelope(t *testing.T, conn *websocket.Conn, seqID uint32, msgID protocol.MsgID, payload proto.Message) {
	t.Helper()
	payloadData, err := proto.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload error = %v", err)
	}
	wireMessage, err := proto.Marshal(&kickpb.Envelope{
		SeqId:   seqID,
		MsgId:   msgID.Uint32(),
		Payload: payloadData,
	})
	if err != nil {
		t.Fatalf("Marshal Envelope error = %v", err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, wireMessage); err != nil {
		t.Fatalf("WriteMessage error = %v", err)
	}
}

func readEnvelope(t *testing.T, conn *websocket.Conn) *kickpb.Envelope {
	t.Helper()
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error = %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("messageType = %d, want BinaryMessage", messageType)
	}

	envelope := &kickpb.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		t.Fatalf("Unmarshal Envelope error = %v", err)
	}
	return envelope
}
