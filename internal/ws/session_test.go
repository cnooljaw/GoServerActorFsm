package ws

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

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
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}
	defer conn.Close()

	requestPayload, err := proto.Marshal(&kickpb.KickRequest{
		HammerType: 1,
		KickShrew:  true,
		NumOfShrew: 1,
		ComboId:    5,
		Shrews: []*kickpb.KickShrew{
			{ShrewIndex: 2, ProtectType: 0},
		},
	})
	if err != nil {
		t.Fatalf("Marshal KickRequest error = %v", err)
	}

	wireMessage, err := proto.Marshal(&kickpb.Envelope{
		SeqId:   42,
		MsgId:   protocol.KickReqID.Uint32(),
		Payload: requestPayload,
	})
	if err != nil {
		t.Fatalf("Marshal Envelope error = %v", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, wireMessage); err != nil {
		t.Fatalf("WriteMessage error = %v", err)
	}

	messageType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error = %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("messageType = %d, want BinaryMessage", messageType)
	}

	envelope := &kickpb.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		t.Fatalf("Unmarshal response Envelope error = %v", err)
	}
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

func TestSessionUsesEnvelopeSeqIDAsOnlyRequestReplyID(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}
	defer conn.Close()

	requestPayload, err := proto.Marshal(&kickpb.KickRequest{
		HammerType: 1,
		KickShrew:  true,
		NumOfShrew: 1,
		ComboId:    5,
		Shrews: []*kickpb.KickShrew{
			{ShrewIndex: 2, ProtectType: 0},
		},
	})
	if err != nil {
		t.Fatalf("Marshal KickRequest error = %v", err)
	}

	wireMessage, err := proto.Marshal(&kickpb.Envelope{
		SeqId:   88,
		MsgId:   protocol.KickReqID.Uint32(),
		Payload: requestPayload,
	})
	if err != nil {
		t.Fatalf("Marshal Envelope error = %v", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, wireMessage); err != nil {
		t.Fatalf("WriteMessage error = %v", err)
	}

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error = %v", err)
	}

	envelope := &kickpb.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		t.Fatalf("Unmarshal response Envelope error = %v", err)
	}
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
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}
	defer conn.Close()

	requestPayload, err := proto.Marshal(&kickpb.KickRequest{
		HammerType: 1,
		KickShrew:  true,
		NumOfShrew: 1,
		ComboId:    9,
		Shrews: []*kickpb.KickShrew{
			{ShrewIndex: 6, ProtectType: 0},
		},
	})
	if err != nil {
		t.Fatalf("Marshal KickRequest error = %v", err)
	}

	wireMessage, err := proto.Marshal(&kickpb.Envelope{
		SeqId:   77,
		MsgId:   protocol.KickReqID.Uint32(),
		Payload: requestPayload,
	})
	if err != nil {
		t.Fatalf("Marshal Envelope error = %v", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, wireMessage); err != nil {
		t.Fatalf("WriteMessage error = %v", err)
	}
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage error = %v", err)
	}

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
